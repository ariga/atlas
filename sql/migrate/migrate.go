// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
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
)

type (
	// FS describes the methods needed to work with the Dir.
	FS interface {
		List() ([]fs.FileInfo, error)
		Read(name string) ([]byte, error)
		Write(name string, data []byte, perm fs.FileMode) error
		Remove(name string) error
	}

	// Dir represents a versioned  migration directory. Aka the place where all versioned migration files are stored.
	Dir struct {
		fs FS
	}

	// DirOption allows for configuring the Dir using functional arguments.
	DirOption func(*Dir) error
)

// NewDir creates a new Dir.
func NewDir(opts ...DirOption) (*Dir, error) {
	d := new(Dir)
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// WithFS sets the fs.FS used by the Dir.
func WithFS(fs FS) DirOption {
	return func(dir *Dir) error {
		dir.fs = fs
		return nil
	}
}

// Local implements FS for a local path.
type localFS struct {
	dir string
}

func (fs localFS) List() ([]fs.FileInfo, error) {
	return ioutil.ReadDir(fs.dir)
}

func (fs localFS) Read(name string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(fs.dir, name))
}

func (fs localFS) Write(name string, data []byte, perm fs.FileMode) error {
	return ioutil.WriteFile(filepath.Join(fs.dir, name), data, perm)
}

func (fs localFS) Remove(name string) error {
	return os.Remove(filepath.Join(fs.dir, name))
}

// WithLocal configures the FS used by the Dir to work on the given local path.
func WithLocal(path string) DirOption {
	return func(d *Dir) error {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("sql/migrate: %q is not a dir", path)
		}
		d.fs = localFS{dir: path}
		return nil
	}
}
