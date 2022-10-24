// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

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
func (p *planApply) PlanChanges(ctx context.Context, name string, changes []schema.Change, opts ...migrate.PlanOption) (*migrate.Plan, error) {
	s := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name:          name,
			Reversible:    true,
			Transactional: true,
		},
	}
	for _, o := range opts {
		o(&s.PlanOptions)
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
// if the driver is unable to produce a plan to it, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
}

// state represents the state of a planning. It's not part of
// planApply so that multiple planning/applying can be called
// in parallel.
type state struct {
	conn
	migrate.Plan
	migrate.PlanOptions
	skipFKs bool
}

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (s *state) plan(ctx context.Context, changes []schema.Change) (err error) {
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddTable:
			err = s.addTable(ctx, c)
		case *schema.DropTable:
			err = s.dropTable(c)
		case *schema.ModifyTable:
			err = s.modifyTable(ctx, c)
		case *schema.RenameTable:
			s.renameTable(c)
		case *schema.DropIndex:
			s.dropIndexes(c.I.Table, c.I)
		case *schema.DropForeignKey:
			s.dropForeignKeys(c.F.Table, c.F)
		case *schema.DropSchema:
			// skip
			return nil
		default:
			err = fmt.Errorf("unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// addTable builds and executes the query for creating a table in a schema.
func (s *state) addTable(ctx context.Context, add *schema.AddTable) error {
	var (
		errs []string
		b    = Build("CREATE TABLE").Ident(add.T.Name)
	)
	if sqlx.Has(add.Extra, &schema.IfNotExists{}) {
		b.P("IF NOT EXISTS")
	}
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(add.T.Columns, func(i int, b *sqlx.Builder) {
			if err := s.column(b, add.T.Columns[i]); err != nil {
				errs = append(errs, err.Error())
			}
		})
		// Primary keys with auto-increment are inlined on the column definition.
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			s.fks(b, add.T.ForeignKeys...)
		}
		for _, attr := range add.T.Attrs {
			if c, ok := attr.(*schema.Check); ok {
				b.Comma()
				check(b, c)
			}
		}
	})
	if pk := add.T.PrimaryKey; pk != nil {
		b.P("PRIMARY KEY")
		s.indexParts(b, pk.Parts)
	}
	if len(errs) > 0 {
		return fmt.Errorf("create table %q: %s", add.T.Name, strings.Join(errs, ", "))
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  add,
		Reverse: Build("DROP TABLE").Table(add.T).String(),
		Comment: fmt.Sprintf("create %q table", add.T.Name),
	})
	return s.addIndexes(add.T, add.T.Indexes...)
}

// dropTable builds and executes the query for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) error {
	s.skipFKs = true
	b := Build("DROP TABLE").Ident(drop.T.Name)
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		b.P("IF EXISTS")
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
	})
	return nil
}

