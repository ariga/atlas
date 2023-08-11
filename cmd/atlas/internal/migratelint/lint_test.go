// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint_test

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestGitChangeDetector(t *testing.T) {
	// Prepare environment.
	root := filepath.Join(t.TempDir(), t.Name(), strconv.FormatInt(time.Now().Unix(), 10))
	mdir := filepath.Join(root, "migrations")
	require.NoError(t, os.MkdirAll(mdir, 0755))
	git := func(args ...string) {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git("init")
	// Config a fake Git user for the working directory.
	git("config", "user.name", "a8m")
	git("config", "user.email", "a8m@atlasgo.io")
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "1_applied.sql"), []byte("1_applied.sql"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "2_applied.sql"), []byte("2_applied.sql"), 0644))
	git("add", ".")
	git("commit", "-m", "applied migrations")
	git("checkout", "-b", "feature")
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "3_new.sql"), []byte("3_new.sql"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "4_new.sql"), []byte("4_new.sql"), 0644))
	git("add", ".")
	git("commit", "-am", "new migrations")

	// Test change detector.
	dir, err := migrate.NewLocalDir(mdir)
	require.NoError(t, err)
	cs, err := migratelint.NewGitChangeDetector(dir, migratelint.WithWorkDir(root))
	require.NoError(t, err)
	base, feat, err := cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 2)
	require.Len(t, feat, 2)
	require.Equal(t, "1_applied.sql", base[0].Name())
	require.Equal(t, "2_applied.sql", base[1].Name())
	require.Equal(t, "3_new.sql", feat[0].Name())
	require.Equal(t, "4_new.sql", feat[1].Name())

	require.NoError(t, os.WriteFile(filepath.Join(mdir, "5_new.sql"), []byte("5_new.sql"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(mdir, "6_new.sql"), []byte("6_new.sql"), 0644))
	git("checkout", "-b", "feature-1")
	git("add", ".")
	git("commit", "-am", "new migrations")
	base, feat, err = cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 2)
	require.Len(t, feat, 4)
	require.Equal(t, "5_new.sql", feat[2].Name())
	require.Equal(t, "6_new.sql", feat[3].Name())

	// Compare feature and feature-1.
	cs, err = migratelint.NewGitChangeDetector(dir, migratelint.WithWorkDir(root), migratelint.WithBase("feature"))
	require.NoError(t, err)
	base, feat, err = cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 4)
	require.Len(t, feat, 2)
	require.Equal(t, "1_applied.sql", base[0].Name())
	require.Equal(t, "2_applied.sql", base[1].Name())
	require.Equal(t, "3_new.sql", base[2].Name())
	require.Equal(t, "4_new.sql", base[3].Name())
	require.Equal(t, "5_new.sql", feat[0].Name())
	require.Equal(t, "6_new.sql", feat[1].Name())
}

