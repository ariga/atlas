// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

var (
	ColorCyan         = color.CyanString
	ColorGreen        = color.HiGreenString
	ColorRed          = color.HiRedString
	ColorRedBgWhiteFg = color.New(color.FgHiWhite, color.BgHiRed).SprintFunc()
	ColorYellow       = color.YellowString
	// ColorTemplateFuncs are globally available functions to color strings in a report template.
	ColorTemplateFuncs = template.FuncMap{
		"cyan":         ColorCyan,
		"green":        ColorGreen,
		"red":          ColorRed,
		"redBgWhiteFg": ColorRedBgWhiteFg,
		"yellow":       ColorYellow,
	}
)

type (
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

	// FileChecks represents a set of checks to run before applying a file.
	FileChecks struct {
		Name  string     `json:"Name,omitempty"`  // File/group name.
		Stmts []*Check   `json:"Stmts,omitempty"` // Checks statements executed.
		Error *StmtError `json:"Error,omitempty"` // Assertion error.
		Start time.Time  `json:"Start,omitempty"` // Start assertion time.
		End   time.Time  `json:"End,omitempty"`   // End assertion time.
	}

	// Check represents an assertion and its status.
	Check struct {
		Stmt  string  `json:"Stmt,omitempty"`  // Assertion statement.
		Error *string `json:"Error,omitempty"` // Assertion error, if any.
	}

	// StmtError groups a statement with its execution error.
	StmtError struct {
		Stmt string `json:"Stmt,omitempty"` // SQL statement that failed.
		Text string `json:"Text,omitempty"` // Error message as returned by the database.
	}
)

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
		"default": func(report *MigrateStatus) (string, error) {
			var buf bytes.Buffer
			t, err := template.New("report").
				Funcs(template.FuncMap{
					"add": add,
				}).
				Funcs(ColorTemplateFuncs).
				Parse(`Migration Status:
{{- if eq .Status "OK" }} {{ green .Status }}{{ end }}
{{- if eq .Status "PENDING" }} {{ yellow .Status }}{{ end }}
  {{ yellow "--" }} Current Version: {{ cyan .Current }}
{{- if gt .Total 0 }}{{ printf " (%s statements applied)" (yellow "%d" .Count) }}{{ end }}
  {{ yellow "--" }} Next Version:    {{ if .Next }}{{ cyan .Next }}{{ if .FromCheckpoint }} (checkpoint){{ end }}{{ else }}UNKNOWN{{ end }}
{{- if gt .Total 0 }}{{ printf " (%s statements left)" (yellow "%d" .Left) }}{{ end }}
  {{ yellow "--" }} Executed Files:  {{ len .Applied }}{{ if gt .Total 0 }} (last one partially){{ end }}
  {{ yellow "--" }} Pending Files:   {{ add (len .Pending) (len .OutOfOrder) }}{{ if .OutOfOrder }} ({{ if .Pending }}{{ len .OutOfOrder }} {{ end }}out of order){{ end }}
{{- if gt .Total 0 }}

Last migration attempt had errors:
  {{ yellow "--" }} SQL:   {{ .SQL }}
  {{ yellow "--" }} {{ red "ERROR:" }} {{ .Error }}
{{- else if and .OutOfOrder .Error }}

  {{ red "ERROR:" }} {{ .Error }}
{{- end }}
`)
			if err != nil {
				return "", err
			}
			err = t.Execute(&buf, report)
			return buf.String(), err
		},
	}, ColorTemplateFuncs)

	// MigrateStatusTemplate holds the default template of the 'migrate status' command.
	MigrateStatusTemplate = template.Must(template.New("report").Funcs(StatusTemplateFuncs).Parse("{{ default . }}"))
)

