// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

var (
	// ColorTemplateFuncs are globally available functions to color strings in a report template.
	ColorTemplateFuncs = template.FuncMap{
		"cyan":         color.CyanString,
		"green":        color.HiGreenString,
		"red":          color.HiRedString,
		"redBgWhiteFg": color.New(color.FgHiWhite, color.BgHiRed).SprintFunc(),
		"yellow":       color.YellowString,
	}
)

type (
	// A Report represents collected information about the execution of a 'atlas migrate' command execution.
	Report interface {
		report()
	}

	// ReportWriter writes a Report.
	ReportWriter interface {
		WriteReport(Report) error
	}

	// A TemplateWriter is ReportWriter that writes output according to a template.
	TemplateWriter struct {
		T *template.Template
		W io.Writer
	}

	// Env holds the environment information.
	Env struct {
		Driver string         `json:"Driver,omitempty"` // Driver name.
		URL    *sqlclient.URL `json:"URL,omitempty"`    // URL to dev database.
		Dir    string         `json:"Dir,omitempty"`    // Path to migration directory.
	}

	// Files is a slice of migrate.File. Implements json.Marshaler.
	Files []migrate.File

	// File wraps migrate.File to implement json.Marshaler.
	File struct{ migrate.File }
)

// WriteReport implements ReportWriter.
func (w *TemplateWriter) WriteReport(r Report) error {
	return w.T.Execute(w.W, r)
}

// MarshalJSON implements json.Marshaler.
func (f File) MarshalJSON() ([]byte, error) {
	type local struct {
		Name        string `json:"Name,omitempty"`
		Version     string `json:"Version,omitempty"`
		Description string `json:"Description,omitempty"`
	}
	return json.Marshal(local{f.Name(), f.Version(), f.Desc()})
}

// MarshalJSON implements json.Marshaler.
func (f Files) MarshalJSON() ([]byte, error) {
	files := make([]File, len(f))
	for i := range f {
		files[i] = File{f[i]}
	}
	return json.Marshal(files)
}

// NewEnv returns an initialized Env.
func NewEnv(c *sqlclient.Client, dir migrate.Dir) Env {
	e := Env{
		Driver: c.Name,
		URL:    c.URL,
	}
	if p, ok := dir.(interface{ Path() string }); ok {
		e.Dir = p.Path()
	}
	return e
}

var (
	// StatusTemplateFuncs are global functions available in status report templates.
	StatusTemplateFuncs = merge(template.FuncMap{
		"json":       jsonEncode,
		"json_merge": jsonMerge,
		"table":      table,
		"default": func(report *StatusReport) (string, error) {
			var buf bytes.Buffer
			t, err := template.New("report").Funcs(ColorTemplateFuncs).Parse(`Migration Status:
{{- if eq .Status "OK"      }} {{ green .Status }}{{ end }}
{{- if eq .Status "PENDING" }} {{ yellow .Status }}{{ end }}
  {{ yellow "--" }} Current Version: {{ cyan .Current }}
{{- if gt .Total 0 }}{{ printf " (%s statements applied)" (yellow "%d" .Count) }}{{ end }}
  {{ yellow "--" }} Next Version:    {{ cyan .Next }}
{{- if gt .Total 0 }}{{ printf " (%s statements left)" (yellow "%d" .Left) }}{{ end }}
  {{ yellow "--" }} Executed Files:  {{ len .Applied }}{{ if gt .Total 0 }} (last one partially){{ end }}
  {{ yellow "--" }} Pending Files:   {{ len .Pending }}
{{ if gt .Total 0 }}
Last migration attempt had errors:
  {{ yellow "--" }} SQL:   {{ .SQL }}
  {{ yellow "--" }} {{ red "ERROR:" }} {{ .Error }}
{{ end }}`)
			if err != nil {
				return "", err
			}
			err = t.Execute(&buf, report)
			return buf.String(), err
		},
	}, ColorTemplateFuncs)

	// DefaultStatusTemplate holds the default template of the 'migrate status' command.
	DefaultStatusTemplate = template.Must(template.New("report").Funcs(StatusTemplateFuncs).Parse("{{ default . }}"))
)

