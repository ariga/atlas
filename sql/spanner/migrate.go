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
	planned, err := sqlx.DetachCycles(changes)
	if err != nil {
		return err
	}
	for _, c := range planned {
		switch c := c.(type) {
		case *schema.AddTable:
			err = s.addTable(ctx, c)
		case *schema.DropTable:
			err = s.dropTable(c)
		case *schema.ModifyTable:
			err = s.modifyTable(ctx, c)
		case *schema.DropIndex:
			s.dropIndexes(c.I.Table, c.I)
		case *schema.DropForeignKey:
			s.dropForeignKeys(c.F.Table, c.F)
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
		b    = s.Build("CREATE TABLE").Ident(add.T.Name)
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
		Reverse: s.Build("DROP TABLE").Table(add.T).String(),
		Comment: fmt.Sprintf("create %q table", add.T.Name),
	})
	return s.addIndexes(add.T, add.T.Indexes...)
}

// dropTable builds and executes the query for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) error {
	s.skipFKs = true
	b := s.Build("DROP TABLE").Ident(drop.T.Name)
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
func (s *state) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	if len(modify.Changes) > 0 {
		if err := s.alterTable(modify.T, modify.Changes); err != nil {
			return err
		}
	}
	return nil
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (s *state) alterTable(t *schema.Table, changes []schema.Change) error {
	build := func(change schema.Change) (string, error) {
		b := s.Build("ALTER TABLE").Table(t)
		switch change := change.(type) {
		default:
			return "", fmt.Errorf("unsupported change type %T", change)
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			if err := s.column(b, change.C); err != nil {
				return "", err
			}
		}
		return b.String(), nil
	}
	for _, change := range changes {
		cmd, err := build(change)
		if err != nil {
			return fmt.Errorf("alter table %q: %v", t.Name, err)
		}
		change := &migrate.Change{
			Cmd: cmd,
			Source: &schema.ModifyTable{
				T:       t,
				Changes: changes,
			},
			Comment: fmt.Sprintf("modify %q table", t.Name),
		}
		s.append(change)
	}
	return nil
}

func (s *state) column(b *sqlx.Builder, c *schema.Column) error {
	t, err := FormatType(c.Type.Type)
	if err != nil {
		return err
	}
	b.Ident(c.Name).P(t)
	// TODO: respect spanner semantics for when NOT NULL can be specified.
	if !c.Type.Null {
		b.P("NOT")
		b.P("NULL")
	}
	if c.Default != nil {
		x, ok := sqlx.DefaultValue(c)
		if ok {
			b.P("DEFAULT")
			b.Wrap(func(b *sqlx.Builder) {
				b.P(x)
			})
		}
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
		b := s.Build("CREATE")
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
			Reverse: s.Build("DROP INDEX").Ident(idx.Name).String(),
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
		b := s.Build("ALTER TABLE")
		b.Ident(t.Name)
		b.P("ADD")
		s.fks(b, foreignKeys...)
		fkName := fk.Symbol
		// TODO: derive fk name if we don't have one in Symbol.
		s.append(&migrate.Change{
			Cmd:    b.String(),
			Source: &schema.AddForeignKey{F: fk},
			Reverse: s.Build("ALTER TABLE").
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

func (s *state) append(c *migrate.Change) {
	s.Changes = append(s.Changes, c)
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
func (s *state) Build(phrases ...string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '`', Schema: sqlx.P("")}
	return b.P(phrases...)
}
