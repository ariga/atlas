// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A Migrate provides migration capabilities for schema elements.
type migrate struct{ *Driver }

// Migrate returns a PostgreSQL schema executor.
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
		}
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			m.fks(b, add.T.ForeignKeys...)
		}
	})
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	if err := m.addIndexes(ctx, add.T, add.T.Indexes...); err != nil {
		return err
	}
	return m.addComments(ctx, add.T)
}

// dropTable builds and executes the query for dropping a table from a schema.
func (m *migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	b := Build("DROP TABLE").Table(drop.T)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	return nil
}

func (m *migrate) addComments(ctx context.Context, t *schema.Table) error {
	var c schema.Comment
	if sqlx.Has(t.Attrs, &c) {
		b := Build("COMMENT ON TABLE").Table(t).P("IS", "'"+c.Text+"'")
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("add comment to table: %w", err)
		}
	}
	for i := range t.Columns {
		if sqlx.Has(t.Columns[i].Attrs, &c) {
			b := Build("COMMENT ON COLUMN").Table(t)
			b.WriteByte('.')
			b.Ident(t.Columns[i].Name).P("IS", "'"+c.Text+"'")
			if _, err := m.ExecContext(ctx, b.String()); err != nil {
				return fmt.Errorf("add comment to column: %w", err)
			}
		}
	}
	for i := range t.Indexes {
		if sqlx.Has(t.Indexes[i].Attrs, &c) {
			b := Build("COMMENT ON INDEX").Ident(t.Indexes[i].Name).P("IS", "'"+c.Text+"'")
			if _, err := m.ExecContext(ctx, b.String()); err != nil {
				return fmt.Errorf("add comment to index: %w", err)
			}
		}
	}
	return nil
}

func (m *migrate) addIndexes(ctx context.Context, t *schema.Table, indexes ...*schema.Index) error {
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
		m.indexParts(b, idx.Parts)
		m.indexAttrs(b, idx.Attrs)
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}
func (m *migrate) column(b *sqlx.Builder, c *schema.Column) {
	b.Ident(c.Name).P(c.Type.Raw)
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
			panic(fmt.Sprintf("unexpected collumn attribute: %T", attr))
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
			for _, attr := range parts[i].Attrs {
				m.partAttr(b, attr)
			}
		})
	})
}

func (m *migrate) partAttr(b *sqlx.Builder, attr schema.Attr) {
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

func (m *migrate) indexAttrs(b *sqlx.Builder, attrs []schema.Attr) {
	for _, attr := range attrs {
		switch attr := attr.(type) {
		case *schema.Comment:
		case *IndexPredicate:
			b.P("WHERE").P(attr.P)
		default:
			panic(fmt.Sprintf("unexpected index attribute: %T", attr))
		}
	}
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

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '"'}
	return b.P(phrase)
}
