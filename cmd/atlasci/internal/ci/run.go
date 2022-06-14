// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"text/template"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
)

// Runner is used to execute CI jobs.
type Runner struct {
	// DevClient configures the "dev driver" to calculate
	// migration changes by the driver.
	Dev *sqlclient.Client

	// RunChangeDetector configures the ChangeDetector to
	// be used by the runner.
	ChangeDetector ChangeDetector

	// Scan is used for scanning the migration directory.
	Scan migrate.Scanner

	// Analyzer defines the analysis to be run in the CI job.
	Analyzer sqlcheck.Analyzer

	// ReportWriter writes the summary report.
	ReportWriter ReportWriter
}

// Run executes the CI job.
func (r *Runner) Run(ctx context.Context) error {
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	if err != nil {
		return err
	}
	// Bring the dev environment to the base point.
	for _, f := range base {
		stmt, err := r.Scan.Stmts(f)
		if err != nil {
			return err
		}
		for _, s := range stmt {
			if _, err := r.Dev.ExecContext(ctx, s); err != nil {
				return err
			}
		}
	}
	l := &DevLoader{Dev: r.Dev, Scan: r.Scan}
	files, err := l.LoadChanges(ctx, feat)
	if err != nil {
		return err
	}
	var sum SummaryReport
	for _, f := range files {
		fr, err := NewFileReport(f)
		if err != nil {
			return err
		}
		if err := r.Analyzer.Analyze(ctx, &sqlcheck.Pass{
			File:     f,
			Dev:      r.Dev,
			Reporter: fr,
		}); err != nil {
			return err
		}
		sum.Files = append(sum.Files, fr)
	}
	return r.ReportWriter.WriteReport(sum)
}

var (
	// TemplateFuncs are global functions available in templates.
	TemplateFuncs = template.FuncMap{
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
	}
	// DefaultTemplate is the default template used by the CI job.
	DefaultTemplate = template.Must(template.New("report").
			Funcs(TemplateFuncs).
			Parse(`
{{- range $f := .Files }}
	{{- range $r := $f.Reports }}
		{{- if $r.Text }}
			{{- printf "%s:\n\n" $r.Text }}
		{{- else if $r.Diagnostics }}
			{{- printf "Unnamed diagnostics for file %s:\n\n" $f.File.Name }}
		{{- end }}
		{{- range $d := $r.Diagnostics }}
			{{- printf "\tL%d: %s\n" ($f.Line $d.Pos) $d.Text }}
		{{- end }}
		{{- if $r.Diagnostics }}
			{{- print "\n" }}
		{{- end }}
	{{- end }}
{{- end -}}
`))
)

type (
	// A SummaryReport contains a summary of the analysis of all files.
	// It is used as an input to templates to report the CI results.
	SummaryReport struct {
		Files []*FileReport // All files and their report.
	}

	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		Name    string
		Text    string
		Reports []sqlcheck.Report
	}

	// ReportWriter is a type of report writer that writes a summary of analysis reports.
	ReportWriter interface {
		WriteReport(SummaryReport) error
	}

	// A TemplateWriter is a type of writer that writes output according to a template.
	TemplateWriter struct {
		T *template.Template
		W io.Writer
	}
)

// NewFileReport returns a new FileReport.
func NewFileReport(f migrate.File) (*FileReport, error) {
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &FileReport{Name: f.Name(), Text: string(buf)}, nil
}

// Line returns the line number from a position.
func (f *FileReport) Line(pos int) int {
	return strings.Count(f.Text[:pos], "\n") + 1
}

// WriteReport implements sqlcheck.ReportWriter.
func (f *FileReport) WriteReport(r sqlcheck.Report) {
	f.Reports = append(f.Reports, r)
}

// WriteReport implements ReportWriter.
func (w *TemplateWriter) WriteReport(r SummaryReport) error {
	return w.T.Execute(w.W, r)
}

// MigrationAnalyzer implements sqlcheck.Analyzer. It validates the migration dir.
type MigrationAnalyzer struct {
	Dir migrate.Dir
}

// Analyze implements the sqlcheck.Analyzer interface.
func (m *MigrationAnalyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	diags := make([]sqlcheck.Diagnostic, 0)
	err := migrate.Validate(m.Dir)
	if err != nil {
		switch {
		case errors.Is(err, migrate.ErrChecksumMismatch):
			diags = append(diags, sqlcheck.Diagnostic{
				Pos:  0,
				Text: "There was a checksum mismatch",
			})
		case errors.Is(err, migrate.ErrChecksumFormat):
			diags = append(diags, sqlcheck.Diagnostic{
				Pos:  0,
				Text: "The checksum file format is invalid",
			})
		case errors.Is(err, migrate.ErrChecksumNotFound):
			diags = append(diags, sqlcheck.Diagnostic{
				Pos:  0,
				Text: "The checksum file was not found",
			})
		default:
			return err
		}
	}
	if len(diags) > 0 {
		p.Reporter.WriteReport(sqlcheck.Report{
			Text:        "Migration directory could not be verified",
			Diagnostics: diags,
		})

	}
	return nil
}
