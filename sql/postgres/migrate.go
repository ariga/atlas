// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// A planApply provides migration capabilities for schema elements.
type planApply struct{ conn }

// PlanChanges returns a migration plan for the given schema changes.
func (p *planApply) PlanChanges(ctx context.Context, name string, changes []schema.Change) (*migrate.Plan, error) {
	s := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name:          name,
			Reversible:    true,
			Transactional: true,
		},
	}
	if err := s.plan(ctx, changes); err != nil {
		return nil, err
	}
	for _, c := range s.Changes {
		if c.Reverse == "" {
			s.Reversible = false
		}
	}
	return &s.Plan, nil
}

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to do so, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change) error {
	plan, err := p.PlanChanges(ctx, "apply", changes)
	if err != nil {
		return err
	}
	for _, c := range plan.Changes {
		if _, err := p.ExecContext(ctx, c.Cmd); err != nil {
			return err
		}
	}
	return nil
}

// state represents the state of a planning. It is not part of
// planApply so that multiple planning/applying can be called
// in parallel.
type state struct {
	conn
	migrate.Plan
}

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (s *state) plan(ctx context.Context, changes []schema.Change) error {
	planned := s.topLevel(changes)
	planned, err := sqlx.DetachCycles(planned)
	if err != nil {
		return err
	}
	for _, c := range planned {
		switch c := c.(type) {
		case *schema.AddTable:
			err = s.addTable(ctx, c)
		case *schema.DropTable:
			s.dropTable(c)
		case *schema.ModifyTable:
			err = s.modifyTable(ctx, c)
		default:
			err = fmt.Errorf("unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// topLevel executes first the changes for creating or dropping schemas (top-level schema elements).
func (s *state) topLevel(changes []schema.Change) []schema.Change {
	planned := make([]schema.Change, 0, len(changes))
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddSchema:
			b := Build("CREATE SCHEMA").Ident(c.S.Name)
			if sqlx.Has(c.Extra, &schema.IfNotExists{}) {
				b.P("IF NOT EXISTS")
			}
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Reverse: Build("DROP SCHEMA").Ident(c.S.Name).String(),
				Comment: fmt.Sprintf("Add new schema named %q", c.S.Name),
			})
		case *schema.DropSchema:
			b := Build("DROP SCHEMA").Ident(c.S.Name)
			if sqlx.Has(c.Extra, &schema.IfExists{}) {
				b.P("IF NOT EXISTS")
			}
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Comment: fmt.Sprintf("Drop schema named %q", c.S.Name),
			})
		default:
			planned = append(planned, c)
		}
	}
	return planned
}

// addTable builds and executes the query for creating a table in a schema.
func (s *state) addTable(ctx context.Context, add *schema.AddTable) error {
	// Create enum types before using them in the `CREATE TABLE` statement.
	if err := s.addTypes(ctx, add.T.Columns...); err != nil {
		return err
	}
	b := Build("CREATE TABLE").Table(add.T)
	if sqlx.Has(add.Extra, &schema.IfNotExists{}) {
		b.P("IF NOT EXISTS")
	}
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(add.T.Columns, func(i int, b *sqlx.Builder) {
			s.column(b, add.T.Columns[i])
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().P("PRIMARY KEY")
			s.indexParts(b, pk.Parts)
		}
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			s.fks(b, add.T.ForeignKeys...)
		}
	})
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  add,
		Comment: fmt.Sprintf("create %q table", add.T.Name),
		Reverse: Build("DROP TABLE").Table(add.T).String(),
	})
	s.addIndexes(add.T, add.T.Indexes...)
	s.addComments(add.T)
	return nil
}

// dropTable builds and executes the query for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) {
	b := Build("DROP TABLE").Table(drop.T)
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		b.P("IF EXISTS")
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
	})
}

