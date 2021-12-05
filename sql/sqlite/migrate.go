// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A migrate provides migration capabilities for schema elements.
type migrate struct{ *Driver }

// Migrate returns a SQLite schema executor.
func (d *Driver) Migrate() schema.Execer {
	return &migrate{Driver: d}
}

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (m *migrate) Exec(ctx context.Context, changes []schema.Change) (err error) {
	for _, c := range changes {
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
	return nil
}

// addTable builds and executes the query for creating a table in a schema.
func (m *migrate) addTable(ctx context.Context, add *schema.AddTable) error {
	b := Build("CREATE TABLE").Ident(add.T.Name)
	b.Wrap(func(b *sqlx.Builder) {
		b.MapComma(add.T.Columns, func(i int, b *sqlx.Builder) {
			m.column(b, add.T.Columns[i])
		})
		// Primary keys with auto-increment are inlined on the column definition.
		if pk := add.T.PrimaryKey; pk != nil && !autoincPK(pk) {
			b.Comma().P("PRIMARY KEY")
			m.indexParts(b, pk.Parts)
		}
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			m.fks(b, add.T.ForeignKeys...)
		}
	})
	if p := (WithoutRowID{}); sqlx.Has(add.T.Attrs, &p) {
		b.P("WITHOUT ROWID")
	}
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return m.addIndexes(ctx, add.T, add.T.Indexes...)
}

// dropTable builds and executes the query for dropping a table from a schema.
func (m *migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	if err := m.skipConstraints(ctx, func(ctx context.Context) error {
		_, err := m.ExecContext(ctx, Build("DROP TABLE").Ident(drop.T.Name).String())
		return err
	}); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	return nil
}

// modifyTable builds and executes the queries for bringing the table into its modified state.
// If the modification contains changes that are not index creation/deletion or a simple column
// addition, the changes are applied using a temporary table following the procedure mentioned
// in: https://www.sqlite.org/lang_altertable.html#making_other_kinds_of_table_schema_changes.
func (m *migrate) modifyTable(ctx context.Context, modify *schema.ModifyTable) error {
	if alterable(modify) {
		return m.alterTable(ctx, modify)
	}
	if err := m.skipConstraints(ctx, func(ctx context.Context) error {
		newT := *modify.T
		indexes := newT.Indexes
		newT.Indexes = nil
		newT.Name = "new_" + newT.Name
		// Create a new table with a temporary name, and copy the existing rows to it.
		if err := m.addTable(ctx, &schema.AddTable{T: &newT}); err != nil {
			return err
		}
		if err := m.copyRows(ctx, modify.T, &newT, modify.Changes); err != nil {
			return err
		}
		// Drop the current table, and rename the new one to its real name.
		stmt := Build("DROP TABLE").Ident(modify.T.Name).String()
		if _, err := m.ExecContext(ctx, stmt); err != nil {
			return err
		}
		stmt = Build("ALTER TABLE").Ident(newT.Name).P("RENAME TO").Ident(modify.T.Name).String()
		if _, err := m.ExecContext(ctx, stmt); err != nil {
			return err
		}
		return m.addIndexes(ctx, modify.T, indexes...)
	}); err != nil {
		return fmt.Errorf("modify table: %w", err)
	}
	return nil
}

func (m *migrate) column(b *sqlx.Builder, c *schema.Column) {
	b.Ident(c.Name).P(mustFormat(c.Type.Type))
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	if x, ok := c.Default.(*schema.RawExpr); ok {
		b.P("DEFAULT", x.X)
	}
	if sqlx.Has(c.Attrs, &AutoIncrement{}) {
		b.P("PRIMARY KEY AUTOINCREMENT")
	}
}

