// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/cmd/atlasci/internal/ci"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestGitChangeDetector(t *testing.T) {
	cs, err := ci.NewGitChangeDetector("", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: root cannot be empty")

	cs, err = ci.NewGitChangeDetector("testdata", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: dir cannot be nil")

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
	commit, err := w.Commit("first migration file", &git.CommitOptions{
		Author: &object.Signature{Name: "a8m"},
	})
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
	commit, err = w.Commit("second and third migration files", &git.CommitOptions{
		Author: &object.Signature{Name: "a8m"},
	})
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

func TestLatestChanges(t *testing.T) {
	files := []migrate.File{
		testFile{name: "1.sql", content: "CREATE TABLE t1 (id INT)"},
		testFile{name: "2.sql", content: "CREATE TABLE t2 (id INT)\nDROP TABLE users"},
	}
	base, feat, err := ci.LatestChanges(testDir{files: files}, 0).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Equal(t, files, base)
	require.Empty(t, feat)

	base, feat, err = ci.LatestChanges(testDir{files: files}, 2).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = ci.LatestChanges(testDir{files: files}, -1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = ci.LatestChanges(testDir{files: files}, 1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Equal(t, files[:1], base)
	require.Equal(t, files[1:], feat)
}

func TestDevLoader_LoadChanges(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	defer c.Close()
	err = c.ApplyChanges(ctx, []schema.Change{
		&schema.AddTable{
			T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
		},
	})
	require.NoError(t, err)
	l := &ci.DevLoader{Dev: c, Scan: testDir{}}
	diff, err := l.LoadChanges(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, diff)

	files := []migrate.File{
		testFile{name: "1.sql", content: "CREATE TABLE t1 (id INT)\nINSERT INTO t1 (id) VALUES (1)"},
		testFile{name: "2.sql", content: "CREATE TABLE t2 (id INT)\nDROP TABLE users"},
		testFile{name: "3.sql", content: "CREATE TABLE t3 (id INT)\nDROP TABLE t3"},
	}
	diff, err = l.LoadChanges(ctx, files)
	require.NoError(t, err)
	require.Len(t, diff, 3)

	// File 1.
	require.Equal(t, files[0], diff[0].File)
	require.Len(t, diff[0].Changes, 2)
	require.Zero(t, diff[0].Changes[0].Pos)
	require.Equal(t, "CREATE TABLE t1 (id INT)", diff[0].Changes[0].Stmt)
	require.IsType(t, (*schema.AddTable)(nil), diff[0].Changes[0].Changes[0])
	require.Equal(t, "INSERT INTO t1 (id) VALUES (1)", diff[0].Changes[1].Stmt)
	require.Empty(t, diff[0].Changes[1].Changes)

	// File 2.
	require.Equal(t, files[1], diff[1].File)
	require.Len(t, diff[1].Changes, 2)
	require.Zero(t, diff[1].Changes[0].Pos)
	require.Equal(t, "CREATE TABLE t2 (id INT)", diff[1].Changes[0].Stmt)
	require.IsType(t, (*schema.AddTable)(nil), diff[1].Changes[0].Changes[0])
	require.Zero(t, diff[1].Changes[0].Pos)
	require.Equal(t, "DROP TABLE users", diff[1].Changes[1].Stmt)
	require.IsType(t, (*schema.DropTable)(nil), diff[1].Changes[1].Changes[0])

	// File 3.
	require.Equal(t, files[2], diff[2].File)
	require.IsType(t, (*schema.AddTable)(nil), diff[2].Changes[0].Changes[0])
	require.IsType(t, (*schema.DropTable)(nil), diff[2].Changes[1].Changes[0])
	require.Empty(t, diff[2].Sum)
}

type testDir struct {
	ci.DirScanner
	files []migrate.File
}

func (t testDir) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (t testDir) Files() ([]migrate.File, error) {
	return t.files, nil
}

func (testDir) Stmts(f migrate.File) ([]string, error) {
	return strings.Split(string(f.Bytes()), "\n"), nil
}

type testFile struct {
	fs.File
	name, content string
}

func (f testFile) Name() string {
	return f.name
}

func (f testFile) Bytes() []byte {
	return []byte(f.content)
}