// modifyTable builds and executes the queries for bringing the table into its modified state.
func (s *state) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	var (
		changes     []schema.Change
		addI, dropI []*schema.Index
	)
	for _, change := range skipAutoChanges(modify.Changes) {
		switch change := change.(type) {
		case *schema.DropAttr:
			return fmt.Errorf("unsupported change type: %T", change)
		case *schema.AddIndex:
			addI = append(addI, change.I)
		case *schema.DropIndex:
			dropI = append(dropI, change.I)
		case *schema.ModifyIndex:
			// Index modification requires rebuilding the index.
			addI = append(addI, change.To)
			dropI = append(dropI, change.From)
		case *schema.ModifyForeignKey:
			// Foreign-key modification is translated into 2 steps.
			// Dropping the current foreign key and creating a new one.
			changes = append(changes, &schema.DropForeignKey{
				F: change.From,
			}, &schema.AddForeignKey{
				F: change.To,
			})
		case *schema.AddColumn:
			if err := s.addTypes(ctx, change.C); err != nil {
				return err
			}
			changes = append(changes, change)
		case *schema.ModifyColumn:
			from, ok1 := change.From.Type.Type.(*schema.EnumType)
			to, ok2 := change.To.Type.Type.(*schema.EnumType)
			switch {
			// Enum was added.
			case !ok1 && ok2:
				if err := s.addTypes(ctx, change.To); err != nil {
					return err
				}
			// Enum was changed.
			case ok1 && ok2 && from.T == to.T:
				if err := s.alterType(from, to); err != nil {
					return err
				}
			// Not an enum, or was dropped.
			default:
				changes = append(changes, change)
			}
		default:
			changes = append(changes, change)
		}
	}
	s.dropIndexes(modify.T, dropI...)
	if len(changes) > 0 {
		s.alterTable(modify.T, changes)
	}
	s.addIndexes(modify.T, addI...)
	return nil
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (s *state) alterTable(t *schema.Table, changes []schema.Change) {
	var (
		b          = Build("ALTER TABLE").Table(t)
		reversible = true
		reverse    = b.Clone()
	)
	b.MapComma(changes, func(i int, b *sqlx.Builder) {
		switch change := changes[i].(type) {
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			s.column(b, change.C)
			reverse.P("DROP COLUMN").Ident(change.C.Name)
		case *schema.DropColumn:
			b.P("DROP COLUMN").Ident(change.C.Name)
			reversible = false
		case *schema.ModifyColumn:
			s.alterColumn(b, change.Change, change.To)
			s.alterColumn(reverse, change.Change, change.From)
		case *schema.AddForeignKey:
			b.P("ADD")
			s.fks(b, change.F)
			reverse.P("DROP CONSTRAINT").Ident(change.F.Symbol)
		case *schema.DropForeignKey:
			b.P("DROP CONSTRAINT").Ident(change.F.Symbol)
			reversible = false
		}
	})
	change := &migrate.Change{
		Cmd: b.String(),
		Source: &schema.ModifyTable{
			T:       t,
			Changes: changes,
		},
		Comment: fmt.Sprintf("Modify %q table", t.Name),
	}
	if reversible {
		change.Reverse = reverse.String()
	}
	s.append(change)
}

func (s *state) addComments(t *schema.Table) {
	var c schema.Comment
	if sqlx.Has(t.Attrs, &c) {
		b := Build("COMMENT ON TABLE").Table(t)
		s.append(&migrate.Change{
			Cmd:     b.Clone().P("IS", quote(c.Text)).String(),
			Comment: fmt.Sprintf("add comment to table: %q", t.Name),
			Reverse: b.Clone().P("IS NULL").String(),
		})
	}
	for i := range t.Columns {
		if sqlx.Has(t.Columns[i].Attrs, &c) {
			b := Build("COMMENT ON COLUMN").Table(t)
			b.WriteByte('.')
			b.Ident(t.Columns[i].Name)
			s.append(&migrate.Change{
				Cmd:     b.Clone().P("IS", quote(c.Text)).String(),
				Comment: fmt.Sprintf("add comment to column: %q on table: %q", t.Columns[i].Name, t.Name),
				Reverse: b.Clone().P("IS NULL").String(),
			})
		}
	}
	for i := range t.Indexes {
		if sqlx.Has(t.Indexes[i].Attrs, &c) {
			b := Build("COMMENT ON INDEX").Ident(t.Indexes[i].Name).P("IS", quote(c.Text))
			s.append(&migrate.Change{
				Cmd:     b.Clone().P("IS", quote(c.Text)).String(),
				Comment: fmt.Sprintf("add comment to index: %q on table: %q", t.Indexes[i].Name, t.Name),
				Reverse: b.Clone().P("IS NULL").String(),
			})
		}
	}
}

func (s *state) dropIndexes(t *schema.Table, indexes ...*schema.Index) {
	for _, idx := range indexes {
		s.append(&migrate.Change{
			Cmd:     Build("DROP INDEX").Ident(idx.Name).String(),
			Comment: fmt.Sprintf("Drop index %q to table: %q", idx.Name, t.Name),
		})
	}
}

