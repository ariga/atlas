// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"bytes"
	"context"
	"io"
	"sync"
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
		fr := NewFileReport(f)
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

// DefaultTemplate is the default template used by the CI job.
var DefaultTemplate = template.Must(template.New("report").
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

type (
	// A SummaryReport contains a summary of the analysis of all files.
	// It is used as an input to templates to report the CI results.
	SummaryReport struct {
		Files []*FileReport // All files and their report.
	}

	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		migrate.File
		Reports []sqlcheck.Report

		// file content.
		buf     []byte
		bufOnce sync.Once
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
func NewFileReport(f migrate.File) *FileReport {
	return &FileReport{File: f}
}

// Contents returns the contents of the file.
func (f *FileReport) Contents() []byte {
	f.bufOnce.Do(func() {
		// Assume no error as the file was
		// already loaded in previous steps.
		f.buf, _ = io.ReadAll(f.File)
	})
	return f.buf
}

// Line returns the line number from a position.
func (f *FileReport) Line(pos int) int {
	return bytes.Count(f.Contents()[:pos], []byte("\n")) + 1
}

// WriteReport implements sqlcheck.ReportWriter.
func (f *FileReport) WriteReport(r sqlcheck.Report) {
	f.Reports = append(f.Reports, r)
}

// WriteReport implements ReportWriter.
func (w *TemplateWriter) WriteReport(r SummaryReport) error {
	return w.T.Execute(w.W, r)
}
