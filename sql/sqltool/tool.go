// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqltool

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
)

var (
	// GolangMigrateFormatter returns migrate.Formatter compatible with golang-migrate/migrate.
	GolangMigrateFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.up.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.down.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// GooseFormatter returns migrate.Formatter compatible with pressly/goose.
	GooseFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- +goose Up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- +goose Down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// FlywayFormatter returns migrate.Formatter compatible with Flyway.
	FlywayFormatter = templateFormatter(
		"V{{ now }}{{ with .Name }}__{{ . }}{{ end }}.sql",
		`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
		"U{{ now }}{{ with .Name }}__{{ . }}{{ end }}.sql",
		`{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
	// LiquibaseFormatter returns migrate.Formatter compatible with Liquibase.
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
	// DbmateFormatter returns migrate.Formatter compatible with amacneil/dbmate.
	DbmateFormatter = templateFormatter(
		"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
		`-- migrate:up
{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}
-- migrate:down
{{ range rev .Changes }}{{ if .Reverse }}{{ with .Comment }}-- reverse: {{ println . }}{{ end }}{{ printf "%s;\n" .Reverse }}{{ end }}{{ end }}`,
	)
)

// GolangMigrateDir wraps migrate.LocalDir and provides implementation compatible with golang-migrate/migrate.
type GolangMigrateDir struct{ *migrate.LocalDir }

// NewGolangMigrateDir returns a new GolangMigrateDir.
func NewGolangMigrateDir(path string) (*GolangMigrateDir, error) {
	dir, err := migrate.NewLocalDir(path)
	if err != nil {
		return nil, err
	}
	return &GolangMigrateDir{dir}, nil
}

// Files implements Scanner.Files. It looks for all files with up.sql suffix and orders them by filename.
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
		ret[i] = &GolangMigrateFile{LocalFile: migrate.NewLocalFile(n, b)}
	}
	return ret, nil
}

// GolangMigrateFile wraps migrate.LocalFile with custom description function.
type GolangMigrateFile struct {
	*migrate.LocalFile
}

// Desc implements File.Desc.
func (f *GolangMigrateFile) Desc() string {
	return strings.TrimSuffix(f.LocalFile.Desc(), ".up")
}

// GooseDir wraps migrate.LocalDir and provides implementation compatible with pressly/goose.
type GooseDir struct{ *migrate.LocalDir }

// NewGooseDir returns a new GooseDir.
func NewGooseDir(path string) (*GooseDir, error) {
	dir, err := migrate.NewLocalDir(path)
	if err != nil {
		return nil, err
	}
	return &GooseDir{dir}, nil
}

// GooseFile wraps migrate.LocalFile with custom statements function.
type GooseFile struct {
	*migrate.LocalFile
}

// Files implements Scanner.Files. It looks for all files with up.sql suffix and orders them by filename.
func (d *GooseDir) Files() ([]migrate.File, error) {
	files, err := d.LocalDir.Files()
	if err != nil {
		return nil, err
	}
	for i, f := range files {
		files[i] = &GooseFile{f.(*migrate.LocalFile)}
	}
	return files, nil
}

// Stmts implements Scanner.Stmts. It understands the migration format used by pressly/goose sql migration files.
func (f *GooseFile) Stmts() ([]string, error) {
	var (
		state, line int
		stmts       []string
		buf         strings.Builder
		sc          = bufio.NewScanner(bytes.NewReader(f.Bytes()))
	)
	for sc.Scan() {
		line++
		s := sc.Text()
		// Handle pragmas.
		if strings.HasPrefix(s, pragma) {
			switch strings.TrimSpace(strings.TrimPrefix(s, pragma)) {
			case "Up":
				switch state {
				case none: // found the "up" part of the file
					state = up
				default:
					return nil, unexpectedGoosePragmaErr(f, line, "Up")
				}
			case "Down":
				switch state {
				case up: // found the "down" part
					if rest := strings.TrimSpace(buf.String()); len(rest) > 0 {
						return nil, unexpectedGoosePragmaErr(f, line, "Down")
					}
					return stmts, nil
				default:
					return nil, unexpectedGoosePragmaErr(f, line, "Down")
				}
			case "StatementBegin":
				switch state {
				case up:
					state = begin // begin of a statement
				default:
					return nil, unexpectedGoosePragmaErr(f, line, "StatementBegin")
				}
			case "StatementEnd":
				switch state {
				case begin:
					state = end // end of a statement
				default:
					return nil, unexpectedGoosePragmaErr(f, line, "StatementEnd")
				}
			}
		}
		// Write the line of the statement.
		if !rePragma.MatchString(s) {
			if _, err := buf.WriteString(s + "\n"); err != nil {
				return nil, err
			}
		}
		switch state {
		case up: // end of statement if line ends with semicolon
			if s := strings.TrimSpace(s); strings.HasSuffix(s, ";") && !strings.HasPrefix(s, "--") {
				stmts = append(stmts, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
		case end: // end of statement marked by pragma
			stmts = append(stmts, strings.TrimSpace(buf.String()))
			buf.Reset()
			state = up // back in up block
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("sql/migrate: scanning migration %q: %w", f.Name(), err)
	}
	if state == none {
		return nil, fmt.Errorf("sql/migrate: empty migration %q", f.Name())
	}
	return stmts, nil
}

const (
	none int = iota // state when parsing goose sql file
	up
	begin
	end
	pragma = "-- +goose"
)

var rePragma = regexp.MustCompile("-- \\+goose Up|Down|StatementBegin|StatementEnd")

func unexpectedGoosePragmaErr(f *GooseFile, line int, pragma string) error {
	return fmt.Errorf(
		"sql/migrate: goose: %s:%d unexpected pragma '%s'",
		f.Name(), line, pragma,
	)
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
	tf, err := migrate.NewTemplateFormatter(tpls...)
	if err != nil {
		panic(err)
	}
	return tf
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
