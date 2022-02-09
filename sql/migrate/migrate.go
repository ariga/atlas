// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"ariga.io/atlas/sql/schema"
)

type (
	// A Plan defines a planned changeset that its execution brings the database to
	// the new desired state. Additional information is calculated by the different
	// drivers to indicate if the changeset is transactional (can be rolled-back) and
	// reversible (a down file can be generated to it).
	Plan struct {
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
		PlanChanges(context.Context, []schema.Change) (*Plan, error)

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
	// FS describes the methods needed to work with the Planner.
	FS interface {
		// List all present migration files. Usually *.sql files.
		List() ([]string, error)

		// Read the contents of a file by name.
		Read(name string) ([]byte, error)

		// Write the contents to the file by name.
		Write(name string, data []byte, perm fs.FileMode) error

		// Remove a file by name.
		Remove(name string) error
	}

	// Printer wraps the methods for naming and dumping a migration plan to one or more files.
	Printer interface {
		// Print prints the given Plan.
		// The first return argument contains a slice of filenames.
		// The second return argument is meant to hold the contents for each filename.
		// The length of the filenames-slice and contents-slice must be equal.
		Print(*Plan) ([]string, [][]byte, error)
	}

	// Planner can plan the steps to take to migrate from one state to another. It uses the enclosed FS to write
	// those changes to versioned migration files.
	Planner struct {
		drv Driver
		fs  FS
		pr  Printer
		// // templates holds the filename and content templates to use when dumping a migration.
		// templates []struct{ N, C *template.Template }
	}
)

// New creates a new Planner.
func New(drv Driver, fs FS) *Planner {
	return &Planner{
		drv: drv,
		fs:  fs,
	}
}

// localFS implements FS for a local path.
type localFS struct {
	dir  string
	glob string
}

func (fs *localFS) List() ([]string, error) {
	return filepath.Glob(filepath.Join(fs.dir, fs.glob))
}

func (fs *localFS) Read(name string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(fs.dir, name))
}

func (fs *localFS) Write(name string, data []byte, perm fs.FileMode) error {
	return ioutil.WriteFile(filepath.Join(fs.dir, name), data, perm)
}

func (fs *localFS) Remove(name string) error {
	return os.Remove(filepath.Join(fs.dir, name))
}

// NewLocalFS configures the FS used by a Planner to work on the given local path.
func NewLocalFS(path, glob string) (*localFS, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("sql/migrate: %q is not a dir", path)
	}
	return &localFS{dir: path, glob: glob}, nil
}

// localFS implements Printer for a golang-migrate/migrate compatible migration files.
type goMigratePrinter struct{}

func (goMigratePrinter) Print(plan *Plan) ([]string, [][]byte, error) {
	return nil, nil, nil
}

// Plan calculates the migration Plan required for moving the current state (from) state to
// the next state (to). A StateReader can be a directory, static schema elements or a Driver connection.
func (p *Planner) Plan(ctx context.Context, from StateReader, to StateReader) (*Plan, error) {
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
}
