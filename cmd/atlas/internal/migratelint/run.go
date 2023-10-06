// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"text/template"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"golang.org/x/exp/slices"
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

	// Analyzers defines the analysis to be run in the CI job.
	Analyzers []sqlcheck.Analyzer

	// ReportWriter writes the summary report.
	ReportWriter ReportWriter

	// summary report. reset on each run.
	sum *SummaryReport
}

// Run executes the CI job.
func (r *Runner) Run(ctx context.Context) error {
	switch err := r.summary(ctx); err.(type) {
	case nil:
		if err := r.ReportWriter.WriteReport(r.sum); err != nil {
			return err
		}
		// If any of the analyzers returns
		// an error, fail silently.
		for _, f := range r.sum.Files {
			if f.Error != "" {
				return SilentError{error: errors.New(f.Error)}
			}
		}
		return nil
	case *FileError:
		if err := r.ReportWriter.WriteReport(r.sum); err != nil {
			return err
		}
		return SilentError{error: err}
	default:
		return err
	}
}

// A list of steps in CI report.
const (
	StepIntegrityCheck = "Migration Integrity Check"
	StepDetectChanges  = "Detect New Migration Files"
	StepLoadChanges    = "Replay Migration Files"
	StepAnalyzeFile    = "Analyze %s"
)

func (r *Runner) summary(ctx context.Context) error {
	r.sum = NewSummaryReport(r.Dev, r.Dir)

	// Integrity check.
	switch err := migrate.Validate(r.Dir); {
	case errors.Is(err, migrate.ErrChecksumNotFound):
	case err != nil:
		err := &FileError{File: migrate.HashFileName, Err: err}
		r.sum.Files = append(r.sum.Files, &FileReport{Name: migrate.HashFileName, Error: err.Error()})
		return r.sum.StepError(StepIntegrityCheck, fmt.Sprintf("File %s is invalid", migrate.HashFileName), err)
	default:
		// If the hash file exists, it is valid.
		if _, err := fs.Stat(r.Dir, migrate.HashFileName); err == nil {
			r.sum.StepResult(StepIntegrityCheck, fmt.Sprintf("File %s is valid", migrate.HashFileName), nil)
		}
	}

	// Detect new migration files.
	base, feat, err := r.ChangeDetector.DetectChanges(ctx)
	switch err := err.(type) {
	// No error.
	case nil:
		r.sum.StepResult(StepDetectChanges, fmt.Sprintf("Found %d new migration files (from %d total)", len(feat), len(base)+len(feat)), nil)
	// Error that should be reported, but not halt the lint.
	case interface{ StepReport() *StepReport }:
		r.sum.Steps = append(r.sum.Steps, err.StepReport())
	default:
		return r.sum.StepError(StepDetectChanges, "Failed find new migration files", err)
	}

	// Load files into changes.
	l := &DevLoader{Dev: r.Dev}
	diff, err := l.LoadChanges(ctx, base, feat)
	if err != nil {
		if fr := (&FileError{}); errors.As(err, &fr) {
			r.sum.Files = append(r.sum.Files, &FileReport{Name: fr.File, Error: err.Error()})
		}
		return r.sum.StepError(StepLoadChanges, "Failed loading changes on dev database", err)
	}
	r.sum.StepResult(StepLoadChanges, fmt.Sprintf("Loaded %d changes on dev database", len(diff.Files)), nil)
	r.sum.WriteSchema(r.Dev, diff)

	// Analyze files.
	return r.analyze(ctx, diff.Files)
}

