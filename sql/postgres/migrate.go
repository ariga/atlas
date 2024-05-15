// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DefaultPlan provides basic planning capabilities for PostgreSQL dialects.
// Note, it is recommended to call Open, create a new Driver and use its
// migrate.PlanApplier when a database connection is available.
var DefaultPlan migrate.PlanApplier = &planApply{conn: &conn{ExecQuerier: sqlx.NoRows}}

// A planApply provides migration capabilities for schema elements.
type planApply struct{ *conn }

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
	if err := verifyChanges(ctx, changes); err != nil {
		return nil, err
	}
	if err := s.plan(changes); err != nil {
		return nil, err
	}
	if err := sqlx.SetReversible(&s.Plan); err != nil {
		return nil, err
	}
	return &s.Plan, nil
}

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to do so, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
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

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (s *state) plan(changes []schema.Change) error {
	if s.SchemaQualifier != nil {
		if err := sqlx.CheckChangesScope(s.PlanOptions, changes); err != nil {
			return err
		}
	}
	planned, err := s.topLevel(changes)
	if err != nil {
		return err
	}
	if planned, err = detachCycles(planned); err != nil {
		return err
	}
	for _, c := range s.sortChanges(planned) {
		switch c := c.(type) {
		case *schema.AddTable:
			err = s.addTable(c)
		case *schema.ModifyTable:
			err = s.modifyTable(c)
		case *schema.RenameTable:
			s.renameTable(c)
		case *schema.AddFunc:
			err = s.addFunc(c)
		case *schema.AddProc:
			err = s.addProc(c)
		case *schema.ModifyFunc:
			err = s.modifyFunc(c)
		case *schema.ModifyProc:
			err = s.modifyProc(c)
		case *schema.RenameFunc:
			err = s.renameFunc(c)
		case *schema.RenameProc:
			err = s.renameProc(c)
		case *schema.AddView:
			err = s.addView(c)
		case *schema.DropView:
			err = s.dropView(c)
		case *schema.ModifyView:
			err = s.modifyView(c)
		case *schema.RenameView:
			s.renameView(c)
		case *schema.DropTable:
			err = s.dropTable(c)
		case *schema.DropFunc:
			err = s.dropFunc(c)
		case *schema.DropProc:
			err = s.dropProc(c)
		case *schema.AddObject:
			err = s.addObject(c)
		case *schema.ModifyObject:
			err = s.modifyObject(c)
		case *schema.DropObject:
			err = s.dropObject(c)
		case *schema.AddTrigger:
			err = s.addTrigger(c)
		case *schema.DropTrigger:
			err = s.dropTrigger(c)
		case *schema.RenameTrigger:
			err = s.renameTrigger(c)
		case *schema.ModifyTrigger:
			err = s.modifyTrigger(c)
		default:
			err = fmt.Errorf("unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// topLevel executes first the changes for creating or dropping schemas and
// create objects that tables might depend on.
func (s *state) topLevel(changes []schema.Change) ([]schema.Change, error) {
	planned := make([]schema.Change, 0, len(changes))
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddSchema:
			b := s.Build("CREATE SCHEMA")
			// Add the 'IF NOT EXISTS' clause if it is explicitly specified, or if the schema name is 'public'.
			// That is because the 'public' schema is automatically created by PostgreSQL in every new database,
			// and running the command with this clause will fail in case the schema already exists.
			if sqlx.Has(c.Extra, &schema.IfNotExists{}) || c.S.Name == "public" {
				b.P("IF NOT EXISTS")
			}
			b.Ident(c.S.Name)
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Reverse: s.Build("DROP SCHEMA").Ident(c.S.Name).P("CASCADE").String(),
				Comment: fmt.Sprintf("Add new schema named %q", c.S.Name),
			})
			if cm := (schema.Comment{}); sqlx.Has(c.S.Attrs, &cm) {
				s.append(s.schemaComment(c, c.S, cm.Text, ""))
			}
		case *schema.ModifySchema:
			for i := range c.Changes {
				switch change := c.Changes[i].(type) {
				// Add schema attributes to an existing schema only if
				// it is different from the default server configuration.
				case *schema.AddAttr:
					a, ok := change.A.(*schema.Comment)
					if !ok {
						return nil, fmt.Errorf("unexpected schema AddAttr: %T", change.A)
					}
					s.append(s.schemaComment(c, c.S, a.Text, ""))
				case *schema.ModifyAttr:
					to, ok1 := change.To.(*schema.Comment)
					from, ok2 := change.From.(*schema.Comment)
					if !ok1 || !ok2 {
						return nil, fmt.Errorf("unexpected schema ModifyAttr: (%T, %T)", change.To, change.From)
					}
					s.append(s.schemaComment(c, c.S, to.Text, from.Text))
				default:
					return nil, fmt.Errorf("unsupported ModifySchema change: %T", change)
				}
			}
		case *schema.DropSchema:
			b := s.Build("DROP SCHEMA")
			if sqlx.Has(c.Extra, &schema.IfExists{}) {
				b.P("IF EXISTS")
			}
			b.Ident(c.S.Name).P("CASCADE")
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Source:  c,
				Comment: fmt.Sprintf("Drop schema named %q", c.S.Name),
			})
		case *schema.RenameObject:
			e1, ok1 := c.From.(*schema.EnumType)
			e2, ok2 := c.To.(*schema.EnumType)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("unsupported rename types %T -> %T", c.From, c.To)
			}
			s.append(&migrate.Change{
				Source:  c,
				Cmd:     s.Build("ALTER TYPE").Ident(e1.T).P("RENAME TO").Ident(e2.T).String(),
				Reverse: s.Build("ALTER TYPE").Ident(e2.T).P("RENAME TO").Ident(e1.T).String(),
				Comment: fmt.Sprintf("rename an enum from %q to %q", e1.T, e2.T),
			})
		default:
			planned = append(planned, c)
		}
	}
	return planned, nil
}

