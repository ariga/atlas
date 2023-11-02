// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"archive/tar"
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqltool"

	"github.com/stretchr/testify/require"
)

func TestHashSum(t *testing.T) {
	// Sum file gets created.
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	v := time.Now().UTC().Format("20060102150405")
	plan := &migrate.Plan{Version: v, Name: "plan", Changes: []*migrate.Change{{Cmd: "cmd"}}}
	pl := migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, 2, countFiles(t, d))
	requireFileEqual(t, d, v+"_plan.sql", "cmd;\n")
	require.FileExists(t, filepath.Join(p, "atlas.sum"))

	// Disable sum.
	p = t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.PlanWithChecksum(false))
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, 1, countFiles(t, d))
	requireFileEqual(t, d, v+"_plan.sql", "cmd;\n")

	// Files not ending with .sql get ignored.
	p = t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, os.WriteFile(filepath.Join(p, "include.sql"), nil, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(p, "exclude.txt"), nil, 0600))
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, 4, countFiles(t, d))
	c, err := os.ReadFile(filepath.Join(p, "atlas.sum"))
	require.NoError(t, err)
	require.Contains(t, string(c), "include.sql")
	require.NotContains(t, string(c), "exclude.txt")

	// Files with directive in first line get ignored.
	p = t.TempDir()
	d, err = migrate.NewLocalDir(p)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, os.WriteFile(filepath.Join(p, "include.sql"), []byte("//atlas:sum\nfoo"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(p, "exclude_1.sql"), []byte("//atlas:sum ignore\nbar"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(p, "exclude_2.sql"), []byte("atlas:sum ignore"), 0600))
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, 5, countFiles(t, d))
	requireFileEqual(t, d, v+"_plan.sql", "cmd;\n")
	c, err = os.ReadFile(filepath.Join(p, "atlas.sum"))
	require.NoError(t, err)
	require.Contains(t, string(c), "include")
	require.NotContains(t, string(c), "exclude_1.sql")
	require.NotContains(t, string(c), "exclude_2.sql")
}

//go:embed testdata/migrate/atlas.sum
var hash []byte

func TestValidate(t *testing.T) {
	// Add the sum file form the testdata/migrate dir without any files in it - should fail.
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	require.NoError(t, d.WriteFile("atlas.sum", hash))
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))

	td := "testdata/migrate"
	d, err = migrate.NewLocalDir(td)
	require.NoError(t, err)

	// testdata/migrate is valid.
	require.Nil(t, migrate.Validate(d))

	// Making a manual change to the sum file should raise validation error.
	f, err := os.OpenFile(filepath.Join(td, "atlas.sum"), os.O_RDWR, os.ModeAppend)
	require.NoError(t, err)
	_, err = f.WriteString("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	t.Cleanup(func() {
		require.NoError(t, os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644))
	})
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))
	require.NoError(t, os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644))
	f, err = os.OpenFile(filepath.Join(td, "atlas.sum"), os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	require.NoError(t, err)
	_, err = f.WriteString("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, migrate.ErrChecksumFormat, migrate.Validate(d))
	require.NoError(t, os.WriteFile(filepath.Join(td, "atlas.sum"), hash, 0644))

	// Changing the filename should raise validation error.
	require.NoError(t, os.Rename(filepath.Join(td, "1_initial.up.sql"), filepath.Join(td, "1_first.up.sql")))
	t.Cleanup(func() {
		require.NoError(t, os.Rename(filepath.Join(td, "1_first.up.sql"), filepath.Join(td, "1_initial.up.sql")))
	})
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))

	// Removing it as well (move it out of the dir).
	require.NoError(t, os.Rename(filepath.Join(td, "1_first.up.sql"), filepath.Join(td, "..", "bak")))
	t.Cleanup(func() {
		require.NoError(t, os.Rename(filepath.Join(td, "..", "bak"), filepath.Join(td, "1_first.up.sql")))
	})
	require.Equal(t, migrate.ErrChecksumMismatch, migrate.Validate(d))
}

func TestHash_MarshalText(t *testing.T) {
	d, err := migrate.NewLocalDir("testdata/migrate")
	require.NoError(t, err)
	h, err := d.Checksum()
	require.NoError(t, err)
	ac, err := h.MarshalText()
	require.Equal(t, hash, ac)
}

func TestHash_UnmarshalText(t *testing.T) {
	d, err := migrate.NewLocalDir("testdata/migrate")
	require.NoError(t, err)
	h, err := d.Checksum()
	require.NoError(t, err)
	var ac migrate.HashFile
	require.NoError(t, ac.UnmarshalText(hash))
	require.Equal(t, h, ac)
}

