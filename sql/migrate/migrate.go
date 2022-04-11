// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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

// Realm returns a StateReader for the static Realm object.
func Realm(r *schema.Realm) StateReader {
	return StateReaderFunc(func(context.Context) (*schema.Realm, error) {
		return r, nil
	})
}

// Schema returns a StateReader for the static Schema object.
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

// Conn returns a StateReader for a Driver.
func Conn(drv Driver, opts *schema.InspectRealmOption) StateReader {
	return StateReaderFunc(func(ctx context.Context) (*schema.Realm, error) {
		return drv.InspectRealm(ctx, opts)
	})
}

type (
	// Dir describes the methods needed for a Planner to manage migration files.
	Dir interface {
		fs.FS

		// WriteFile writes the data to the named file.
		WriteFile(string, []byte) error
	}

	// Formatter wraps the Format method.
	Formatter interface {
		// Format formats the given Plan into one or more migration files.
		Format(*Plan) ([]File, error)
	}

	// File represents a single migration file.
	File interface {
		io.Reader
		// Name returns the name of the FormattedFile.
		Name() string
	}

	// Planner can plan the steps to take to migrate from one state to another. It uses the enclosed FS to write
	// those changes to versioned migration files.
	Planner struct {
		drv Driver      // driver to use
		dir Dir         // where migration files are stored and read from
		fmt Formatter   // how to format a plan to migration files
		dsr StateReader // how to read a state from the migration directory
		sum bool        // whether to create a sum file for the migration directory
	}

	// PlannerOption allows managing a Planner using functional arguments.
	PlannerOption func(*Planner)
)

// NewPlanner creates a new Planner.
func NewPlanner(drv Driver, dir Dir, opts ...PlannerOption) *Planner {
	p := &Planner{drv: drv, dir: dir, sum: true}
	for _, opt := range opts {
		opt(p)
	}
	if p.fmt == nil {
		p.fmt = DefaultFormatter
	}
	if p.dsr == nil {
		p.dsr = GlobStateReader(p.dir, p.drv, "*.sql")
	}
	return p
}

// WithFormatter sets the Formatter of a Planner.
func WithFormatter(fmt Formatter) PlannerOption {
	return func(p *Planner) {
		p.fmt = fmt
	}
}

// WithStateReader sets the StateReader of a Planner.
func WithStateReader(dsr StateReader) PlannerOption {
	return func(p *Planner) {
		p.dsr = dsr
	}
}

// DisableChecksum disables the hash-sum functionality for the migration directory.
func DisableChecksum() PlannerOption {
	return func(p *Planner) {
		p.sum = false
	}
}

// GlobStateReader creates a StateReader that loads all files from Dir matching
// glob in lexicographic order and uses the Driver to create a migration state.
//
// If the given Driver implements the Emptier interface the IsEmpty method will be used to determine if the Driver is
// connected to an "empty" database. This behavior was added to support SQLite flavors.
func GlobStateReader(dir Dir, drv Driver, glob string) StateReaderFunc {
	var errNotClean = errors.New("sql/migrate: connected database is not clean")
	return func(ctx context.Context) (realm *schema.Realm, err error) {
		// Collect the migration files.
		names, err := fs.Glob(dir, glob)
		if err != nil {
			return nil, err
		}
		// We need an empty database state to reliably replay the migration directory.
		if c, ok := drv.(interface {
			// The IsClean method can be added to a Driver to override how to
			// determine if a connected database is in a clean state.
			// This interface exists only to support SQLite favors.
			IsClean(context.Context) (bool, error)
		}); ok {
			e, err := c.IsClean(ctx)
			if err != nil {
				return nil, fmt.Errorf("sql/migrate: checking database state: %w", err)
			}
			if !e {
				return nil, errNotClean
			}
		} else {
			realm, err = drv.InspectRealm(ctx, nil)
			if err != nil {
				return nil, err
			}
			if len(realm.Schemas) > 0 {
				return nil, errNotClean
			}
		}
		// Sort files lexicographically.
		sort.Slice(names, func(i, j int) bool {
			return names[i] < names[j]
		})
		// If the driver supports it, acquire a lock while replaying the migration changes.
		if l, ok := drv.(schema.Locker); ok {
			unlock, err := l.Lock(ctx, "atlas_migration_directory_state", 0)
			if err != nil {
				return nil, fmt.Errorf("sql/migrate: acquiring database lock: %w", err)
			}
			defer unlock()
		}
		// Clean up after ourselves.
		defer func() {
			if e, ok := drv.(interface {
				// The Clean method can be added to a Driver to change how to clean a database.
				// This interface exists only to support SQLite favors.
				Clean(context.Context) error
			}); ok {
				if derr := e.Clean(ctx); derr != nil {
					err = wrap(derr, err)
				}
				return
			}
			realm, derr := drv.InspectRealm(ctx, nil)
			if derr != nil {
				err = wrap(derr, err)
			}
			del := make([]schema.Change, len(realm.Schemas))
			for i, s := range realm.Schemas {
				del[i] = &schema.DropSchema{S: s, Extra: []schema.Clause{&schema.IfExists{}}}
			}
			if derr := drv.ApplyChanges(ctx, del); derr != nil {
				err = wrap(derr, err)
			}
		}()
		for _, n := range names {
			f, err := dir.Open(n)
			if err != nil {
				return nil, err
			}
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				t := sc.Text()
				if !strings.HasPrefix(strings.TrimSpace(t), "--") {
					if _, err := drv.ExecContext(ctx, t); err != nil {
						f.Close()
						return nil, err
					}
				}
			}
			f.Close()
			if err := sc.Err(); err != nil {
				return nil, err
			}
		}
		realm, err = drv.InspectRealm(ctx, nil)
		return
	}
}

