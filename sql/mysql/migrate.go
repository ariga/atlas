package mysql

import (
	"context"
	"fmt"
	"strings"
	"text/template"

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
	var b strings.Builder
	if err := createTmpl.Execute(&b, add.T); err != nil {
		return err
	}
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

var createTmpl = template.Must(template.New("create_table").
	Funcs(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"ident": func(s string) string { return "`" + s + "`" },
		"attrs": attrs,
	}).
	Parse(`
CREATE TABLE {{ ident $.Name }} (
	{{- $nc := len $.Columns }}
	{{- range $i, $c := $.Columns }}
		{{- $comma := or (ne $i (add $nc -1)) $.PrimaryKey }}
		{{ ident $c.Name }} {{ $c.Type.Raw }} {{ if not $c.Type.Null }}NOT {{ end }}NULL{{ with $attr := attrs $c }} {{ . }}{{ end }}{{ if $comma }},{{ end }}
	{{- end }}
	{{- with $.PrimaryKey }}
		PRIMARY KEY ({{ range $i, $p := .Parts }}{{ if $i }}, {{ end }}{{ ident $p.C.Name }}{{ end }})
	{{- end }}
)`))

func attrs(c *schema.Column) string {
	var attr []string
	if x, ok := c.Default.(*schema.RawExpr); ok {
		attr = append(attr, "DEFAULT", x.X)
	}
	for i := range c.Attrs {
		switch a := c.Attrs[i].(type) {
		case *OnUpdate:
			attr = append(attr, "ON UPDATE", a.A)
		case *AutoIncrement:
			attr = append(attr, "AUTO_INCREMENT")
		case *schema.Collation:
			attr = append(attr, "COLLATE", a.V)
		case *schema.Comment:
			attr = append(attr, "COMMENT", "'"+strings.ReplaceAll(a.Text, "'", "\\'")+"'")
		}
	}
	return strings.Join(attr, " ")
}