func TestLocalDir(t *testing.T) {
	// Files don't work.
	d, err := migrate.NewLocalDir("migrate.go")
	require.ErrorContains(t, err, "sql/migrate: \"migrate.go\" is not a dir")
	require.Nil(t, d)

	// Does not create a dir for you.
	d, err = migrate.NewLocalDir("foo/bar")
	require.EqualError(t, err, "sql/migrate: stat foo/bar: no such file or directory")
	require.Nil(t, d)

	// Open and WriteFile work.
	d, err = migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, d)
	require.NoError(t, d.WriteFile("name", []byte("content")))
	f, err := d.Open("name")
	require.NoError(t, err)
	i, err := f.Stat()
	require.NoError(t, err)
	require.Equal(t, i.Name(), "name")
	c, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, "content", string(c))

	// Default Dir implementation.
	d, err = migrate.NewLocalDir("testdata/migrate/sub")
	require.NoError(t, err)
	require.NotNil(t, d)

	files, err := d.Files()
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.Equal(t, "1.a_sub.up.sql", files[0].Name())
	require.Equal(t, "2.10.x-20_description.sql", files[1].Name())
	require.Equal(t, "3_partly.sql", files[2].Name())

	stmts, err := files[0].Stmts()
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE t_sub(c int);", "ALTER TABLE t_sub ADD c1 int;"}, stmts)
	require.Equal(t, "1.a", files[0].Version())
	require.Equal(t, "sub.up", files[0].Desc())

	stmts, err = files[1].Stmts()
	require.NoError(t, err)
	require.Equal(t, []string{"ALTER TABLE t_sub ADD c2 int;"}, stmts)
	require.Equal(t, "2.10.x-20", files[1].Version())
	require.Equal(t, "description", files[1].Desc())
}

func TestCheckpointDir(t *testing.T) {
	local, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	for _, d := range []migrate.CheckpointDir{&migrate.MemDir{}, local} {
		files, err := d.Files()
		require.NoError(t, err)
		require.Empty(t, files)
		cks, err := d.CheckpointFiles()
		require.NoError(t, err)
		require.Empty(t, cks)
		require.NoError(t, migrate.Validate(d))

		require.NoError(t, d.WriteFile("1.sql", []byte("create table t1(c int);")))
		sum, err := d.Checksum()
		require.NoError(t, err)
		require.NoError(t, migrate.WriteSumFile(d, sum))
		require.NoError(t, migrate.Validate(d))
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 1)
		cks, err = d.CheckpointFiles()
		require.NoError(t, err)
		require.Empty(t, cks)

		require.NoError(t, d.WriteCheckpoint("2_checkpoint.sql", "", []byte("create table t1(c int);")))
		sum, err = d.Checksum()
		require.NoError(t, err)
		require.NoError(t, migrate.WriteSumFile(d, sum))
		require.NoError(t, migrate.Validate(d))
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 2)
		require.Equal(t, []string{"1.sql", "2_checkpoint.sql"}, []string{files[0].Name(), files[1].Name()})
		files = migrate.SkipCheckpointFiles(files)
		require.Len(t, files, 1)
		require.Equal(t, "1.sql", files[0].Name())
		cks, err = d.CheckpointFiles()
		require.NoError(t, err)
		require.Len(t, cks, 1)
		require.Equal(t, "2_checkpoint.sql", cks[0].Name())

		require.NoError(t, d.WriteFile("3.sql", []byte("create table t2(c int);")))
		sum, err = d.Checksum()
		require.NoError(t, err)
		require.NoError(t, migrate.WriteSumFile(d, sum))
		require.NoError(t, migrate.Validate(d))
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 3)
		require.Equal(t, []string{"1.sql", "2_checkpoint.sql", "3.sql"}, []string{files[0].Name(), files[1].Name(), files[2].Name()})
		files = migrate.SkipCheckpointFiles(files)
		require.Len(t, files, 2)
		require.Equal(t, []string{"1.sql", "3.sql"}, []string{files[0].Name(), files[1].Name()})
		cks, err = d.CheckpointFiles()
		require.NoError(t, err)
		require.Len(t, cks, 1)
		require.Equal(t, "2_checkpoint.sql", cks[0].Name())

		require.NoError(t, d.WriteCheckpoint("4_checkpoint.sql", "v4", []byte("create table t1(c int);\ncreate table t2(c int);")))
		sum, err = d.Checksum()
		require.NoError(t, err)
		require.NoError(t, migrate.WriteSumFile(d, sum))
		require.NoError(t, migrate.Validate(d))
		files, err = d.Files()
		require.NoError(t, err)
		require.Len(t, files, 4)
		require.Equal(t, []string{"1.sql", "2_checkpoint.sql", "3.sql", "4_checkpoint.sql"}, []string{files[0].Name(), files[1].Name(), files[2].Name(), files[3].Name()})
		files = migrate.SkipCheckpointFiles(files)
		require.Len(t, files, 2)
		require.Equal(t, []string{"1.sql", "3.sql"}, []string{files[0].Name(), files[1].Name()})
		cks, err = d.CheckpointFiles()
		require.NoError(t, err)
		require.Len(t, cks, 2)
		require.Equal(t, []string{"2_checkpoint.sql", "4_checkpoint.sql"}, []string{cks[0].Name(), cks[1].Name()})
		tag, err := cks[0].(migrate.CheckpointFile).CheckpointTag()
		require.NoError(t, err)
		require.Equal(t, "", tag)
		tag, err = cks[1].(migrate.CheckpointFile).CheckpointTag()
		require.NoError(t, err)
		require.Equal(t, "v4", tag)
	}
}