// analyze runs the analysis on the given files.
func (r *Runner) analyze(ctx context.Context, files []*sqlcheck.File) error {
	for _, f := range files {
		var (
			es []string
			nl = nolintRules(f)
			fr = NewFileReport(f)
		)
		if nl.ignored {
			continue
		}
		for _, az := range r.Analyzers {
			err := az.Analyze(ctx, &sqlcheck.Pass{
				File:     f,
				Dev:      r.Dev,
				Reporter: nl.reporterFor(fr, az),
			})
			// If the last report was skipped,
			// skip emitting its error.
			if err != nil && !nl.skipped {
				es = append(es, err.Error())
			}
		}
		fr.Error = strings.Join(es, "; ")
		r.sum.Files = append(r.sum.Files, fr)
		r.sum.StepResult(
			fmt.Sprintf(StepAnalyzeFile, f.Name()),
			fmt.Sprintf("%d reports were found in analysis", len(fr.Reports)),
			fr,
		)
	}
	return nil
}

var (
	// TemplateFuncs are global functions available in templates.
	TemplateFuncs = template.FuncMap{
		"json": func(v any, args ...string) (string, error) {
			var (
				b   []byte
				err error
			)
			switch len(args) {
			case 0:
				b, err = json.Marshal(v)
			case 1:
				b, err = json.MarshalIndent(v, "", args[0])
			default:
				b, err = json.MarshalIndent(v, args[0], args[1])
			}
			return string(b), err
		},
	}
	// DefaultTemplate is the default template used by the CI job.
	DefaultTemplate = template.Must(template.New("report").
			Funcs(TemplateFuncs).
			Parse(`
{{- range $f := .Files }}
	{{- /* If there is an error but not diagnostics, print it. */}}
	{{- if and $f.Error (not $f.Reports) }}
		{{- printf "%s: %s\n" $f.Name $f.Error }}
	{{- else }}
		{{- range $r := $f.Reports }}
			{{- if $r.Text }}
				{{- printf "%s: %s:\n\n" $f.Name $r.Text }}
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
	// JSONTemplate is the JSON template used by CI wrappers.
	JSONTemplate = template.Must(template.New("json").
			Funcs(TemplateFuncs).
			Parse("{{ json . }}"))
)

type (
	// A SummaryReport contains a summary of the analysis of all files.
	// It is used as an input to templates to report the CI results.
	SummaryReport struct {
		URL string `json:"URL,omitempty"` // URL of the report, if exists.

		// Env holds the environment information.
		Env struct {
			Driver string         `json:"Driver,omitempty"` // Driver name.
			URL    *sqlclient.URL `json:"URL,omitempty"`    // URL to dev database.
			Dir    string         `json:"Dir,omitempty"`    // Path to migration directory.
		}

		// Schema versions found by the runner.
		Schema struct {
			Current string `json:"Current,omitempty"` // Current schema.
			Desired string `json:"Desired,omitempty"` // Desired schema.
		}

		// Steps of the analysis. Added in verbose mode.
		Steps []*StepReport `json:"Steps,omitempty"`

		// Files reports. Non-empty in case there are findings.
		Files []*FileReport `json:"Files,omitempty"`
	}

	// StepReport contains a summary of the analysis of a single step.
	StepReport struct {
		Name   string      `json:"Name,omitempty"`   // Step name.
		Text   string      `json:"Text,omitempty"`   // Step description.
		Error  string      `json:"Error,omitempty"`  // Error that cause the execution to halt.
		Result *FileReport `json:"Result,omitempty"` // Result of the step. For example, a diagnostic.
	}

	// FileReport contains a summary of the analysis of a single file.
	FileReport struct {
		Name    string            `json:"Name,omitempty"`    // Name of the file.
		Text    string            `json:"Text,omitempty"`    // Contents of the file.
		Reports []sqlcheck.Report `json:"Reports,omitempty"` // List of reports.
		Error   string            `json:"Error,omitempty"`   // File specific error.
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

	// SilentError is returned in case the wrapped error is already
	// printed by the runner and should not be printed by its caller
	SilentError struct{ error }
)

// NewSummaryReport returns a new SummaryReport.
func NewSummaryReport(c *sqlclient.Client, dir migrate.Dir) *SummaryReport {
	sum := &SummaryReport{
		Env: struct {
			Driver string         `json:"Driver,omitempty"`
			URL    *sqlclient.URL `json:"URL,omitempty"`
			Dir    string         `json:"Dir,omitempty"`
		}{
			Driver: c.Name,
			URL:    c.URL,
		},
		Files: make([]*FileReport, 0),
	}
	if p, ok := dir.(interface{ Path() string }); ok {
		sum.Env.Dir = p.Path()
	}
	return sum
}

// StepResult appends step result to the summary.
func (r *SummaryReport) StepResult(name, text string, result *FileReport) {
	r.Steps = append(r.Steps, &StepReport{
		Name:   name,
		Text:   text,
		Result: result,
	})
}

// StepError appends step error to the summary.
func (r *SummaryReport) StepError(name, text string, err error) error {
	r.Steps = append(r.Steps, &StepReport{
		Name:  name,
		Text:  text,
		Error: err.Error(),
	})
	return err
}

// WriteSchema writes the current and desired schema to the summary.
func (r *SummaryReport) WriteSchema(c *sqlclient.Client, diff *Changes) {
	if curr, err := c.MarshalSpec(diff.From); err == nil {
		r.Schema.Current = string(curr)
	}
	if desired, err := c.MarshalSpec(diff.To); err == nil {
		r.Schema.Desired = string(desired)
	}
}

// DiagnosticsCount returns the total number of diagnostics in the report.
func (r *SummaryReport) DiagnosticsCount() int {
	var n int
	for _, f := range r.Files {
		for _, r := range f.Reports {
			n += len(r.Diagnostics)
		}
	}
	return n
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

func nolintRules(f *sqlcheck.File) *skipRules {
	s := &skipRules{pos2rules: make(map[int][]string)}
	if l, ok := f.File.(*migrate.LocalFile); ok {
		ds := l.Directive("nolint")
		// A file directive without specific classes/codes
		// (e.g. atlas:nolint) ignores the entire file.
		if s.ignored = len(ds) == 1 && ds[0] == ""; s.ignored {
			return s
		}
		// A file directive with specific classes/codes applies these
		// rules on all statements (e.g., atlas:nolint destructive).
		for _, d := range ds {
			for _, c := range f.Changes {
				s.pos2rules[c.Stmt.Pos] = append(s.pos2rules[c.Stmt.Pos], strings.Split(d, " ")...)
			}
		}
	}
	for _, c := range f.Changes {
		for _, d := range c.Stmt.Directive("nolint") {
			s.pos2rules[c.Stmt.Pos] = append(s.pos2rules[c.Stmt.Pos], strings.Split(d, " ")...)
		}
	}
	return s
}

type skipRules struct {
	pos2rules map[int][]string // statement positions to rules
	ignored   bool             // file is ignored. i.e., no analysis is performed
	skipped   bool             // if the last report was skipped by the rules
}

func (s *skipRules) reporterFor(rw sqlcheck.ReportWriter, az sqlcheck.Analyzer) sqlcheck.ReportWriter {
	return sqlcheck.ReportWriterFunc(func(r sqlcheck.Report) {
		var (
			ds     = make([]sqlcheck.Diagnostic, 0, len(r.Diagnostics))
			az, ok = az.(sqlcheck.NamedAnalyzer)
		)
		for _, d := range r.Diagnostics {
			switch rules := s.pos2rules[d.Pos]; {
			case
				// A directive without specific classes/codes
				// (e.g. atlas:nolint) ignore all diagnostics.
				len(rules) == 1 && rules[0] == "",
				// Match a specific code/diagnostic. e.g. atlas:nolint DS101.
				slices.Contains(rules, d.Code),
				// Skip the entire analyzer (class of changes).
				ok && slices.Contains(rules, az.Name()):
			default:
				ds = append(ds, d)
			}
		}
		if s.skipped = len(ds) == 0; !s.skipped {
			rw.WriteReport(sqlcheck.Report{Text: r.Text, Diagnostics: ds})
		}
	})
}
