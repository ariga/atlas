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
var DefaultPlan migrate.PlanApplier = &planApply{
	conn: &conn{ExecQuerier: sqlx.NoRows},
}

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
			Transactional: false,
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
		case *schema.ModifyTable:
			if err := s.modifyTable(change); err != nil {
				return err
			}
		case *schema.RenameTable:
			s.renameTable(change)
		default:
			return fmt.Errorf("ydb: unsupported change type: %T", change)
		}
	}
	return nil
}

// tablePath returns the full YDB path for a table.
func (s *state) tablePath(table *schema.Table) string {
	if s.database == "" {
		return table.Name
	}
	return s.database + "/" + table.Name
}

// addTable builds and executes the query for creating a table in a schema.
func (s *state) addTable(addTable *schema.AddTable) error {
	var errs []string
	builder := s.Build("CREATE TABLE")

	if sqlx.Has(addTable.Extra, &schema.IfNotExists{}) {
		builder.P("IF NOT EXISTS")
	}

	builder.Ident(s.tablePath(addTable.T))
	builder.WrapIndent(func(b *sqlx.Builder) {
		b.MapIndent(
			addTable.T.Columns,
			func(i int, b *sqlx.Builder) {
				if err := s.column(b, addTable.T.Columns[i]); err != nil {
					errs = append(errs, err.Error())
				}
			},
		)

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
		Ident(s.tablePath(addTable.T)).
		String()

	s.append(&migrate.Change{
		Cmd:     builder.String(),
		Source:  addTable,
		Comment: fmt.Sprintf("create %q table", addTable.T.Name),
		Reverse: reverse,
	})
	return nil
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

	builder := s.Build("DROP TABLE")
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		builder.P("IF EXISTS")
	}
	builder.Ident(s.tablePath(drop.T))

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
		Cmd:     builder.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
		Reverse: reverse,
	})
	return nil
}