// modifyTable builds and executes the queries for bringing the table into its modified state.
// If the modification contains changes that are not index creation/deletion or a simple column
// addition, the changes are applied using a temporary table following the procedure mentioned
func (s *state) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	if alterable(modify) {
		return s.alterTable(modify)
	}
	s.skipFKs = true
	newT := *modify.T
	indexes := newT.Indexes
	fks := newT.ForeignKeys
	newT.Indexes = nil
	newT.ForeignKeys = nil
	newT.Name = "new_" + newT.Name

	// Create a new table with a temporary name, and copy the existing rows to it.
	if err := s.addTable(ctx, &schema.AddTable{T: &newT}); err != nil {
		return err
	}
	if err := s.copyRows(modify.T, &newT, modify.Changes); err != nil {
		return err
	}

	// TODO(tmc): either drop or fix-up FKs in the mix here.

	// Drop the current table, create the new one, and copy the rows back.
	s.append(&migrate.Change{
		Cmd:     Build("DROP TABLE").Ident(modify.T.Name).String(),
		Source:  modify,
		Comment: fmt.Sprintf("drop %q table after copying rows", modify.T.Name),
	})
	// walk foreign keys and swap out prefixes.

	newNewT := newT
	newNewT.Name = modify.T.Name
	if err := s.addTable(ctx, &schema.AddTable{T: &newNewT}); err != nil {
		return err
	}
	var reverseChanges []schema.Change
	for _, c := range modify.Changes {
		reverseChanges = append(reverseChanges, c)
	}

	// TODO(tmc): Prepare the reverse changes.
	if err := s.copyRows(&newT, &newNewT, reverseChanges); err != nil {
		return err
	}
	/*
		s.append(&migrate.Change{
			Cmd:     Build("ALTER TABLE").Ident(newT.Name).P("RENAME TO").Ident(modify.T.Name).String(),
			Source:  modify,
			Comment: fmt.Sprintf("rename temporary table %q to %q", newT.Name, modify.T.Name),
		})
	*/
	// TODO: do we need to prefix/re-prefix the indexes?
	if err := s.addIndexes(modify.T, indexes...); err != nil {
		return err
	}
	// TODO: do we need to prefix/re-prefix the fks
	if err := s.addForeignKeys(modify.T, fks...); err != nil {
		return err
	}
	// drop the intermediate table:
	s.append(&migrate.Change{
		Cmd:     Build("DROP TABLE").Ident(newT.Name).String(),
		Source:  modify,
		Comment: fmt.Sprintf("drop temporary table %q", newT.Name),
	})
	return nil
}

func (s *state) renameTable(c *schema.RenameTable) {
	s.append(&migrate.Change{
		Source:  c,
		Comment: fmt.Sprintf("rename a table from %q to %q", c.From.Name, c.To.Name),
		Cmd:     Build("ALTER TABLE").Table(c.From).P("RENAME TO").Table(c.To).String(),
		Reverse: Build("ALTER TABLE").Table(c.To).P("RENAME TO").Table(c.From).String(),
	})
}

func (s *state) column(b *sqlx.Builder, c *schema.Column) error {
	t, err := FormatType(c.Type.Type)
	if err != nil {
		return err
	}
	b.Ident(c.Name).P(t)
	if !c.Type.Null {
		b.P("NOT")
		b.P("NULL")
	}
	if c.Default != nil {
		x, err := defaultValue(c)
		if err != nil {
			return err
		}
		b.P("DEFAULT")
		b.Wrap(func(b *sqlx.Builder) {
			b.P(x)
		})
	}
	return nil
}

func (s *state) dropIndexes(t *schema.Table, indexes ...*schema.Index) error {
	rs := &state{conn: s.conn}
	if err := rs.addIndexes(t, indexes...); err != nil {
		return err
	}
	for i := range rs.Changes {
		s.append(&migrate.Change{
			Cmd:     rs.Changes[i].Reverse,
			Reverse: rs.Changes[i].Cmd,
			Comment: fmt.Sprintf("drop index %q from table: %q", indexes[i].Name, t.Name),
		})
	}
	return nil
}

func (s *state) addIndexes(t *schema.Table, indexes ...*schema.Index) error {
	for _, idx := range indexes {
		b := Build("CREATE")
		if idx.Unique {
			b.P("UNIQUE")
		}
		b.P("INDEX")
		if idx.Name != "" {
			b.Ident(idx.Name)
		}
		b.P("ON").Ident(t.Name)
		s.indexParts(b, idx.Parts)
		if p := (IndexPredicate{}); sqlx.Has(idx.Attrs, &p) {
			b.P("WHERE").P(p.P)
		}
		s.append(&migrate.Change{
			Cmd:     b.String(),
			Source:  &schema.AddIndex{I: idx},
			Reverse: Build("DROP INDEX").Ident(idx.Name).String(),
			Comment: fmt.Sprintf("create index %q to table: %q", idx.Name, t.Name),
		})
	}
	return nil
}

func (s *state) indexParts(b *sqlx.Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(parts, func(i int, b *sqlx.Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(sqlx.MayWrap(part.X.(*schema.RawExpr).X))
			}
			if parts[i].Desc {
				b.P("DESC")
			}
		})
	})
}