type (
	// StatusReporter is used to gather information about migration status.
	StatusReporter struct {
		// Client configures the connection to the database to file a StatusReport for.
		Client *sqlclient.Client
		// Dir is used for scanning and validating the migration directory.
		Dir migrate.Dir
		// ReportWriter writes the summary report.
		ReportWriter ReportWriter
		// Schema name the revision table resides in.
		Schema string
	}

	// StatusReport contains a summary of the migration status of a database.
	StatusReport struct {
		Report    `json:"-"`
		Env       `json:"Env"`
		Available Files               `json:"Available,omitempty"` // Available migration files
		Pending   Files               `json:"Pending,omitempty"`   // Pending migration files
		Applied   []*migrate.Revision `json:"Applied,omitempty"`   // Applied migration files
		Current   string              `json:"Current,omitempty"`   // Current migration version
		Next      string              `json:"Next,omitempty"`      // Next migration version
		Count     int                 `json:"Count,omitempty"`     // Count of applied statements of the last revision
		Total     int                 `json:"Total,omitempty"`     // Total statements of the last migration
		Status    string              `json:"Status,omitempty"`    // Status of migration (OK, PENDING)
		Error     string              `json:"Error,omitempty"`     // Last Error that occurred
		SQL       string              `json:"SQL,omitempty"`       // SQL that caused the last Error
	}
)

// NewStatusReport returns a new StatusReport.
func NewStatusReport(c *sqlclient.Client, dir migrate.Dir) (*StatusReport, error) {
	files, err := dir.Files()
	if err != nil {
		return nil, err
	}
	return &StatusReport{
		Env:       NewEnv(c, dir),
		Available: files,
	}, nil
}

// Report creates and writes a StatusReport.
func (r *StatusReporter) Report(ctx context.Context) error {
	rep, err := NewStatusReport(r.Client, r.Dir)
	if err != nil {
		return err
	}
	// Check if there already is a revision table in the defined schema.
	// Inspect schema and check if the table does already exist.
	sch, err := r.Client.InspectSchema(ctx, r.Schema, &schema.InspectOptions{Tables: []string{revision.Table}})
	if err != nil && !schema.IsNotExistError(err) {
		return err
	}
	if schema.IsNotExistError(err) || func() bool { _, ok := sch.Table(revision.Table); return !ok }() {
		// Either schema or table does not exist.
		rep.Pending = rep.Available
	} else {
		// Both exist, fetch their data.
		rrw, err := NewEntRevisions(ctx, r.Client, WithSchema(r.Schema))
		if err != nil {
			return err
		}
		if err := rrw.Migrate(ctx); err != nil {
			return err
		}
		ex, err := migrate.NewExecutor(r.Client.Driver, r.Dir, rrw)
		if err != nil {
			return err
		}
		rep.Pending, err = ex.Pending(ctx)
		if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
			return err
		}
		rep.Applied, err = rrw.ReadRevisions(ctx)
		if err != nil {
			return err
		}
	}
	switch len(rep.Pending) {
	case len(rep.Available):
		rep.Current = "No migration applied yet"
	default:
		rep.Current = rep.Applied[len(rep.Applied)-1].Version
	}
	if len(rep.Pending) == 0 {
		rep.Status = "OK"
		rep.Next = "Already at latest version"
	} else {
		rep.Status = "PENDING"
		rep.Next = rep.Pending[0].Version()
	}
	// If the last one is partially applied (and not manually resolved).
	if len(rep.Applied) != 0 {
		last := rep.Applied[len(rep.Applied)-1]
		if !last.Type.Has(migrate.RevisionTypeResolved) && last.Applied < last.Total {
			rep.SQL = strings.ReplaceAll(last.ErrorStmt, "\n", " ")
			rep.Error = strings.ReplaceAll(last.Error, "\n", " ")
			rep.Count = last.Applied
			idx := migrate.FilesLastIndex(rep.Available, func(f migrate.File) bool {
				return f.Version() == last.Version
			})
			if idx == -1 {
				return fmt.Errorf("migration file with version %q not found", last.Version)
			}
			stmts, err := rep.Available[idx].Stmts()
			if err != nil {
				return err
			}
			rep.Total = len(stmts)
		}
	}
	return r.ReportWriter.WriteReport(rep)
}

