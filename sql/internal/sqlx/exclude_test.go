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

	r = schema.NewRealm(schema.New("s1"), schema.New("s2"), schema.New("s3"))
	r, err = ExcludeRealm(r, []string{"s*.*", "s*.*.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	r, err = ExcludeRealm(r, []string{"s*"})
	require.NoError(t, err)
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

func TestExcludeRealm_Selector(t *testing.T) {
	r := schema.NewRealm(
		schema.New("s1").
			AddTables(
				schema.NewTable("t1"),
			).
			AddViews(
				schema.NewView("v1", "SELECT * FROM t1"),
			).
			AddFuncs(
				&schema.Func{Name: "f1"},
				&schema.Func{Name: "f2"},
				&schema.Func{Name: "f3"},
			).
			AddProcs(
				&schema.Proc{Name: "p1"},
				&schema.Proc{Name: "p2"},
				&schema.Proc{Name: "p3"},
			),
	)
	r, err := ExcludeRealm(r, []string{"*.*[type=table]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables)
	require.Len(t, r.Schemas[0].Views, 1)

	r, err = ExcludeRealm(r, []string{"*.f1[type=function]"})
	require.NoError(t, err)
	require.Len(t, r.Schemas[0].Funcs, 2)
	require.Equal(t, []string{"f2", "f3"}, []string{r.Schemas[0].Funcs[0].Name, r.Schemas[0].Funcs[1].Name})

	r, err = ExcludeRealm(r, []string{"*.p2[type=procedure]"})
	require.NoError(t, err)
	require.Len(t, r.Schemas[0].Procs, 2)
	require.Equal(t, []string{"p1", "p3"}, []string{r.Schemas[0].Procs[0].Name, r.Schemas[0].Procs[1].Name})

	r, err = ExcludeRealm(r, []string{"*.*[type=procedure|function]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Funcs)
	require.Empty(t, r.Schemas[0].Procs)

	r.Schemas[0].AddTables(
		schema.NewTable("t1"),
		schema.NewTable("t2"),
		schema.NewTable("t3"),
	)
	r, err = ExcludeRealm(r, []string{"*.*[type=view]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Views)
	require.Len(t, r.Schemas[0].Tables, 3)

	r, err = ExcludeRealm(r, []string{"*.t[12][type=table]"})
	require.Empty(t, r.Schemas[0].Views)
	require.Len(t, r.Schemas[0].Tables, 1)

	r.Schemas[0].AddViews(
		schema.NewView("v1", "SELECT * FROM t1"),
	)
	r, err = ExcludeRealm(r, []string{"*.*[type=view|table]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Views)
	require.Empty(t, r.Schemas[0].Tables)

	r.Schemas[0].
		AddTables(
			schema.NewTable("t1").AddColumns(schema.NewColumn("c1")).AddIndexes(schema.NewIndex("i1")).AddChecks(schema.NewCheck().SetName("k1")),
		).
		AddViews(
			schema.NewView("v1", "SELECT * FROM t1").AddColumns(schema.NewColumn("c1")),
		)
	r, err = ExcludeRealm(r, []string{"*.*[type=table].*1[type=column|check]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables[0].Attrs)
	require.Empty(t, r.Schemas[0].Tables[0].Columns)
	require.Len(t, r.Schemas[0].Tables[0].Indexes, 1)
	require.Len(t, r.Schemas[0].Views[0].Columns, 1)

	r, err = ExcludeRealm(r, []string{"*.*[type=table|view].*1[type=column|check|index]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables[0].Indexes)
	require.Empty(t, r.Schemas[0].Views[0].Columns)
}

func TestExcludeRealm_Checks(t *testing.T) {
	r := schema.NewRealm(
		schema.New("s1").AddTables(
			schema.NewTable("t1").AddChecks(schema.NewCheck().SetName("c1")),
		),
	)
	r, err := ExcludeRealm(r, []string{"s1.t1.c1"})
	require.NoError(t, err)
	require.Len(t, r.Schemas[0].Tables[0].Attrs, 0)
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
	r := schema.NewRealm(
		schema.New("s1").AddTables(
			schema.NewTable("t1"),
			schema.NewTable("t2"),
		),
	)
	_, err := ExcludeSchema(r.Schemas[0], []string{"t2"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 1)
	require.Len(t, r.Schemas[0].Tables, 1)
}
