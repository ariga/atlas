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

	// Dir is used for scanning and validating the migration directory.
	Dir migrate.Dir

	// Analyzer defines the analysis to be run in the CI job.
	Analyzer sqlcheck.Analyzer

	// ReportWriter writes the summary report.
	ReportWriter ReportWriter
}

// Run executes the CI job.
func (r *Runner) Run(ctx context.Context) error {
	sum, err := r.summary(ctx)
	switch err := err.(type) {
	case nil:
		return r.ReportWriter.WriteReport(sum)
	case *FileError:
		return r.ReportWriter.WriteReport(&SummaryReport{
			Files: []*FileReport{{Name: err.File, Error: err.Error()}},
		})
	default:
		return err
	}
}

func (r *Runner) summary(ctx context.Context) (*SummaryReport, error) {
	// Validate sum file in case it exists.
	if err := migrate.Validate(r.Dir); err != nil && !errors.Is(err, migrate.ErrChecksumNotFound) {
		return nil, &FileError{File: migrate.HashFileName, Err: err}
	}
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	if err != nil {
		return nil, err
	}
	l := &DevLoader{Dev: r.Dev}
	files, err := l.LoadChanges(ctx, base, feat)
	if err != nil {
		return nil, err
	}
	sum := &SummaryReport{Files: make([]*FileReport, 0, len(files))}
	for _, f := range files {
		fr := NewFileReport(f)
		if err := r.Analyzer.Analyze(ctx, &sqlcheck.Pass{
			File:     f,
			Dev:      r.Dev,
			Reporter: fr,
		}); err != nil {
			fr.Error = err.Error()
		}
		sum.Files = append(sum.Files, fr)
	}
	return sum, nil
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
	{{- if $f.Error }}
		{{- printf "%s: %s\n" $f.Name $f.Error }}
	{{- else }}
		{{- range $r := $f.Reports }}
			{{- if $r.Text }}
				{{- printf "%s:\n\n" $r.Text }}
			{{- else if $r.Diagnostics }}
				{{- printf "Unnamed diagnostics for file %s:\n\n" $f.Name }}
			{{- end }}
			{{- range $d := $r.Diagnostics }}
				{{- printf "\tL%d: %s\n" ($f.Line $d.Pos) $d.Text }}
			{{- end }}
			{{- if $r.Diagnostics }}
				{{- print "\n" }}
			{{- end }}
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
		Name    string            // Name of the file.
		Text    string            // Contents of the file.
		Reports []sqlcheck.Report // List of reports.
		Error   string            // File specific error.
	}

	// ReportWriter is a type of report writer that writes a summary of analysis reports.
	ReportWriter interface {
		WriteReport(*SummaryReport) error
	}

	// A TemplateWriter is a type of writer that writes output according to a template.
	TemplateWriter struct {
		T *template.Template
		W io.Writer
	}
)

// NewFileReport returns a new FileReport.
func NewFileReport(f migrate.File) *FileReport {
	return &FileReport{Name: f.Name(), Text: string(f.Bytes())}
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
func (w *TemplateWriter) WriteReport(r *SummaryReport) error {
	return w.T.Execute(w.W, r)
}
