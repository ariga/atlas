// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	// MigrateStatusTemplate holds the default template of the 'migrate status' command.
	MigrateStatusTemplate = template.Must(template.New("report").Funcs(StatusTemplateFuncs).Parse("{{ default . }}"))
)

// MigrateStatus contains a summary of the migration status of a database.
type MigrateStatus struct {
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

// NewMigrateStatus returns a new MigrateStatus.
func NewMigrateStatus(c *sqlclient.Client, dir migrate.Dir) (*MigrateStatus, error) {
	files, err := dir.Files()
	if err != nil {
		return nil, err
	}
	return &MigrateStatus{
		Env:       NewEnv(c, dir),
		Available: files,
	}, nil
}

// Left returns the amount of statements left to apply (if any).
func (r *MigrateStatus) Left() int { return r.Total - r.Count }

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
		"dec":        dec,
		"upper":      strings.ToUpper,
		"json":       jsonEncode,
		"json_merge": jsonMerge,
	})

	// MigrateApplyTemplate holds the default template of the 'migrate apply' command.
	MigrateApplyTemplate = template.Must(template.
				New("report").
				Funcs(ApplyTemplateFuncs).
				Parse(`{{- if not .Pending -}}
No migration files to execute
{{- else -}}
Migrating to version {{ cyan .Target }}{{ with .Current }} from {{ cyan . }}{{ end }} ({{ len .Pending }} migrations in total):
{{ range $i, $f := .Applied }}
  {{ yellow "--" }} migrating version {{ cyan $f.File.Version }}{{ range $f.Applied }}
    {{ cyan "->" }} {{ . }}{{ end }}
  {{- with .Error }}
    {{ redBgWhiteFg .Text }}
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
{{- end }}
`))
)

type (
	// MigrateApply contains a summary of a migration applying attempt on a database.
	MigrateApply struct {
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

	// AppliedFile is part of an MigrateApply containing information about an applied file in a migration attempt.
	AppliedFile struct {
		migrate.File
		Start   time.Time
		End     time.Time
		Skipped int      // Amount of skipped SQL statements in a partially applied file.
		Applied []string // SQL statements applied with success
		Error   *StmtError
	}
)

// NewMigrateApply returns an MigrateApply.
func NewMigrateApply(client *sqlclient.Client, dir migrate.Dir) *MigrateApply {
	return &MigrateApply{
		Env:   NewEnv(client, dir),
		Start: time.Now(),
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

// CountStmts returns the amount of applied statements.
func (a *MigrateApply) CountStmts() (n int) {
	for _, f := range a.Applied {
		n += len(f.Applied)
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
		Changes Changes `json:"Changes,omitempty"`
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
func NewSchemaApply(env Env, applied, pending []*migrate.Change, err *StmtError) *SchemaApply {
	return &SchemaApply{
		Env: env,
		Changes: Changes{
			Applied: applied,
			Pending: pending,
			Error:   err,
		},
	}
}

// NewSchemaPlan returns a SchemaApply only with pending changes.
func NewSchemaPlan(env Env, pending []*migrate.Change, err *StmtError) *SchemaApply {
	return NewSchemaApply(env, nil, pending, err)
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
	*sqlclient.Client `json:"-"`
	Realm             *schema.Realm `json:"Schema,omitempty"` // Inspected realm.
	Error             error         `json:"Error,omitempty"`  // General error that occurred during inspection.
}

var (
	// InspectTemplateFuncs are global functions available in inspect report templates.
	InspectTemplateFuncs = template.FuncMap{
		"sql":  sqlInspect,
		"json": jsonEncode,
	}

	// SchemaInspectTemplate holds the default template of the 'schema inspect' command.
	SchemaInspectTemplate = template.Must(template.New("inspect").
				Funcs(InspectTemplateFuncs).
				Parse(`{{ with .Error }}{{ .Error }}{{ else }}{{ $.MarshalHCL }}{{ end }}`))
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
	if s.Error != nil {
		return json.Marshal(struct{ Error string }{s.Error.Error()})
	}
	type (
		Column struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
			Null bool   `json:"null,omitempty"`
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
		}
		Schema struct {
			Name   string  `json:"name"`
			Tables []Table `json:"tables,omitempty"`
		}
	)
	var realm struct {
		Schemas []Schema `json:"schemas,omitempty"`
	}
	for _, s1 := range s.Realm.Schemas {
		s2 := Schema{Name: s1.Name}
		for _, t1 := range s1.Tables {
			t2 := Table{Name: t1.Name}
			for _, c1 := range t1.Columns {
				t2.Columns = append(t2.Columns, Column{
					Name: c1.Name,
					Type: c1.Type.Raw,
					Null: c1.Type.Null,
				})
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

func sqlInspect(report *SchemaInspect, indent ...string) (string, error) {
	if report.Error != nil {
		return report.Error.Error(), nil
	}
	var changes schema.Changes
	for _, s := range report.Realm.Schemas {
		// Generate commands for creating the schemas on realm-mode.
		if report.Client.URL.Schema == "" {
			changes = append(changes, &schema.AddSchema{S: s})
		}
		for _, t := range s.Tables {
			changes = append(changes, &schema.AddTable{T: t})
		}
		for _, v := range s.Views {
			changes = append(changes, &schema.AddView{V: v})
		}
		for _, o := range s.Objects {
			changes = append(changes, &schema.AddObject{O: o})
		}
	}
	return fmtPlan(report.Client, changes, indent)
}

// SchemaDiff contains a summary of the 'schema diff' command.
type SchemaDiff struct {
	*sqlclient.Client
	Changes []schema.Change
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

func sqlDiff(diff *SchemaDiff, indent ...string) (string, error) {
	return fmtPlan(diff.Client, diff.Changes, indent)
}

func fmtPlan(client *sqlclient.Client, changes schema.Changes, indent []string) (string, error) {
	if len(indent) > 1 {
		return "", fmt.Errorf("unexpected number of arguments: %d", len(indent))
	}
	plan, err := client.PlanChanges(context.Background(), "plan", changes, func(o *migrate.PlanOptions) {
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
