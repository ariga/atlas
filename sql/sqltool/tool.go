// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqltool

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
)

var (
	// GolangMigrateFormatter is a migrate.Formatter compatible with golang-migrate/migrate.
	GolangMigrateFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.up.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.down.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// GooseFormatter returns a migrate.Formatter compatible with pressly/goose.
	GooseFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- +goose Up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- +goose Down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// FlywayFormatter returns a migrate.Formatter compatible with Flyway.
	FlywayFormatter = templateFormatter(
		"V{{ now }}{{ with .Name }}__{{ . }}{{ end }}.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"U{{ now }}{{ with .Name }}__{{ . }}{{ end }}.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// LiquibaseFormatter returns a migrate.Formatter compatible with Liquibase.
	LiquibaseFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`{{- $now := now -}}
--liquibase formatted sql

{{- range $index, $change := .Changes }}
--changeset atlas:{{ $now }}-{{ inc $index }}
{{ with $change.Comment }}--comment: {{ . }}{{ end }}
{{ $change.Cmd }};
{{ with $change.Reverse }}--rollback: {{ . }};{{ end }}
{{ end }}`,
	)
	// DbmateFormatter returns a migrate.Formatter compatible with amacneil/dbmate.
	DbmateFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- migrate:up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- migrate:down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
)

// GolangMigrateDir wraps a migrate.LocalDir and provides a migrate.Scanner
// implementation compatible with golang-migrate/migrate.
type GolangMigrateDir struct{ *migrate.LocalDir }

// NewGolangMigrateDir returns a new GolangMigrateDir.
func NewGolangMigrateDir(path string) (*GolangMigrateDir, error) {
	dir, err := migrate.NewLocalDir(path)
	if err != nil {
		return nil, err
	}
	return &GolangMigrateDir{dir}, nil
}

// Files implements Scanner.Files. It looks for all files with up.sql suffix and orders them by filename-
func (d *GolangMigrateDir) Files() ([]migrate.File, error) {
	names, err := fs.Glob(d, "*.up.sql")
	if err != nil {
		return nil, err
	}
	// Sort files lexicographically.
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	ret := make([]migrate.File, len(names))
	for i, n := range names {
		b, err := fs.ReadFile(d, n)
		if err != nil {
			return nil, fmt.Errorf("sql/migrate: read file %q: %w", n, err)
		}
		ret[i] = migrate.NewLocalFile(n, b)
	}
	return ret, nil
}

// Desc implements Scanner.Desc.
func (d *GolangMigrateDir) Desc(f migrate.File) (string, error) {
	desc, err := d.LocalDir.Desc(f)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(desc, ".up"), nil
}

// funcs contains the template.FuncMap for the different formatters.
var funcs = template.FuncMap{
	"inc": func(x int) int { return x + 1 },
	// now formats the current time in a lexicographically ascending order while maintaining human readability.
	"now": func() string { return time.Now().UTC().Format("20060102150405") },
	"rev": reverse,
}

// templateFormatter parses the given templates and passes them on to the migrate.NewTemplateFormatter.
func templateFormatter(templates ...string) migrate.Formatter {
	tpls := make([]*template.Template, len(templates))
	for i, t := range templates {
		tpls[i] = template.Must(template.New("").Funcs(funcs).Parse(t))
	}
	fmt, err := migrate.NewTemplateFormatter(tpls...)
	if err != nil {
		panic(err)
	}
	return fmt
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