func TestLatestChanges(t *testing.T) {
	files := []migrate.File{
		testFile{name: "1.sql", content: "CREATE TABLE t1 (id INT)"},
		testFile{name: "2.sql", content: "CREATE TABLE t2 (id INT)\nDROP TABLE users"},
	}
	base, feat, err := migratelint.LatestChanges(testDir{files: files}, 0).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Equal(t, files, base)
	require.Empty(t, feat)

	base, feat, err = migratelint.LatestChanges(testDir{files: files}, 2).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = migratelint.LatestChanges(testDir{files: files}, -1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = migratelint.LatestChanges(testDir{files: files}, 1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Equal(t, files[:1], base)
	require.Equal(t, files[1:], feat)
}

func TestDevLoader_LoadChanges(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	defer c.Close()
	l := &migratelint.DevLoader{Dev: c}
	diff, err := l.LoadChanges(ctx, nil, nil)
	require.NoError(t, err)
	require.Empty(t, diff.Files)

	diff, err = l.LoadChanges(ctx, []migrate.File{
		testFile{name: "base.sql", content: "---\n\nCREATE INVALID users (id INT);\n"},
	}, nil)
	require.Error(t, err)
	require.Nil(t, diff)
	fr := err.(*migratelint.FileError)
	require.Equal(t, `executing statement: near "INVALID": syntax error`, fr.Err.Error())
	require.Equal(t, 5, fr.Pos)

	base := []migrate.File{
		testFile{name: "base.sql", content: "CREATE TABLE users (id INT);"},
	}
	files := []migrate.File{
		testFile{name: "1.sql", content: "CREATE TABLE t1 (id INT);\nINSERT INTO t1 (id) VALUES (1);"},
		testFile{name: "2.sql", content: "CREATE TABLE t2 (id INT);\nDROP TABLE users;"},
		testFile{name: "3.sql", content: "CREATE TABLE t3 (id INT);\nDROP TABLE t3;"},
		testFile{name: "4.sql", content: "ALTER TABLE t2 RENAME id TO oid;"},
	}
	diff, err = l.LoadChanges(ctx, base, files)
	require.NoError(t, err)
	require.Len(t, diff.Files, 4)

	// File 1.
	require.Equal(t, files[0], diff.Files[0].File)
	require.Len(t, diff.Files[0].Changes, 2)
	require.Zero(t, diff.Files[0].Changes[0].Stmt.Pos)
	require.Equal(t, "CREATE TABLE t1 (id INT);", diff.Files[0].Changes[0].Stmt.Text)
	require.IsType(t, (*schema.AddTable)(nil), diff.Files[0].Changes[0].Changes[0])
	require.Equal(t, "INSERT INTO t1 (id) VALUES (1);", diff.Files[0].Changes[1].Stmt.Text)
	require.Empty(t, diff.Files[0].Changes[1].Changes)

	// File 2.
	require.Equal(t, files[1], diff.Files[1].File)
	require.Len(t, diff.Files[1].Changes, 2)
	require.Zero(t, diff.Files[1].Changes[0].Stmt.Pos)
	require.Equal(t, "CREATE TABLE t2 (id INT);", diff.Files[1].Changes[0].Stmt.Text)
	require.IsType(t, (*schema.AddTable)(nil), diff.Files[1].Changes[0].Changes[0])
	require.Zero(t, diff.Files[1].Changes[0].Stmt.Pos)
	require.Equal(t, "DROP TABLE users;", diff.Files[1].Changes[1].Stmt.Text)
	require.IsType(t, (*schema.DropTable)(nil), diff.Files[1].Changes[1].Changes[0])

	// File 3.
	require.Equal(t, files[2], diff.Files[2].File)
	require.IsType(t, (*schema.AddTable)(nil), diff.Files[2].Changes[0].Changes[0])
	require.IsType(t, (*schema.DropTable)(nil), diff.Files[2].Changes[1].Changes[0])
	require.Empty(t, diff.Files[2].Sum)

	// File 3.
	require.Equal(t, files[3], diff.Files[3].File)
	require.IsType(t, (*schema.ModifyTable)(nil), diff.Files[3].Changes[0].Changes[0])
	require.IsType(t, (*schema.RenameColumn)(nil), diff.Files[3].Changes[0].Changes[0].(*schema.ModifyTable).Changes[0])

	// Changes.
	changes, err := c.RealmDiff(diff.From, diff.To)
	require.NoError(t, err)
	require.Len(t, changes, 3)

	err = c.ApplyChanges(ctx, []schema.Change{
		&schema.AddTable{
			T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
		},
	})
	require.NoError(t, err)
	_, err = l.LoadChanges(ctx, base, files)
	require.ErrorAs(t, err, new(*migrate.NotCleanError))
}

type testDir struct {
	migrate.Dir
	files []migrate.File
}

func (t testDir) Path() string {
	return "migrations"
}

func (t testDir) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (t testDir) Files() ([]migrate.File, error) {
	return t.files, nil
}

type testFile struct {
	migrate.File
	name, content string
}

func (f testFile) Name() string {
	return f.name
}

func (f testFile) Bytes() []byte {
	return []byte(f.content)
}

func (f testFile) Stmts() ([]string, error) {
	return strings.Split(f.content, "\n"), nil
}

func (f testFile) StmtDecls() (stmts []*migrate.Stmt, err error) {
	return migrate.Stmts(f.content)
}
