// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci_test

import (
	"context"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"arigo.io/atlasci/internal/ci"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/require"
)

func TestGitChangeDetector(t *testing.T) {
	cs, err := ci.NewGitChangeDetector("", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: root cannot be \"\"")

	cs, err = ci.NewGitChangeDetector("testdata", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: dir cannot be <nil>")

	tmp, err := util.TempDir(osfs.Default, "", "testdata")
	require.NoError(t, err)
	fs := osfs.New(tmp)
	require.NoError(t, fs.MkdirAll("migrations", 0755))
	r, err := git.Init(filesystem.NewStorage(fs, cache.NewObjectLRUDefault()), fs)
	require.NoError(t, err)
	w, err := r.Worktree()
	require.NoError(t, err)

	_, err = fs.Create("migrations/1_applied.sql")
	require.NoError(t, err)
	_, err = w.Add("migrations/1_applied.sql")
	require.NoError(t, err)
	commit, err := w.Commit("first migration file", &git.CommitOptions{})
	require.NoError(t, err)
	_, err = r.CommitObject(commit)
	require.NoError(t, err)

	// OK
	d, err := migrate.NewLocalDir(filepath.Join(fs.Root(), "migrations"))
	require.NoError(t, err)
	cs, err = ci.NewGitChangeDetector(fs.Root(), d)
	require.NoError(t, err)
	require.NotNil(t, cs)

	// No diff.
	base, feat, err := cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 1)
	require.Empty(t, feat)
	require.Equal(t, "1_applied.sql", base[0].Name())

	// Feature branch.
	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.NewBranchReferenceName("feature"),
		Keep:   true,
	})
	require.NoError(t, err)
	_, err = fs.Create("migrations/2_new.sql")
	require.NoError(t, err)
	_, err = fs.Create("migrations/3_new_the_second.sql")
	require.NoError(t, err)
	_, err = w.Add("migrations/2_new.sql")
	require.NoError(t, err)
	_, err = w.Add("migrations/3_new_the_second.sql")
	require.NoError(t, err)
	commit, err = w.Commit("second and third migration files", &git.CommitOptions{})
	require.NoError(t, err)
	_, err = r.CommitObject(commit)
	require.NoError(t, err)

	base, feat, err = cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 1)
	require.Len(t, feat, 2)
	require.Equal(t, "1_applied.sql", base[0].Name())
	require.Equal(t, "2_new.sql", feat[0].Name())
	require.Equal(t, "3_new_the_second.sql", feat[1].Name())
}
