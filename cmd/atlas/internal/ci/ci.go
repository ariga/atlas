// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// A ChangeDetector takes a migration directory and splits it into the "base" files (already merged) and new ones.
	ChangeDetector interface {
		// DetectChanges splits the files of a migration directory into the "base" files (already merged) and new ones.
		DetectChanges(context.Context) ([]migrate.File, []migrate.File, error)
	}

	// A ChangeLoader takes a set of migration files and will create multiple schema.Changes out of it.
	// It will also label migration files as either "generated" or "handcrafted".
	ChangeLoader interface {
		// LoadChanges converts each of the given migration files into one Changes.
		LoadChanges(context.Context, []migrate.File) ([]*sqlcheck.File, error)
	}

	// DirScanner stitches migrate.Dir and migrate.Scanner into one interface.
	DirScanner interface {
		migrate.Dir
		migrate.Scanner
	}
)

type (
	// GitChangeDetector implements the ChangeDetector interface by utilizing a git repository.
	GitChangeDetector struct {
		work string     // path to the git working directory (i.e. -C)
		base string     // name of the base branch (e.g. master)
		path string     // path of the migration directory relative to the repository root (in slash notation)
		dir  DirScanner // the migration directory to load migration files from
	}

	// GitChangeDetectorOption allows configuring GitChangeDetector with functional arguments.
	GitChangeDetectorOption func(*GitChangeDetector) error
)

// NewGitChangeDetector configures a new GitChangeDetector.
func NewGitChangeDetector(dir DirScanner, opts ...GitChangeDetectorOption) (*GitChangeDetector, error) {
	if dir == nil {
		return nil, errors.New("internal/ci: dir cannot be nil")
	}
	d := &GitChangeDetector{dir: dir}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	if d.base == "" {
		d.base = "master"
	}
	if d.path == "" {
		d.path = "migrations"
	}
	return d, nil
}

// WithWorkDir configures the git working directory for a GitChangeDetector.
func WithWorkDir(work string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.work = work
		return nil
	}
}

// WithBase configures the git base branch name for a GitChangeDetector.
func WithBase(base string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.base = base
		return nil
	}
}

// WithMigrationsPath configures the git base branch name for a GitChangeDetector.
func WithMigrationsPath(path string) GitChangeDetectorOption {
	return func(d *GitChangeDetector) error {
		d.path = filepath.ToSlash(path)
		return nil
	}
}

// DetectChanges implements the ChangeDetector interface.
func (d *GitChangeDetector) DetectChanges(ctx context.Context) ([]migrate.File, []migrate.File, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, nil, fmt.Errorf("lookup git: %w", err)
	}
	var args []string
	if d.work != "" {
		args = append(args, "-C", d.work)
	}
	args = append(args, "--no-pager", "diff", "--name-only", "--diff-filter=A", d.base, "HEAD", d.path)
	buf, err := exec.CommandContext(ctx, "git", args...).
		CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff: %w", err)
	}
	diff := strings.Split(string(buf), "\n")
	names := make(map[string]struct{}, len(diff))
	for i := range diff {
		names[filepath.Base(diff[i])] = struct{}{}
	}
	files, err := d.dir.Files()
	if err != nil {
		return nil, nil, fmt.Errorf("reading migration directory: %w", err)
	}
	// Iterate over the migration files. If we find a file, that has been added in the diff between base and head,
	// every migration file preceding it can be considered old, the file itself and everything thereafter new,
	// since Atlas assumes a linear migration history.
	for i, f := range files {
		if _, ok := names[f.Name()]; ok {
			return files[:i], files[i:], nil
		}
	}
	return files, nil, nil
}

var _ ChangeDetector = (*GitChangeDetector)(nil)

// latestChange implements the ChangeDetector by selecting the latest N files.
type latestChange struct {
	n   int        // number of (latest) files considered new.
	dir DirScanner // migration directory to load migration files from.
}

// LatestChanges implements the ChangeDetector interface by selecting the latest N files as new.
// It is useful for executing analysis on files in development before they are committed or on
// all files in a directory.
func LatestChanges(dir DirScanner, n int) ChangeDetector {
	return &latestChange{n: n, dir: dir}
}

