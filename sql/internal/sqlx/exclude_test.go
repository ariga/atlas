// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestExcludeRealm_Schemas(t *testing.T) {
	r := schema.NewRealm(schema.New("s1"), schema.New("s2"), schema.New("s3"))
	require.NoError(t, ExcludeRealm(r, []string{"s4"}))
	require.Len(t, r.Schemas, 3)

	require.NoError(t, ExcludeRealm(r, []string{"s1", "s2.t2", "s3.t3.c3"}))
	require.Len(t, r.Schemas, 2)
	require.Equal(t, "s2", r.Schemas[0].Name)
	require.Equal(t, "s3", r.Schemas[1].Name)

	require.NoError(t, ExcludeRealm(r, []string{"*"}))
	require.Empty(t, r.Schemas)

	r = schema.NewRealm(schema.New("s1"), schema.New("s2"), schema.New("s3"))
	require.NoError(t, ExcludeRealm(r, []string{"s*.*", "s*.*.*"}))
	require.Len(t, r.Schemas, 3)
	require.NoError(t, ExcludeRealm(r, []string{"s*"}))
	require.Empty(t, r.Schemas)
}

func TestExcludeRealm_Tables(t *testing.T) {
	r := schema.NewRealm(
		schema.New("s0"),
		schema.New("s1").AddTables(
			schema.NewTable("t1"),
		),
		schema.New("s2").AddTables(
			schema.NewTable("t1"),
			schema.NewTable("t2"),
		),
		schema.New("s3").AddTables(
			schema.NewTable("t1"),
			schema.NewTable("t2"),
			schema.NewTable("t3"),
		),
	)
	require.NoError(t, ExcludeRealm(r, []string{"s4"}))
	require.Len(t, r.Schemas, 4)
	require.NoError(t, ExcludeRealm(r, []string{"s0"}))
	require.Len(t, r.Schemas, 3)
	require.Equal(t, "s1", r.Schemas[0].Name)
	require.Equal(t, "s2", r.Schemas[1].Name)
	require.Equal(t, "s3", r.Schemas[2].Name)

	require.NoError(t, ExcludeRealm(r, []string{"*.t1.*"}))
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Len(t, r.Schemas[1].Tables, 2)
	require.Len(t, r.Schemas[2].Tables, 3)

	require.NoError(t, ExcludeRealm(r, []string{"*.t1"}))
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Tables)
	require.Len(t, r.Schemas[1].Tables, 1)
	require.Len(t, r.Schemas[2].Tables, 2)

	require.NoError(t, ExcludeRealm(r, []string{"s[12].t2"}))
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Tables)
	require.Empty(t, r.Schemas[1].Tables)
	require.Len(t, r.Schemas[2].Tables, 2)

	require.NoError(t, ExcludeRealm(r, []string{"*.t[23].*"}))
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[2].Tables, 2)
	require.NoError(t, ExcludeRealm(r, []string{"*.t[23]"}))
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[2].Tables)
}

func TestExcludeRealm_Columns(t *testing.T) {
	r := schema.NewRealm(
		schema.New("s1").AddTables(
			func() *schema.Table {
				t := schema.NewTable("t1").AddColumns(schema.NewColumn("c1"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
		),
		schema.New("s2").AddTables(
			func() *schema.Table {
				t := schema.NewTable("t1").AddColumns(schema.NewColumn("c1"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
			func() *schema.Table {
				t := schema.NewTable("t2").AddColumns(schema.NewColumn("c1"), schema.NewColumn("c2"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]), schema.NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
		),
		schema.New("s3").AddTables(
			func() *schema.Table {
				t := schema.NewTable("t1").AddColumns(schema.NewColumn("c1"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]))
				return t
			}(),
			func() *schema.Table {
				t := schema.NewTable("t2").AddColumns(schema.NewColumn("c1"), schema.NewColumn("c2"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]), schema.NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
			func() *schema.Table {
				t := schema.NewTable("t3").AddColumns(schema.NewColumn("c1"), schema.NewColumn("c2"), schema.NewColumn("c3"))
				t.AddIndexes(schema.NewIndex("i1").AddColumns(t.Columns[0]), schema.NewIndex("i2").AddColumns(t.Columns[1]))
				return t
			}(),
		),
	)
	require.NoError(t, ExcludeRealm(r, []string{"s[23].t[23].c1"}))
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

	require.NoError(t, ExcludeRealm(r, []string{"s[23].t*.*"}))
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

	require.NoError(t, ExcludeRealm(r, []string{"*.*.*"}))
	require.Len(t, r.Schemas, 3)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Empty(t, r.Schemas[0].Tables[0].Columns)
	require.Empty(t, r.Schemas[0].Tables[0].Indexes)
}

func TestExcludeSchema(t *testing.T) {
	r := schema.NewRealm(
		schema.New("s1").AddTables(
			schema.NewTable("t1"),
			schema.NewTable("t2"),
		),
	)
	require.NoError(t, ExcludeSchema(r.Schemas[0], []string{"t2"}))
	require.Len(t, r.Schemas, 1)
	require.Len(t, r.Schemas[0].Tables, 1)
}