// Left returns the amount of statements left to apply (if any).
func (r *StatusReport) Left() int { return r.Total - r.Count }

func table(report *StatusReport) (string, error) {
	var buf strings.Builder
	tbl := tablewriter.NewWriter(&buf)
	tbl.SetRowLine(true)
	tbl.SetAutoMergeCellsByColumnIndex([]int{0})
	tbl.SetHeader([]string{
		"Version",
		"Description",
		"Status",
		"Count",
		"Executed At",
		"Execution Time",
		"Error",
		"SQL",
	})
	for _, r := range report.Applied {
		tbl.Append([]string{
			r.Version,
			r.Description,
			r.Type.String(),
			fmt.Sprintf("%d/%d", r.Applied, r.Total),
			r.ExecutedAt.Format("2006-01-02 15:04:05 MST"),
			r.ExecutionTime.String(),
			r.Error,
			r.ErrorStmt,
		})
	}
	for i, f := range report.Pending {
		var c string
		if i == 0 {
			if r := report.Applied[len(report.Applied)-1]; f.Version() == r.Version && r.Applied < r.Total {
				stmts, err := f.Stmts()
				if err != nil {
					return "", err
				}
				c = fmt.Sprintf("%d/%d", len(stmts)-r.Applied, len(stmts))
			}
		}
		tbl.Append([]string{
			f.Version(),
			f.Desc(),
			"pending",
			c,
			"", "", "", "",
		})
	}
	tbl.Render()
	return buf.String(), nil
}

var (
	// ApplyTemplateFuncs are global functions available in apply report templates.
	ApplyTemplateFuncs = merge(ColorTemplateFuncs, template.FuncMap{
		"dec":        dec,
		"json":       jsonEncode,
		"json_merge": jsonMerge,
	})

	// DefaultApplyTemplate holds the default template of the 'migrate apply' command.
	DefaultApplyTemplate = template.Must(template.
				New("report").
				Funcs(ApplyTemplateFuncs).
				Parse(`Migrating to version {{ cyan .Target }}{{ with .Current }} from {{ cyan . }}{{ end }} ({{ len .Pending }} migrations in total):
{{ range $i, $f := .Applied }}
  {{ yellow "--" }} migrating version {{ cyan $f.File.Version }}{{ range $f.Applied }}
    {{ cyan "->" }} {{ . }}{{ end }}
  {{- with .Error }}
    {{ redBgWhiteFg .Error }}
  {{- else }}
  {{ yellow "--" }} ok ({{ yellow (.End.Sub .Start).String }})
  {{- end }}
{{ end }}
  {{ cyan "-------------------------" }}
  {{ yellow "--" }} {{ .End.Sub .Start }}
{{- $files := len .Applied }}
{{- $stmts := .CountStmts }}
{{- if .Error }}
  {{ yellow "--" }} {{ dec $files }} migrations ok (1 with errors)
  {{ yellow "--" }} {{ dec $stmts }} sql statements ok (1 with errors)
{{- else }}
  {{ yellow "--" }} {{ len .Applied }} migrations 
  {{ yellow "--" }} {{ .CountStmts  }} sql statements
{{- end }}
`))
)

