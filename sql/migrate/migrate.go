// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"ariga.io/atlas/sql/schema"
)

type (
	// A Plan defines a planned changeset that its execution brings the database to
	// the new desired state. Additional information is calculated by the different
	// drivers to indicate if the changeset is transactional (can be rolled-back) and
	// reversible (a down file can be generated to it).
	Plan struct {
		// Name of the plan. Provided by the user or auto-generated.
		Name string

		// Reversible describes if the changeset is reversible.
		Reversible bool

		// Transactional describes if the changeset is transactional.
		Transactional bool

		// Changes defines the list of changeset in the plan.
		Changes []*Change
	}

	// A Change of migration.
	Change struct {
		// Cmd or statement to execute.
		Cmd string

		// Args for placeholder parameters in the statement above.
		Args []interface{}

		// A Comment describes the change.
		Comment string

		// Reverse contains the "reversed statement" if
		// command is reversible.
		Reverse string

		// The Source that caused this change, or nil.
		Source schema.Change
	}
)

type (
	// The Driver interface must be implemented by the different dialects to support database
	// migration authoring/planning and applying. ExecQuerier, Inspector and Differ, provide
	// basic schema primitives for inspecting database schemas, calculate the difference between
	// schema elements, and executing raw SQL statements. The PlanApplier interface wraps the
	// methods for generating migration plan for applying the actual changes on the database.
	Driver interface {
		schema.Differ
		schema.ExecQuerier
		schema.Inspector
		PlanApplier
	}

	// PlanApplier wraps the methods for planning and applying changes
	// on the database.
	PlanApplier interface {
		// PlanChanges returns a migration plan for applying the given changeset.
		PlanChanges(context.Context, string, []schema.Change) (*Plan, error)

		// ApplyChanges is responsible for applying the given changeset.
		// An error may return from ApplyChanges if the driver is unable
		// to execute a change.
		ApplyChanges(context.Context, []schema.Change) error
	}

	// StateReader wraps the method for reading a database/schema state.
	// The types below provides a few builtin options for reading a state
	// from a migration directory, a static object (e.g. a parsed file).
	//
	// In next Go version, the State will be replaced with the following
	// union type `interface { Realm | Schema }`.
	StateReader interface {
		ReadState(ctx context.Context) (*schema.Realm, error)
	}

	// The StateReaderFunc type is an adapter to allow the use of
	// ordinary functions as state readers.
	StateReaderFunc func(ctx context.Context) (*schema.Realm, error)
)

// ReadState calls f(ctx).
func (f StateReaderFunc) ReadState(ctx context.Context) (*schema.Realm, error) {
	return f(ctx)
}

// ErrNoPlan is returned by Plan when there is no change between the two states.
var ErrNoPlan = errors.New("sql/migrate: no plan for matched states")

// Realm returns a state reader for the static Realm object.
func Realm(r *schema.Realm) StateReader {
	return StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		return r, nil
	})
}

// Schema returns a state reader for the static Schema object.
func Schema(s *schema.Schema) StateReader {
	return StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		r := &schema.Realm{Schemas: []*schema.Schema{s}}
		if s.Realm != nil {
			r.Attrs = s.Realm.Attrs
		}
		s.Realm = r
		return r, nil
	})
}

type (
	// Dir represents a versioned migration directory.
	Dir struct {
		fs        fs.FS
		conn      Driver
		pattern   string
		templates []struct{ N, T *template.Template }
	}

	// DirOption allows configuring the Dir
	// using functional options.
	DirOption func(*Dir) error
)

// NewDir creates a new workspace directory based
// on the given configuration options.
func NewDir(opts ...DirOption) (*Dir, error) {
	d := &Dir{
		fs: dirFS{
			dir: "migrations",
			FS:  os.DirFS("migrations"),
		},
		pattern: "*.sql",
	}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	if len(d.templates) == 0 {
		d.templates = []struct{ N, T *template.Template }{defaultTemplate}
	}
	return d, nil
}