// MigrateStatus contains a summary of the migration status of a database.
type MigrateStatus struct {
	context.Context `json:"-"`
	Env             `json:"Env"`
	Available       Files               `json:"Available,omitempty"`  // Available migration files
	OutOfOrder      Files               `json:"OutOfOrder,omitempty"` // OutOfOrder migration files
	Pending         Files               `json:"Pending,omitempty"`    // Pending migration files
	Applied         []*migrate.Revision `json:"Applied,omitempty"`    // Applied migration files
	Current         string              `json:"Current,omitempty"`    // Current migration version
	Next            string              `json:"Next,omitempty"`       // Next migration version
	Count           int                 `json:"Count,omitempty"`      // Count of applied statements of the last revision
	Total           int                 `json:"Total,omitempty"`      // Total statements of the last migration
	Status          string              `json:"Status,omitempty"`     // Status of migration (OK, PENDING)
	Error           string              `json:"Error,omitempty"`      // Last Error that occurred
	SQL             string              `json:"SQL,omitempty"`        // SQL that caused the last Error
}

// Left returns the amount of statements left to apply (if any).
func (r *MigrateStatus) Left() int { return r.Total - r.Count }

// FromCheckpoint reports if we start from a checkpoint version
// Hence, the first file to be executed on the database is checkpoint.
func (r *MigrateStatus) FromCheckpoint() bool {
	if len(r.Applied) > 0 || len(r.Pending) == 0 || r.Pending[0].Version() != r.Next {
		return false
	}
	ck, ok := r.Pending[0].(migrate.CheckpointFile)
	return ok && ck.IsCheckpoint()
}

