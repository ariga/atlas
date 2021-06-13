package mysql

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// A Migrate provides migration capabilities for schema elements.
type Migrate struct{ *Driver }

// Exec executes the changes on the database. An error is returned
// if one of the operations fail, or a change is not supported.
func (m *Migrate) Exec(ctx context.Context, changes []schema.Change) (err error) {
	for _, c := range changes {
		switch c := c.(type) {
		case *schema.AddTable:
			err = m.addTable(ctx, c)
		case *schema.DropTable:
			err = m.dropTable(ctx, c)
		case *schema.ModifyTable:
			err = m.modifyTable(ctx, c)
		default:
			err = fmt.Errorf("mysql: unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return
}

// addTable builds and executes the query for creating a table in a schema.
func (m *Migrate) addTable(ctx context.Context, add *schema.AddTable) error {
	b := Build("CREATE TABLE").Table(add.T)
	b.Wrap(func(b *Builder) {
		b.MapComma(add.T.Columns, func(i int, b *Builder) {
			column(b, add.T.Columns[i])
		})
		if pk := add.T.PrimaryKey; pk != nil {
			b.Comma().P("PRIMARY KEY")
			indexParts(b, pk.Parts)
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
			indexParts(b, idx.Parts)
			attrs(b, idx.Attrs...)
		})
		if len(add.T.ForeignKeys) > 0 {
			b.Comma()
			fks(b, add.T.ForeignKeys...)
		}
	})
	attrs(b, add.T.Attrs...)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: create table: %w", err)
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
			return fmt.Errorf("mysql: unsupported change type: %T", change)
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

// dropTable builds and executes the query for dropping a table from a schema.
func (m *Migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	b := Build("DROP TABLE").Table(drop.T)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: drop table: %w", err)
	}
	return nil
}

// alterTable modifies the given table by executing on it a list of changes in one SQL statement.
func (m *Migrate) alterTable(ctx context.Context, t *schema.Table, changes []schema.Change) error {
	b := Build("ALTER TABLE").Table(t)
	b.MapComma(changes, func(i int, b *Builder) {
		switch change := changes[i].(type) {
		case *schema.AddColumn:
			b.P("ADD COLUMN")
			column(b, change.C)
		case *schema.ModifyColumn:
			b.P("MODIFY COLUMN")
			column(b, change.To)
		case *schema.DropColumn:
			b.P("DROP COLUMN").Ident(change.C.Name)
		case *schema.AddIndex:
			b.P("ADD")
			if change.I.Unique {
				b.P("UNIQUE")
			}
			b.P("INDEX").Ident(change.I.Name)
			indexParts(b, change.I.Parts)
			attrs(b, change.I.Attrs...)
		case *schema.DropIndex:
			b.P("DROP INDEX").Ident(change.I.Name)
		case *schema.AddForeignKey:
			b.P("ADD")
			fks(b, change.F)
		case *schema.DropForeignKey:
			b.P("DROP FOREIGN KEY").Ident(change.F.Symbol)
		case *schema.AddAttr:
			attrs(b, change.A)
		case *schema.ModifyAttr:
			attrs(b, change.To)
		}
	})
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: alter table: %w", err)
	}
	return nil
}

func fks(b *Builder, fks ...*schema.ForeignKey) {
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

func column(b *Builder, c *schema.Column) {
	b.Ident(c.Name).P(c.Type.Raw)
	if !c.Type.Null {
		b.P("NOT")
	}
	b.P("NULL")
	if x, ok := c.Default.(*schema.RawExpr); ok {
		b.P("DEFAULT", x.X)
	}
	attrs(b, c.Attrs...)
}

func indexParts(b *Builder, parts []*schema.IndexPart) {
	b.Wrap(func(b *Builder) {
		b.MapComma(parts, func(i int, b *Builder) {
			switch part := parts[i]; {
			case part.C != nil:
				b.Ident(part.C.Name)
			case part.X != nil:
				b.WriteString(part.X.(*schema.RawExpr).X)
			}
			if indexCollation(parts[i].Attrs).V == "D" {
				b.P("DESC")
			}
		})
	})
}

func attrs(b *Builder, attrs ...schema.Attr) {
	for i := range attrs {
		switch a := attrs[i].(type) {
		case *OnUpdate:
			b.P("ON UPDATE", a.A)
		case *AutoIncrement:
			b.P("AUTO_INCREMENT")
			if a.V != 0 {
				b.P(strconv.FormatInt(a.V, 10))
			}
		case *schema.Charset:
			b.P("CHARACTER SET", a.V)
		case *schema.Collation:
			b.P("COLLATE", a.V)
		case *schema.Comment:
			b.P("COMMENT", "'"+strings.ReplaceAll(a.Text, "'", "\\'")+"'")
		}
	}
}

// A Builder provides a syntactic sugar API for writing SQL statements.
type Builder struct{ bytes.Buffer }

// Build instantiates a new builder and writes the given phrase to it.
func Build(phrase string) *Builder {
	b := &Builder{}
	return b.P(phrase)
}

// P writes a list of phrases to the builder separated and
// suffixed with whitespace.
func (b *Builder) P(phrases ...string) *Builder {
	for _, p := range phrases {
		if p == "" {
			continue
		}
		if b.Len() > 0 && b.lastByte() != ' ' {
			b.WriteByte(' ')
		}
		b.WriteString(p)
		if p[len(p)-1] != ' ' {
			b.WriteByte(' ')
		}
	}
	return b
}

// Ident writes the given string quoted as an SQL identifier.
func (b *Builder) Ident(s string) *Builder {
	if s != "" {
		b.WriteByte('`')
		b.WriteString(s)
		b.WriteByte('`')
		b.WriteByte(' ')
	}
	return b
}

// Table writes the table identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) Table(t *schema.Table) *Builder {
	if t.Schema != nil {
		b.Ident(t.Schema.Name)
		b.rewriteLastByte('.')
	}
	b.Ident(t.Name)
	return b
}

// Comma writes a comma. If the current buffer ends
// with whitespace, it will be replaced instead.
func (b *Builder) Comma() *Builder {
	if b.lastByte() == ' ' {
		b.rewriteLastByte(',')
		b.WriteByte(' ')
	} else {
		b.WriteString(", ")
	}
	return b
}

// MapComma maps the slice x using the function f and joins the result with
// a comma separating between the written elements.
func (b *Builder) MapComma(x interface{}, f func(i int, b *Builder)) *Builder {
	s := reflect.ValueOf(x)
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			b.Comma()
		}
		f(i, b)
	}
	return b
}

// Wrap wraps the written string with parentheses.
func (b *Builder) Wrap(f func(b *Builder)) *Builder {
	b.WriteByte('(')
	f(b)
	if b.lastByte() != ' ' {
		b.WriteByte(')')
	} else {
		b.rewriteLastByte(')')
	}
	return b
}

// String overrides the Buffer.String method and ensure no spaces pad the returned statement.
func (b *Builder) String() string {
	return strings.TrimSpace(b.Buffer.String())
}

func (b *Builder) lastByte() byte {
	if b.Len() == 0 {
		return 0
	}
	buf := b.Buffer.Bytes()
	return buf[len(buf)-1]
}

func (b *Builder) rewriteLastByte(c byte) {
	if b.Len() == 0 {
		return
	}
	buf := b.Buffer.Bytes()
	buf[len(buf)-1] = c
}