func (s *state) addTypes(ctx context.Context, columns ...*schema.Column) error {
	for _, c := range columns {
		e, ok := c.Type.Type.(*schema.EnumType)
		if !ok {
			continue
		}
		if e.T == "" {
			return fmt.Errorf("missing enum name for column %q", c.Name)
		}
		c.Type.Raw = e.T
		if exists, err := s.enumExists(ctx, e.T); err != nil {
			return err
		} else if exists {
			continue
		}
		b := Build("CREATE TYPE").Ident(e.T).P("AS ENUM")
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(e.Values, func(i int, b *sqlx.Builder) {
				b.WriteString("'" + e.Values[i] + "'")
			})
		})
		s.append(&migrate.Change{
			Cmd:     b.String(),
			Comment: fmt.Sprintf("create enum type %q", e.T),
			Reverse: Build("DROP TYPE").Ident(e.T).String(),
		})
	}
	return nil
}

func (s *state) alterType(from, to *schema.EnumType) error {
	if len(from.Values) > len(to.Values) {
		return fmt.Errorf("dropping enum (%q) value is not supported", from.T)
	}
	for i := range from.Values {
		if from.Values[i] != to.Values[i] {
			return fmt.Errorf("replacing or reordering enum (%q) value is not supported: %q != %q", to.T, to.Values, from.Values)
		}
	}
	for _, v := range to.Values[len(from.Values):] {
		s.append(&migrate.Change{
			Cmd:     Build("ALTER TYPE").Ident(from.T).P("ADD VALUE", quote(v)).String(),
			Comment: fmt.Sprintf("add value to enum type: %q", from.T),
		})
	}
	return nil
}