func TestMemDir(t *testing.T) {
	var d migrate.MemDir
	files, err := d.Files()
	require.NoError(t, err)
	require.Empty(t, files)
	require.NoError(t, migrate.Validate(&d))

	require.NoError(t, d.WriteFile("1.sql", []byte("create table t(c int);")))
	files, err = d.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "1.sql", files[0].Name())
	require.Equal(t, "1", files[0].Version())
	require.Equal(t, "", files[0].Desc())
	require.EqualValues(t, "create table t(c int);", files[0].Bytes())
	hs1, err := d.Checksum()
	require.NoError(t, err)
	hs2, err := migrate.NewHashFile(files)
	require.NoError(t, err)
	require.Equal(t, hs1, hs2)

	// Will fail without checksum file.
	require.ErrorIs(t, migrate.Validate(&d), migrate.ErrChecksumNotFound)

	// Will not return the non-sql checksum file.
	files, err = d.Files()
	require.NoError(t, err)
	require.Len(t, files, 1) // 1.sql

	// Sync with additional directory.
	var d2, d3 migrate.MemDir
	d.SyncWrites(d2.WriteFile, d3.WriteFile)
	require.NoError(t, d.WriteFile("2.sql", []byte("create table t2(c int);")))
	for _, d1 := range []migrate.MemDir{d, d2, d3} {
		f, err := d1.Open("2.sql")
		require.NoError(t, err)
		c, err := io.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, "create table t2(c int);", string(c))
	}
}

func TestOpenMemDir(t *testing.T) {
	dev1 := migrate.OpenMemDir("dev")
	require.NoError(t, dev1.WriteFile("1.sql", []byte("create table t1(c int);")))
	// Open the same dir.
	dev2 := migrate.OpenMemDir("dev")
	files2, err := dev2.Files()
	require.NoError(t, err)
	require.Len(t, files2, 1)
	require.NoError(t, dev2.WriteFile("2.sql", []byte("create table t2(c int);")))
	files1, err := dev1.Files()
	require.NoError(t, err)
	require.Len(t, files1, 2)
	files2, err = dev2.Files()
	require.NoError(t, err)
	require.Len(t, files2, 2)
	// Open a new dir.
	etc := migrate.OpenMemDir("etc")
	files, err := etc.Files()
	require.NoError(t, err)
	require.Empty(t, files)

	// Closing dir and opening it should not
	// clean it if there are active references.
	require.NoError(t, dev1.Close())
	dev1 = migrate.OpenMemDir("dev")
	files1, err = dev1.Files()
	require.NoError(t, err)
	require.Len(t, files1, 2)

	// Cleanup directory on close.
	require.NoError(t, dev1.Close())
	require.NoError(t, dev2.Close())
	dev1 = migrate.OpenMemDir("dev")
	files1, err = dev1.Files()
	require.NoError(t, err)
	require.Empty(t, files1)
}