type (
	// ApplyReport contains a summary of a migration applying attempt on a database.
	ApplyReport struct {
		Report `json:"-"`
		Env
		Pending Files          `json:"Pending,omitempty"` // Pending migration files
		Applied []*AppliedFile `json:"Applied,omitempty"` // Applied files
		Current string         `json:"Current,omitempty"` // Current migration version
		Target  string         `json:"Target,omitempty"`  // Target migration version
		Start   time.Time
		End     time.Time
		// Error is set even then, if it was not caused by a statement in a migration file,
		// but by Atlas, e.g. when committing or rolling back a transaction.
		Error string `json:"Error,omitempty"`
	}

	// AppliedFile is part of an ApplyReport containing information about an applied file in a migration attempt.
	AppliedFile struct {
		migrate.File
		Start   time.Time
		End     time.Time
		Skipped int      // Amount of skipped SQL statements in a partially applied file.
		Applied []string // SQL statements applied with success
		Error   *struct {
			SQL   string // SQL statement that failed.
			Error string // Error returned by the database.
		}
	}
)

// NewApplyReport returns an ApplyReport.
func NewApplyReport(client *sqlclient.Client, dir migrate.Dir) *ApplyReport {
	return &ApplyReport{
		Env: NewEnv(client, dir),
	}
}

// Log implements migrate.Logger.
func (a *ApplyReport) Log(e migrate.LogEntry) {
	switch e := e.(type) {
	case migrate.LogExecution:
		a.Start = time.Now()
		a.Current = e.From
		a.Target = e.To
		a.Pending = e.Files
	case migrate.LogFile:
		if l := len(a.Applied); l > 0 {
			f := a.Applied[l-1]
			f.End = time.Now()
		}
		a.Applied = append(a.Applied, &AppliedFile{
			File:    File{e.File},
			Start:   time.Now(),
			Skipped: e.Skip,
		})
	case migrate.LogStmt:
		f := a.Applied[len(a.Applied)-1]
		f.Applied = append(f.Applied, e.SQL)
	case migrate.LogError:
		if l := len(a.Applied); l > 0 {
			f := a.Applied[len(a.Applied)-1]
			f.End = time.Now()
			a.End = f.End
			f.Error = &struct {
				SQL   string
				Error string
			}{e.SQL, e.Error.Error()}
		}
	case migrate.LogDone:
		f := a.Applied[len(a.Applied)-1]
		f.End = time.Now()
		a.End = f.End
	}
}

// CountStmts returns the amount of applied statements.
func (a *ApplyReport) CountStmts() (n int) {
	for _, f := range a.Applied {
		n += len(f.Applied)
	}
	return
}

// MarshalJSON implements json.Marshaler.
func (f *AppliedFile) MarshalJSON() ([]byte, error) {
	type local struct {
		Name        string    `json:"Name,omitempty"`
		Version     string    `json:"Version,omitempty"`
		Description string    `json:"Description,omitempty"`
		Start       time.Time `json:"Start,omitempty"`
		End         time.Time `json:"End,omitempty"`
		Skipped     int       `json:"Skipped,omitempty"`
		Stmts       []string  `json:"Applied,omitempty"`
		Error       *struct {
			SQL   string
			Error string
		} `json:"Error,omitempty"`
	}
	return json.Marshal(local{
		Name:        f.Name(),
		Version:     f.Version(),
		Description: f.Desc(),
		Start:       f.Start,
		End:         f.End,
		Skipped:     f.Skipped,
		Stmts:       f.Applied,
		Error:       f.Error,
	})
}

func merge(maps ...template.FuncMap) template.FuncMap {
	switch len(maps) {
	case 0:
		return nil
	case 1:
		return maps[0]
	default:
		m := maps[0]
		for _, e := range maps[1:] {
			for k, v := range e {
				m[k] = v
			}
		}
		return m
	}
}

func jsonEncode(v any, args ...string) (string, error) {
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
}

func jsonMerge(objects ...string) (string, error) {
	var r map[string]any
	for i := range objects {
		if err := json.Unmarshal([]byte(objects[i]), &r); err != nil {
			return "", fmt.Errorf("json_merge: %w", err)
		}
	}
	b, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("json_merge: %w", err)
	}
	return string(b), nil
}

func dec(i int) int {
	return i - 1
}
