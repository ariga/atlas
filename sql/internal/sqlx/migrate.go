// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/schema"
)

type (
	// A Migrate provides a generic schema.Execer for executing schema changes.
	//
	// The MigrateDriver is required for supporting database/dialect specific
	// SQL statements, like writing table or column specific attributes.
	Migrate struct {
		MigrateDriver
	}

	// A MigrateDriver wraps all required methods for building the SQL statements
	// for applying the changeset on the database. See sql/schema/mysql/migrate.go
	// for an implementation example.
	MigrateDriver interface {
		schema.ExecQuerier

		// QuoteChar returns the character that is used for quoting SQL identifiers.
		// For example, '`' in MySQL, and '"' in PostgreSQL.
		QuoteChar() byte

		// WriteTableAttr writes the given table attribute
		// to the SQL statement through the given builder.
		WriteTableAttr(*Builder, schema.Attr)

		// WriteColumnAttr writes the given column attribute
		// to the SQL statement through the given builder.
		WriteColumnAttr(*Builder, schema.Attr)

		// WriteIndexAttr writes the given index attribute
		// to the SQL statement through the given builder.
		WriteIndexAttr(*Builder, schema.Attr)

		// WriteIndexPartAttr writes the given index-part attribute
		// to the SQL statement through the given builder.
		WriteIndexPartAttr(*Builder, schema.Attr)
	}
)

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (m *Migrate) Exec(ctx context.Context, changes []schema.Change) (err error) {
	planned := DetachCycles(changes)
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
func (m *Migrate) addTable(ctx context.Context, add *schema.AddTable) error {
	b := m.Build("CREATE TABLE").Table(add.T)
	b.Wrap(func(b *Builder) {
		b.MapComma(add.T.Columns, func(i int, b *Builder) {
			m.column(b, add.T.Columns[i])
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().P("PRIMARY KEY")
			m.indexParts(b, pk.Parts)
		}
		if len(add.T.Indexes) > 0 {
			b.Comma()
		}
		b.MapComma(add.T.Indexes, func(i int, b *Builder) {
			idx := add.T.Indexes[i]
			if idx.Unique {
				b.P("UNIQUE")
			}
			b.P("INDEX").Ident(idx.Name)
			m.indexParts(b, idx.Parts)
			attrs(b, m.WriteIndexAttr, idx.Attrs...)
		})
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			m.fks(b, add.T.ForeignKeys...)
		}
	})
	attrs(b, m.WriteTableAttr, add.T.Attrs...)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

// dropTable builds and executes the query for dropping a table from a schema.
func (m *Migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	b := m.Build("DROP TABLE").Table(drop.T)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	return nil
}

// modifyTable builds and executes the queries for bringing the table into its modified state.
func (m *Migrate) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	var changes [2][]schema.Change
	for _, change := range modify.Changes {
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
func (m *Migrate) alterTable(ctx context.Context, t *schema.Table, changes []schema.Change) error {
	b := m.Build("ALTER TABLE").Table(t)
	b.MapComma(changes, func(i int, b *Builder) {
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
			attrs(b, m.WriteIndexAttr, change.I.Attrs...)
		case *schema.DropIndex:
			b.P("DROP INDEX").Ident(change.I.Name)
		case *schema.AddForeignKey:
			b.P("ADD")
			m.fks(b, change.F)
		case *schema.DropForeignKey:
			b.P("DROP FOREIGN KEY").Ident(change.F.Symbol)
		case *schema.AddAttr:
			m.WriteTableAttr(b, change.A)
		case *schema.ModifyAttr:
			m.WriteTableAttr(b, change.To)
		}
	})
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("alter table: %w", err)
	}
	return nil
}

func (m *Migrate) column(b *Builder, c *schema.Column) {
	b.Ident(c.Name).P(c.Type.Raw)
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	if x, ok := c.Default.(*schema.RawExpr); ok {
		b.P("DEFAULT", x.X)
	}
	attrs(b, m.WriteColumnAttr, c.Attrs...)
}

func (m *Migrate) indexParts(b *Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *Builder) {
		b.MapComma(parts, func(i int, b *Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
			attrs(b, m.WriteIndexPartAttr, parts[i].Attrs...)
		})
	})
}

func (m *Migrate) fks(b *Builder, fks ...*schema.ForeignKey) {
	b.MapComma(fks, func(i int, b *Builder) {
		fk := fks[i]
		if fk.Symbol != "" {
			b.P("CONSTRAINT").Ident(fk.Symbol)
		}
		b.P("FOREIGN KEY")
		b.Wrap(func(b *Builder) {
			b.MapComma(fk.Columns, func(i int, b *Builder) {
				b.Ident(fk.Columns[i].Name)
			})
		})
		b.P("REFERENCES").Table(fk.RefTable)
		b.Wrap(func(b *Builder) {
			b.MapComma(fk.RefColumns, func(i int, b *Builder) {
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

// Build instantiates a new builder and writes the given phrase to it.
func (m *Migrate) Build(phrase string) *Builder {
	b := &Builder{QuoteChar: m.QuoteChar()}
	return b.P(phrase)
}

func attrs(b *Builder, fn func(*Builder, schema.Attr), attrs ...schema.Attr) {
	for _, a := range attrs {
		fn(b, a)
	}
}