func table(report *MigrateStatus) (string, error) {
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

// MigrateSetTemplate holds the default template of the 'migrate set' command.
var MigrateSetTemplate = template.Must(template.New("set").
	Funcs(ColorTemplateFuncs).Parse(`
{{- if and (not .Current) .Revisions -}}
All revisions deleted ({{ len .Revisions }} in total):
{{ else if and .Current .Revisions -}}
Current version is {{ cyan .Current.Version }} ({{ .Summary }}):
{{ end }}
{{- if .Revisions }}
{{ range .ByVersion }}
  {{- $text := .ColoredVersion }}{{ with .Description }}{{ $text = printf "%s (%s)" $text . }}{{ end }}
  {{- printf "  %s\n" $text }}
{{- end }}
{{ end -}}
`))

type (
	// MigrateSet contains a summary of the migrate set command.
	MigrateSet struct {
		context.Context `json:"-"`
		// Revisions that were added, removed or updated.
		Revisions []RevisionOp `json:"Revisions,omitempty"`
		// Current version in the revisions table.
		Current *migrate.Revision `json:"Latest,omitempty"`
	}
	// RevisionOp represents an operation done on a revision.
	RevisionOp struct {
		*migrate.Revision
		Op string `json:"Op,omitempty"`
	}
)

// ByVersion returns all revisions sorted by version.
func (r *MigrateSet) ByVersion() []RevisionOp {
	sort.Slice(r.Revisions, func(i, j int) bool {
		return r.Revisions[i].Version < r.Revisions[j].Version
	})
	return r.Revisions
}

// Set records revision that was added.
func (r *MigrateSet) Set(rev *migrate.Revision) {
	r.Revisions = append(r.Revisions, RevisionOp{Revision: rev, Op: "set"})
}

// Removed records revision that was added.
func (r *MigrateSet) Removed(rev *migrate.Revision) {
	r.Revisions = append(r.Revisions, RevisionOp{Revision: rev, Op: "remove"})
}

// Summary returns a summary of the set operation.
func (r *MigrateSet) Summary() string {
	var s, d int
	for i := range r.Revisions {
		switch r.Revisions[i].Op {
		case "set":
			s++
		default:
			d++
		}
	}
	var sum []string
	if s > 0 {
		sum = append(sum, fmt.Sprintf("%d set", s))
	}
	if d > 0 {
		sum = append(sum, fmt.Sprintf("%d removed", d))
	}
	return strings.Join(sum, ", ")
}

// ColoredVersion returns the version of the revision with a color.
func (r *RevisionOp) ColoredVersion() string {
	c := color.HiGreenString("+")
	if r.Op != "set" {
		c = color.HiRedString("-")
	}
	return c + " " + r.Version
}

var (
	// ApplyTemplateFuncs are global functions available in apply report templates.
	ApplyTemplateFuncs = merge(ColorTemplateFuncs, template.FuncMap{
		"add":        add,
		"upper":      strings.ToUpper,
		"json":       jsonEncode,
		"json_merge": jsonMerge,
		"indent_ln":  indentLn,
	})

	// MigrateApplyTemplate holds the default template of the 'migrate apply' command.
	MigrateApplyTemplate = template.Must(template.
				New("report").
				Funcs(ApplyTemplateFuncs).
				Parse(`{{- if not .Pending -}}
{{- println "No migration files to execute" }}
{{- else -}}
Migrating to version {{ cyan .Target }}{{ with .Current }} from {{ cyan . }}{{ end }} ({{ len .Pending }} migrations in total):
{{ range $i, $f := .Applied }}
	{{- println }}
	{{- $checkFailed := false }}
	{{- range $cf := $f.Checks }}
		{{- println " " (yellow "--") "checks before migrating version" (cyan $f.File.Version) }}
		{{- range $s := $cf.Stmts }}
			{{- if $s.Error }}
				{{- println "   " (red "->") (indent_ln $s.Stmt 7) }}
			{{- else }}
				{{- println "   " (cyan "->") (indent_ln $s.Stmt 7) }}
			{{- end }}
		{{- end }}
		{{- with $cf.Error }}
			{{- $checkFailed = true }}
			{{- println "   " (redBgWhiteFg .Text) }}
		{{- else }}
			{{- printf "  %s ok (%s)\n\n" (yellow "--") (yellow ($cf.End.Sub $cf.Start).String) }}
		{{- end }}
	{{- end }}
	{{- if $checkFailed }}
		{{- continue }} {{- /* No statements were applied. */}}
	{{- end }}
	{{- println " " (yellow "--") "migrating version" (cyan $f.File.Version) }}
	{{- range $f.Applied }}
		{{- println "   " (cyan "->") (indent_ln . 7) }}
	{{- end }}
	{{- with .Error }}
		{{- println "   " (redBgWhiteFg .Text) }}
	{{- else }}
		{{- printf "  %s ok (%s)\n" (yellow "--") (yellow (.End.Sub .Start).String) }}
	{{- end }}
{{- end }}
{{- println }}
{{- println " " (cyan "-------------------------") }}
{{- println " " (.Summary "  ") }}
{{- end -}}
`))
)

type (
	// MigrateApply contains a summary of a migration applying attempt on a database.
	MigrateApply struct {
		Env
		context.Context `json:"-"`
		Pending         Files          `json:"Pending,omitempty"` // Pending migration files
		Applied         []*AppliedFile `json:"Applied,omitempty"` // Applied files
		Current         string         `json:"Current,omitempty"` // Current migration version
		Target          string         `json:"Target,omitempty"`  // Target migration version
		Start           time.Time
		End             time.Time
		// Error is set even then, if it was not caused by a statement in a migration file,
		// but by Atlas, e.g. when committing or rolling back a transaction.
		Error string `json:"Error,omitempty"`
	}

	// AppliedFile is part of an MigrateApply containing information about an applied file in a migration attempt.
	AppliedFile struct {
		migrate.File
		Start   time.Time
		End     time.Time
		Skipped int           // Amount of skipped SQL statements in a partially applied file.
		Applied []string      // SQL statements applied with success
		Checks  []*FileChecks // Assertion checks
		Error   *StmtError
	}
)

// NewMigrateApply returns an MigrateApply.
func NewMigrateApply(ctx context.Context, client *sqlclient.Client, dir migrate.Dir) *MigrateApply {
	return &MigrateApply{
		Context: ctx,
		Env:     NewEnv(client, dir),
		Start:   time.Now(),
	}
}

// Log implements migrate.Logger.
func (a *MigrateApply) Log(e migrate.LogEntry) {
	switch e := e.(type) {
	case migrate.LogExecution:
		// Do not set start time if it
		// was set by the constructor.
		if a.Start.IsZero() {
			a.Start = time.Now()
		}
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
	case migrate.LogChecks:
		f := a.Applied[len(a.Applied)-1]
		f.Checks = append(f.Checks, &FileChecks{
			Name:  e.Name,
			Start: time.Now(),
			Stmts: make([]*Check, 0, len(e.Stmts)),
		})
	case migrate.LogCheck:
		var (
			f  = a.Applied[len(a.Applied)-1]
			cf = f.Checks[len(f.Checks)-1]
			ck = &Check{Stmt: e.Stmt}
		)
		if e.Error != nil {
			m := e.Error.Error()
			ck.Error = &m
		}
		cf.Stmts = append(cf.Stmts, ck)
	case migrate.LogChecksDone:
		f := a.Applied[len(a.Applied)-1]
		cf := f.Checks[len(f.Checks)-1]
		cf.End = time.Now()
		if e.Error != nil {
			cf.Error = &StmtError{
				Text: e.Error.Error(),
				Stmt: cf.Stmts[len(cf.Stmts)-1].Stmt,
			}
		}
	case migrate.LogStmt:
		f := a.Applied[len(a.Applied)-1]
		f.Applied = append(f.Applied, e.SQL)
	case migrate.LogError:
		if l := len(a.Applied); l > 0 {
			f := a.Applied[len(a.Applied)-1]
			f.End = time.Now()
			a.End = f.End
			f.Error = &StmtError{
				Stmt: e.SQL,
				Text: e.Error.Error(),
			}
		}
	case migrate.LogDone:
		n := time.Now()
		if l := len(a.Applied); l > 0 {
			a.Applied[l-1].End = n
		}
		a.End = n
	}
}

// Summary returns a footer of the migration attempt.
func (a *MigrateApply) Summary(ident string) string {
	var (
		passedC, failedC int
		passedS, failedS int
		passedF, failedF int
		lines            = make([]string, 0, 3)
	)
	for _, f := range a.Applied {
		// For each check file, count the
		// number of failed assertions.
		for _, cf := range f.Checks {
			for _, s := range cf.Stmts {
				if s.Error != nil {
					failedC++
				} else {
					passedC++
				}
			}
		}
		passedS += len(f.Applied)
		if f.Error != nil {
			failedF++
			// Last statement failed (not an assertion).
			if len(f.Checks) == 0 || f.Checks[len(f.Checks)-1].Error == nil {
				passedS--
				failedS++
			}
		} else {
			passedF++
		}
	}
	// Execution time.
	lines = append(lines, fmt.Sprintf(a.End.Sub(a.Start).String()))
	// Executed files.
	switch {
	case passedF > 0 && failedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s ok, %d with errors", passedF, plural(passedF), failedF))
	case passedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s", passedF, plural(passedF)))
	case failedF > 0:
		lines = append(lines, fmt.Sprintf("%d migration%s with errors", failedF, plural(failedF)))
	}
	// Executed checks.
	switch {
	case passedC > 0 && failedC > 0:
		lines = append(lines, fmt.Sprintf("%d check%s ok, %d failure%s", passedC, plural(passedC), failedC, plural(failedC)))
	case passedC > 0:
		lines = append(lines, fmt.Sprintf("%d check%s", passedC, plural(passedC)))
	case failedC > 0:
		lines = append(lines, fmt.Sprintf("%d check error%s", failedC, plural(failedC)))
	}
	// Executed statements.
	switch {
	case passedS > 0 && failedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s ok, %d with errors", passedS, plural(passedS), failedS))
	case passedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s", passedS, plural(passedS)))
	case failedS > 0:
		lines = append(lines, fmt.Sprintf("%d sql statement%s with errors", failedS, plural(failedS)))
	}
	var b strings.Builder
	for i, l := range lines {
		b.WriteString(ColorYellow("--"))
		b.WriteByte(' ')
		b.WriteString(l)
		if i < len(lines)-1 {
			b.WriteByte('\n')
			b.WriteString(ident)
		}
	}
	return b.String()
}

func plural(n int) (s string) {
	if n > 1 {
		s += "s"
	}
	return
}

// MarshalJSON implements json.Marshaler.
func (a *MigrateApply) MarshalJSON() ([]byte, error) {
	type Alias MigrateApply
	var v struct {
		*Alias
		Message string `json:"Message,omitempty"`
	}
	v.Alias = (*Alias)(a)
	switch {
	case a.Error != "":
	case len(v.Applied) == 0:
		v.Message = "No migration files to execute"
	default:
		v.Message = fmt.Sprintf("Migrated to version %s from %s (%d migrations in total)", v.Target, v.Current, len(v.Pending))
	}
	return json.Marshal(v)
}

// MarshalJSON implements json.Marshaler.
func (f *AppliedFile) MarshalJSON() ([]byte, error) {
	type local struct {
		Name        string     `json:"Name,omitempty"`
		Version     string     `json:"Version,omitempty"`
		Description string     `json:"Description,omitempty"`
		Start       time.Time  `json:"Start,omitempty"`
		End         time.Time  `json:"End,omitempty"`
		Skipped     int        `json:"Skipped,omitempty"`
		Stmts       []string   `json:"Applied,omitempty"`
		Error       *StmtError `json:"Error,omitempty"`
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

// SchemaPlanTemplate holds the default template of the 'schema apply --dry-run' command.
var SchemaPlanTemplate = template.Must(template.
	New("plan").
	Funcs(ApplyTemplateFuncs).
	Parse(`{{- with .Changes.Pending -}}
-- Planned Changes:
{{ range . -}}
{{- if .Comment -}}
{{- printf "-- %s%s\n" (slice .Comment 0 1 | upper ) (slice .Comment 1) -}}
{{- end -}}
{{- printf "%s;\n" .Cmd -}}
{{- end -}}
{{- else -}}
Schema is synced, no changes to be made
{{ end -}}
`))

type (
	// SchemaApply contains a summary of a 'schema apply' execution on a database.
	SchemaApply struct {
		Env
		context.Context `json:"-"`
		Changes         Changes `json:"Changes,omitempty"`
		// General error that occurred during execution.
		// e.g., when committing or rolling back a transaction.
		Error string `json:"Error,omitempty"`
	}
	// Changes represents a list of changes that are pending or applied.
	Changes struct {
		Applied []*migrate.Change `json:"Applied,omitempty"` // SQL changes applied with success
		Pending []*migrate.Change `json:"Pending,omitempty"` // SQL changes that were not applied
		Error   *StmtError        `json:"Error,omitempty"`   // Error that occurred during applying
	}
)

// NewSchemaApply returns a SchemaApply.
func NewSchemaApply(ctx context.Context, env Env, applied, pending []*migrate.Change, err *StmtError) *SchemaApply {
	return &SchemaApply{
		Env:     env,
		Context: ctx,
		Changes: Changes{
			Applied: applied,
			Pending: pending,
			Error:   err,
		},
	}
}

// NewSchemaPlan returns a SchemaApply only with pending changes.
func NewSchemaPlan(ctx context.Context, env Env, pending []*migrate.Change, err *StmtError) *SchemaApply {
	return NewSchemaApply(ctx, env, nil, pending, err)
}

// MarshalJSON implements json.Marshaler.
func (c Changes) MarshalJSON() ([]byte, error) {
	var v struct {
		Applied []string   `json:"Applied,omitempty"`
		Pending []string   `json:"Pending,omitempty"`
		Error   *StmtError `json:"Error,omitempty"`
	}
	for i := range c.Applied {
		v.Applied = append(v.Applied, c.Applied[i].Cmd)
	}
	for i := range c.Pending {
		v.Pending = append(v.Pending, c.Pending[i].Cmd)
	}
	v.Error = c.Error
	return json.Marshal(v)
}

// SchemaInspect contains a summary of the 'schema inspect' command.
type SchemaInspect struct {
	context.Context   `json:"-"`
	*sqlclient.Client `json:"-"`
	Realm             *schema.Realm `json:"Schema,omitempty"` // Inspected realm.
}

var (
	// InspectTemplateFuncs are global functions available in inspect report templates.
	InspectTemplateFuncs = template.FuncMap{
		"sql":     sqlInspect,
		"json":    jsonEncode,
		"mermaid": mermaid,
	}

	// SchemaInspectTemplate holds the default template of the 'schema inspect' command.
	SchemaInspectTemplate = template.Must(template.New("inspect").
				Funcs(InspectTemplateFuncs).
				Parse(`{{ $.MarshalHCL }}`))
)

// MarshalHCL returns the default HCL representation of the schema.
// Used by the template declared above.
func (s *SchemaInspect) MarshalHCL() (string, error) {
	spec, err := s.MarshalSpec(s.Realm)
	if err != nil {
		return "", err
	}
	return string(spec), nil
}

// MarshalJSON implements json.Marshaler.
func (s *SchemaInspect) MarshalJSON() ([]byte, error) {
	type (
		Attrs struct {
			Comment string `json:"comment,omitempty"`
			Charset string `json:"charset,omitempty"`
			Collate string `json:"collate,omitempty"`
		}
		Column struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
			Null bool   `json:"null,omitempty"`
			Attrs
		}
		IndexPart struct {
			Desc   bool   `json:"desc,omitempty"`
			Column string `json:"column,omitempty"`
			Expr   string `json:"expr,omitempty"`
		}
		Index struct {
			Name   string      `json:"name,omitempty"`
			Unique bool        `json:"unique,omitempty"`
			Parts  []IndexPart `json:"parts,omitempty"`
		}
		ForeignKey struct {
			Name       string   `json:"name"`
			Columns    []string `json:"columns,omitempty"`
			References struct {
				Table   string   `json:"table"`
				Columns []string `json:"columns,omitempty"`
			} `json:"references"`
		}
		Table struct {
			Name        string       `json:"name"`
			Columns     []Column     `json:"columns,omitempty"`
			Indexes     []Index      `json:"indexes,omitempty"`
			PrimaryKey  *Index       `json:"primary_key,omitempty"`
			ForeignKeys []ForeignKey `json:"foreign_keys,omitempty"`
			Attrs
		}
		Schema struct {
			Name   string  `json:"name"`
			Tables []Table `json:"tables,omitempty"`
			Attrs
		}
	)
	var (
		realm struct {
			Schemas []Schema `json:"schemas,omitempty"`
		}
		setAttrs = func(from []schema.Attr, to *Attrs) {
			for i := range from {
				switch a := from[i].(type) {
				case *schema.Comment:
					to.Comment = a.Text
				case *schema.Charset:
					to.Charset = a.V
				case *schema.Collation:
					to.Collate = a.V
				}
			}
		}
	)
	for _, s1 := range s.Realm.Schemas {
		s2 := Schema{Name: s1.Name}
		setAttrs(s1.Attrs, &s2.Attrs)
		for _, t1 := range s1.Tables {
			t2 := Table{Name: t1.Name}
			setAttrs(t1.Attrs, &t2.Attrs)
			for _, c1 := range t1.Columns {
				c2 := Column{
					Name: c1.Name,
					Type: c1.Type.Raw,
					Null: c1.Type.Null,
				}
				setAttrs(c1.Attrs, &c2.Attrs)
				t2.Columns = append(t2.Columns, c2)
			}
			idxParts := func(idx *schema.Index) (parts []IndexPart) {
				for _, p1 := range idx.Parts {
					p2 := IndexPart{Desc: p1.Desc}
					switch {
					case p1.C != nil:
						p2.Column = p1.C.Name
					case p1.X != nil:
						switch t := p1.X.(type) {
						case *schema.Literal:
							p2.Expr = t.V
						case *schema.RawExpr:
							p2.Expr = t.X
						}
					}
					parts = append(parts, p2)
				}
				return parts
			}
			for _, idx1 := range t1.Indexes {
				t2.Indexes = append(t2.Indexes, Index{
					Name:   idx1.Name,
					Unique: idx1.Unique,
					Parts:  idxParts(idx1),
				})
			}
			if t1.PrimaryKey != nil {
				t2.PrimaryKey = &Index{Parts: idxParts(t1.PrimaryKey)}
			}
			for _, fk1 := range t1.ForeignKeys {
				fk2 := ForeignKey{Name: fk1.Symbol}
				for _, c1 := range fk1.Columns {
					fk2.Columns = append(fk2.Columns, c1.Name)
				}
				fk2.References.Table = fk1.RefTable.Name
				for _, c1 := range fk1.RefColumns {
					fk2.References.Columns = append(fk2.References.Columns, c1.Name)
				}
				t2.ForeignKeys = append(t2.ForeignKeys, fk2)
			}
			s2.Tables = append(s2.Tables, t2)
		}
		realm.Schemas = append(realm.Schemas, s2)
	}
	return json.Marshal(realm)
}

// MarshalSQL returns the default SQL representation of the schema.
func (s *SchemaInspect) MarshalSQL(indent ...string) (string, error) {
	return sqlInspect(s, indent...)
}

func sqlInspect(report *SchemaInspect, indent ...string) (string, error) {
	var changes schema.Changes
	for _, s := range report.Realm.Schemas {
		// Generate commands for creating the schemas on realm-mode.
		if report.Client.URL.Schema == "" {
			changes = append(changes, &schema.AddSchema{S: s})
		}
		for _, o := range s.Objects {
			changes = append(changes, &schema.AddObject{O: o})
		}
		for _, t := range s.Tables {
			changes = append(changes, &schema.AddTable{T: t})
			for _, r := range t.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, v := range s.Views {
			changes = append(changes, &schema.AddView{V: v})
			for _, r := range v.Triggers {
				changes = append(changes, &schema.AddTrigger{T: r})
			}
		}
		for _, f := range s.Funcs {
			changes = append(changes, &schema.AddFunc{F: f})
		}
		for _, p := range s.Procs {
			changes = append(changes, &schema.AddProc{P: p})
		}
	}
	return fmtPlan(report.Context, report.Client, changes, indent)
}

// SchemaDiff contains a summary of the 'schema diff' command.
type SchemaDiff struct {
	context.Context   `json:"-"`
	*sqlclient.Client `json:"-"`
	From, To          *schema.Realm
	Changes           []schema.Change
}

var (
	// SchemaDiffFuncs are global functions available in diff report templates.
	SchemaDiffFuncs = template.FuncMap{
		"sql": sqlDiff,
	}
	// SchemaDiffTemplate holds the default template of the 'schema diff' command.
	SchemaDiffTemplate = template.Must(template.
				New("schema_diff").
				Funcs(SchemaDiffFuncs).
				Parse(`{{- with .Changes -}}
{{ sql $ }}
{{- else -}}
Schemas are synced, no changes to be made.
{{ end -}}
`))
)

// MarshalSQL returns the default SQL representation of the schema.
func (s *SchemaDiff) MarshalSQL(indent ...string) (string, error) {
	return sqlDiff(s, indent...)
}

func sqlDiff(diff *SchemaDiff, indent ...string) (string, error) {
	return fmtPlan(diff.Context, diff.Client, diff.Changes, indent)
}

func fmtPlan(ctx context.Context, client *sqlclient.Client, changes schema.Changes, indent []string) (string, error) {
	if len(indent) > 1 {
		return "", fmt.Errorf("unexpected number of arguments: %d", len(indent))
	}
	plan, err := client.PlanChanges(ctx, "plan", changes, func(o *migrate.PlanOptions) {
		o.Mode = migrate.PlanModeDump
		// Disable tables qualifier in schema-mode.
		if client.URL.Schema != "" {
			o.SchemaQualifier = new(string)
		}
		if len(indent) > 0 {
			o.Indent = indent[0]
		}
	})
	if err != nil {
		return "", err
	}
	switch files, err := migrate.DefaultFormatter.Format(plan); {
	case err != nil:
		return "", err
	case len(files) != 1:
		return "", fmt.Errorf("unexpected number of files: %d", len(files))
	default:
		return string(files[0].Bytes()), nil
	}
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

func mermaid(i *SchemaInspect, _ ...string) (string, error) {
	ft, ok := i.Driver.(interface {
		FormatType(schema.Type) (string, error)
	})
	if !ok {
		return "", fmt.Errorf("mermaid: driver does not support FormatType")
	}
	var (
		b       strings.Builder
		qualify = len(i.Realm.Schemas) > 1
		funcs   = template.FuncMap{
			"nospace":    strings.NewReplacer(" ", "_").Replace,
			"formatType": ft.FormatType,
			"tableName": func(t *schema.Table) string {
				if qualify {
					return fmt.Sprintf("%[1]s_%[2]s[\"%[1]s.%[2]s\"]", t.Schema.Name, t.Name)
				}
				return t.Name
			},
			"tableIdent": func(t *schema.Table) string {
				if qualify {
					return fmt.Sprintf("%s_%s", t.Schema.Name, t.Name)
				}
				return t.Name
			},
			"pkfk": func(t *schema.Table, c *schema.Column) string {
				var pkfk []string
				if t.PrimaryKey != nil && slices.ContainsFunc(t.PrimaryKey.Parts, func(p *schema.IndexPart) bool { return p.C == c }) {
					pkfk = append(pkfk, "PK")
				}
				if c.ForeignKeys != nil {
					pkfk = append(pkfk, "FK")
				}
				return strings.Join(pkfk, ",")
			},
			"card": func(fk *schema.ForeignKey) string {
				var (
					hasU = func(t *schema.Table, cs []*schema.Column) bool {
						if t.PrimaryKey != nil && slices.EqualFunc(t.PrimaryKey.Parts, cs, func(p *schema.IndexPart, c *schema.Column) bool {
							return p.C != nil && p.C.Name == c.Name
						}) {
							return true
						}
						return slices.ContainsFunc(t.Indexes, func(idx *schema.Index) bool {
							return idx.Unique && slices.EqualFunc(idx.Parts, cs, func(p *schema.IndexPart, c *schema.Column) bool {
								return p.C != nil && p.C.Name == c.Name
							})
						})
					}
					from, to = "}", "{"
				)
				if hasU(fk.Table, fk.Columns) {
					from = "|"
				}
				if hasU(fk.RefTable, fk.RefColumns) {
					to = "|"
				}
				return fmt.Sprintf("%so--o%s", from, to)
			},
		}
		t = template.Must(template.New("mermaid").
			Funcs(funcs).
			Parse(`erDiagram
{{- range $s := .Schemas }}
  {{- range $t := $s.Tables }}
    {{ tableName $t }} {
    {{- range $c := $t.Columns }}
      {{ formatType $c.Type.Type | nospace }} {{ nospace $c.Name }}{{ with pkfk $t $c }} {{ . }}{{ end }}
    {{- end }}
    }
    {{- range $fk := $t.ForeignKeys }}
    {{ tableIdent $t }} {{ card $fk }} {{ tableIdent $fk.RefTable }} : {{ $fk.Symbol }}
    {{- end }}
  {{- end }}
{{- end }}
`))
	)
	if err := t.Execute(&b, i.Realm); err != nil {
		return "", err
	}
	return b.String(), nil
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

func add(a, b int) int {
	return a + b
}

func indentLn(input string, indent int) string {
	pad := strings.Repeat(" ", indent)
	return strings.ReplaceAll(input, "\n", "\n"+pad)
}