func TestLocalFile_Directive(t *testing.T) {
	f := migrate.NewLocalFile("1.sql", []byte(`-- atlas:lint ignore
alter table users drop column id;
`))
	require.Empty(t, f.Directive("lint"), "statement directives are ignored")

	f = migrate.NewLocalFile("1.sql", []byte(`-- atlas:lint ignore

alter table users drop column id;

-- atlas:lint DS102
alter table pets drop column id;
`))
	require.Equal(t, []string{"ignore"}, f.Directive("lint"), "single directive")

	f = migrate.NewLocalFile("1.sql", []byte(`-- atlas:lint ignore
-- atlas:txmode none

alter table users drop column id;

-- atlas:lint DS102
alter table pets drop column id;
`))
	require.Equal(t, []string{"ignore"}, f.Directive("lint"), "first directive from two")
	require.Equal(t, []string{"none"}, f.Directive("txmode"), "second directive from two")

	f = migrate.NewLocalFile("1.sql", []byte(`-- atlas:nolint

alter table users drop column id;
`))
	require.Equal(t, []string{""}, f.Directive("nolint"), "directives without arguments returned as empty string")

	f = migrate.NewLocalFile("1.sql", nil)
	require.Empty(t, f.Directive("lint"))
	f = migrate.NewLocalFile("1.sql", []byte("-- atlas:lint ignore"))
	require.Empty(t, f.Directive("lint"))
	f = migrate.NewLocalFile("1.sql", []byte("-- atlas:lint ignore\n"))
	require.Empty(t, f.Directive("lint"))
	f = migrate.NewLocalFile("1.sql", []byte("-- atlas:lint ignore\n\n"))
	require.Equal(t, []string{"ignore"}, f.Directive("lint"), "double newline as directive separator")
}

func TestLocalFile_AddDirective(t *testing.T) {
	f := migrate.NewLocalFile("1.sql", []byte("SELECT 1;"))
	f.AddDirective("lint", "ignore")
	require.Equal(t, []string{"ignore"}, f.Directive("lint"))
	require.Equal(t, "-- atlas:lint ignore\n\nSELECT 1;", string(f.Bytes()))
	f.AddDirective("checkpoint")
	require.Equal(t, []string{"ignore"}, f.Directive("lint"))
	require.Equal(t, []string{""}, f.Directive("checkpoint"))
	require.Equal(t, `-- atlas:checkpoint
-- atlas:lint ignore

SELECT 1;`, string(f.Bytes()))

	f = migrate.NewLocalFile("1.sql", []byte("-- atlas:directive statement directive\nSELECT 1;"))
	f.AddDirective("lint", "ignore")
	require.Equal(t, []string{"ignore"}, f.Directive("lint"))
	require.Equal(t, `-- atlas:lint ignore

-- atlas:directive statement directive
SELECT 1;`, string(f.Bytes()))
}

func TestLocalFile_CheckpointTag(t *testing.T) {
	// Not a checkpoint.
	for _, b := range []string{
		"SELECT 1;",
		"-- atlas:checkpoint\nSELECT 1;",
		"-- atlas:checkpoint tag\nSELECT 1;",
	} {
		f := migrate.NewLocalFile("1.sql", []byte(b))
		require.False(t, f.IsCheckpoint())
		tag, err := f.CheckpointTag()
		require.ErrorIs(t, err, migrate.ErrNotCheckpoint)
		require.Empty(t, tag)
	}
	// Checkpoint.
	f := migrate.NewLocalFile("1.sql", []byte("-- atlas:checkpoint\n\nSELECT 1;"))
	require.True(t, f.IsCheckpoint())
	tag, err := f.CheckpointTag()
	require.NoError(t, err)
	require.Empty(t, tag)
	f = migrate.NewLocalFile("1.sql", []byte("-- atlas:checkpoint tag\n\nSELECT 1;"))
	require.True(t, f.IsCheckpoint())
	tag, err = f.CheckpointTag()
	require.NoError(t, err)
	require.Equal(t, "tag", tag)
}

func TestDirTar(t *testing.T) {
	d := migrate.OpenMemDir("")
	defer d.Close()

	err := d.WriteFile("1.sql", []byte("create table t(c int);"))
	require.NoError(t, err)

	b, err := migrate.ArchiveDir(d)
	require.NoError(t, err)
	f, err := fileNames(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, []string{"1.sql"}, f)

	// Test with Golang migrate dir.
	dir1, err := sqltool.NewGolangMigrateDir("./testdata/golang-migrate")
	require.NoError(t, err)
	b, err = migrate.ArchiveDir(dir1)
	require.NoError(t, err)

	// With sumfile.
	checksum, err := d.Checksum()
	require.NoError(t, err)
	err = migrate.WriteSumFile(d, checksum)
	require.NoError(t, err)

	b, err = migrate.ArchiveDir(d)
	require.NoError(t, err)
	f, err = fileNames(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, []string{"atlas.sum", "1.sql"}, f)

	dir, err := migrate.UnarchiveDir(b)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "1.sql", files[0].Name())
	require.Equal(t, "create table t(c int);", string(files[0].Bytes()))
}

func fileNames(r io.Reader) ([]string, error) {
	var out []string
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		out = append(out, hdr.Name)
	}
	return out, nil
}
