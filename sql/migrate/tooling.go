// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"text/template"
	"time"
)

var (
	// funcs contains the template.FuncMap for the different formatters.
	funcs = template.FuncMap{
		"now": func() string { return time.Now().Format("20060102150405") },
		"rev": reverse,
	}
)

// NewGolangMigrateFormatter returns a migrate.Formatter computable with golang-migrate/migrate.
func NewGolangMigrateFormatter() (Formatter, error) {
	return templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.up.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.down.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
}

// NewGooseFormatter returns a migrate.Formatter computable with pressly/goose.
func NewGooseFormatter() (Formatter, error) {
	return templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- +goose Up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- +goose Down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
}

// templateFormatter parses the given templates and passes them on to the migrate.NewTemplateFormatter.
func templateFormatter(templates ...string) (fmt Formatter, err error) {
	tpls := make([]*template.Template, len(templates))
	for i, t := range templates {
		tpls[i], err = template.New("").Funcs(funcs).Parse(t)
		if err != nil {
			return nil, err
		}
	}
	return NewTemplateFormatter(tpls...)
}
