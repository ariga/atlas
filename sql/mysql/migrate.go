// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A migrate provides migration capabilities for schema elements.
type migrate struct{ *Driver }

// Migrate returns a MySQL schema executor.
func (d *Driver) Migrate() schema.Execer {
	return &migrate{Driver: d}
}

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (m *migrate) Exec(ctx context.Context, changes []schema.Change) (err error) {
	planned := sqlx.DetachCycles(changes)
	for _, c := range planned {
		switch c := c.(type) {
		case *schema.AddTable:
			err = m.addTable(ctx, c)
		case *schema.DropTable:
			err = m.dropTable(ctx, c)
		case *schema.ModifyTable:
			err = m.modifyTable(ctx, c)
		default:
			err = fmt.Errorf("unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return
}

// addTable builds and executes the query for creating a table in a schema.
func (m *migrate) addTable(ctx context.Context, add *schema.AddTable) error {
	b := Build("CREATE TABLE").Table(add.T)
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(add.T.Columns, func(i int, b *sqlx.Builder) {
			m.column(b, add.T.Columns[i])
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().P("PRIMARY KEY")
			m.indexParts(b, pk.Parts)
			m.attr(b, pk.Attrs...)
		}
		if len(add.T.Indexes) > 0 {
			b.Comma()
		}
		b.MapComma(add.T.Indexes, func(i int, b *sqlx.Builder) {
			idx := add.T.Indexes[i]
			if idx.Unique {
				b.P("UNIQUE")
			}
			b.P("INDEX").Ident(idx.Name)
			m.indexParts(b, idx.Parts)
			m.attr(b, idx.Attrs...)
		})
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			m.fks(b, add.T.ForeignKeys...)
		}
	})
	m.tableAttr(b, add.T.Attrs...)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

// dropTable builds and executes the query for dropping a table from a schema.
func (m *migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	b := Build("DROP TABLE").Table(drop.T)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	return nil
}

// modifyTable builds and executes the queries for bringing the table into its modified state.
func (m *migrate) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	var changes [2][]schema.Change
	for _, change := range skipAutoChanges(modify.Changes) {
		switch change := change.(type) {
		// Constraints should be dropped before dropping columns, because if a column
		// is a part of multi-column constraints (like, unique index), ALTER TABLE
		// might fail if the intermediate state violates the constraints.
		case *schema.DropIndex:
			changes[0] = append(changes[0], change)
		case *schema.ModifyForeignKey:
			// Foreign-key modification is translated into 2 steps.
			// Dropping the current foreign key and creating a new one.
			changes[0] = append(changes[0], &schema.DropForeignKey{
				F: change.From,
			})
			// Drop the auto-created index for referenced if the reference was changed.
			if change.Change.Is(schema.ChangeRefTable | schema.ChangeRefColumn) {
				changes[0] = append(changes[0], &schema.DropIndex{
					I: &schema.Index{
						Name:  change.From.Symbol,
						Table: modify.T,
					},
				})
			}
			changes[1] = append(changes[1], &schema.AddForeignKey{
				F: change.To,
			})
		// Index modification requires rebuilding the index.
		case *schema.ModifyIndex:
			changes[0] = append(changes[0], &schema.DropIndex{
				I: change.From,
			})
			changes[1] = append(changes[1], &schema.AddIndex{
				I: change.To,
			})
		case *schema.DropAttr:
			return fmt.Errorf("unsupported change type: %T", change)
		default:
			changes[1] = append(changes[1], change)
		}
	}
	for i := range changes {
		if len(changes[i]) > 0 {
			if err := m.alterTable(ctx, modify.T, changes[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (m *migrate) alterTable(ctx context.Context, t *schema.Table, changes []schema.Change) error {
	b := Build("ALTER TABLE").Table(t)
	b.MapComma(changes, func(i int, b *sqlx.Builder) {
		switch change := changes[i].(type) {
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			m.column(b, change.C)
		case *schema.ModifyColumn:
			b.P("MODIFY COLUMN")
			m.column(b, change.To)
		case *schema.DropColumn:
			b.P("DROP COLUMN").Ident(change.C.Name)
		case *schema.AddIndex:
			b.P("ADD")
			if change.I.Unique {
				b.P("UNIQUE")
			}
			b.P("INDEX").Ident(change.I.Name)
			m.indexParts(b, change.I.Parts)
			m.attr(b, change.I.Attrs...)
		case *schema.DropIndex:
			b.P("DROP INDEX").Ident(change.I.Name)
		case *schema.AddForeignKey:
			b.P("ADD")
			m.fks(b, change.F)
		case *schema.DropForeignKey:
			b.P("DROP FOREIGN KEY").Ident(change.F.Symbol)
		case *schema.AddAttr:
			m.tableAttr(b, change.A)
		case *schema.ModifyAttr:
			m.tableAttr(b, change.To)
		}
	})
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("alter table: %w", err)
	}
	return nil
}

func (m *migrate) column(b *sqlx.Builder, c *schema.Column) {
	b.Ident(c.Name).P(m.mustFormat(c.Type.Type))
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	if x, ok := c.Default.(*schema.RawExpr); ok {
		v := x.X
		// Ensure string default values are quoted.
		if _, ok := c.Type.Type.(*schema.StringType); ok {
			v = quote(v)
		}
		b.P("DEFAULT", v)
	}
	for _, a := range c.Attrs {
		switch a := a.(type) {
		case *OnUpdate:
			b.P("ON UPDATE", a.A)
		case *AutoIncrement:
			b.P("AUTO_INCREMENT")
			if a.V != 0 {
				b.P(strconv.FormatInt(a.V, 10))
			}
		default:
			m.attr(b, a)
		}
	}
}

func (m *migrate) indexParts(b *sqlx.Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(parts, func(i int, b *sqlx.Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
			for _, a := range parts[i].Attrs {
				if c, ok := a.(*schema.Collation); ok && c.V == "D" {
					b.P("DESC")
				}
			}
		})
	})
}

func (m *migrate) fks(b *sqlx.Builder, fks ...*schema.ForeignKey) {
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

// tableAttr writes the given table attribute to the SQL
// statement builder when a table is created or altered.
func (m *migrate) tableAttr(b *sqlx.Builder, attrs ...schema.Attr) {
	for _, a := range attrs {
		switch a := a.(type) {
		case *AutoIncrement:
			b.P("AUTO_INCREMENT")
			if a.V != 0 {
				b.P(strconv.FormatInt(a.V, 10))
			}
		case *schema.Charset:
			b.P("CHARACTER SET", a.V)
		default:
			m.attr(b, a)
		}
	}
}

func (*migrate) attr(b *sqlx.Builder, attrs ...schema.Attr) {
	for _, a := range attrs {
		switch a := a.(type) {
		case *schema.Collation:
			b.P("COLLATE", a.V)
		case *schema.Comment:
			b.P("COMMENT", quote(a.Text))
		}
	}
}

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '`'}
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
search:
	for i, c := range changes {
		// Simple case for skipping key dropping, if its columns are dropped.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html#alter-table-add-drop-column
		c, ok := c.(*schema.DropIndex)
		if !ok {
			continue
		}
		for _, p := range c.I.Parts {
			if p.C == nil || !dropC[p.C.Name] {
				continue search
			}
		}
		changes = append(changes[:i], changes[i+1:]...)
	}
	return changes
}

func quote(s string) string {
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") ||
		strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s
	}
	return strconv.Quote(s)
}
