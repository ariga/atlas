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

// A migrate provides migration capabilities for schema elements.
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
	// Create enum types before using them in the `CREATE TABLE` statement.
	if err := m.addTypes(ctx, add.T.Columns...); err != nil {
		return err
	}
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

// modifyTable builds and executes the queries for bringing the table into its modified state.
func (m *migrate) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	var (
		changes     []schema.Change
		addI, dropI []*schema.Index
	)
	for _, change := range modify.Changes {
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
			if err := m.addTypes(ctx, change.C); err != nil {
				return err
			}
			changes = append(changes, change)
		case *schema.ModifyColumn:
			from, ok1 := change.From.Type.Type.(*schema.EnumType)
			to, ok2 := change.To.Type.Type.(*schema.EnumType)
			switch {
			// Enum was added.
			case !ok1 && ok2:
				if err := m.addTypes(ctx, change.To); err != nil {
					return err
				}
			// Enum was changed.
			case ok1 && ok2 && from.T == to.T:
				if err := m.alterType(ctx, from, to); err != nil {
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
	if err := m.dropIndexes(ctx, dropI...); err != nil {
		return err
	}
	if len(changes) > 0 {
		if err := m.alterTable(ctx, modify.T, changes); err != nil {
			return err
		}
	}
	return m.addIndexes(ctx, modify.T, addI...)
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (m *migrate) alterTable(ctx context.Context, t *schema.Table, changes []schema.Change) error {
	b := Build("ALTER TABLE").Table(t)
	b.MapComma(changes, func(i int, b *sqlx.Builder) {
		switch change := changes[i].(type) {
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			m.column(b, change.C)
		case *schema.DropColumn:
			b.P("DROP COLUMN").Ident(change.C.Name)
		case *schema.ModifyColumn:
			m.alterColumn(b, change)
		case *schema.AddForeignKey:
			b.P("ADD")
			m.fks(b, change.F)
		case *schema.DropForeignKey:
			b.P("DROP FOREIGN KEY").Ident(change.F.Symbol)
		}
	})
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("alter table: %w", err)
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

func (m *migrate) dropIndexes(ctx context.Context, indexes ...*schema.Index) error {
	for _, idx := range indexes {
		b := Build("DROP INDEX").Ident(idx.Name)
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("drop index: %w", err)
		}
	}
	return nil
}

func (m *migrate) addTypes(ctx context.Context, columns ...*schema.Column) error {
	for _, c := range columns {
		e, ok := c.Type.Type.(*schema.EnumType)
		if !ok {
			continue
		}
		if e.T == "" {
			return fmt.Errorf("missing enum name for column %q", c.Name)
		}
		c.Type.Raw = e.T
		if exists, err := m.enumExists(ctx, e.T); err != nil {
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
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("create enum type: %w", err)
		}
	}
	return nil
}

func (m *migrate) alterType(ctx context.Context, from, to *schema.EnumType) error {
	if len(from.Values) > len(to.Values) {
		return fmt.Errorf("dropping enum (%q) value is not supported", from.T)
	}
	for i := range from.Values {
		if from.Values[i] != to.Values[i] {
			return fmt.Errorf("replacing or reordering enum (%q) value is not supported: %q != %q", to.T, to.Values, from.Values)
		}
	}
	for _, v := range to.Values[len(from.Values):] {
		b := Build("ALTER TYPE").Ident(from.T).P("ADD VALUE", "'"+v+"'")
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("adding a new value %q for enum %q: %w", v, to.T, err)
		}
	}
	return nil
}

func (m *migrate) enumExists(ctx context.Context, name string) (bool, error) {
	rows, err := m.Driver.QueryContext(ctx, "SELECT * FROM pg_type WHERE typname = $1 AND typtype = 'e'", name)
	if err != nil {
		return false, fmt.Errorf("check index existance: %w", err)
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
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
	b.Ident(c.Name).P(m.mustFormat(c.Type.Type))
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

func (m *migrate) alterColumn(b *sqlx.Builder, c *schema.ModifyColumn) {
	for k := c.Change; !k.Is(schema.NoChange); {
		b.P("ALTER COLUMN").Ident(c.To.Name)
		switch {
		case k.Is(schema.ChangeType):
			b.P("TYPE").P(m.mustFormat(c.To.Type.Type))
			if collate := (schema.Collation{}); sqlx.Has(c.To.Attrs, &collate) {
				b.P("COLLATE", collate.V)
			}
			k &= ^schema.ChangeType
		case k.Is(schema.ChangeNull) && c.To.Type.Null:
			b.P("DROP NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeNull) && !c.To.Type.Null:
			b.P("SET NOT NULL")
			k &= ^schema.ChangeNull
		case k.Is(schema.ChangeDefault) && c.To.Default == nil:
			b.P("DROP DEFAULT")
			k &= ^schema.ChangeDefault
		case k.Is(schema.ChangeDefault) && c.To.Default != nil:
			x, ok := c.To.Default.(*schema.RawExpr)
			if !ok {
				panic(fmt.Sprintf("unexpected column default: %T", c.To.Default))
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
