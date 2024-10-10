// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint_test

import (
	"bytes"
	"context"
	_ "embed"
	"testing"
	"text/template"

	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

func TestRunner_Run(t *testing.T) {
	ctx := context.Background()
	b := &bytes.Buffer{}
	c, err := sqlclient.Open(ctx, "sqlite://run?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	color.NoColor = true

	t.Run("checksum mismatch", func(t *testing.T) {
		var (
			d = &migrate.MemDir{}
			r = &migratelint.Runner{
				Dir: d,
				Dev: c,
				ReportWriter: &migratelint.TemplateWriter{
					T: migratelint.DefaultTemplate,
					W: b,
				},
			}
			err = &migrate.ChecksumError{}
		)
		// File was added at the end.
		require.NoError(t, migrate.WriteSumFile(d, migrate.HashFile{}))
		require.NoError(t, d.WriteFile("1.sql", []byte("content")))
		require.ErrorAs(t, r.Run(ctx), &err)
		require.Regexp(t, `Analyzing changes \(1 migration in total\):

  Error: checksum mismatch \(atlas.sum\): L2: 1\.sql was added

  -------------------------
  -- .*
  -- 1 version with errors
`, b.String())
		// File was edited.
		b.Reset()
		require.NoError(t, migrate.WriteSumFile(d, must(d.Checksum())))
		require.NoError(t, d.WriteFile("1.sql", []byte("content changed")))
		require.ErrorAs(t, r.Run(ctx), &err)
		require.Regexp(t, `Analyzing changes \(1 migration in total\):

  Error: checksum mismatch \(atlas.sum\): L2: 1\.sql was edited

  -------------------------
  -- .*
  -- 1 version with errors
`, b.String())
		// File was removed.
		b.Reset()
		h := must(d.Checksum())
		*d = migrate.MemDir{}
		require.NoError(t, migrate.WriteSumFile(d, h))
		require.ErrorAs(t, r.Run(ctx), &err)
		require.Regexp(t, `Analyzing changes \(1 migration in total\):

  Error: checksum mismatch \(atlas.sum\): L2: 1\.sql was removed

  -------------------------
  -- .*
  -- 1 version with errors
`, b.String())
		// File was added in the middle.
		b.Reset()
		require.NoError(t, d.WriteFile("1.sql", []byte("content")))
		require.NoError(t, d.WriteFile("3.sql", []byte("content")))
		require.NoError(t, migrate.WriteSumFile(d, must(d.Checksum())))
		require.NoError(t, d.WriteFile("2.sql", []byte("content")))
		require.ErrorAs(t, r.Run(ctx), &err)
		require.Regexp(t, `Analyzing changes \(1 migration in total\):

  Error: checksum mismatch \(atlas.sum\): L3: 2\.sql was added

  -------------------------
  -- .*
  -- 1 version with errors
`, b.String())
	})

	az := &testAnalyzer{
		reports: []sqlcheck.Report{
			{Text: "Report 1", Diagnostics: []sqlcheck.Diagnostic{{Pos: 1, Text: "Diagnostic 1", Code: "TS101"}}},
		},
	}
	r := &migratelint.Runner{
		Dir: &migrate.MemDir{},
		Dev: c,
		ChangeDetector: testDetector{
			base: []migrate.File{
				migrate.NewLocalFile("1.sql", []byte("CREATE TABLE users (id INT);")),
			},
			feat: []migrate.File{
				migrate.NewLocalFile("2.sql", []byte("CREATE TABLE pets (id INT);\nDROP TABLE users;")),
			},
		},
		Analyzers: []sqlcheck.Analyzer{az},
		ReportWriter: &migratelint.TemplateWriter{
			T: migratelint.DefaultTemplate,
			W: b,
		},
	}
	require.NoError(t, r.Run(ctx))

	require.Len(t, az.passes, 1)
	changes := az.passes[0].File.Changes
	require.Len(t, changes, 2)
	require.Equal(t, "CREATE TABLE pets (id INT);", changes[0].Stmt.Text)
	require.Equal(t, "DROP TABLE users;", changes[1].Stmt.Text)
	require.Regexp(t, `Analyzing changes from version 1 to 2 \(1 migration in total\):

  -- analyzing version 2
    -- Report 1:
      -- L1: Diagnostic 1 https://atlasgo.io/lint/analyzers#TS101
  -- ok \(.*\)

  -------------------------
  -- .*
  -- 1 version with warnings
  -- 2 schema changes
  -- 1 diagnostic
`, b.String())

	b.Reset()
	az.reports = append(az.reports, sqlcheck.Report{Text: "Report 2", Diagnostics: []sqlcheck.Diagnostic{{Pos: 2, Text: "Diagnostic 2", Code: "TS101"}}})
	r.ReportWriter.(*migratelint.TemplateWriter).T = template.Must(template.New("").
		Funcs(migratelint.TemplateFuncs).
		Parse(`
Env:
{{ .Env.Driver }}

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
sqlite3

Steps:
{"Name":"Detect New Migration Files","Text":"Found 1 new migration files (from 2 total)"}
{"Name":"Replay Migration Files","Text":"Loaded 1 changes on dev database"}
{"Name":"Analyze 2.sql","Text":"2 reports were found in analysis","Result":{"Name":"2.sql","Text":"CREATE TABLE pets (id INT);\nDROP TABLE users;","Reports":[{"Text":"Report 1","Diagnostics":[{"Pos":1,"Text":"Diagnostic 1","Code":"TS101"}]},{"Text":"Report 2","Diagnostics":[{"Pos":2,"Text":"Diagnostic 2","Code":"TS101"}]}]}}

Files:
{"Name":"2.sql","Text":"CREATE TABLE pets (id INT);\nDROP TABLE users;","Reports":[{"Text":"Report 1","Diagnostics":[{"Pos":1,"Text":"Diagnostic 1","Code":"TS101"}]},{"Text":"Report 2","Diagnostics":[{"Pos":2,"Text":"Diagnostic 2","Code":"TS101"}]}]}


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

	b.Reset()
	r.ReportWriter.(*migratelint.TemplateWriter).T = template.Must(template.New("").
		Funcs(migratelint.TemplateFuncs).
		Parse(`{"DiagnosticsCount": {{ .DiagnosticsCount }}, "FilesCount": {{ len .Files }}}`))
	require.NoError(t, r.Run(ctx))
	require.Equal(t, `{"DiagnosticsCount": 2, "FilesCount": 1}`, b.String())

	// Suggested fixes.
	az.reports = append(az.reports, sqlcheck.Report{
		Text: "Report 3", Diagnostics: []sqlcheck.Diagnostic{
			{Pos: 2, Text: "Diagnostic 3", Code: "TS101", SuggestedFixes: []sqlcheck.SuggestedFix{
				{TextEdit: &sqlcheck.TextEdit{NewText: "Not shown"}},
			}},
			{Pos: 2, Text: "Diagnostic 4", Code: "TS101", SuggestedFixes: []sqlcheck.SuggestedFix{
				{Message: `Add a pre-migration check to ensure table "users" is empty before dropping it`},
			}},
			{Pos: 2, Text: `Adding a non-nullable "int" column "oops" will fail in case table "groups" is not empty`, Code: "TS101", SuggestedFixes: []sqlcheck.SuggestedFix{
				{Message: `Add a pre-migration check to ensure "oops" column is not empty before changing its nullability`},
			}},
		},
	})
	b.Reset()
	r.ReportWriter = &migratelint.TemplateWriter{T: migratelint.DefaultTemplate, W: b}
	require.NoError(t, r.Run(ctx))
	require.Regexp(t, `Analyzing changes from version 1 to 2 \(1 migration in total\):

  -- analyzing version 2
    -- Report 1:
      -- L1: Diagnostic 1 https://atlasgo.io/lint/analyzers#TS101
    -- Report 2:
      -- L1: Diagnostic 2 https://atlasgo.io/lint/analyzers#TS101
    -- Report 3:
      -- L1: Diagnostic 3 https://atlasgo.io/lint/analyzers#TS101
      -- L1: Diagnostic 4 https://atlasgo.io/lint/analyzers#TS101
      -- L1: Adding a non-nullable "int" column "oops" will fail in case table "groups" is not
         empty https://atlasgo.io/lint/analyzers#TS101
    -- suggested fixes:
      -> Add a pre-migration check to ensure table "users" is empty before dropping it
      -> Add a pre-migration check to ensure "oops" column is not empty before changing its
         nullability
  -- ok (.+)

  -------------------------
  --.+
  -- 1 version with warnings
  -- 2 schema changes
  -- 5 diagnostics
`, b.String())
}

type testAnalyzer struct {
	passes  []*sqlcheck.Pass
	reports []sqlcheck.Report
}

func (t *testAnalyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	t.passes = append(t.passes, p)
	for _, r := range t.reports {
		p.Reporter.WriteReport(r)
	}
	return nil
}

type testDetector struct {
	base, feat []migrate.File
}

func (t testDetector) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	return t.base, t.feat, nil
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
