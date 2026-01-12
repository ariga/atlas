// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DefaultPlan provides basic planning capabilities for YDB dialect.
// Note, it is recommended to call Open, create a new Driver and use its
// migrate.PlanApplier when a database connection is available.
var DefaultPlan migrate.PlanApplier = &planApply{conn: &conn{ExecQuerier: sqlx.NoRows}}

// A planApply provides migration capabilities for schema elements.
type planApply struct{ *conn }

// PlanChanges returns a migration plan for the given schema changes.
func (p *planApply) PlanChanges(
	_ context.Context,
	name string,
	changes []schema.Change,
	opts ...migrate.PlanOption,
) (*migrate.Plan, error) {
	state := &state{
		conn: p.conn,
		Plan: migrate.Plan{
			Name:          name,
			Transactional: true,
		},
	}
	for _, opt := range opts {
		opt(&state.PlanOptions)
	}
	if err := state.plan(changes); err != nil {
		return nil, err
	}
	if err := sqlx.SetReversible(&state.Plan); err != nil {
		return nil, err
	}
	return &state.Plan, nil
}

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to do so, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(
	ctx context.Context,
	changes []schema.Change,
	opts ...migrate.PlanOption,
) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
}

// state represents the state of a planning. It is not part of
// planApply so that multiple planning/applying can be called
// in parallel.
type state struct {
	*conn
	migrate.Plan
	migrate.PlanOptions
}

// plan processes the changes and generates migration statements.
func (s *state) plan(changes []schema.Change) error {
	for _, change := range changes {
		switch change := change.(type) {
		case *schema.AddTable:
			if err := s.addTable(change); err != nil {
				return err
			}
		case *schema.DropTable:
			if err := s.dropTable(change); err != nil {
				return err
			}
		default:
			return fmt.Errorf("ydb: unsupported change type: %T", change)
		}
	}
	return nil
}

// addTable builds and executes the query for creating a table in a schema.
func (s *state) addTable(addTable *schema.AddTable) error {
	var errs []string
	b := s.Build("CREATE TABLE")

	b.Table(addTable.T)
	b.WrapIndent(func(b *sqlx.Builder) {
		b.MapIndent(addTable.T.Columns, func(i int, b *sqlx.Builder) {
			if err := s.column(b, addTable.T.Columns[i]); err != nil {
				errs = append(errs, err.Error())
			}
		})
		if primaryKey := addTable.T.PrimaryKey; primaryKey != nil {
			b.Comma().NL().P("PRIMARY KEY")
			s.indexParts(b, primaryKey.Parts)
		} else {
			errs = append(errs, "ydb: primary key is mandatory")
		}
		// inline secondary indexes
		for _, idx := range addTable.T.Indexes {
			b.Comma().NL()
			s.indexDef(b, idx)
		}
	})

	if len(errs) > 0 {
		return fmt.Errorf("create table %q: %s", addTable.T.Name, strings.Join(errs, ", "))
	}

	reverse := s.Build("DROP TABLE").
		Table(addTable.T).
		String()

	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  addTable,
		Comment: fmt.Sprintf("create %q table", addTable.T.Name),
		Reverse: reverse,
	})
	return nil
}

// indexDef writes an inline index definition for CREATE TABLE.
func (s *state) indexDef(b *sqlx.Builder, idx *schema.Index) {
	b.P("INDEX").Ident(idx.Name).P("GLOBAL ON")
	s.indexParts(b, idx.Parts)
}

// dropTable builds and executes the query for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) error {
	reverseState := &state{
		conn:        s.conn,
		PlanOptions: s.PlanOptions,
	}

	if err := reverseState.addTable(&schema.AddTable{T: drop.T}); err != nil {
		return fmt.Errorf("calculate reverse for drop table %q: %w", drop.T.Name, err)
	}

	b := s.Build("DROP TABLE")
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		b.P("IF EXISTS")
	}
	b.Table(drop.T)

	// The reverse of 'DROP TABLE' might be a multi-statement operation
	reverse := func() any {
		cmd := make([]string, len(reverseState.Changes))
		for i, c := range reverseState.Changes {
			cmd[i] = c.Cmd
		}
		if len(cmd) == 1 {
			return cmd[0]
		}
		return cmd
	}()

	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
		Reverse: reverse,
	})
	return nil
}

// column writes the column definition to the builder.
func (s *state) column(b *sqlx.Builder, c *schema.Column) error {
	t, err := FormatType(c.Type.Type)
	if err != nil {
		return err
	}

	b.Ident(c.Name).P(t)

	if !c.Type.Null {
		b.P("NOT NULL")
	}
	return nil
}

// indexParts writes the index parts (columns) to the builder.
func (s *state) indexParts(b *sqlx.Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(parts, func(i int, b *sqlx.Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
		})
	})
}

// append adds changes to the plan.
func (s *state) append(c ...*migrate.Change) {
	s.Changes = append(s.Changes, c...)
}

// Build instantiates a new builder and writes the given phrase to it.
func (s *state) Build(phrases ...string) *sqlx.Builder {
	return (*Driver).StmtBuilder(nil, s.PlanOptions).
		P(phrases...)
}