// DetectChanges implements the ChangeDetector interface.
func (d *latestChange) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	files, err := d.dir.Files()
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: reading migration directory: %w", err)
	}
	// In case n is -1 or greater than the
	// number of files, return all files.
	if len(files) <= d.n || d.n < 0 {
		return nil, files, nil
	}
	return files[:len(files)-d.n], files[len(files)-d.n:], nil
}

// DevLoader implements the ChangesLoader interface using a dev-driver.
type DevLoader struct {
	// Dev environment used as a sandbox instantiated to the starting point (e.g. base branch).
	Dev *sqlclient.Client
	// Scan is used for scanning the migration directory.
	Scan migrate.Scanner
}

// LoadChanges implements the ChangesLoader interface.
func (d *DevLoader) LoadChanges(ctx context.Context, base, files []migrate.File) (_ []*sqlcheck.File, err error) {
	// Lock database so no one else interferes with our change detection.
	l, ok := d.Dev.Driver.(schema.Locker)
	if !ok {
		return nil, errors.New("driver does not support locking")
	}
	unlock, err := l.Lock(ctx, "atlas_ci_change_detection", 0)
	if err != nil {
		return nil, fmt.Errorf("acquiring database lock: %w", err)
	}
	defer unlock()
	// We need an empty database state to reliably replay the migration directory.
	if err := migrate.IsClean(ctx, d.Dev.Driver); err != nil {
		return nil, err
	}
	// Clean up after ourselves.
	defer func() {
		if err2 := migrate.Clean(ctx, d.Dev.Driver); err2 != nil {
			if err != nil {
				err = fmt.Errorf("%w: %v", err, err2)
				return
			}
			err = err2
		}
	}()
	// Bring the dev environment to the base point.
	for _, f := range base {
		stmt, err := d.Scan.Stmts(f)
		if err != nil {
			return nil, &FileError{File: f.Name(), Err: fmt.Errorf("scanning statements: %w", err)}
		}
		for _, s := range stmt {
			if _, err := d.Dev.ExecContext(ctx, s); err != nil {
				return nil, &FileError{File: f.Name(), Err: fmt.Errorf("executing statement: %q: %w", s, err)}
			}
		}
	}
	diff := make([]*sqlcheck.File, len(files))
	current, err := d.Dev.InspectRealm(ctx, nil)
	if err != nil {
		return nil, err
	}
	for i, f := range files {
		diff[i] = &sqlcheck.File{
			File: f,
		}
		stmts, err := d.Scan.Stmts(f)
		if err != nil {
			return nil, &FileError{File: f.Name(), Err: fmt.Errorf("scanning statements: %w", err)}
		}
		start := current
		for _, s := range stmts {
			if _, err := d.Dev.ExecContext(ctx, s); err != nil {
				return nil, &FileError{File: f.Name(), Err: fmt.Errorf("executing statement: %w", err)}
			}
			target, err := d.Dev.InspectRealm(ctx, nil)
			if err != nil {
				return nil, err
			}
			changes, err := d.Dev.RealmDiff(current, target)
			if err != nil {
				return nil, err
			}
			current = target
			p, err := pos(f, s)
			if err != nil {
				return nil, err
			}
			diff[i].Changes = append(diff[i].Changes, &sqlcheck.Change{
				Pos:     p,
				Stmt:    s,
				Changes: changes,
			})
		}
		if diff[i].Sum, err = d.Dev.RealmDiff(start, current); err != nil {
			return nil, err
		}
	}
	return diff, nil
}

// pos returns the position of a statement in migration file.
func pos(f migrate.File, stmt string) (int, error) {
	i := bytes.Index(f.Bytes(), []byte(stmt))
	if i == -1 {
		return 0, &FileError{File: f.Name(), Err: fmt.Errorf("statement %q was not found in %q", stmt, f.Bytes())}
	}
	return i, nil
}

// FileError represents an error that occurred while processing a file.
type FileError struct {
	File string
	Err  error
}

func (e FileError) Error() string { return e.Err.Error() }
