// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"io/fs"
	"os"
	"strconv"
	"testing"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestPlanner_WritePlan(t *testing.T) {
	mfs := &mockFS{}
	plan := &migrate.Plan{
		Name: "add_t1_and_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);", Reverse: "DROP TABLE t1 IF EXISTS"},
			{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2"},
		},
	}

	// DefaultFormatter
	pl := migrate.New(nil, mfs, migrate.DefaultFormatter)
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := strconv.FormatInt(time.Now().Unix(), 10)
	require.Len(t, mfs.files, 2)
	require.Equal(t, &file{
		n: v + "_add_t1_and_t2.up.sql",
		b: bytes.NewBufferString("CREATE TABLE t1(c int);\nCREATE TABLE t2(c int);\n"),
	}, mfs.files[0])
	require.Equal(t, &file{
		n: v + "_add_t1_and_t2.down.sql",
		b: bytes.NewBufferString("DROP TABLE t2;\nDROP TABLE t1 IF EXISTS;\n"),
	}, mfs.files[1])

	// Custom formatter (creates only "up" migration files).
	fmt, err := migrate.NewTemplateFormatter(
		template.Must(template.New("").Parse("{{.Name}}.sql")),
		template.Must(template.New("").Parse("{{range .Changes}}{{println .Cmd}}{{end}}")),
	)
	require.NoError(t, err)
	pl = migrate.New(nil, mfs, fmt)
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Len(t, mfs.files, 3)
	require.Equal(t, &file{
		n: "add_t1_and_t2.sql",
		b: bytes.NewBufferString("CREATE TABLE t1(c int);\nCREATE TABLE t2(c int)\n"),
	}, mfs.files[2])
}

func TestPlanner_Plan(t *testing.T) {
	var (
		mfs = &mockFS{}
		drv = &mockDriver{}
		ctx = context.Background()
	)

	// nothing to do
	pl := migrate.New(drv, mfs, migrate.DefaultFormatter)
	plan, err := pl.Plan(ctx, "empty", migrate.Realm(nil), migrate.Realm(nil))
	require.ErrorIs(t, err, migrate.ErrNoPlan)
	require.Nil(t, plan)

	// there are changes
	drv.changes = []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c", "int"))},
		&schema.AddTable{T: schema.NewTable("t2").AddColumns(schema.NewIntColumn("c", "int"))},
	}
	drv.plan = &migrate.Plan{
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);"},
			{Cmd: "CREATE TABLE t2(c int);"},
		},
	}
	plan, err = pl.Plan(ctx, "", migrate.Realm(nil), migrate.Realm(nil))
	require.NoError(t, err)
	require.Equal(t, drv.plan, plan)
}

func TestGlobStateReader(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	localFS, err := migrate.NewLocalDir("testdata")
	require.NoError(t, err)

	_, err = localFS.GlobStateReader(drv, "*.up.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, drv.executed, []string{"CREATE TABLE t(c int);"})

	_, err = localFS.GlobStateReader(drv, "*.down.sql").ReadState(ctx)
	require.NoError(t, err)
	require.Equal(t, drv.executed, []string{"CREATE TABLE t(c int);", "DROP TABLE IF EXISTS t;"})
}

func TestLocalDir(t *testing.T) {
	_, err := migrate.NewLocalDir("does_not_exist")
	require.Error(t, os.ErrNotExist)

	d, err := migrate.NewLocalDir(t.TempDir())
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
}

type (
	mockFS struct {
		files []*file
	}
	file struct {
		n string
		b *bytes.Buffer
	}
	fileInfo struct {
		n string
	}
)

func (f *file) Read(b []byte) (int, error) { return f.b.Read(b) }
func (f *file) Close() error               { return nil }
func (f *file) Stat() (fs.FileInfo, error) { return fileInfo{f.n}, nil }

func (i fileInfo) Name() string       { return i.n }
func (i fileInfo) Size() int64        { return 0 }
func (i fileInfo) Mode() fs.FileMode  { return 0 }
func (i fileInfo) ModTime() time.Time { return time.Time{} }
func (i fileInfo) IsDir() bool        { return false }
func (i fileInfo) Sys() interface{}   { return nil }

func (fs *mockFS) Open(name string) (fs.File, error) {
	for _, f := range fs.files {
		if f.n == name {
			return f, nil
		}
	}
	f := &file{n: name, b: new(bytes.Buffer)}
	fs.files = append(fs.files, f)
	return f, nil
}

func (fs *mockFS) WriteFile(name string, d []byte) error {
	fs.files = append(fs.files, &file{n: name, b: bytes.NewBuffer(d)})
	return nil
}

type mockDriver struct {
	migrate.Driver
	plan     *migrate.Plan
	changes  []schema.Change
	executed []string
}

func (m *mockDriver) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	m.executed = append(m.executed, query)
	return nil, nil
}

func (m mockDriver) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return &schema.Realm{}, nil
}
func (m mockDriver) RealmDiff(_, _ *schema.Realm) ([]schema.Change, error) {
	return m.changes, nil
}
func (m mockDriver) PlanChanges(context.Context, string, []schema.Change) (*migrate.Plan, error) {
	return m.plan, nil
}
