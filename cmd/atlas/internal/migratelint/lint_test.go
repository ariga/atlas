// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migratelint_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migratelint"
	"ariga.io/atlas/sql/migrate"
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
