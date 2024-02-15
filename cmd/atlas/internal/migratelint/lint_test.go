// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestDirChangeDetector(t *testing.T) {
	base, head := &migrate.MemDir{}, &migrate.MemDir{}
	require.NoError(t, base.WriteFile("1.sql", []byte("create table t1 (id int)")))
	require.NoError(t, head.WriteFile("1.sql", []byte("create table t1 (id int)")))
	require.NoError(t, base.WriteFile("2.sql", []byte("create table t2 (id int)")))
	require.NoError(t, head.WriteFile("2.sql", []byte("create table t2 (id int)")))

	baseF, newF, err := migratelint.DirChangeDetector{Base: base, Head: head}.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, baseF, 2)
	require.Empty(t, newF)

	require.NoError(t, head.WriteFile("3.sql", []byte("create table t3 (id int)")))
	baseF, newF, err = migratelint.DirChangeDetector{Base: base, Head: head}.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, baseF, 2)
	require.Len(t, newF, 1)
}

func TestLatestChanges(t *testing.T) {
	dir := &migrate.MemDir{}
	require.NoError(t, dir.WriteFile("1.sql", []byte("CREATE TABLE t1 (id INT)")))
	require.NoError(t, dir.WriteFile("2.sql", []byte("CREATE TABLE t2 (id INT)")))
	base, feat, err := migratelint.LatestChanges(dir, 0).DetectChanges(context.Background())
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	require.Equal(t, files, base)
	require.Empty(t, feat)

	base, feat, err = migratelint.LatestChanges(dir, 2).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = migratelint.LatestChanges(dir, -1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Empty(t, base)
	require.Equal(t, files, feat)

	base, feat, err = migratelint.LatestChanges(dir, 1).DetectChanges(context.Background())
	require.NoError(t, err)
	require.Equal(t, files[:1], base)
	require.Equal(t, files[1:], feat)
}

func TestDevLoader_LoadChanges(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer c.Close()
	l := &migratelint.DevLoader{Dev: c}
	diff, err := l.LoadChanges(ctx, nil, nil)
	require.NoError(t, err)
	require.Empty(t, diff.Files)

	diff, err = l.LoadChanges(ctx, []migrate.File{
		migrate.NewLocalFile("0_base.sql", []byte("---\n\nCREATE INVALID users (id INT);\n")),
	}, nil)
	require.Error(t, err)
	require.Nil(t, diff)
	var fr *migratelint.FileError
	require.True(t, errors.As(err, &fr))
	require.Equal(t, `executing statement: near "INVALID": syntax error`, fr.Err.Error())
	require.Equal(t, 5, fr.Pos)

	base := []migrate.File{
		migrate.NewLocalFile("0_base.sql", []byte("CREATE TABLE users (id INT);")),
	}
	files := []migrate.File{
		migrate.NewLocalFile("1.sql", []byte("CREATE TABLE t1 (id INT);\nINSERT INTO t1 (id) VALUES (1);")),
		migrate.NewLocalFile("2.sql", []byte("CREATE TABLE t2 (id INT);\nDROP TABLE users;")),
		migrate.NewLocalFile("3.sql", []byte("CREATE TABLE t3 (id INT);\nDROP TABLE t3;")),
		migrate.NewLocalFile("4.sql", []byte("ALTER TABLE t2 RENAME id TO oid;")),
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

func TestDevLoader_LoadCheckpoints(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://ci?mode=memory&_fk=1")
	require.NoError(t, err)
	defer c.Close()
	dir := &migrate.MemDir{}
	l := &migratelint.DevLoader{Dev: c}
	require.NoError(t, dir.WriteFile("1.sql", []byte("CREATE TABLE t1 (id INT);")))
	require.NoError(t, dir.WriteFile("2.sql", []byte("CREATE TABLE t2 (id INT);")))
	require.NoError(t, dir.WriteCheckpoint("3_checkpoint.sql", "", []byte("CREATE TABLE t1 (id INT);\nCREATE TABLE t2 (id INT);")))
	require.NoError(t, dir.WriteFile("4.sql", []byte("CREATE TABLE t3 (id INT);")))

	files, err := dir.Files()
	require.NoError(t, err)
	// Base contains a checkpoint file.
	diff, err := l.LoadChanges(ctx, files[:3], files[3:])
	require.NoError(t, err)
	require.Len(t, diff.Files, 1)
	require.Equal(t, "4.sql", diff.Files[0].File.Name())
	isAddTable(t, diff.Files[0].Changes[0].Changes[0], "t3")

	// Changed files contain a checkpoint file.
	diff, err = l.LoadChanges(ctx, files[:2], files[2:])
	require.NoError(t, err)
	require.Len(t, diff.Files, 2)
	require.Equal(t, "3_checkpoint.sql", diff.Files[0].File.Name())
	require.Len(t, diff.Files[0].Changes, 2)
	isAddTable(t, diff.Files[0].Changes[0].Changes[0], "t1")
	isAddTable(t, diff.Files[0].Changes[1].Changes[0], "t2")
	require.Equal(t, "4.sql", diff.Files[1].File.Name())
	isAddTable(t, diff.Files[1].Changes[0].Changes[0], "t3")

	// Both base and changed files contain a checkpoint file.
	require.NoError(t, dir.WriteCheckpoint("5_checkpoint.sql", "", []byte("CREATE TABLE t1(id INT);\nCREATE TABLE t2(id INT);\nCREATE TABLE t3(id INT);")))
	files, err = dir.Files()
	require.NoError(t, err)
	diff, err = l.LoadChanges(ctx, files[:3], files[3:])
	require.NoError(t, err)
	require.Len(t, diff.Files, 2)
	require.Equal(t, "4.sql", diff.Files[0].File.Name())
	isAddTable(t, diff.Files[0].Changes[0].Changes[0], "t3")
	require.Equal(t, "5_checkpoint.sql", diff.Files[1].File.Name())
	require.Len(t, diff.Files[1].Changes, 3)
	isAddTable(t, diff.Files[1].Changes[0].Changes[0], "t1")
	isAddTable(t, diff.Files[1].Changes[1].Changes[0], "t2")
	isAddTable(t, diff.Files[1].Changes[2].Changes[0], "t3")
}

func isAddTable(t *testing.T, c schema.Change, name string) {
	require.IsType(t, (*schema.AddTable)(nil), c)
	require.Equal(t, name, c.(*schema.AddTable).T.Name)
}