type (
	// dirFS wraps the os.DirFS with additional writing capabilities.
	dirFS struct {
		fs.FS
		dir string
	}
	// FileRemoveWriter wraps the WriteFile and RemoveFile methods
	// to allow editing the migration directory on development.
	FileRemoveWriter interface {
		RemoveFile(name string) error
		WriteFile(name string, data []byte, perm fs.FileMode) error
	}
)

// WriteFile implements the FileWriter interface.
func (d *dirFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(filepath.Join(d.dir, name), data, perm)
}

// RemoveFile implements the FileWriter interface.
func (d *dirFS) RemoveFile(name string) error {
	return os.Remove(filepath.Join(d.dir, name))
}

// DirFS configures the FS used by the migration directory.
func DirFS(fs fs.FS) DirOption {
	return func(d *Dir) error {
		d.fs = fs
		return nil
	}
}

// DirPath configures the FS used by the migration directory
// to point the given OS directory.
func DirPath(path string) DirOption {
	return func(d *Dir) error {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("sql/migrate: %q not a dir", path)
		}
		d.fs = &dirFS{FS: os.DirFS(path), dir: path}
		return nil
	}
}

// DirGlob configures the glob/pattern for reading
// migration files from the directory. For example:
//
//	migrate.NewDir(
//		migrate.DirPath("migrations"),
//		migrate.DirGlob("*.up.sql"),
//	)
//
func DirGlob(pattern string) DirOption {
	return func(d *Dir) error {
		d.pattern = pattern
		return nil
	}
}

// DirConn provides a Driver connection to a database. It is usually connected to
// an ephemeral database for emulating migration changes on it and calculating the
// "current state" to be compared with the "desired state".
func DirConn(conn Driver) DirOption {
	return func(d *Dir) error {
		d.conn = conn
		return nil
	}
}

var (
	// TemplateFuncs defines the global functions available for the templates.
	TemplateFuncs = template.FuncMap{
		"hasSuffix": strings.HasSuffix,
		"timestamp": func() int64 {
			return time.Now().Unix()
		},
	}
	defaultTemplate = struct {
		N, T *template.Template
	}{
		N: template.Must(template.New("name").
			Funcs(TemplateFuncs).
			Parse("{{timestamp}}_{{.Name}}.sql")),
		T: template.Must(template.New("name").
			Funcs(TemplateFuncs).
			Parse(`
{{- range $c := .Changes }}
	{{- $cmd := $c.Cmd }}
	{{- if not (hasSuffix $c.Cmd ";") }}
		{{- $cmd = print $cmd ";" }}
	{{- end }}
	{{- println $cmd }}
{{- end }}`)),
	}
)

// DirTemplates configures template files for writing
// the Plan object to the migration directory.
//
//	migrate.NewDir(
//		migrate.DirPath("migrations"),
//		migrate.DirTemplates("{{timestamp}}{{.Name}}.up.sql", "{{range c := .Changes}}{{println c.Cmd}}{{end}}"),
//	)
//
//	migrate.NewDir(
//		migrate.DirPath("migrations"),
//		migrate.DirTemplates(
//			"{{timestamp}}{{.Name}}.up.sql", "{{range $c := .Changes}}{{println $c.Cmd}}{{end}}",
//			"{{timestamp}}{{.Name}}.down.sql", "{{range $c := .Changes}}{{println $c.Reverse}}{{end}}",
//		),
//	)
//
func DirTemplates(nameFileTmpl ...string) DirOption {
	return func(d *Dir) error {
		if n := len(nameFileTmpl); n == 0 || n%2 == 1 {
			return fmt.Errorf("odd or zero argument count")
		}
		for i := 0; i < len(nameFileTmpl); i += 2 {
			if err := d.addTemplate(nameFileTmpl[i], nameFileTmpl[i+1]); err != nil {
				return err
			}
		}
		return nil
	}
}