// Plan calculates the migration Plan required for moving the current state (from) state to
// the next state (to). A StateReader can be a directory, static schema elements or a Driver connection.
func (p *Planner) Plan(ctx context.Context, name string, to StateReader) (*Plan, error) {
	current, err := p.dsr.ReadState(ctx)
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

// WritePlan writes the given Plan to the Dir based on the configured Formatter.
func (p *Planner) WritePlan(plan *Plan) error {
	files, err := p.fmt.Format(plan)
	if err != nil {
		return err
	}
	for _, f := range files {
		d, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		if err := p.dir.WriteFile(f.Name(), d); err != nil {
			return err
		}
	}
	if p.sum {
		sum, err := HashSum(p.dir)
		if err != nil {
			return err
		}
		return WriteSumFile(p.dir, sum)
	}
	return nil
}

// LocalDir implements Dir for a local path.
type LocalDir struct {
	dir string
}

// NewLocalDir returns a new the Dir used by a Planner to work on the given local path.
func NewLocalDir(path string) (*LocalDir, error) {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
		return &LocalDir{dir: path}, nil
	} else if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("sql/migrate: %q is not a dir", path)
	}
	return &LocalDir{dir: path}, nil
}

// Open implements fs.FS.
func (d *LocalDir) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(d.dir, name))
}

// WriteFile implements Dir.WriteFile.
func (d *LocalDir) WriteFile(name string, b []byte) error {
	return os.WriteFile(filepath.Join(d.dir, name), b, 0600)
}

var _ Dir = (*LocalDir)(nil)

var (
	// templateFuncs contains the template.FuncMap for the DefaultFormatter.
	templateFuncs = template.FuncMap{
		"now": func() string { return time.Now().Format("20060102150405") },
		"rev": reverse,
	}
	// DefaultFormatter is a default implementation for Formatter.
	DefaultFormatter = &TemplateFormatter{
		templates: []struct{ N, C *template.Template }{
			{
				N: template.Must(template.New("").Funcs(templateFuncs).Parse(
					"{{ now }}{{ with .Name }}_{{ . }}{{ end }}.sql",
				)),
				C: template.Must(template.New("").Funcs(templateFuncs).Parse(
					`{{ range .Changes }}{{ with .Comment }}-- {{ println . }}{{ end }}{{ printf "%s;\n" .Cmd }}{{ end }}`,
				)),
			},
		},
	}
)

