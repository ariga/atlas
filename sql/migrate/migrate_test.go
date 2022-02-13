// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"bytes"
	"io"
	"strconv"
	"testing"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestPlanner_WritePlan(t *testing.T) {
	var mfs = &mockFS{}
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

type (
	mockFS struct {
		files []*file
	}
	file struct {
		n string
		b *bytes.Buffer
	}
)

func (f *file) Read(b []byte) (int, error) {
	return f.b.Read(b)
}

func (f *file) Write(b []byte) (int, error) {
	return f.b.Write(b)
}

func (f *file) Close() error { return nil }

func (f *file) Name() string { return f.n }

func (fs *mockFS) Open(name string) (io.ReadWriteCloser, error) {
	for _, f := range fs.files {
		if f.n == name {
			return f, nil
		}
	}
	f := &file{n: name, b: new(bytes.Buffer)}
	fs.files = append(fs.files, f)
	return f, nil
}

var _ io.ReadWriteCloser = (*file)(nil)
