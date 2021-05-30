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
	}).
	Parse(`
CREATE TABLE {{ ident $.Name }} (
	{{- $nc := len $.Columns }}
	{{- range $i, $c := $.Columns }}
		{{- $comma := or (ne $i (add $nc -1)) (len $.PrimaryKey) }}
		{{ ident $c.Name }} {{ $c.Type.Raw }} {{ if not $c.Type.Null }}NOT {{ end }} NULL{{ if $comma }},{{ end }}
	{{- end }}
	{{- with $.PrimaryKey }}
		PRIMARY KEY ({{ range $i, $c := . }}{{ if $i }}, {{ end }}{{ ident $c.Name }}{{ end }})
	{{- end }}
)`))