// TemplateFormatter implements Formatter by using templates.
type TemplateFormatter struct {
	templates []struct{ N, C *template.Template }
}

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
	files := make([]File, 0, len(t.templates))
	for _, tpl := range t.templates {
		var n, c bytes.Buffer
		if err := tpl.N.Execute(&n, plan); err != nil {
			return nil, err
		}
		if err := tpl.C.Execute(&c, plan); err != nil {
			return nil, err
		}
		files = append(files, &templateFile{
			Buffer: &c,
			n:      n.String(),
		})
	}
	return files, nil
}

type templateFile struct {
	*bytes.Buffer
	n string
}

// Name implements the File interface.
func (f *templateFile) Name() string { return f.n }

// Filename of the migration directory integrity sum file.
const hashFile = "atlas.sum"

// HashFile represents the integrity sum file of the migration dir.
type HashFile []struct{ N, H string }

// HashSum reads the whole dir, sorts the files by name and creates a HashSum from its contents.
func HashSum(dir Dir) (HashFile, error) {
	var (
		hs HashFile
		h  = sha256.New()
	)
	err := fs.WalkDir(dir, "", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// If this is the integrity sum file do not include it into the sum.
		if filepath.Base(path) == hashFile {
			return nil
		}
		if !d.IsDir() {
			f, err := dir.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := h.Write([]byte(path)); err != nil {
				return err
			}
			if _, err = io.Copy(h, f); err != nil {
				return err
			}
			hs = append(hs, struct{ N, H string }{path, base64.StdEncoding.EncodeToString(h.Sum(nil))})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hs, nil
}

var (
	// ErrChecksumFormat is returned from Validate if the sum files format is invalid.
	ErrChecksumFormat = errors.New("checksum file format invalid")
	// ErrChecksumMismatch is returned from Validate if the hash sums don't match.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrChecksumNotFound is returned from Validate if the hash file does not exist.
	ErrChecksumNotFound = errors.New("checksum file not found")
)

// Validate checks if the migration dir is in sync with its sum file.
// If they don't match ErrChecksumMismatch is returned.
func Validate(dir Dir) error {
	f, err := dir.Open(hashFile)
	if os.IsNotExist(err) {
		// If there are no migration files yet this is okay.
		files, err := fs.ReadDir(dir, "/")
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		return ErrChecksumNotFound
	}
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	var fh HashFile
	if err := fh.UnmarshalText(b); err != nil {
		return err
	}
	mh, err := HashSum(dir)
	if err != nil {
		return err
	}
	if fh.Sum() != mh.Sum() {
		return ErrChecksumMismatch
	}
	return nil
}

// WriteSumFile writes the given HashFile to the Dir. If the file does not exist, it is created.
func WriteSumFile(dir Dir, sum HashFile) error {
	b, err := sum.MarshalText()
	if err != nil {
		return err
	}
	return dir.WriteFile(hashFile, b)
}

// Sum returns the checksum of the represented hash file.
func (s HashFile) Sum() string {
	sha := sha256.New()
	for _, f := range s {
		sha.Write([]byte(f.N))
		sha.Write([]byte(f.H))
	}
	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

// MarshalText implements encoding.TextMarshaler.
func (s HashFile) MarshalText() ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, f := range s {
		fmt.Fprintf(buf, "%s h1:%s\n", f.N, f.H)
	}
	return []byte(fmt.Sprintf("h1:%s\n%s", s.Sum(), buf)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *HashFile) UnmarshalText(b []byte) error {
	sc := bufio.NewScanner(bytes.NewReader(b))
	// The first line contains the sum.
	sc.Scan()
	sum := strings.TrimPrefix(sc.Text(), "h1:")
	for sc.Scan() {
		li := strings.SplitN(sc.Text(), "h1:", 2)
		if len(li) != 2 {
			return ErrChecksumFormat
		}
		*s = append(*s, struct{ N, H string }{strings.TrimSpace(li[0]), li[1]})
	}
	if sum != s.Sum() {
		return ErrChecksumMismatch
	}
	return sc.Err()
}

// reverse changes for the down migration.
func reverse(changes []*Change) []*Change {
	n := len(changes)
	rev := make([]*Change, n)
	if n%2 == 1 {
		rev[n/2] = changes[n/2]
	}
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = changes[j], changes[i]
	}
	return rev
}

func wrap(err1, err2 error) error {
	if err2 != nil {
		return fmt.Errorf("sql/migrate: %w: %v", err2, err1)
	}
	return err1
}
