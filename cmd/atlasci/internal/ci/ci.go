// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
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
		root string     // path to the git repository root
		base string     // name of the base branch (e.g. master)
		path string     // path of the migration directory relative to the repository root (in slash notation)
		dir  DirScanner // the migration directory to load migration files from
	}

	// GitChangeDetectorOption allows configuring GitChangeDetector with functional arguments.
	GitChangeDetectorOption func(*GitChangeDetector) error
)

// NewGitChangeDetector configures a new GitChangeDetector.
func NewGitChangeDetector(root string, dir DirScanner, opts ...GitChangeDetectorOption) (*GitChangeDetector, error) {
	if root == "" {
		return nil, errors.New("internal/ci: root cannot be empty")
	}
	if dir == nil {
		return nil, errors.New("internal/ci: dir cannot be nil")
	}
	d := &GitChangeDetector{root: root, dir: dir}
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
func (d *GitChangeDetector) DetectChanges(_ context.Context) ([]migrate.File, []migrate.File, error) {
	// Fetch all the files of the migration directory.
	files, err := d.dir.Files()
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: reading migration directory: %w", err)
	}
	// Diff the base tree and head tree.
	r, err := git.PlainOpen(d.root)
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: open git repo: %w", err)
	}
	baseT, err := treeObject(r, plumbing.NewBranchReferenceName(d.base))
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: %w", err)
	}
	headT, err := treeObject(r, plumbing.HEAD)
	if err != nil {
		return nil, nil, fmt.Errorf("internal/ci: %w", err)
	}
	diff, err := baseT.Diff(headT)
	if err != nil {
		return nil, nil, err
	}
	if len(diff) == 0 {
		return files, nil, nil
	}
	// Since isNewFile assumes a sorted slice of changes, make sure to sort it.
	sort.Sort(diff)
	// Iterate over the migration files. If we find a file, that has been added in the diff between base and head,
	// every migration file preceding it can be considered old, the file itself and everything thereafter new,
	// since Atlas assumes a linear migration history.
	for i, f := range files {
		ok, err := d.isNewFile(diff, f)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		return files[:i], files[i:], nil
	}
	return files, nil, nil
}

var _ ChangeDetector = (*GitChangeDetector)(nil)

// isNewFile determines if the given file has been added in the given tree diff.
// It assumes the diff is sorted by filename.
func (d *GitChangeDetector) isNewFile(diff object.Changes, file migrate.File) (bool, error) {
	// Git uses "/" as path separator regardless of platform.
	n := filepath.ToSlash(filepath.Join(d.path, file.Name()))
	i := sort.Search(len(diff), func(i int) bool {
		return diff[i].To.Name >= n
	})
	if i <= len(diff) && strings.HasSuffix(diff[i].To.Name, n) {
		a, err := diff[i].Action()
		if err != nil {
			return false, err
		}
		return a == merkletrie.Insert, nil
	}
	// If the file has not been found in the diff, it must be an old one.
	return false, nil
}

// treeObject returns the object.Tree for a git reference name (e.g. refs/head/master).
func treeObject(r *git.Repository, n plumbing.ReferenceName) (*object.Tree, error) {
	ref, err := r.Reference(n, true)
	if err != nil {
		return nil, fmt.Errorf("reference feature branch: %w", err)
	}
	c, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("de-reference commit %q: %w", ref.Hash(), err)
	}
	t, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("get tree for commit %q: %w", c.Hash, err)
	}
	return t, nil
}

// DevLoader implements the ChangesLoader interface using a dev-driver.
type DevLoader struct {
	// Dev environment used as a sandbox instantiated to the starting point (e.g. base branch).
	Dev *sqlclient.Client
	// Scan is used for scanning the migration directory.
	Scan migrate.Scanner
}

// LoadChanges implements the ChangesLoader interface.
func (d *DevLoader) LoadChanges(ctx context.Context, files []migrate.File) ([]*sqlcheck.File, error) {
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
			return nil, err
		}
		for _, s := range stmts {
			if _, err := d.Dev.ExecContext(ctx, s); err != nil {
				return nil, err
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
			// In case the change is recognized by Atlas.
			if len(changes) > 0 {
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
		}
	}
	return diff, nil
}

// pos returns the position of a statement in migration file.
func pos(f migrate.File, stmt string) (int, error) {
	buf, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}
	i := bytes.Index(buf, []byte(stmt))
	if i == -1 {
		return 0, fmt.Errorf("statement %q was not found in %q", stmt, buf)
	}
	return i, nil
}
