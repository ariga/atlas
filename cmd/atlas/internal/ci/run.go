// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// summary report. reset on each run.
	sum *SummaryReport
}

// Run executes the CI job.
func (r *Runner) Run(ctx context.Context) error {
	err := r.summary(ctx)
	if err2 := r.ReportWriter.WriteReport(r.sum); err2 != nil {
		if err != nil {
			err2 = fmt.Errorf("%w: %v", err, err2)
		}
		err = err2
	}
	return err
}

const (
	stepIntegrityCheck = "Migration Integrity Check"
	stepDetectChanges  = "Detect New Migration Files"
	stepLoadChanges    = "Replay Migration Files"
	stepAnalyzeFile    = "Analyze %s"
)

func (r *Runner) summary(ctx context.Context) error {
	r.sum = NewSummaryReport(r.Dev, r.Dir)

	// Integrity check.
	switch err := migrate.Validate(r.Dir); {
	case errors.Is(err, migrate.ErrChecksumNotFound):
	case err != nil:
		r.sum.Files = append(r.sum.Files, &FileReport{Name: migrate.HashFileName, Error: err})
		return r.sum.StepError(stepIntegrityCheck, fmt.Sprintf("File %s is invalid", migrate.HashFileName), err)
	default:
		r.sum.StepResult(stepIntegrityCheck, fmt.Sprintf("File %s is valid", migrate.HashFileName), nil)
	}

	// Detect new migration files.
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	if err != nil {
		return r.sum.StepError(stepDetectChanges, "Failed find new migration files", err)
	}
	r.sum.StepResult(stepDetectChanges, fmt.Sprintf("Found %d new migration files (from %d total)", len(feat), len(base)+len(feat)), nil)

	// Load files into changes.
	l := &DevLoader{Dev: r.Dev}
	diff, err := l.LoadChanges(ctx, base, feat)
	if err != nil {
		if fr := (&FileError{}); errors.As(err, fr) {
			r.sum.Files = append(r.sum.Files, &FileReport{Name: fr.File, Error: err})
		}
		return r.sum.StepError(stepDetectChanges, "Failed loading changes on dev database", err)
	}
	r.sum.StepResult(stepLoadChanges, fmt.Sprintf("Loaded %d changes on dev database", len(diff.Files)), nil)
	r.sum.WriteSchema(r.Dev, diff)

	// Analyze files.
	for _, f := range diff.Files {
		fr := NewFileReport(f)
		if err := r.Analyzer.Analyze(ctx, &sqlcheck.Pass{
			File:     f,
			Dev:      r.Dev,
			Reporter: fr,
		}); err != nil {
			fr.Error = err
		}
		r.sum.Files = append(r.sum.Files, fr)
		r.sum.StepResult(
			fmt.Sprintf(stepAnalyzeFile, f.Name()),
			fmt.Sprintf("%d reports were found in analysis", len(fr.Reports)),
			fr,
		)
	}
	return nil
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
		// Env holds the environment information.
		Env struct {
			Driver string         // Driver name.
			URL    *sqlclient.URL // URL to dev database.
			Dir    string         // Path to migration directory.
		}

		// Steps of the analysis. Added in verbose mode.
		Steps []struct {
			Name   string // Step name.
			Text   string // Step description.
			Error  error  // Error that cause the execution to halt.
			Result any    // Result of the step. For example, a diagnostic.
		}

		// Schema versions found by the runner.
		Schema struct {
			Current string // Current schema.
			Desired string // Desired schema.
		}

		// Files reports. Non-empty in case there are findings.
		Files []*FileReport
	}

	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		Name    string            // Name of the file.
		Text    string            // Contents of the file.
		Reports []sqlcheck.Report // List of reports.
		Error   error             // File specific error.
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

// NewSummaryReport returns a new SummaryReport.
func NewSummaryReport(c *sqlclient.Client, dir migrate.Dir) *SummaryReport {
	sum := &SummaryReport{
		Env: struct {
			Driver string
			URL    *sqlclient.URL
			Dir    string
		}{
			Driver: c.Name,
			URL:    c.URL,
		},
	}
	if p, ok := dir.(interface{ Path() string }); ok {
		sum.Env.Dir = p.Path()
	}
	return sum
}

// StepResult appends step result to the summary.
func (f *SummaryReport) StepResult(name, text string, result any) {
	f.Steps = append(f.Steps, struct {
		Name   string
		Text   string
		Error  error
		Result any
	}{
		Name:   name,
		Text:   text,
		Result: result,
	})
}

// StepError appends step error to the summary.
func (f *SummaryReport) StepError(name, text string, err error) error {
	f.Steps = append(f.Steps, struct {
		Name   string
		Text   string
		Error  error
		Result any
	}{
		Name:  name,
		Text:  text,
		Error: err,
	})
	return err
}

// WriteSchema writes the current and desired schema to the summary.
func (f *SummaryReport) WriteSchema(c *sqlclient.Client, diff *Changes) {
	if curr, err := c.MarshalSpec(diff.From); err == nil {
		f.Schema.Current = string(curr)
	}
	if desired, err := c.MarshalSpec(diff.To); err == nil {
		f.Schema.Desired = string(desired)
	}
}

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
