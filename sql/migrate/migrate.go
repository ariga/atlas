// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	// Dir describes the methods needed to work with the Planner.
	Dir interface {
		// Open opens the named file.
		Open(name string) (io.ReadWriteCloser, error)
	}

	// Formatter wraps the Format method for naming and dumping a migration plan to one or more Files.
	Formatter interface {
		// Format formats the given Plan.
		// The returned map contains the contents for each file to dump.
		Format(*Plan) ([]File, error)
	}

	// File represents a migration file.
	File interface {
		io.Reader
		Name() string
	}

	// Planner can plan the steps to take to migrate from one state to another. It uses the enclosed FS to write
	// those changes to versioned migration files.
	Planner struct {
		drv Driver
		dir Dir
		fmt Formatter
	}
)

// New creates a new Planner.
func New(drv Driver, dir Dir, fmt Formatter) *Planner {
	return &Planner{
		drv: drv,
		dir: dir,
		fmt: fmt,
	}
}

// Plan calculates the migration Plan required for moving the current state (from) state to
// the next state (to). A StateReader can be a directory, static schema elements or a Driver connection.
func (p *Planner) Plan(ctx context.Context, name string, from StateReader, to StateReader) (*Plan, error) {
	current, err := from.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	desired, err := to.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	changes, err := p.drv.RealmDiff(current, desired)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, ErrNoPlan
	}
	return p.drv.PlanChanges(ctx, name, changes)
}

// WritePlan writes the given plan to the Dir based on the given Formatter.
func (p *Planner) WritePlan(plan *Plan) error {
	fs, err := p.fmt.Format(plan)
	if err != nil {
		return err
	}
	for _, f := range fs {
		fh, err := p.dir.Open(f.Name())
		if err != nil {
			return err
		}
		_, err = io.Copy(fh, f)
		fh.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// LocalDir implements Dir for a local path.
type LocalDir struct {
	dir string
}

// Open implements Dir.Open.
func (d *LocalDir) Open(name string) (File, error) {
	return os.Create(filepath.Join(d.dir, name))
}

// NewLocalDir returns a new the Dir used by a Planner to work on the given local path.
func NewLocalDir(path string) (*LocalDir, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("sql/migrate: %q is not a dir", path)
	}
	return &LocalDir{dir: path}, nil
}

// TemplateFormatter implements Formatter by using templates.
type TemplateFormatter struct {
	templates []struct{ N, C *template.Template }
}

var (
	// templateFuncs contains the template.FuncMap for the DefaultFormatter.
	templateFuncs = template.FuncMap{
		"now": time.Now,
		"sem": ensureSemicolonSuffix,
		"rev": reverse,
	}
	// DefaultFormatter is a default implementation for Formatter. Compatible with golang-migrate/migrate.
	DefaultFormatter = &TemplateFormatter{
		templates: []struct{ N, C *template.Template }{
			{
				N: template.Must(template.New("").Funcs(templateFuncs).Parse(
					"{{ now.Unix }}{{ with .Name }}_{{ . }}{{ end }}.up.sql",
				)),
				C: template.Must(template.New("").Funcs(templateFuncs).Parse(
					"{{ range .Changes }}{{ println (sem .Cmd) }}{{ end }}",
				)),
			},
			{
				N: template.Must(template.New("").Funcs(templateFuncs).Parse(
					"{{ now.Unix }}{{ with .Name }}_{{ . }}{{ end }}.down.sql",
				)),
				C: template.Must(template.New("").Funcs(templateFuncs).Parse(
					"{{ range rev .Changes }}{{ println (sem .Reverse) }}{{ end }}",
				)),
			},
		},
	}
)

// NewTemplateFormatter creates a new Formatter working with the given templates.
//
//	migrate.NewTemplateFormatter(
//		template.Must(template.New("").Parse("{{now.Unix}}{{.Name}}.sql")),                 // name template
//		template.Must(template.New("").Parse("{{range .Changes}}{{println .Cmd}}{{end}}")), // content template
//	)
//
func NewTemplateFormatter(templates ...*template.Template) (*TemplateFormatter, error) {
	if n := len(templates); n == 0 || n%2 == 1 {
		return nil, fmt.Errorf("zero or odd number of templates given")
	}
	t := new(TemplateFormatter)
	for i := 0; i < len(templates); i += 2 {
		t.templates = append(t.templates, struct{ N, C *template.Template }{templates[i], templates[i+1]})
	}
	return t, nil
}

// Format implements the Formatter interface.
func (t *TemplateFormatter) Format(plan *Plan) ([]File, error) {
	fs := make([]File, 0, len(t.templates))
	for _, tpl := range t.templates {
		var n, c bytes.Buffer
		if err := tpl.N.Execute(&n, plan); err != nil {
			return nil, err
		}
		if err := tpl.C.Execute(&c, plan); err != nil {
			return nil, err
		}
		fs = append(fs, &templateFile{
			Buffer: &c,
			n:      n.String(),
		})
	}
	return fs, nil
}

type templateFile struct {
	*bytes.Buffer
	n string
}

// Name implements the File interface.
func (f *templateFile) Name() string { return f.n }

// reverse changes for the down migration.
func reverse(changes []*Change) []*Change {
	rev := make([]*Change, len(changes))
	for i, j := 0, len(changes)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = changes[j], changes[i]
	}
	return rev
}

// ensure an SQL statement has a trailing ";".
func ensureSemicolonSuffix(s string) string {
	if !strings.HasSuffix(s, ";") {
		return s + ";"
	}
	return s
}