func (s *state) dropForeignKeys(t *schema.Table, foreignKeys ...*schema.ForeignKey) error {
	rs := &state{conn: s.conn}
	if err := rs.addForeignKeys(t, foreignKeys...); err != nil {
		return err
	}
	for i := range rs.Changes {
		s.append(&migrate.Change{
			Cmd:     rs.Changes[i].Reverse,
			Reverse: rs.Changes[i].Cmd,
			Comment: fmt.Sprintf("drop foreignKey %q from table: %q", foreignKeys[i].Symbol, t.Name),
		})
	}
	return nil
}

func (s *state) addForeignKeys(t *schema.Table, foreignKeys ...*schema.ForeignKey) error {
	for _, fk := range foreignKeys {
		b := Build("ALTER TABLE")
		b.Ident(t.Name)
		b.P("ADD")
		s.fks(b, foreignKeys...)
		fkName := fk.Symbol
		// TODO: derive fk name if we don't have one in Symbol.
		s.append(&migrate.Change{
			Cmd:    b.String(),
			Source: &schema.AddForeignKey{F: fk},
			Reverse: Build("ALTER TABLE").
				Ident(t.Name).
				P(fmt.Sprintf("DROP CONSTRAINT %v", fkName)).
				String(),
			Comment: fmt.Sprintf("create foreignKey %q to table: %q", fk.Symbol, t.Name),
		})
	}
	return nil
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
		if fk.RefTable != nil {
			b.P("REFERENCES").Ident(fk.RefTable.Name)
			b.Wrap(func(b *sqlx.Builder) {
				b.MapComma(fk.RefColumns, func(i int, b *sqlx.Builder) {
					b.Ident(fk.RefColumns[i].Name)
				})
			})
		}
	})
}

func (s *state) copyRows(from *schema.Table, to *schema.Table, changes []schema.Change) error {
	var (
		args       []any
		fromC, toC []string
	)
	for _, column := range to.Columns {
		// Skip generated columns in INSERT as they are computed.
		if sqlx.Has(column.Attrs, &schema.GeneratedExpr{}) {
			continue
		}
		// Find a change that associated with this column, if exists.
		var change schema.Change
		for i := range changes {
			switch c := changes[i].(type) {
			case *schema.AddColumn:
				if c.C.Name != column.Name {
					break
				}
				if change != nil {
					return fmt.Errorf("duplicate changes for column: %q: %T, %T", column.Name, change, c)
				}
				change = changes[i]
			case *schema.ModifyColumn:
				if c.To.Name != column.Name {
					break
				}
				if change != nil {
					return fmt.Errorf("duplicate changes for column: %q: %T, %T", column.Name, change, c)
				}
				change = changes[i]
			case *schema.DropColumn:
				if c.C.Name == column.Name {
					return fmt.Errorf("unexpected drop column: %q", column.Name)
				}
			}
		}
		switch change := change.(type) {
		// We expect that new columns are added with DEFAULT/GENERATED
		// values or defined as nullable if the table is not empty.
		case *schema.AddColumn:
		// Column modification requires special handling if it was
		// converted from nullable to non-nullable with default value.
		case *schema.ModifyColumn:
			toC = append(toC, column.Name)
			if !column.Type.Null && column.Default != nil && change.Change.Is(schema.ChangeNull|schema.ChangeDefault) {
				fromC = append(fromC, fmt.Sprintf("IFNULL(`%[1]s`, ?) AS `%[1]s`", column.Name))
				x, err := defaultValue(column)
				if err != nil {
					return err
				}
				args = append(args, x)
			} else {
				fromC = append(fromC, column.Name)
			}
		// Columns without changes should be transferred as-is.
		case nil:
			toC = append(toC, column.Name)
			fromC = append(fromC, column.Name)
		}
	}
	stmt := fmt.Sprintf("INSERT INTO `%s` (%s) SELECT %s FROM `%s`", to.Name, identComma(toC), identComma(fromC), from.Name)
	s.append(&migrate.Change{
		Cmd:     stmt,
		Args:    args,
		Comment: fmt.Sprintf("copy rows from old table %q to new temporary table %q", from.Name, to.Name),
	})
	return nil
}