func (s *state) enumExists(ctx context.Context, name string) (bool, error) {
	rows, err := s.QueryContext(ctx, "SELECT * FROM pg_type WHERE typname = $1 AND typtype = 'e'", name)
	if err != nil {
		return false, fmt.Errorf("check index existance: %w", err)
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}

func (s *state) addIndexes(t *schema.Table, indexes ...*schema.Index) {
	for _, idx := range indexes {
		b := Build("CREATE")
		if idx.Unique {
			b.P("UNIQUE")
		}
		b.P("INDEX")
		if idx.Name != "" {
			b.Ident(idx.Name)
		}
		b.P("ON").Table(t)
		s.indexParts(b, idx.Parts)
		s.indexAttrs(b, idx.Attrs)
		s.append(&migrate.Change{
			Cmd:     b.String(),
			Reverse: Build("DROP INDEX").Ident(idx.Name).String(),
			Comment: fmt.Sprintf("Create index %q to table: %q", idx.Name, t.Name),
		})
	}
}

func (s *state) column(b *sqlx.Builder, c *schema.Column) {
	b.Ident(c.Name).P(mustFormat(c.Type.Type))
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	if x, ok := c.Default.(*schema.RawExpr); ok {
		b.P("DEFAULT", x.X)
	}
	for _, attr := range c.Attrs {
		switch attr := attr.(type) {
		case *schema.Comment:
		case *schema.Collation:
			b.P("COLLATE").Ident(attr.V)
		case *Identity:
			if attr.Generation == "" {
				attr.Generation = "BY DEFAULT"
			}
			b.P("GENERATED", attr.Generation, "AS IDENTITY")
		default:
			panic(fmt.Sprintf("unexpected column attribute: %T", attr))
		}
	}
}

func (s *state) alterColumn(b *sqlx.Builder, k schema.ChangeKind, c *schema.Column) {
	for !k.Is(schema.NoChange) {
		b.P("ALTER COLUMN").Ident(c.Name)
		switch {
		case k.Is(schema.ChangeType):
			b.P("TYPE").P(mustFormat(c.Type.Type))
			if collate := (schema.Collation{}); sqlx.Has(c.Attrs, &collate) {
				b.P("COLLATE", collate.V)
			}
			k &= ^schema.ChangeType
		case k.Is(schema.ChangeNull) && c.Type.Null:
			b.P("DROP NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeNull) && !c.Type.Null:
			b.P("SET NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeDefault) && c.Default == nil:
			b.P("DROP DEFAULT")
			k &= ^schema.ChangeDefault
		case k.Is(schema.ChangeDefault) && c.Default != nil:
			x, ok := c.Default.(*schema.RawExpr)
			if !ok {
				panic(fmt.Sprintf("unexpected column default: %T", c.Default))
			}
			b.P("SET DEFAULT", x.X)
			k &= ^schema.ChangeDefault
		default:
			panic(fmt.Sprintf("unexpected column change: %d", k))
		}
		if !k.Is(schema.NoChange) {
			b.Comma()
		}
	}
}

func (s *state) indexParts(b *sqlx.Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(parts, func(i int, b *sqlx.Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
			for _, attr := range parts[i].Attrs {
				s.partAttr(b, attr)
			}
		})
	})
}

func (s *state) partAttr(b *sqlx.Builder, attr schema.Attr) {
	switch attr := attr.(type) {
	case *IndexColumnProperty:
		switch {
		case attr.Desc && attr.NullsLast:
			b.P("DESC NULL LAST")
		case attr.Desc:
			// Rows in descending order are stored
			// with nulls first by default.
			b.P("DESC")
		case attr.Asc && attr.NullsFirst:
			b.P("NULL FIRST")
		case attr.Asc && attr.NullsLast:
			// Do nothing, since B-tree indexes store
			// rows in ascending order with nulls last.
		}
	case *schema.Collation:
		b.P("COLLATE").Ident(attr.V)
	default:
		panic(fmt.Sprintf("unexpected index part attribute: %T", attr))
	}
}

func (s *state) indexAttrs(b *sqlx.Builder, attrs []schema.Attr) {
	// Avoid appending the default method.
	if t := (IndexType{}); sqlx.Has(attrs, &t) && strings.ToLower(t.T) != "btree" {
		b.P("USING").P(t.T)
	}
	if p := (IndexPredicate{}); sqlx.Has(attrs, &p) {
		b.P("WHERE").P(p.P)
	}
	for _, attr := range attrs {
		switch attr.(type) {
		case *schema.Comment, *ConType, *IndexType, *IndexPredicate:
		default:
			panic(fmt.Sprintf("unexpected index attribute: %T", attr))
		}
	}
}

func (s *state) fks(b *sqlx.Builder, fks ...*schema.ForeignKey) {
	b.MapComma(fks, func(i int, b *sqlx.Builder) {
		fk := fks[i]
		if fk.Symbol != "" {
			b.P("CONSTRAINT").Ident(fk.Symbol)
		}
		b.P("FOREIGN KEY")
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(fk.Columns, func(i int, b *sqlx.Builder) {
				b.Ident(fk.Columns[i].Name)
			})
		})
		b.P("REFERENCES").Table(fk.RefTable)
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(fk.RefColumns, func(i int, b *sqlx.Builder) {
				b.Ident(fk.RefColumns[i].Name)
			})
		})
		if fk.OnUpdate != "" {
			b.P("ON UPDATE", string(fk.OnUpdate))
		}
		if fk.OnDelete != "" {
			b.P("ON DELETE", string(fk.OnDelete))
		}
	})
}

func (s *state) append(c *migrate.Change) {
	s.Changes = append(s.Changes, c)
}

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '"'}
	return b.P(phrase)
}

// skipAutoChanges filters unnecessary changes that are automatically
// happened by the database when ALTER TABLE is executed.
func skipAutoChanges(changes []schema.Change) []schema.Change {
	dropC := make(map[string]bool)
	for _, c := range changes {
		if c, ok := c.(*schema.DropColumn); ok {
			dropC[c.C.Name] = true
		}
	}
	for i, c := range changes {
		switch c := c.(type) {
		// Indexes involving the column are automatically dropped
		// with it. This true for multi-columns indexes as well.
		// See https://www.postgresql.org/docs/current/sql-altertable.html
		case *schema.DropIndex:
			for _, p := range c.I.Parts {
				if p.C == nil && dropC[p.C.Name] {
					changes = append(changes[:i], changes[i+1:]...)
					break
				}
			}
		// Simple case for skipping constraint dropping,
		// if the child table columns were dropped.
		case *schema.DropForeignKey:
			for _, c := range c.F.Columns {
				if dropC[c.Name] {
					changes = append(changes[:i], changes[i+1:]...)
					break
				}
			}
		}
	}
	return changes
}

func quote(s string) string {
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return s
	}
	return "'" + s + "'"
}
