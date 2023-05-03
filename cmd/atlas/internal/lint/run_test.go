// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package lint_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"ariga.io/atlas/cmd/atlas/internal/lint"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestRunner_Run(t *testing.T) {
	ctx := context.Background()
	b := &bytes.Buffer{}
	c, err := sqlclient.Open(ctx, "sqlite://run?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	r := &lint.Runner{
		Dir: testDir{},
		Dev: c,
		ChangeDetector: testDetector{
			base: []migrate.File{
				testFile{name: "1.sql", content: "CREATE TABLE users (id INT);"},
			},
			feat: []migrate.File{
				testFile{name: "2.sql", content: "CREATE TABLE pets (id INT);\nDROP TABLE users;"},
			},
		},
		Analyzers: []sqlcheck.Analyzer{
			&testAnalyzer{},
		},
		ReportWriter: &lint.TemplateWriter{
			T: lint.DefaultTemplate,
			W: b,
		},
	}
	require.NoError(t, r.Run(ctx))

	passes := r.Analyzers[0].(*testAnalyzer).passes
	require.Len(t, passes, 1)
	changes := passes[0].File.Changes
	require.Len(t, changes, 2)
	require.Equal(t, "CREATE TABLE pets (id INT);", changes[0].Stmt.Text)
	require.Equal(t, "DROP TABLE users;", changes[1].Stmt.Text)
	require.Equal(t, `2.sql: Report 1:

	L1: Diagnostic 1

`, b.String())

	b.Reset()
	r.ReportWriter.(*lint.TemplateWriter).T = template.Must(template.New("").
		Funcs(lint.TemplateFuncs).
		Parse(`
Env:
{{ .Env.Driver }}, {{ .Env.Dir }}

Steps:
{{ range $s := .Steps }}
	{{- if $s.Error }}
		"Error in step " {{ $s.Name }} ": " {{ $s.Error }} 
	{{- else }}
		{{- json $s }}
	{{- end }}
{{ end }}
{{- if .Files }}
Files:
{{ range $f := .Files }}
	{{- json $f }}
{{ end }}
{{- end }}

Current Schema:
{{ .Schema.Current }}
Desired Schema:
{{ .Schema.Desired }}
`))
	require.NoError(t, r.Run(ctx))
	require.Equal(t, `
Env:
sqlite3, migrations

Steps:
{"Name":"Detect New Migration Files","Text":"Found 1 new migration files (from 2 total)"}
{"Name":"Replay Migration Files","Text":"Loaded 1 changes on dev database"}
{"Name":"Analyze 2.sql","Text":"1 reports were found in analysis","Result":{"Name":"2.sql","Text":"CREATE TABLE pets (id INT);\nDROP TABLE users;","Reports":[{"Text":"Report 2","Diagnostics":[{"Pos":1,"Text":"Diagnostic 1","Code":""},{"Pos":2,"Text":"Diagnostic 2","Code":""}]}]}}

Files:
{"Name":"2.sql","Text":"CREATE TABLE pets (id INT);\nDROP TABLE users;","Reports":[{"Text":"Report 2","Diagnostics":[{"Pos":1,"Text":"Diagnostic 1","Code":""},{"Pos":2,"Text":"Diagnostic 2","Code":""}]}]}


Current Schema:
table "users" {
  schema = schema.main
  column "id" {
    null = true
    type = int
  }
}
schema "main" {
}

Desired Schema:
table "pets" {
  schema = schema.main
  column "id" {
    null = true
    type = int
  }
}
schema "main" {
}

`, b.String())
}

type testAnalyzer struct {
	passes []*sqlcheck.Pass
}

func (t *testAnalyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	t.passes = append(t.passes, p)
	r := sqlcheck.Report{
		Text: fmt.Sprintf("Report %d", len(t.passes)),
	}
	for i := 1; i <= len(t.passes); i++ {
		r.Diagnostics = append(r.Diagnostics, sqlcheck.Diagnostic{
			Pos:  i,
			Text: fmt.Sprintf("Diagnostic %d", i),
		})
	}
	p.Reporter.WriteReport(r)
	return nil
}

type testDetector struct {
	base, feat []migrate.File
}

func (t testDetector) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	return t.base, t.feat, nil
}
