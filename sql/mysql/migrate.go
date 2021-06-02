package mysql

import (
	"context"
	"fmt"
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
		default:
			err = fmt.Errorf("mysql: unsupported change %T", c)
		}
		if err != nil {
			return err
		}
	}
	return
}

func (m *Migrate) addTable(ctx context.Context, add *schema.AddTable) error {
	b := &strings.Builder{}
	b.WriteString("CREATE TABLE ")
	ident(b, add.T.Name)
	b.WriteString(" (\n")
	for i, c := range add.T.Columns {
		if i != 0 {
			b.WriteString(",\n")
		}
		b.WriteString("  ")
		ident(b, c.Name)
		b.WriteString(" " + c.Type.Raw)
		if !c.Type.Null {
			b.WriteString(" NOT")
		}
		b.WriteString(" NULL")
		if x, ok := c.Default.(*schema.RawExpr); ok {
			b.WriteString(" DEFAULT " + x.X)
		}
		attrs(b, c.Attrs)

	}
	if pk := add.T.PrimaryKey; pk != nil {
		b.WriteString(",\n  ")
		b.WriteString("PRIMARY KEY ")
		parts(b, pk.Parts)
	}
	for _, idx := range add.T.Indexes {
		b.WriteString(",\n  ")
		if idx.Unique {
			b.WriteString("UNIQUE ")
		}
		b.WriteString("INDEX ")
		if idx.Name != "" {
			ident(b, idx.Name)
			b.WriteByte(' ')
		}
		parts(b, idx.Parts)
	}
	b.WriteString("\n)")
	attrs(b, add.T.Attrs)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: create table: %w", err)
	}
	return nil
}

func (m *Migrate) dropTable(ctx context.Context, drop *schema.DropTable) error {
	var b strings.Builder
	b.WriteString("DROP TABLE ")
	if drop.T.Schema != nil {
		ident(&b, drop.T.Schema.Name)
		b.WriteByte('.')
	}
	ident(&b, drop.T.Name)
	if _, err := m.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("mysql: drop table: %w", err)
	}
	return nil
}

// ident writes the given identifier in MySQL format.
func ident(b *strings.Builder, ident string) {
	b.WriteByte('`')
	b.WriteString(ident)
	b.WriteByte('`')
}

func attrs(b *strings.Builder, attrs []schema.Attr) {
	for i := range attrs {
		switch a := attrs[i].(type) {
		case *OnUpdate:
			b.WriteString(" ON UPDATE " + a.A)
		case *AutoIncrement:
			b.WriteString(" AUTO_INCREMENT")
			if a.V != 0 {
				b.WriteByte(' ')
				b.WriteString(strconv.FormatInt(a.V, 10))
			}
		case *schema.Charset:
			b.WriteString(" CHARACTER SET " + a.V)
		case *schema.Collation:
			b.WriteString(" COLLATE " + a.V)
		case *schema.Comment:
			b.WriteString(" COMMENT '" + strings.ReplaceAll(a.Text, "'", "\\'") + "'")
		}
	}
}

func parts(b *strings.Builder, parts []*schema.IndexPart) {
	b.WriteByte('(')
	for i, part := range parts {
		if i > 0 {
			b.WriteString(", ")
		}
		switch {
		case part.C != nil:
			ident(b, part.C.Name)
		case part.X != nil:
			b.WriteString(part.X.(*schema.RawExpr).X)
		}
	}
	b.WriteByte(')')
}