// alterTable alters the table with the given changes. Assuming the changes are "alterable".
func (s *state) alterTable(modify *schema.ModifyTable) error {
	for _, change := range modify.Changes {
		switch change := change.(type) {
		case *schema.AddIndex:
			if err := s.addIndexes(modify.T, change.I); err != nil {
				return err
			}
		case *schema.DropIndex:
			if err := s.dropIndexes(modify.T, change.I); err != nil {
				return err
			}
		case *schema.RenameIndex:
			if err := s.addIndexes(modify.T, change.To); err != nil {
				return err
			}
			if err := s.dropIndexes(modify.T, change.From); err != nil {
				return err
			}
		case *schema.AddColumn:
			b := Build("ALTER TABLE").Ident(modify.T.Name)
			r := b.Clone()
			if err := s.column(b.P("ADD COLUMN"), change.C); err != nil {
				return err
			}
			s.append(&migrate.Change{
				Source:  change,
				Cmd:     b.String(),
				Reverse: r.P("DROP COLUMN").Ident(change.C.Name).String(),
				Comment: fmt.Sprintf("add column %q to table: %q", change.C.Name, modify.T.Name),
			})
		case *schema.RenameColumn:
			b := Build("ALTER TABLE").Ident(modify.T.Name).P("RENAME COLUMN")
			r := b.Clone()
			s.append(&migrate.Change{
				Source:  change,
				Cmd:     b.Ident(change.From.Name).P("TO").Ident(change.To.Name).String(),
				Reverse: r.Ident(change.To.Name).P("TO").Ident(change.From.Name).String(),
				Comment: fmt.Sprintf("rename a column from %q to %q", change.From.Name, change.To.Name),
			})
		default:
			return fmt.Errorf("unexpected change in alter table: %T", change)
		}
	}
	return nil
}

func (s *state) append(c *migrate.Change) {
	s.Changes = append(s.Changes, c)
}

func alterable(modify *schema.ModifyTable) bool {
	for _, change := range modify.Changes {
		switch change := change.(type) {
		case *schema.RenameColumn, *schema.RenameIndex, *schema.DropIndex, *schema.AddIndex:
		case *schema.AddColumn:
			if len(change.C.Indexes) > 0 || len(change.C.ForeignKeys) > 0 || change.C.Default != nil {
				return false
			}
			// TODO(tmc): check if this is true for spanner
			if x := (schema.GeneratedExpr{}); sqlx.Has(change.C.Attrs, &x) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// checks writes the CHECK constraint to the builder.
func check(b *sqlx.Builder, c *schema.Check) {
	expr := c.Expr
	// Expressions should be wrapped with parens.
	if t := strings.TrimSpace(expr); !strings.HasPrefix(t, "(") || !strings.HasSuffix(t, ")") {
		expr = "(" + t + ")"
	}
	if c.Name != "" {
		b.P("CONSTRAINT").Ident(c.Name)
	}
	b.P("CHECK", expr)
}

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '`'}
	return b.P(phrase)
}

func defaultValue(c *schema.Column) (string, error) {
	switch x := c.Default.(type) {
	case *schema.Literal:
		return sqlx.SingleQuote(x.V)
	case *schema.RawExpr:
		return x.X, nil
	default:
		return "", fmt.Errorf("unexpected default value type: %T", x)
	}
}

func identComma(c []string) string {
	b := &sqlx.Builder{QuoteChar: '`'}
	b.MapComma(c, func(i int, b *sqlx.Builder) {
		if strings.ContainsRune(c[i], '`') {
			b.WriteString(c[i])
		} else {
			b.Ident(c[i])
		}
	})
	return b.String()
}
