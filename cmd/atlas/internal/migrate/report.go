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

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Reporter is used to gather information about migration status.
type Reporter struct {
	// Client configures the connection to the database to file a StatusReport for.
	Client *sqlclient.Client
	// Dir is used for scanning and validating the migration directory.
	Dir migrate.Dir
	// ReportWriter writes the summary report.
	ReportWriter ReportWriter
	// Schema name the revision table resides in.
	Schema string
}

// Status creates and writes a StatusReport.
func (r *Reporter) Status(ctx context.Context) error {
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
		"table":  table,
		"cyan":   color.CyanString,
		"green":  color.HiGreenString,
		"red":    color.HiRedString,
		"yellow": color.YellowString,
		"default": func(report *StatusReport) (string, error) {
			var buf bytes.Buffer
			t, err := template.New("report").Funcs(template.FuncMap{
				"cyan":   color.CyanString,
				"green":  color.HiGreenString,
				"red":    color.HiRedString,
				"yellow": color.YellowString,
			}).Parse(`Migration Status:
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
	}
	DefaultTemplate = template.Must(template.New("report").Funcs(TemplateFuncs).Parse("{{ default . }}"))
)

type (
	// Env holds the environment information.
	Env struct {
		Driver string         `json:"Driver,omitempty"` // Driver name.
		URL    *sqlclient.URL `json:"URL,omitempty"`    // URL to dev database.
		Dir    string         `json:"Dir,omitempty"`    // Path to migration directory.
	}

	// StatusReport contains a summary of the migration status of a database.
	StatusReport struct {
		Env           `json:"Env"`
		Available     Files               `json:"Available,omitempty"` // Available migration files.
		Pending       Files               `json:"Pending,omitempty"`   // Pending migration files.
		Applied       []*migrate.Revision `json:"Applied,omitempty"`   // Applied migration files.
		Current, Next string              // Current and Next migration version.
		Count, Total  int                 // Count of Total statements applied of the last revision.
		Status        string              // Status of migration (OK, PENDING).
		Error         string              `json:"Error,omitempty"` // Last Error that occurred.
		SQL           string              `json:"SQL,omitempty"`   // SQL that caused the last Error.
	}

	// ReportWriter writes a StatusReport.
	ReportWriter interface {
		WriteReport(*StatusReport) error
	}

	// A TemplateWriter is ReportWrite that writes output according to a template.
	TemplateWriter struct {
		T *template.Template
		W io.Writer
	}

	// Files is a slice of migrate.File. Implements json.Marshaler.
	Files []migrate.File
)

// NewStatusReport returns a new StatusReport.
func NewStatusReport(c *sqlclient.Client, dir migrate.Dir) (*StatusReport, error) {
	files, err := dir.Files()
	if err != nil {
		return nil, err
	}
	sum := &StatusReport{
		Env: Env{
			Driver: c.Name,
			URL:    c.URL,
		},
		Available: files,
	}
	if p, ok := dir.(interface{ Path() string }); ok {
		sum.Env.Dir = p.Path()
	}
	return sum, nil
}

// WriteReport implements ReportWriter.
func (w *TemplateWriter) WriteReport(r *StatusReport) error {
	return w.T.Execute(w.W, r)
}

// MarshalJSON implements json.Marshaler.
func (fs Files) MarshalJSON() ([]byte, error) {
	type file struct {
		Name        string `json:"Name,omitempty"`
		Version     string `json:"Version,omitempty"`
		Description string `json:"Description,omitempty"`
	}
	files := make([]file, len(fs))
	for i, f := range fs {
		files[i] = file{f.Name(), f.Version(), f.Desc()}
	}
	return json.Marshal(files)
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