func (m *migrate) addIndexes(ctx context.Context, t *schema.Table, indexes ...*schema.Index) error {
	for _, idx := range indexes {
		// PRIMARY KEY or UNIQUE columns automatically create indexes with the generated name.
		// See: sqlite/build.c#sqlite3CreateIndex. Therefore, we ignore such PKs, but create
		// the inlined UNIQUE constraints manually with custom name, because SQLite does not
		// allow creating indexes with such names manually. Note, this case is possible if
		// "apply" schema that was inspected from the database as-is.
		if strings.HasPrefix(idx.Name, "sqlite_autoindex") {
			if i := (IndexOrigin{}); sqlx.Has(idx.Attrs, &i) && i.O == "p" {
				continue
			}
			// Use the following format: <Table>_<Columns>.
			names := make([]string, len(idx.Parts)+1)
			names[0] = t.Name
			for i, p := range idx.Parts {
				if p.C == nil {
					return fmt.Errorf("unexpected index part %s (%d)", idx.Name, i)
				}
				names[i+1] = p.C.Name
			}
			idx.Name = strings.Join(names, "_")
		}
		b := Build("CREATE")
		if idx.Unique {
			b.P("UNIQUE")
		}
		b.P("INDEX")
		if idx.Name != "" {
			b.Ident(idx.Name)
		}
		b.P("ON").Ident(t.Name)
		m.indexParts(b, idx.Parts)
		if p := (IndexPredicate{}); sqlx.Has(idx.Attrs, &p) {
			b.P("WHERE").P(p.P)
		}
		if _, err := m.ExecContext(ctx, b.String()); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
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
		b.P("REFERENCES").Ident(fk.RefTable.Name)
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

// skipConstraints runs f without enforcement on the foreign keys constraints if
// they are enabled.
func (m *migrate) skipConstraints(ctx context.Context, f func(context.Context) error) (err error) {
	var enabled bool
	if err = m.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return fmt.Errorf("checking foreign_keys enforcement: %w", err)
	}
	if !enabled {
		return f(ctx)
	}
	if _, err := m.ExecContext(ctx, "PRAGMA foreign_keys = off"); err != nil {
		return fmt.Errorf("disabling the enforcement of foreign-keys constraints: %w", err)
	}
	defer func() {
		if _, rerr := m.ExecContext(ctx, "PRAGMA foreign_keys = on"); rerr != nil && err == nil {
			err = rerr
		}
	}()
	return f(ctx)
}

func (m *migrate) copyRows(ctx context.Context, from *schema.Table, to *schema.Table, changes []schema.Change) error {
	var (
		args       []interface{}
		fromC, toC []string
	)
	for _, column := range to.Columns {
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
		// We expect that new columns are added with DEFAULT values,
		// or defined as nullable if the table is not empty.
		case *schema.AddColumn:
		// Column modification requires special handling if it was
		// converted from nullable to non-nullable with default value.
		case *schema.ModifyColumn:
			toC = append(toC, column.Name)
			if !column.Type.Null && column.Default != nil && change.Change.Is(schema.ChangeNull|schema.ChangeDefault) {
				fromC = append(fromC, fmt.Sprintf("IFNULL(`%[1]s`, ?) AS `%[1]s`", column.Name))
				args = append(args, column.Default.(*schema.RawExpr).X)
			} else {
				fromC = append(fromC, column.Name)
			}
		// Columns without changes, should transfer as-is.
		case nil:
			toC = append(toC, column.Name)
			fromC = append(fromC, column.Name)
		}
	}
	stmt := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", to.Name, strings.Join(toC, ", "), strings.Join(fromC, ", "), from.Name)
	if _, err := m.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("copy rows from %q to %q: %w", from.Name, to.Name, err)
	}
	return nil
}

// alterTable alters the table with the given changes. Assuming the changes are "alterable".
func (m *migrate) alterTable(ctx context.Context, modify *schema.ModifyTable) error {
	for _, change := range modify.Changes {
		switch change := change.(type) {
		case *schema.AddIndex:
			if err := m.addIndexes(ctx, modify.T, change.I); err != nil {
				return err
			}
		case *schema.DropIndex:
			b := Build("DROP INDEX").Ident(change.I.Name)
			if _, err := m.ExecContext(ctx, b.String()); err != nil {
				return fmt.Errorf("drop index %q to table: %q", change.I.Name, modify.T.Name)
			}
		case *schema.AddColumn:
			b := Build("ALTER TABLE").Ident(modify.T.Name).P("ADD COLUMN")
			m.column(b, change.C)
			if _, err := m.ExecContext(ctx, b.String()); err != nil {
				return fmt.Errorf("add column %q to table: %q", change.C.Name, modify.T.Name)
			}
		default:
			return fmt.Errorf("unexpected change in alter table: %T", change)
		}
	}
	return nil
}

func alterable(modify *schema.ModifyTable) bool {
	for _, change := range modify.Changes {
		switch change := change.(type) {
		case *schema.DropIndex, *schema.AddIndex:
		case *schema.AddColumn:
			if len(change.C.Indexes) > 0 || len(change.C.ForeignKeys) > 0 || change.C.Default != nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func autoincPK(pk *schema.Index) bool {
	return sqlx.Has(pk.Attrs, &AutoIncrement{}) ||
		len(pk.Parts) == 1 && pk.Parts[0].C != nil && sqlx.Has(pk.Parts[0].C.Attrs, &AutoIncrement{})
}

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *sqlx.Builder {
	b := &sqlx.Builder{QuoteChar: '`'}
	return b.P(phrase)
}