// modifyTable builds the statements that bring the table into its modified state.
func (s *state) modifyTable(modify *schema.ModifyTable) error {
	var (
		alterOps     []schema.Change
		addIndexOps  []*schema.AddIndex
		dropIndexOps []*schema.DropIndex
	)

	for _, change := range modify.Changes {
		switch change := change.(type) {
		case *schema.AddColumn:
			alterOps = append(alterOps, change)

		case *schema.DropColumn:
			alterOps = append(alterOps, change)

		case *schema.AddIndex:
			addIndexOps = append(addIndexOps, change)

		case *schema.DropIndex:
			dropIndexOps = append(dropIndexOps, change)

		case *schema.ModifyIndex:
			// Index modification requires rebuilding the index.
			dropIndexOps = append(dropIndexOps, &schema.DropIndex{I: change.From})
			addIndexOps = append(addIndexOps, &schema.AddIndex{I: change.To})

		case *schema.RenameIndex:
			s.renameIndex(modify, change)

		default:
			return fmt.Errorf("ydb: unsupported table change: %T", change)
		}
	}

	// Drop indexes first, then alter table, then add indexes
	if err := s.dropIndexes(modify, modify.T, dropIndexOps...); err != nil {
		return err
	}

	if len(alterOps) > 0 {
		if err := s.alterTable(modify.T, alterOps); err != nil {
			return err
		}
	}

	if err := s.addIndexes(modify, modify.T, addIndexOps...); err != nil {
		return err
	}

	return nil
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (s *state) alterTable(table *schema.Table, changes []schema.Change) error {
	var reverse []schema.Change

	buildFunc := func(changes []schema.Change) (string, error) {
		builder := s.Build("ALTER TABLE").Ident(s.tablePath(table))

		err := builder.MapCommaErr(
			changes,
			func(i int, builder *sqlx.Builder) error {
				switch change := changes[i].(type) {
				case *schema.AddColumn:
					builder.P("ADD COLUMN")
					if err := s.column(builder, change.C); err != nil {
						return err
					}
					reverse = append(reverse, &schema.DropColumn{C: change.C})

				case *schema.DropColumn:
					builder.P("DROP COLUMN").Ident(change.C.Name)
					reverse = append(reverse, &schema.AddColumn{C: change.C})
				}

				return nil
			},
		)
		if err != nil {
			return "", err
		}

		return builder.String(), nil
	}

	query, err := buildFunc(changes)
	if err != nil {
		return fmt.Errorf("alter table %q: %v", table.Name, err)
	}

	cmd := &migrate.Change{
		Cmd: query,
		Source: &schema.ModifyTable{
			T:       table,
			Changes: changes,
		},
		Comment: fmt.Sprintf("modify %q table", table.Name),
	}

	// Changes should be reverted in a reversed order they were created.
	sqlx.ReverseChanges(reverse)
	if cmd.Reverse, err = buildFunc(reverse); err != nil {
		return fmt.Errorf("reverse alter table %q: %v", table.Name, err)
	}

	s.append(cmd)
	return nil
}

func (s *state) addIndexes(src schema.Change, table *schema.Table, indexes ...*schema.AddIndex) error {
	for _, add := range indexes {
		index := add.I

		builder := s.Build("ALTER TABLE").
			Ident(s.tablePath(table)).
			P("ADD INDEX").
			Ident(index.Name)

		s.buildIndexSpec(builder, index)

		reverseOp := s.Build("ALTER TABLE").
			Ident(s.tablePath(table)).
			P("DROP INDEX").
			Ident(index.Name).
			String()

		s.append(&migrate.Change{
			Cmd:     builder.String(),
			Source:  src,
			Comment: fmt.Sprintf("create index %q to table: %q", index.Name, table.Name),
			Reverse: reverseOp,
		})
	}
	return nil
}

func (s *state) dropIndexes(src schema.Change, table *schema.Table, drops ...*schema.DropIndex) error {
	adds := make([]*schema.AddIndex, len(drops))
	for i, drop := range drops {
		adds[i] = &schema.AddIndex{
			I:     drop.I,
			Extra: drop.Extra,
		}
	}

	reverseState := &state{conn: s.conn, PlanOptions: s.PlanOptions}
	if err := reverseState.addIndexes(src, table, adds...); err != nil {
		return err
	}

	for i, add := range adds {
		s.append(&migrate.Change{
			Cmd:     reverseState.Changes[i].Reverse.(string),
			Source:  src,
			Comment: fmt.Sprintf("drop index %q from table: %q", add.I.Name, table.Name),
			Reverse: reverseState.Changes[i].Cmd,
		})
	}

	return nil
}

// renameTable builds and appends the statement for renaming a table.
func (s *state) renameTable(rename *schema.RenameTable) {
	s.append(&migrate.Change{
		Source:  rename,
		Comment: fmt.Sprintf("rename a table from %q to %q", rename.From.Name, rename.To.Name),
		Cmd:     s.Build("ALTER TABLE").Ident(s.tablePath(rename.From)).P("RENAME TO").Ident(s.tablePath(rename.To)).String(),
		Reverse: s.Build("ALTER TABLE").Ident(s.tablePath(rename.To)).P("RENAME TO").Ident(s.tablePath(rename.From)).String(),
	})
}

// renameIndex builds and appends the statement for renaming an index.
func (s *state) renameIndex(modify *schema.ModifyTable, rename *schema.RenameIndex) {
	s.append(&migrate.Change{
		Source:  rename,
		Comment: fmt.Sprintf("rename an index from %q to %q", rename.From.Name, rename.To.Name),
		Cmd:     s.Build("ALTER TABLE").Ident(s.tablePath(modify.T)).P("RENAME INDEX").Ident(rename.From.Name).P("TO").Ident(rename.To.Name).String(),
		Reverse: s.Build("ALTER TABLE").Ident(s.tablePath(modify.T)).P("RENAME INDEX").Ident(rename.To.Name).P("TO").Ident(rename.From.Name).String(),
	})
}

// column writes the column definition to the builder.
func (s *state) column(builder *sqlx.Builder, column *schema.Column) error {
	t, err := FormatType(column.Type.Type)
	if err != nil {
		return err
	}

	builder.Ident(column.Name).P(t)

	if !column.Type.Null {
		builder.P("NOT NULL")
	}
	return nil
}

// indexDef writes an inline index definition for CREATE TABLE.
func (s *state) indexDef(builder *sqlx.Builder, index *schema.Index) {
	builder.P("INDEX").Ident(index.Name)
	s.buildIndexSpec(builder, index)
}

// buildIndexSpec writes the common index specification:
// GLOBAL [UNIQUE] [SYNC|ASYNC] ON (columns) [COVER (columns)].
func (s *state) buildIndexSpec(builder *sqlx.Builder, idx *schema.Index) {
	indexAttrs := IndexAttributes{}
	hasAttrs := sqlx.Has(idx.Attrs, &indexAttrs)

	builder.P("GLOBAL")

	if idx.Unique {
		builder.P("UNIQUE")
	}

	if hasAttrs && indexAttrs.Async {
		builder.P("ASYNC")
	} else {
		builder.P("SYNC")
	}

	builder.P("ON")
	s.indexParts(builder, idx.Parts)

	if hasAttrs && len(indexAttrs.CoverColumns) > 0 {
		builder.P("COVER")
		s.indexCoverColumns(builder, indexAttrs.CoverColumns)
	}
}

// indexParts writes the index parts (columns) to the builder.
func (s *state) indexParts(builder *sqlx.Builder, parts []*schema.IndexPart) {
	builder.Wrap(func(b *sqlx.Builder) {
		b.MapComma(
			parts,
			func(i int, builder *sqlx.Builder) {
				if parts[i].C != nil {
					builder.Ident(parts[i].C.Name)
				}
			},
		)
	})
}

// indexCoverColumns writes the cover columns to the builder.
func (s *state) indexCoverColumns(builder *sqlx.Builder, coverColumns []*schema.Column) {
	builder.Wrap(func(b *sqlx.Builder) {
		b.MapComma(
			coverColumns,
			func(i int, builder *sqlx.Builder) {
				builder.Ident(coverColumns[i].Name)
			},
		)
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
