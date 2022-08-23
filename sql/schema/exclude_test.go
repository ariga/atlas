// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExcludeRealm_Schemas(t *testing.T) {
	r := NewRealm(New("s1"), New("s2"), New("s3"))
	r, err := ExcludeRealm(r, []string{"s4"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)

	r, err = ExcludeRealm(r, []string{"s1", "s2.t2", "s3.t3.c3"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 2)
	require.Equal(t, "s2", r.Schemas[0].Name)
	require.Equal(t, "s3", r.Schemas[1].Name)

	r, err = ExcludeRealm(r, []string{"*"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas)

	r = NewRealm(New("s1"), New("s2"), New("s3"))
	r, err = ExcludeRealm(r, []string{"s*.*", "s*.*.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	r, err = ExcludeRealm(r, []string{"s*"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas)
}

func TestExcludeRealm_Tables(t *testing.T) {
	r := NewRealm(
		New("s0"),
		New("s1").AddTables(
			NewTable("t1"),
		),
		New("s2").AddTables(
			NewTable("t1"),
			NewTable("t2"),
		),
		New("s3").AddTables(
			NewTable("t1"),
			NewTable("t2"),
			NewTable("t3"),
		),
	)
	r, err := ExcludeRealm(r, []string{"s4"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 4)
	r, err = ExcludeRealm(r, []string{"s0"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Equal(t, "s1", r.Schemas[0].Name)
	require.Equal(t, "s2", r.Schemas[1].Name)
	require.Equal(t, "s3", r.Schemas[2].Name)

	r, err = ExcludeRealm(r, []string{"*.t1.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Len(t, r.Schemas[1].Tables, 2)
	require.Len(t, r.Schemas[2].Tables, 3)

	r, err = ExcludeRealm(r, []string{"*.t1"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Tables)
	require.Len(t, r.Schemas[1].Tables, 1)
	require.Len(t, r.Schemas[2].Tables, 2)

	r, err = ExcludeRealm(r, []string{"s[12].t2"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Tables)
	require.Empty(t, r.Schemas[1].Tables)
	require.Len(t, r.Schemas[2].Tables, 2)

	r, err = ExcludeRealm(r, []string{"*.t[23].*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[2].Tables, 2)
	r, err = ExcludeRealm(r, []string{"*.t[23]"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[2].Tables)
}

func TestExcludeRealm_Columns(t *testing.T) {
	r := NewRealm(
		New("s1").AddTables(
			func() *Table {
				t := NewTable("t1").AddColumns(NewColumn("c1"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
		),
		New("s2").AddTables(
			func() *Table {
				t := NewTable("t1").AddColumns(NewColumn("c1"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
			func() *Table {
				t := NewTable("t2").AddColumns(NewColumn("c1"), NewColumn("c2"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]), NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
		),
		New("s3").AddTables(
			func() *Table {
				t := NewTable("t1").AddColumns(NewColumn("c1"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
			func() *Table {
				t := NewTable("t2").AddColumns(NewColumn("c1"), NewColumn("c2"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]), NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
			func() *Table {
				t := NewTable("t3").AddColumns(NewColumn("c1"), NewColumn("c2"), NewColumn("c3"))
				t.AddIndexes(NewIndex("i1").AddColumns(t.Columns[0]), NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
		),
	)
	r, err := ExcludeRealm(r, []string{"s[23].t[23].c1"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Len(t, r.Schemas[0].Tables[0].Columns, 1)
	require.Len(t, r.Schemas[0].Tables[0].Indexes, 1)

	require.Len(t, r.Schemas[1].Tables, 2)
	require.Len(t, r.Schemas[1].Tables[0].Columns, 1)
	require.Len(t, r.Schemas[1].Tables[0].Indexes, 1)
	require.Len(t, r.Schemas[1].Tables[1].Columns, 1)
	require.Len(t, r.Schemas[1].Tables[1].Indexes, 1)

	require.Len(t, r.Schemas[2].Tables, 3)
	require.Len(t, r.Schemas[2].Tables[0].Columns, 1)
	require.Len(t, r.Schemas[2].Tables[0].Indexes, 1)
	require.Len(t, r.Schemas[2].Tables[1].Columns, 1)
	require.Len(t, r.Schemas[2].Tables[1].Indexes, 1)
	require.Len(t, r.Schemas[2].Tables[2].Columns, 2)
	require.Len(t, r.Schemas[2].Tables[2].Indexes, 1)

	r, err = ExcludeRealm(r, []string{"s[23].t*.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Len(t, r.Schemas[0].Tables[0].Columns, 1)
	require.Len(t, r.Schemas[0].Tables[0].Indexes, 1)

	require.Len(t, r.Schemas[1].Tables, 2)
	require.Empty(t, r.Schemas[1].Tables[0].Columns)
	require.Empty(t, r.Schemas[1].Tables[0].Indexes)
	require.Empty(t, r.Schemas[1].Tables[1].Columns)
	require.Empty(t, r.Schemas[1].Tables[1].Indexes)

	require.Len(t, r.Schemas[2].Tables, 3)
	require.Empty(t, r.Schemas[2].Tables[0].Columns)
	require.Empty(t, r.Schemas[2].Tables[0].Indexes)
	require.Empty(t, r.Schemas[2].Tables[1].Columns)
	require.Empty(t, r.Schemas[2].Tables[1].Indexes)
	require.Empty(t, r.Schemas[2].Tables[2].Columns)
	require.Empty(t, r.Schemas[2].Tables[2].Indexes)

	r, err = ExcludeRealm(r, []string{"*.*.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Empty(t, r.Schemas[0].Tables[0].Columns)
	require.Empty(t, r.Schemas[0].Tables[0].Indexes)
}

func TestExcludeSchema(t *testing.T) {
	r := NewRealm(
		New("s1").AddTables(
			NewTable("t1"),
			NewTable("t2"),
		),
	)
	_, err := ExcludeSchema(r.Schemas[0], []string{"t2"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 1)
	require.Len(t, r.Schemas[0].Tables, 1)
}