// addTable builds and executes the query for creating a table in a schema.
func (s *state) addTable(add *schema.AddTable) error {
	var (
		errs []string
		b    = s.Build("CREATE TABLE")
	)
	if sqlx.Has(add.Extra, &schema.IfNotExists{}) {
		b.P("IF NOT EXISTS")
	}
	b.Table(add.T)
	b.WrapIndent(func(b *sqlx.Builder) {
		b.MapIndent(add.T.Columns, func(i int, b *sqlx.Builder) {
			if err := s.column(b, add.T.Columns[i]); err != nil {
				errs = append(errs, err.Error())
			}
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().NL().P("PRIMARY KEY")
			if err := s.index(b, pk); err != nil {
				errs = append(errs, err.Error())
			}
		}
		for _, idx := range add.T.Indexes {
			_, okU := uniqueConst(idx.Attrs)
			_, okE := excludeConst(idx.Attrs)
			if okU || okE {
				b.Comma().NL()
				if err := s.constraint(b, idx); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			s.fks(b, add.T.ForeignKeys...)
		}
		for _, attr := range add.T.Attrs {
			if c, ok := attr.(*schema.Check); ok {
				b.Comma().NL()
				check(b, c)
			}
		}
	})
	if p := (Partition{}); sqlx.Has(add.T.Attrs, &p) {
		s, err := formatPartition(p)
		if err != nil {
			errs = append(errs, err.Error())
		}
		b.P(s)
	}
	if len(errs) > 0 {
		return fmt.Errorf("create table %q: %s", add.T.Name, strings.Join(errs, ", "))
	}
	s.append(&migrate.Change{
		Cmd:     b.String(),
		Source:  add,
		Comment: fmt.Sprintf("create %q table", add.T.Name),
		Reverse: s.Build("DROP TABLE").Table(add.T).String(),
	})
	for _, idx := range add.T.Indexes {
		_, okU := uniqueConst(idx.Attrs)
		_, okE := excludeConst(idx.Attrs)
		if !okU && !okE {
			// Indexes do not need to be created concurrently on new tables.
			if err := s.addIndexes(add, add.T, &schema.AddIndex{I: idx}); err != nil {
				return err
			}
		}
	}
	s.addComments(add, add.T)
	return nil
}

// dropTable builds and executes the query for dropping a table from a schema.
func (s *state) dropTable(drop *schema.DropTable) error {
	cmd := &changeGroup{}
	rs := &state{
		conn:        s.conn,
		PlanOptions: s.PlanOptions,
	}
	if err := rs.addTable(&schema.AddTable{T: drop.T}); err != nil {
		return fmt.Errorf("calculate reverse for drop table %q: %w", drop.T.Name, err)
	}
	b := s.Build("DROP TABLE")
	if sqlx.Has(drop.Extra, &schema.IfExists{}) {
		b.P("IF EXISTS")
	}
	b.Table(drop.T)
	if sqlx.Has(drop.Extra, &Cascade{}) {
		b.P("CASCADE")
	}
	cmd.main = &migrate.Change{
		Cmd:     b.String(),
		Source:  drop,
		Comment: fmt.Sprintf("drop %q table", drop.T.Name),
		// The reverse of 'DROP TABLE' might be a multi
		// statement operation. e.g., table with indexes.
		Reverse: func() any {
			cmd := make([]string, len(rs.Changes))
			for i, c := range rs.Changes {
				cmd[i] = c.Cmd
			}
			if len(cmd) == 1 {
				return cmd[0]
			}
			return cmd
		}(),
	}
	cmd.append(s)
	return nil
}

// modifyTable builds the statements that bring the table into its modified state.
func (s *state) modifyTable(modify *schema.ModifyTable) error {
	var (
		alter   []schema.Change
		addI    []*schema.AddIndex
		dropI   []*schema.DropIndex
		changes []*migrate.Change
	)
	for _, change := range skipAutoChanges(modify.Changes) {
		switch change := change.(type) {
		case *schema.AddAttr, *schema.ModifyAttr:
			from, to, err := commentChange(change)
			if err != nil {
				return err
			}
			changes = append(changes, s.tableComment(modify, modify.T, to, from))
		case *schema.DropAttr:
			return fmt.Errorf("unsupported change type: %T", change)
		case *schema.AddIndex:
			if c := (schema.Comment{}); sqlx.Has(change.I.Attrs, &c) {
				changes = append(changes, s.indexComment(modify, modify.T, change.I, c.Text, ""))
			}
			_, okU := uniqueConst(change.I.Attrs)
			_, okE := excludeConst(change.I.Attrs)
			// Unlike ADD INDEX statements that are executed separately,
			// ADD CONSTRAINT are added to the ALTER TABLE statement below.
			if okU || okE {
				alter = append(alter, change)
			} else {
				addI = append(addI, change)
			}
		case *schema.DropIndex:
			_, okU := uniqueConst(change.I.Attrs)
			_, okE := excludeConst(change.I.Attrs)
			// Unlike DROP INDEX statements that are executed separately,
			// DROP CONSTRAINT are added to the ALTER TABLE statement below.
			if okU || okE {
				alter = append(alter, change)
			} else {
				dropI = append(dropI, change)
			}
		case *schema.ModifyPrimaryKey:
			// Primary key modification needs to be split into "Drop" and "Add"
			// because the new key may include columns that have not been added yet.
			alter = append(alter, &schema.DropPrimaryKey{
				P: change.From,
			}, &schema.AddPrimaryKey{
				P: change.To,
			})
		case *schema.ModifyIndex:
			k := change.Change
			if change.Change.Is(schema.ChangeComment) {
				from, to, err := commentChange(sqlx.CommentDiff(change.From.Attrs, change.To.Attrs))
				if err != nil {
					return err
				}
				changes = append(changes, s.indexComment(modify, modify.T, change.To, to, from))
				// If only the comment of the index was changed.
				if k &= ^schema.ChangeComment; k.Is(schema.NoChange) {
					continue
				}
			}
			if addU, ok := indexToUnique(change); ok {
				alter = append(alter, addU)
				continue
			}
			// Index (or constraint) modification requires rebuilding the index.
			_, fromU := uniqueConst(change.From.Attrs)
			_, fromE := excludeConst(change.From.Attrs)
			if fromU || fromE {
				alter = append(alter, &schema.DropIndex{I: change.From})
			} else {
				dropI = append(dropI, &schema.DropIndex{I: change.From})
			}
			_, toU := uniqueConst(change.To.Attrs)
			_, toE := excludeConst(change.To.Attrs)
			if toU || toE {
				alter = append(alter, &schema.AddIndex{I: change.To})
			} else {
				addI = append(addI, &schema.AddIndex{I: change.To})
			}
		case *schema.RenameIndex:
			changes = append(changes, &migrate.Change{
				Source:  change,
				Comment: fmt.Sprintf("rename an index from %q to %q", change.From.Name, change.To.Name),
				Cmd:     s.Build("ALTER INDEX").Ident(change.From.Name).P("RENAME TO").Ident(change.To.Name).String(),
				Reverse: s.Build("ALTER INDEX").Ident(change.To.Name).P("RENAME TO").Ident(change.From.Name).String(),
			})
		case *schema.ModifyForeignKey:
			// Foreign-key modification is translated into 2 steps.
			// Dropping the current foreign key and creating a new one.
			alter = append(alter, &schema.DropForeignKey{
				F: change.From,
			}, &schema.AddForeignKey{
				F: change.To,
			})
		case *schema.AddColumn:
			if c := (schema.Comment{}); sqlx.Has(change.C.Attrs, &c) {
				changes = append(changes, s.columnComment(modify, modify.T, change.C, c.Text, ""))
			}
			alter = append(alter, change)
		case *schema.ModifyColumn:
			k := change.Change
			if change.Change.Is(schema.ChangeComment) {
				from, to, err := commentChange(sqlx.CommentDiff(change.From.Attrs, change.To.Attrs))
				if err != nil {
					return err
				}
				changes = append(changes, s.columnComment(modify, modify.T, change.To, to, from))
				// If only the comment of the column was changed.
				if k &= ^schema.ChangeComment; k.Is(schema.NoChange) {
					continue
				}
			}
			alter = append(alter, &schema.ModifyColumn{To: change.To, From: change.From, Change: k, Extra: change.Extra})
		case *schema.RenameColumn:
			// "RENAME COLUMN" cannot be combined with other alterations.
			b := s.Build("ALTER TABLE").Table(modify.T).P("RENAME COLUMN")
			r := b.Clone()
			changes = append(changes, &migrate.Change{
				Source:  change,
				Comment: fmt.Sprintf("rename a column from %q to %q", change.From.Name, change.To.Name),
				Cmd:     b.Ident(change.From.Name).P("TO").Ident(change.To.Name).String(),
				Reverse: r.Ident(change.To.Name).P("TO").Ident(change.From.Name).String(),
			})
		default:
			alter = append(alter, change)
		}
	}
	if err := s.dropIndexes(modify, modify.T, dropI...); err != nil {
		return err
	}
	if len(alter) > 0 {
		if err := s.alterTable(modify.T, alter); err != nil {
			return err
		}
	}
	if err := s.addIndexes(modify, modify.T, addI...); err != nil {
		return err
	}
	s.append(changes...)
	return nil
}

type (
	// AddUniqueConstraint to the table using the given index. Note, if the index
	// name does not match the unique constraint name, PostgreSQL implicitly renames
	// it to the constraint name.
	AddUniqueConstraint struct {
		schema.Change
		Name  string        // Name of the constraint.
		Using *schema.Index // Index to use for the constraint.
	}
	// AddPKConstraint to the table using the given index. Note, if the index
	// name does not match the primary-key constraint name, PostgreSQL implicitly
	// renames it to the constraint name.
	AddPKConstraint struct {
		schema.Change
		Name  string        // Name of the constraint.
		Using *schema.Index // Index to use for the constraint.
	}
)

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (s *state) alterTable(t *schema.Table, changes []schema.Change) error {
	var (
		reverse    []schema.Change
		reversible = true
	)
	// Constraints drop should be executed first.
	sort.SliceStable(changes, func(i, j int) bool {
		return dropConst(changes[i]) && !dropConst(changes[j])
	})
	build := func(alter *changeGroup, changes []schema.Change) (string, error) {
		b := s.Build("ALTER TABLE").Table(t)
		err := b.MapCommaErr(changes, func(i int, b *sqlx.Builder) error {
			switch change := changes[i].(type) {
			case *schema.AddColumn:
				b.P("ADD COLUMN")
				if err := s.column(b, change.C); err != nil {
					return err
				}
				reverse = append(reverse, &schema.DropColumn{C: change.C})
			case *schema.ModifyColumn:
				if err := s.alterColumn(b, alter, t, change); err != nil {
					return err
				}
				if change.Change.Is(schema.ChangeGenerated) {
					reversible = false
				}
				reverse = append(reverse, &schema.ModifyColumn{
					From:   change.To,
					To:     change.From,
					Change: change.Change & ^schema.ChangeGenerated,
				})
			case *schema.DropColumn:
				b.P("DROP COLUMN").Ident(change.C.Name)
				reverse = append(reverse, &schema.AddColumn{C: change.C})
			case *AddUniqueConstraint:
				b.P("ADD")
				drop := change.Using
				if change.Name != "" {
					b.P("CONSTRAINT").Ident(change.Name)
					drop = sqlx.P(*change.Using)
					drop.Name = change.Name
				}
				b.P("UNIQUE USING INDEX").Ident(change.Using.Name)
				// Translated to the DROP CONSTRAINT below,
				// which drops the index as well.
				reverse = append(reverse, &schema.DropIndex{I: drop})
			case *AddPKConstraint:
				b.P("ADD")
				drop := change.Using
				if change.Name != "" {
					b.P("CONSTRAINT").Ident(change.Name)
					drop = sqlx.P(*change.Using)
					drop.Name = change.Name
				}
				b.P("PRIMARY KEY USING INDEX").Ident(change.Using.Name)
				// Translated to the DROP CONSTRAINT below,
				// which drops the index as well.
				reverse = append(reverse, &schema.DropPrimaryKey{P: drop})
			case *schema.AddIndex:
				b.P("ADD")
				if err := s.constraint(b, change.I); err != nil {
					return err
				}
				reverse = append(reverse, &schema.DropIndex{I: change.I})
			case *schema.DropIndex:
				b.P("DROP CONSTRAINT").Ident(change.I.Name)
				reverse = append(reverse, &schema.AddIndex{I: change.I})
			case *schema.AddPrimaryKey:
				b.P("ADD PRIMARY KEY")
				if err := s.index(b, change.P); err != nil {
					return err
				}
				reverse = append(reverse, &schema.DropPrimaryKey{P: change.P})
			case *schema.DropPrimaryKey:
				b.P("DROP CONSTRAINT").Ident(pkName(t, change.P))
				reverse = append(reverse, &schema.AddPrimaryKey{P: change.P})
			case *schema.AddForeignKey:
				s.fks(b.P("ADD"), change.F)
				if sqlx.Has(change.Extra, &NotValid{}) {
					b.P("NOT VALID")
				}
				reverse = append(reverse, &schema.DropForeignKey{F: change.F})
			case *schema.DropForeignKey:
				b.P("DROP CONSTRAINT")
				if sqlx.Has(change.Extra, &schema.IfExists{}) {
					b.P("IF EXISTS")
				}
				b.Ident(change.F.Symbol)
				reverse = append(reverse, &schema.AddForeignKey{F: change.F})
			case *schema.AddCheck:
				check(b.P("ADD"), change.C)
				if sqlx.Has(change.Extra, &NotValid{}) {
					b.P("NOT VALID")
				}
				// Reverse operation is supported if
				// the constraint name is not generated.
				if reversible = reversible && change.C.Name != ""; reversible {
					reverse = append(reverse, &schema.DropCheck{C: change.C})
				}
			case *schema.DropCheck:
				b.P("DROP CONSTRAINT").Ident(change.C.Name)
				reverse = append(reverse, &schema.AddCheck{C: change.C})
			case *schema.ModifyCheck:
				switch {
				case change.From.Name == "":
					return errors.New("cannot modify unnamed check constraint")
				case change.From.Name != change.To.Name:
					return fmt.Errorf("mismatch check constraint names: %q != %q", change.From.Name, change.To.Name)
				case change.From.Expr != change.To.Expr,
					sqlx.Has(change.From.Attrs, &NoInherit{}) && !sqlx.Has(change.To.Attrs, &NoInherit{}),
					!sqlx.Has(change.From.Attrs, &NoInherit{}) && sqlx.Has(change.To.Attrs, &NoInherit{}):
					b.P("DROP CONSTRAINT").Ident(change.From.Name).Comma().P("ADD")
					check(b, change.To)
				default:
					return errors.New("unknown check constraint change")
				}
				reverse = append(reverse, &schema.ModifyCheck{
					From: change.To,
					To:   change.From,
				})
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		return b.String(), nil
	}
	cmd := &changeGroup{}
	stmt, err := build(cmd, changes)
	if err != nil {
		return fmt.Errorf("alter table %q: %v", t.Name, err)
	}
	cmd.main = &migrate.Change{
		Cmd: stmt,
		Source: &schema.ModifyTable{
			T:       t,
			Changes: changes,
		},
		Comment: fmt.Sprintf("modify %q table", t.Name),
	}
	if reversible {
		// Changes should be reverted in
		// a reversed order they were created.
		sqlx.ReverseChanges(reverse)
		if cmd.main.Reverse, err = build(&changeGroup{}, reverse); err != nil {
			return fmt.Errorf("reverse alter table %q: %v", t.Name, err)
		}
	}
	cmd.append(s)
	return nil
}

// changeGroup describes an alter table migrate.Change where its main command
// can be supported by additional statements before and after it is executed.
type changeGroup struct {
	main          *migrate.Change
	before, after []*migrate.Change
}

func (a *changeGroup) append(s *state) {
	s.append(a.before...)
	s.append(a.main)
	s.append(a.after...)
}

func (s *state) alterColumn(b *sqlx.Builder, alter *changeGroup, t *schema.Table, c *schema.ModifyColumn) error {
	for k := c.Change; !k.Is(schema.NoChange); {
		b.P("ALTER COLUMN").Ident(c.To.Name)
		switch {
		case k.Is(schema.ChangeType):
			if err := s.alterType(b, alter, t, c); err != nil {
				return err
			}
			k &= ^schema.ChangeType
		case k.Is(schema.ChangeNull) && c.To.Type.Null:
			if t, ok := c.To.Type.Type.(*SerialType); ok {
				return fmt.Errorf("NOT NULL constraint is required for %s column %q", t.T, c.To.Name)
			}
			b.P("DROP NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeNull) && !c.To.Type.Null:
			b.P("SET NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeDefault) && c.To.Default == nil:
			b.P("DROP DEFAULT")
			k &= ^schema.ChangeDefault
		case k.Is(schema.ChangeDefault) && c.To.Default != nil:
			s.columnDefault(b.P("SET"), c.To)
			k &= ^schema.ChangeDefault
		case k.Is(schema.ChangeAttr):
			toI, ok := identity(c.To.Attrs)
			if !ok {
				return fmt.Errorf("unexpected attribute change (expect IDENTITY): %v", c.To.Attrs)
			}
			// The syntax for altering identity columns is identical to sequence_options.
			// https://www.postgresql.org/docs/current/sql-altersequence.html
			b.P("SET GENERATED", toI.Generation, "SET START WITH", strconv.FormatInt(toI.Sequence.Start, 10), "SET INCREMENT BY", strconv.FormatInt(toI.Sequence.Increment, 10))
			// Skip SEQUENCE RESTART in case the "start value" is less than the "current value" in one
			// of the states (inspected and desired), because this function is used for both UP and DOWN.
			if fromI, ok := identity(c.From.Attrs); (!ok || fromI.Sequence.Last < toI.Sequence.Start) && toI.Sequence.Last < toI.Sequence.Start {
				b.P("RESTART")
			}
			k &= ^schema.ChangeAttr
		case k.Is(schema.ChangeGenerated):
			if sqlx.Has(c.To.Attrs, &schema.GeneratedExpr{}) {
				return fmt.Errorf("unexpected generation expression change (expect DROP EXPRESSION): %v", c.To.Attrs)
			}
			b.P("DROP EXPRESSION")
			k &= ^schema.ChangeGenerated
		default: // e.g. schema.ChangeComment.
			return fmt.Errorf("unexpected column change: %d", k)
		}
		if !k.Is(schema.NoChange) {
			b.Comma()
		}
	}
	return nil
}

// alterType appends the clause(s) to alter the column type and assuming the
// "ALTER COLUMN <Name>" was called before by the alterColumn function.
func (s *state) alterType(b *sqlx.Builder, alter *changeGroup, t *schema.Table, c *schema.ModifyColumn) error {
	// Commands for creating and dropping serial sequences.
	createDropSeq := func(st *SerialType) (string, string, string) {
		seq := fmt.Sprintf(`%s%q`, s.schemaPrefix(t.Schema), st.sequence(t, c.To))
		drop := s.Build("DROP SEQUENCE IF EXISTS").P(seq).String()
		create := s.Build("CREATE SEQUENCE IF NOT EXISTS").P(seq, "OWNED BY").
			P(fmt.Sprintf(`%s%q.%q`, s.schemaPrefix(t.Schema), t.Name, c.To.Name)).
			String()
		return create, drop, seq
	}
	toS, toHas := c.To.Type.Type.(*SerialType)
	fromS, fromHas := c.From.Type.Type.(*SerialType)
	switch {
	// Sequence was dropped.
	case fromHas && !toHas:
		b.P("DROP DEFAULT")
		create, drop, _ := createDropSeq(fromS)
		// Sequence should be deleted after it was dropped
		// from the DEFAULT value.
		alter.after = append(alter.after, &migrate.Change{
			Source:  c,
			Comment: fmt.Sprintf("drop sequence used by serial column %q", c.From.Name),
			Cmd:     drop,
			Reverse: create,
		})
		toT, err := FormatType(c.To.Type.Type)
		if err != nil {
			return err
		}
		fromT, err := FormatType(fromS.IntegerType())
		if err != nil {
			return err
		}
		// Underlying type was changed. e.g. serial to bigint.
		if toT != fromT {
			b.Comma().P("ALTER COLUMN").Ident(c.To.Name).P("TYPE", toT)
		}
	// Sequence was added.
	case !fromHas && toHas:
		create, drop, seq := createDropSeq(toS)
		// Sequence should be created before it is used by the
		// column DEFAULT value.
		alter.before = append(alter.before, &migrate.Change{
			Source:  c,
			Comment: fmt.Sprintf("create sequence for serial column %q", c.To.Name),
			Cmd:     create,
			Reverse: drop,
		})
		b.P("SET DEFAULT", fmt.Sprintf("nextval('%s')", seq))
		toT, err := FormatType(toS.IntegerType())
		if err != nil {
			return err
		}
		fromT, err := FormatType(c.From.Type.Type)
		if err != nil {
			return err
		}
		// Underlying type was changed. e.g. integer to bigserial (bigint).
		if toT != fromT {
			b.Comma().P("ALTER COLUMN").Ident(c.To.Name).P("TYPE", toT)
		}
	// Serial type was changed. e.g. serial to bigserial.
	case fromHas && toHas:
		f, err := FormatType(toS.IntegerType())
		if err != nil {
			return err
		}
		b.P("TYPE", f)
	default:
		var (
			f   string
			err error
		)
		if e, ok := c.To.Type.Type.(*schema.EnumType); ok {
			f = s.enumIdent(e)
		} else if f, err = FormatType(c.To.Type.Type); err != nil {
			return err
		}
		b.P("TYPE", f)
	}
	if collate := (schema.Collation{}); sqlx.Has(c.To.Attrs, &collate) {
		b.P("COLLATE", collate.V)
	}
	if using := (ConvertUsing{}); sqlx.Has(c.Extra, &using) {
		b.P("USING", using.X)
	}
	return nil
}

func (s *state) renameTable(c *schema.RenameTable) {
	s.append(&migrate.Change{
		Source:  c,
		Comment: fmt.Sprintf("rename a table from %q to %q", c.From.Name, c.To.Name),
		Cmd:     s.Build("ALTER TABLE").Table(c.From).P("RENAME TO").Table(c.To).String(),
		Reverse: s.Build("ALTER TABLE").Table(c.To).P("RENAME TO").Table(c.From).String(),
	})
}

func (s *state) addComments(src schema.Change, t *schema.Table) {
	var c schema.Comment
	if sqlx.Has(t.Attrs, &c) && c.Text != "" {
		s.append(s.tableComment(src, t, c.Text, ""))
	}
	for i := range t.Columns {
		if sqlx.Has(t.Columns[i].Attrs, &c) && c.Text != "" {
			s.append(s.columnComment(src, t, t.Columns[i], c.Text, ""))
		}
	}
	for i := range t.Indexes {
		if sqlx.Has(t.Indexes[i].Attrs, &c) && c.Text != "" {
			s.append(s.indexComment(src, t, t.Indexes[i], c.Text, ""))
		}
	}
}

func (s *state) schemaComment(src schema.Change, sc *schema.Schema, to, from string) *migrate.Change {
	b := s.Build("COMMENT ON SCHEMA").Ident(sc.Name).P("IS")
	return &migrate.Change{
		Cmd:     b.Clone().P(quote(to)).String(),
		Source:  src,
		Comment: fmt.Sprintf("set comment to schema: %q", sc.Name),
		Reverse: b.Clone().P(quote(from)).String(),
	}
}

func (s *state) tableComment(src schema.Change, t *schema.Table, to, from string) *migrate.Change {
	b := s.Build("COMMENT ON TABLE").Table(t).P("IS")
	return &migrate.Change{
		Cmd:     b.Clone().P(quote(to)).String(),
		Source:  src,
		Comment: fmt.Sprintf("set comment to table: %q", t.Name),
		Reverse: b.Clone().P(quote(from)).String(),
	}
}

func (s *state) columnComment(src schema.Change, t *schema.Table, c *schema.Column, to, from string) *migrate.Change {
	b := s.Build("COMMENT ON COLUMN").Table(t)
	b.WriteByte('.')
	b.Ident(c.Name).P("IS")
	return &migrate.Change{
		Cmd:     b.Clone().P(quote(to)).String(),
		Source:  src,
		Comment: fmt.Sprintf("set comment to column: %q on table: %q", c.Name, t.Name),
		Reverse: b.Clone().P(quote(from)).String(),
	}
}

func (s *state) indexComment(src schema.Change, t *schema.Table, idx *schema.Index, to, from string) *migrate.Change {
	b := s.Build("COMMENT ON INDEX").SchemaResource(t.Schema, idx.Name).P("IS")
	return &migrate.Change{
		Cmd:     b.Clone().P(quote(to)).String(),
		Source:  src,
		Comment: fmt.Sprintf("set comment to index: %q on table: %q", idx.Name, t.Name),
		Reverse: b.Clone().P(quote(from)).String(),
	}
}

func (s *state) dropIndexes(src schema.Change, t *schema.Table, drops ...*schema.DropIndex) error {
	adds := make([]*schema.AddIndex, len(drops))
	for i, d := range drops {
		adds[i] = &schema.AddIndex{I: d.I, Extra: d.Extra}
	}
	rs := &state{conn: s.conn, PlanOptions: s.PlanOptions}
	if err := rs.addIndexes(src, t, adds...); err != nil {
		return err
	}
	for i, add := range adds {
		s.append(&migrate.Change{
			Cmd:     rs.Changes[i].Reverse.(string),
			Source:  src,
			Comment: fmt.Sprintf("drop index %q from table: %q", add.I.Name, t.Name),
			Reverse: rs.Changes[i].Cmd,
		})
	}
	return nil
}

func (s *state) alterEnum(modify *schema.ModifyObject) error {
	from, ok1 := modify.From.(*schema.EnumType)
	to, ok2 := modify.To.(*schema.EnumType)
	if !ok1 || !ok2 {
		return fmt.Errorf("altering objects (%T) to (%T) is not supported", modify.From, modify.To)
	}
	fromV := make(map[string]int, len(from.Values))
	for i, v := range from.Values {
		fromV[v] = i
	}
	toV := make(map[string]int, len(to.Values))
	for i, v := range from.Values {
		toV[v] = i
	}
	for v := range fromV {
		if _, ok := toV[v]; !ok {
			return fmt.Errorf("dropping value %q from enum %q is not supported", v, from.T)
		}
	}
	var (
		at   int
		name = s.enumIdent(from)
	)
	for i, v := range to.Values {
		b := s.Build("ALTER TYPE").P(name, "ADD VALUE", quote(v))
		switch j, ok := fromV[v]; {
		case !ok:
			if i == 0 && len(from.Values) > 0 {
				b.P("BEFORE").P(quote(from.Values[0]))
			} else if i > 0 && at != len(from.Values) {
				b.P("AFTER").P(quote(to.Values[i-1]))
			}
			s.append(&migrate.Change{
				Cmd:     b.String(),
				Comment: fmt.Sprintf("add value to enum type: %q", from.T),
			})
		case ok && j == at:
			at++
		default:
			return fmt.Errorf("reordering enum %q value %q is not supported", from.T, v)
		}
	}
	return nil
}

func (s *state) addIndexes(src schema.Change, t *schema.Table, adds ...*schema.AddIndex) error {
	for _, add := range adds {
		b, idx := s.Build("CREATE"), add.I
		if idx.Unique {
			b.P("UNIQUE")
		}
		b.P("INDEX")
		if sqlx.Has(add.Extra, &Concurrently{}) {
			b.P("CONCURRENTLY")
		}
		if idx.Name != "" {
			b.Ident(idx.Name)
		}
		b.P("ON").Table(t)
		if err := s.index(b, idx); err != nil {
			return err
		}
		s.append(&migrate.Change{
			Cmd:     b.String(),
			Source:  src,
			Comment: fmt.Sprintf("create index %q to table: %q", idx.Name, t.Name),
			Reverse: func() string {
				b := s.Build("DROP INDEX")
				if sqlx.Has(add.Extra, &Concurrently{}) {
					b.P("CONCURRENTLY")
				}
				// Unlike MySQL, the DROP command is not attached to ALTER TABLE.
				// Therefore, we print indexes with their qualified name, because
				// the connection that executes the statements may not be attached
				// to this schema.
				if t.Schema != nil {
					b.WriteString(s.schemaPrefix(t.Schema))
				}
				b.Ident(idx.Name)
				return b.String()
			}(),
		})
	}
	return nil
}

func (s *state) column(b *sqlx.Builder, c *schema.Column) error {
	f, err := s.formatType(c.Type.Type)
	if err != nil {
		return err
	}
	b.Ident(c.Name).P(f)
	if !c.Type.Null {
		b.P("NOT")
	} else if t, ok := c.Type.Type.(*SerialType); ok {
		return fmt.Errorf("NOT NULL constraint is required for %s column %q", t.T, c.Name)
	}
	b.P("NULL")
	s.columnDefault(b, c)
	for _, attr := range c.Attrs {
		switch a := attr.(type) {
		case *schema.Comment, *schema.Check:
		case *schema.Collation:
			b.P("COLLATE").Ident(a.V)
		case *Identity, *schema.GeneratedExpr:
			// Handled below.
		default:
			return fmt.Errorf("unexpected column attribute: %T", attr)
		}
	}
	switch hasI, hasX := sqlx.Has(c.Attrs, &Identity{}), sqlx.Has(c.Attrs, &schema.GeneratedExpr{}); {
	case hasI && hasX:
		return fmt.Errorf("both identity and generation expression specified for column %q", c.Name)
	case hasI:
		id, _ := identity(c.Attrs)
		b.P("GENERATED", id.Generation, "AS IDENTITY")
		if id.Sequence.Start != defaultSeqStart || id.Sequence.Increment != defaultSeqIncrement {
			b.Wrap(func(b *sqlx.Builder) {
				if id.Sequence.Start != defaultSeqStart {
					b.P("START WITH", strconv.FormatInt(id.Sequence.Start, 10))
				}
				if id.Sequence.Increment != defaultSeqIncrement {
					b.P("INCREMENT BY", strconv.FormatInt(id.Sequence.Increment, 10))
				}
			})
		}
	case hasX:
		x := &schema.GeneratedExpr{}
		sqlx.Has(c.Attrs, x)
		b.P("GENERATED ALWAYS AS", sqlx.MayWrap(x.Expr), "STORED")
	}
	return nil
}

// columnDefault writes the default value of column to the builder.
func (s *state) columnDefault(b *sqlx.Builder, c *schema.Column) {
	if c.Default != nil {
		s.formatDefault(b, c.Type.Type, c.Default)
	}
}

// formatDefault writes the default value of column to the builder.
func (s *state) formatDefault(b *sqlx.Builder, t schema.Type, x schema.Expr) {
	switch x := x.(type) {
	case *schema.Literal:
		v := x.V
		switch t.(type) {
		case *schema.BoolType, *schema.DecimalType, *schema.IntegerType, *schema.FloatType:
		default:
			v = quote(v)
		}
		b.P("DEFAULT", v)
	case *schema.RawExpr:
		// Ignore identity functions added by the Differ.
		if _, ok := t.(*SerialType); !ok {
			b.P("DEFAULT", x.X)
		}
	}
}

func (s *state) indexParts(b *sqlx.Builder, idx *schema.Index) (err error) {
	b.Wrap(func(b *sqlx.Builder) {
		err = b.MapCommaErr(idx.Parts, func(i int, b *sqlx.Builder) error {
			switch part := idx.Parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(sqlx.MayWrap(part.X.(*schema.RawExpr).X))
			}
			return s.partAttrs(b, idx, idx.Parts[i])
		})
	})
	return
}

func (s *state) partAttrs(b *sqlx.Builder, idx *schema.Index, p *schema.IndexPart) error {
	if c := (schema.Collation{}); sqlx.Has(p.Attrs, &c) {
		b.P("COLLATE").Ident(c.V)
	}
	if op := (IndexOpClass{}); sqlx.Has(p.Attrs, &op) {
		d, err := op.DefaultFor(idx, p)
		if err != nil {
			return err
		}
		if !d {
			b.P(op.String())
		}
	}
	if p.Desc {
		b.P("DESC")
	}
	for _, attr := range p.Attrs {
		switch attr := attr.(type) {
		case *IndexColumnProperty:
			switch {
			// Defaults when DESC is specified.
			case p.Desc && attr.NullsFirst:
			case p.Desc && attr.NullsLast:
				b.P("NULLS LAST")
			// Defaults when DESC is not specified.
			case !p.Desc && attr.NullsLast:
			case !p.Desc && attr.NullsFirst:
				b.P("NULLS FIRST")
			}
		// Handled above.
		case *IndexOpClass, *schema.Collation:
		// Handled below.
		case *Operator:
		default:
			return fmt.Errorf("postgres: unexpected index part attribute: %T", attr)
		}
	}
	if _, isE := excludeConst(idx.Attrs); isE {
		switch op := (&Operator{}); {
		case !sqlx.Has(p.Attrs, op):
			return fmt.Errorf("missing operator for exclude constraint %q", idx.Name)
		case op.Name == "":
			return fmt.Errorf("empty operator for exclude constraint %q", idx.Name)
		default:
			b.P("WITH", op.Name)
		}
	}
	return nil
}

func (s *state) index(b *sqlx.Builder, idx *schema.Index) error {
	// Avoid appending the default method.
	if t := (IndexType{}); sqlx.Has(idx.Attrs, &t) && strings.ToUpper(t.T) != IndexTypeBTree {
		b.P("USING", t.T)
	}
	if err := s.indexParts(b, idx); err != nil {
		return err
	}
	if c := (IndexInclude{}); sqlx.Has(idx.Attrs, &c) {
		b.P("INCLUDE")
		b.Wrap(func(b *sqlx.Builder) {
			b.MapComma(c.Columns, func(i int, b *sqlx.Builder) {
				b.Ident(c.Columns[i].Name)
			})
		})
	}
	// Avoid appending the default behavior, which NULL values are distinct.
	if n := (IndexNullsDistinct{}); sqlx.Has(idx.Attrs, &n) && !n.V {
		b.P("NULLS NOT DISTINCT")
	}
	if p, ok := indexStorageParams(idx.Attrs); ok {
		b.P("WITH")
		b.Wrap(func(b *sqlx.Builder) {
			var parts []string
			if p.AutoSummarize {
				parts = append(parts, "autosummarize = true")
			}
			if p.PagesPerRange != 0 && p.PagesPerRange != defaultPagePerRange {
				parts = append(parts, fmt.Sprintf("pages_per_range = %d", p.PagesPerRange))
			}
			b.WriteString(strings.Join(parts, ", "))
		})
	}
	if p := (IndexPredicate{}); sqlx.Has(idx.Attrs, &p) {
		b.P("WHERE").P(p.P)
	}
	for _, attr := range idx.Attrs {
		switch attr.(type) {
		case *schema.Comment, *IndexType, *IndexInclude, *Constraint, *IndexPredicate, *IndexStorageParams, *IndexNullsDistinct:
		default:
			return fmt.Errorf("postgres: unexpected index attribute: %T", attr)
		}
	}
	return nil
}

func (s *state) fks(b *sqlx.Builder, fks ...*schema.ForeignKey) {
	b.MapIndent(fks, func(i int, b *sqlx.Builder) {
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

func (s *state) constraint(b *sqlx.Builder, idx *schema.Index) error {
	if _, isU := uniqueConst(idx.Attrs); isU {
		return s.unique(b, idx)
	}
	if _, isE := excludeConst(idx.Attrs); isE {
		return s.exclude(b, idx)
	}
	return fmt.Errorf("unexpected constraint type for index %q", idx.Name)
}

func (s *state) unique(b *sqlx.Builder, idx *schema.Index) error {
	c, ok := uniqueConst(idx.Attrs)
	if !ok {
		return fmt.Errorf("index %q is not a unique constraint", idx.Name)
	}
	name := c.N
	if name == "" {
		name = idx.Name
	}
	b.P("CONSTRAINT").Ident(name).P("UNIQUE")
	return s.index(b, idx)
}

func (s *state) exclude(b *sqlx.Builder, idx *schema.Index) error {
	c, ok := excludeConst(idx.Attrs)
	if !ok {
		return fmt.Errorf("index %q is not an exclude constraint", idx.Name)
	}
	name := c.N
	if name == "" {
		name = idx.Name
	}
	b.P("CONSTRAINT").Ident(name).P("EXCLUDE")
	return s.index(b, idx)
}

func (s *state) append(c ...*migrate.Change) {
	s.Changes = append(s.Changes, c...)
}

// Build instantiates a new builder and writes the given phrase to it.
func (s *state) Build(phrases ...string) *sqlx.Builder {
	return (*Driver).StmtBuilder(nil, s.PlanOptions).
		P(phrases...)
}

// skipAutoChanges filters unnecessary changes that are automatically
// happened by the database when ALTER TABLE is executed.
func skipAutoChanges(changes []schema.Change) []schema.Change {
	var (
		dropC   = make(map[string]bool)
		planned = make([]schema.Change, 0, len(changes))
	)
	for _, c := range changes {
		if c, ok := c.(*schema.DropColumn); ok {
			dropC[c.C.Name] = true
		}
	}
search:
	for _, c := range changes {
		switch c := c.(type) {
		// Indexes involving the column are automatically dropped
		// with it. This is true for multi-columns indexes as well.
		// See https://www.postgresql.org/docs/current/sql-altertable.html
		case *schema.DropIndex:
			for _, p := range c.I.Parts {
				if p.C != nil && dropC[p.C.Name] {
					continue search
				}
			}
		// Simple case for skipping constraint dropping,
		// if the child table columns were dropped.
		case *schema.DropForeignKey:
			for _, c := range c.F.Columns {
				if dropC[c.Name] {
					continue search
				}
			}
		}
		planned = append(planned, c)
	}
	return planned
}

// commentChange extracts the information for modifying a comment from the given change.
func commentChange(c schema.Change) (from, to string, err error) {
	switch c := c.(type) {
	case *schema.AddAttr:
		toC, ok := c.A.(*schema.Comment)
		if ok {
			to = toC.Text
			return
		}
		err = fmt.Errorf("unexpected AddAttr.(%T) for comment change", c.A)
	case *schema.ModifyAttr:
		fromC, ok1 := c.From.(*schema.Comment)
		toC, ok2 := c.To.(*schema.Comment)
		if ok1 && ok2 {
			from, to = fromC.Text, toC.Text
			return
		}
		err = fmt.Errorf("unsupported ModifyAttr(%T, %T) change", c.From, c.To)
	default:
		err = fmt.Errorf("unexpected change %T", c)
	}
	return
}

// checks writes the CHECK constraint to the builder.
func check(b *sqlx.Builder, c *schema.Check) {
	if c.Name != "" {
		b.P("CONSTRAINT").Ident(c.Name)
	}
	b.P("CHECK", sqlx.MayWrap(c.Expr))
	if sqlx.Has(c.Attrs, &NoInherit{}) {
		b.P("NO INHERIT")
	}
}

func quote(s string) string {
	if sqlx.IsQuoted(s, '\'') {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (s *state) createDropEnum(e *schema.EnumType) (string, string) {
	name := s.enumIdent(e)
	return s.Build("CREATE TYPE").
			P(name, "AS ENUM").
			Wrap(func(b *sqlx.Builder) {
				b.MapComma(e.Values, func(i int, b *sqlx.Builder) {
					b.WriteString(quote(e.Values[i]))
				})
			}).
			String(),
		s.Build("DROP TYPE").P(name).String()
}

func (s *state) enumIdent(e *schema.EnumType) string {
	return s.typeIdent(e.Schema, e.T)
}

func (s *state) domainIdent(d *DomainType) string {
	return s.typeIdent(d.Schema, d.T)
}

func (s *state) compositeIdent(c *CompositeType) string {
	return s.typeIdent(c.Schema, c.T)
}

func (s *state) typeIdent(ns *schema.Schema, name string) string {
	switch {
	// In case the plan uses a specific schema qualifier.
	case s.SchemaQualifier != nil:
		if *s.SchemaQualifier != "" {
			return fmt.Sprintf("%q.%q", *s.SchemaQualifier, name)
		}
	case ns != nil && ns.Name != "":
		return fmt.Sprintf("%q.%q", ns.Name, name)
	}
	return strconv.Quote(name)
}

// schemaPrefix returns the schema prefix based on the planner config.
func (s *state) schemaPrefix(ns *schema.Schema) string {
	switch {
	case s.SchemaQualifier != nil:
		// In case the qualifier is empty, ignore.
		if *s.SchemaQualifier != "" {
			return fmt.Sprintf("%q.", *s.SchemaQualifier)
		}
	case ns != nil && ns.Name != "":
		return fmt.Sprintf("%q.", ns.Name)
	}
	return ""
}

// formatType formats the type but takes into account the qualifier.
func (s *state) formatType(t schema.Type) (string, error) {
	switch t := t.(type) {
	case *schema.EnumType:
		return s.enumIdent(t), nil
	case *DomainType:
		return s.domainIdent(t), nil
	case *CompositeType:
		return s.compositeIdent(t), nil
	case *ArrayType:
		switch t := t.Type.(type) {
		case *schema.EnumType:
			return s.enumIdent(t) + "[]", nil
		case *DomainType:
			return s.domainIdent(t) + "[]", nil
		case *CompositeType:
			return s.compositeIdent(t) + "[]", nil
		}
	}
	return FormatType(t)
}

func pkName(t *schema.Table, pk *schema.Index) string {
	if pk.Name != "" {
		return pk.Name
	}
	// The default naming for primary-key constraints is <Table>_pkey.
	// See: the ChooseIndexName function in PostgreSQL for more reference.
	return t.Name + "_pkey"
}

// dropConst indicates if the given change is a constraint drop.
func dropConst(c schema.Change) bool {
	switch c.(type) {
	case *schema.DropIndex, *schema.DropPrimaryKey, *schema.DropCheck, *schema.DropForeignKey:
		return true
	default:
		return false
	}
}
