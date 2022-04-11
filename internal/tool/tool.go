// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package tool

import (
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
)

var (
	// funcs contains the template.FuncMap for the different formatters.
	funcs = template.FuncMap{
		// now format the current time in a lexicographically ascending order while maintaining human readability.
		"now": func() string { return time.Now().Format("20060102150405") },
		"rev": reverse,
	}
)

// NewGolangMigrateFormatter returns a migrate.Formatter computable with golang-migrate/migrate.
func NewGolangMigrateFormatter() (migrate.Formatter, error) {
	return templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.up.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.down.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
}

// NewGooseFormatter returns a migrate.Formatter computable with pressly/goose.
func NewGooseFormatter() (migrate.Formatter, error) {
	return templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- +goose Up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- +goose Down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
}

// templateFormatter parses the given templates and passes them on to the migrate.NewTemplateFormatter.
func templateFormatter(templates ...string) (fmt migrate.Formatter, err error) {
	tpls := make([]*template.Template, len(templates))
	for i, t := range templates {
		tpls[i], err = template.New("").Funcs(funcs).Parse(t)
		if err != nil {
			return nil, err
		}
	}
	return migrate.NewTemplateFormatter(tpls...)
}

// reverse changes for the down migration.
func reverse(changes []*migrate.Change) []*migrate.Change {
	n := len(changes)
	rev := make([]*migrate.Change, n)
	if n%2 == 1 {
		rev[n/2] = changes[n/2]
	}
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = changes[j], changes[i]
	}
	return rev
}
