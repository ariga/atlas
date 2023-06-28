// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DefaultPlan provides basic planning capabilities for MS-SQL dialects.
// Note, it is recommended to call Open, create a new Driver and use its
// migrate.PlanApplier when a database connection is available.
var DefaultPlan migrate.PlanApplier = &planApply{conn: conn{ExecQuerier: sqlx.NoRows}}

// A planApply provides migration capabilities for schema elements.
type planApply struct{ conn }

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to do so, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
}

// PlanChanges returns a migration plan for the given schema changes.
func (p *planApply) PlanChanges(ctx context.Context, name string, changes []schema.Change, opts ...migrate.PlanOption) (*migrate.Plan, error) {
	s := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name:          name,
			Transactional: true,
		},
	}
	for _, o := range opts {
		o(&s.PlanOptions)
	}
	if err := s.plan(ctx, changes); err != nil {
		return nil, err
	}
	if err := sqlx.SetReversible(&s.Plan); err != nil {
		return nil, err
	}
	return &s.Plan, nil
}

// state represents the state of a planning. It is not part of
// planApply so that multiple planning/applying can be called
// in parallel.
type state struct {
	conn
	migrate.Plan
	migrate.PlanOptions
}

// Build instantiates a new builder and writes the given phrase to it.
func (s *state) Build(phrases ...string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '"', Schema: s.SchemaQualifier, Indent: s.Indent}
	return b.P(phrases...)
}

// plan builds the migration plan for applying the
// given changes on the attached connection.
func (s *state) plan(ctx context.Context, changes []schema.Change) error {
	if s.SchemaQualifier != nil {
		if err := sqlx.CheckChangesScope(s.PlanOptions, changes); err != nil {
			return err
		}
	}
	planned := s.topLevel(changes)
	planned, err := sqlx.DetachCycles(planned)
	if err != nil {
		return err
	}
	for _, c := range planned {
		switch c := c.(type) {
		case *schema.RenameTable:
			s.renameTable(c)
		case *schema.AddTable:
			err = s.addTable(ctx, c)
		case *schema.DropTable:
			err = s.dropTable(ctx, c)
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

func (s *state) topLevel(changes []schema.Change) []schema.Change {
	planned := make([]schema.Change, 0, len(changes))
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddSchema:
			b := s.Build("CREATE SCHEMA")
			b.Ident(c.S.Name)
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Reverse: s.Build("DROP SCHEMA").Ident(c.S.Name).String(),
				Comment: fmt.Sprintf("Add new schema named %q", c.S.Name),
			})
		case *schema.DropSchema:
			b := s.Build("DROP SCHEMA")
			if sqlx.Has(c.Extra, &schema.IfExists{}) {
				b.P("IF EXISTS")
			}
			b.Ident(c.S.Name)
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

func (s *state) addTable(_ context.Context, add *schema.AddTable) error {
	var (
		errs []string
		b    = s.Build("CREATE TABLE")
	)
	b.Table(add.T)
	b.WrapIndent(func(b *sqlx.Builder) {
		b.MapIndent(add.T.Columns, func(i int, b *sqlx.Builder) {
			if err := s.column(b, add.T, add.T.Columns[i]); err != nil {
				errs = append(errs, err.Error())
			}
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().NL().P("PRIMARY KEY")
			// if err := s.index(b, pk); err != nil {
			// 	errs = append(errs, err.Error())
			// }
		}
		// if len(add.T.ForeignKeys) > 0 {
		// 	b.Comma()
		// 	s.fks(b, add.T.ForeignKeys...)
		// }
		// for _, attr := range add.T.Attrs {
		// 	if c, ok := attr.(*schema.Check); ok {
		// 		b.Comma().NL()
		// 		check(b, c)
		// 	}
		// }
	})
	// if p := (Partition{}); sqlx.Has(add.T.Attrs, &p) {
	// 	s, err := formatPartition(p)
	// 	if err != nil {
	// 		errs = append(errs, err.Error())
	// 	}
	// 	b.P(s)
	// }
	if len(errs) > 0 {
		return fmt.Errorf("create table %q: %s", add.T.Name, strings.Join(errs, ", "))
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  add,
		Comment: fmt.Sprintf("create %q table", add.T.Name),
		Reverse: s.Build("DROP TABLE").Table(add.T).String(),
	})
	// for _, idx := range add.T.Indexes {
	// 	// Indexes do not need to be created concurrently on new tables.
	// 	if err := s.addIndexes(add.T, &schema.AddIndex{I: idx}); err != nil {
	// 		return err
	// 	}
	// }
	// s.addComments(add.T)
	return nil
}

func (s *state) dropTable(_ context.Context, _ *schema.DropTable) error {
	return nil
}

func (s *state) modifyTable(_ context.Context, _ *schema.ModifyTable) error {
	return nil
}

func (s *state) renameTable(c *schema.RenameTable) {
	ren := func(old, new *schema.Table) string {
		b := s.Build("EXEC sp_rename")
		b.CommaQuote('\'',
			func() { b.Table(old) },
			func() { b.Ident(new.Name) },
		)
		return b.String()
	}
	s.append(&migrate.Change{
		Source:  c,
		Comment: fmt.Sprintf("rename a table from %q to %q", c.From.Name, c.To.Name),
		Cmd:     ren(c.From, c.To),
		Reverse: ren(c.To, c.From),
	})
}

func (s *state) column(b *sqlx.Builder, t *schema.Table, c *schema.Column) error {
	var (
		computed = &schema.GeneratedExpr{}
		id, hasI = identity(c.Attrs)
	)
	switch hasX := sqlx.Has(c.Attrs, computed); {
	case hasX && hasI:
		return fmt.Errorf("both identity and computed expression specified for column %q", c.Name)
	case hasX:
		b.Ident(c.Name).P("AS", sqlx.MayWrap(computed.Expr), computed.Type)
		if !c.Type.Null {
			b.P("NOT NULL")
		}
	default:
		f, err := s.formatType(t, c)
		if err != nil {
			return err
		}
		b.Ident(c.Name).P(f)
		if !c.Type.Null {
			b.P("NOT")
		}
		b.P("NULL")
		s.columnDefault(b, t, c)
		for _, attr := range c.Attrs {
			switch a := attr.(type) {
			case *schema.Collation:
				b.P("COLLATE").Ident(a.V)
			case *schema.Comment:
			case *schema.GeneratedExpr, *Identity:
				// Handled below.
			default:
				return fmt.Errorf("unexpected column attribute: %T", attr)
			}
		}
		if hasI {
			b.P("IDENTITY").Wrap(func(b *sqlx.Builder) {
				b.P(strconv.FormatInt(id.Seed, 10)).Comma()
				b.P(strconv.FormatInt(id.Increment, 10))
			})
		}
	}
	return nil
}

// columnDefault writes the default value of column to the builder.
func (s *state) columnDefault(b *sqlx.Builder, t *schema.Table, c *schema.Column) {
	if c.Default == nil {
		return
	}
	b.P("CONSTRAINT").Ident(fmt.Sprintf("DEFAULT_%s_%s", t.Name, c.Name))
	switch x := c.Default.(type) {
	case *schema.Literal:
		b.P("DEFAULT", x.V)
	case *schema.RawExpr:
		b.P("DEFAULT", x.X)
	}
}

// formatType formats the type but takes into account the qualifier.
func (s *state) formatType(_ *schema.Table, c *schema.Column) (string, error) {
	return FormatType(c.Type.Type)
}

func (s *state) append(c *migrate.Change) {
	s.Changes = append(s.Changes, c)
}