// ReadState reads the current database/realm state that is stored in the migration directory.
// The given emulator driver is used for playing all migration files against to it.
func (d *Dir) ReadState(ctx context.Context) (*schema.Realm, error) {
	realm, _, err := d.readStateOf(ctx, -1)
	return realm, err
}

// Plan calculates the migration Plan required for moving from the directory state to
// the next state (to). A StateReader, can be another directory, static schema elements
// or a Driver connection.
func (d *Dir) Plan(ctx context.Context, name string, to StateReader) (*Plan, error) {
	current, err := d.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	changes, err := d.conn.RealmDiff(current, desired)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, ErrNoPlan
	}
	return d.conn.PlanChanges(ctx, name, changes)
}

// WritePlan writes the given plan to the directory
// based on the given Write configuration.
func (d *Dir) WritePlan(p *Plan) error {
	rw, ok := d.fs.(FileRemoveWriter)
	if !ok {
		return fmt.Errorf("fs.FS does not support editing: %T", d.fs)
	}
	for _, t := range d.templates {
		var b bytes.Buffer
		if err := t.N.Execute(&b, p); err != nil {
			return err
		}
		if b.String() == "" {
			return errors.New("file name cannot be empty")
		}
		name := b.String()
		b.Reset()
		if err := t.T.Execute(&b, p); err != nil {
			return err
		}
		if err := rw.WriteFile(name, b.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

// Compact compacts the first n migration files into one. If n < 0, all files are selected.
func (d *Dir) Compact(ctx context.Context, name string, n int) error {
	rw, ok := d.fs.(FileRemoveWriter)
	if !ok {
		return fmt.Errorf("fs.FS does not support editing: %T", d.fs)
	}
	desired, files, err := d.readStateOf(ctx, n)
	if err != nil {
		return err
	}
	changes, err := d.conn.RealmDiff(&schema.Realm{}, desired)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return nil
	}
	for _, c := range changes {
		// Add the "IF NOT EXISTS" clause to the schema
		// creation as it is usually created manually.
		if c, ok := c.(*schema.AddSchema); ok {
			c.Extra = append(c.Extra, &schema.IfNotExists{})
		}
	}
	plan, err := d.conn.PlanChanges(ctx, name, changes)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := rw.RemoveFile(f); err != nil {
			return fmt.Errorf("remove file: %q: %w", f, err)
		}
	}
	return d.WritePlan(plan)
}

// readStateOf of first n files. If n < 0, all files are selected.
func (d *Dir) readStateOf(ctx context.Context, n int) (*schema.Realm, []string, error) {
	files, err := fs.Glob(d.fs, d.pattern)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case n == 0 || n >= len(files):
		return nil, nil, fmt.Errorf("sql/migrate: invalid number for selected files: %d", n)
	case n > 0:
		files = files[:n]
	}
	// Files are expected to be sorted lexicographically.
	sort.Slice(files, func(i, j int) bool {
		return files[i] < files[j]
	})
	for _, f := range files {
		buf, err := fs.ReadFile(d.fs, f)
		if err != nil {
			return nil, nil, fmt.Errorf("sql/migrate: scan migration script %q: %w", f, err)
		}
		if _, err := d.conn.ExecContext(ctx, string(buf)); err != nil {
			return nil, nil, fmt.Errorf("sql/migrate: execute migration script %q: %w", f, err)
		}
	}
	realm, err := d.conn.InspectRealm(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	return realm, files, nil
}

func (d *Dir) addTemplate(nameTmpl, fileTmpl string) error {
	nameT, err := template.New("name").Funcs(TemplateFuncs).Parse(nameTmpl)
	if err != nil {
		return err
	}
	fileT, err := template.New("file").Funcs(TemplateFuncs).Parse(fileTmpl)
	if err != nil {
		return err
	}
	d.templates = append(d.templates, struct{ N, T *template.Template }{N: nameT, T: fileT})
	return nil
}
