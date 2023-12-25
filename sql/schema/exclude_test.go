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

func TestExcludeRealm_Enums(t *testing.T) {
	e1, e2, e3 := &EnumType{T: "e1"}, &EnumType{T: "e2"}, &EnumType{T: "e3"}
	s1, s2, s3 := New("s1").AddObjects(e1), New("s2").AddObjects(e1, e2), New("s3").AddObjects(e1, e2, e3)

	r, err := ExcludeRealm(NewRealm(s1, s2, s3), []string{"s1.e1"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Objects)
	require.Len(t, r.Schemas[1].Objects, 2)
	require.Len(t, r.Schemas[2].Objects, 3)

	// Wrong selector.
	r, err = ExcludeRealm(NewRealm(s1, s2, s3), []string{"*.e1[type=table]"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Objects)
	require.Len(t, r.Schemas[1].Objects, 2)
	require.Len(t, r.Schemas[2].Objects, 3)

	// Enum selector.
	r, err = ExcludeRealm(NewRealm(s1, s2, s3), []string{"*.e1[type=enum]"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Objects)
	require.Len(t, r.Schemas[1].Objects, 1)
	require.Equal(t, []Object{e2}, r.Schemas[1].Objects)
	require.Len(t, r.Schemas[2].Objects, 2)
	require.Equal(t, []Object{e2, e3}, r.Schemas[2].Objects)

	// Exclude all.
	r, err = ExcludeRealm(NewRealm(s1, s2, s3), []string{"*.*"})
	require.NoError(t, err)
	require.Len(t, r.Schemas, 3)
	require.Empty(t, r.Schemas[0].Objects)
	require.Empty(t, r.Schemas[1].Objects)
	require.Empty(t, r.Schemas[2].Objects)
}

func TestExcludeSchema_Enums(t *testing.T) {
	e1, e2, e3 := &EnumType{T: "e1"}, &EnumType{T: "e2"}, &EnumType{T: "e3"}
	r := NewRealm(New("s1").AddObjects(e1, e2, e3))

	s, err := ExcludeSchema(r.Schemas[0], []string{"e1"})
	require.NoError(t, err)
	require.Len(t, s.Objects, 2)
	require.Equal(t, []Object{e2, e3}, s.Objects)

	s, err = ExcludeSchema(r.Schemas[0], []string{"e1", "e2"})
	require.NoError(t, err)
	require.Len(t, s.Objects, 1)
	require.Equal(t, []Object{e3}, s.Objects)

	// Wrong selector.
	s, err = ExcludeSchema(r.Schemas[0], []string{"e*[type=view]"})
	require.NoError(t, err)
	require.Len(t, s.Objects, 1)

	// Enum selector.
	s, err = ExcludeSchema(r.Schemas[0], []string{"*[type=enum]"})
	require.NoError(t, err)
	require.Empty(t, s.Objects)
}

func TestExcludeRealm_Selector(t *testing.T) {
	r := NewRealm(
		New("s1").
			AddTables(
				NewTable("t1"),
			).
			AddViews(
				NewView("v1", "SELECT * FROM t1"),
			).
			AddFuncs(
				&Func{Name: "f1"},
				&Func{Name: "f2"},
				&Func{Name: "f3"},
			).
			AddProcs(
				&Proc{Name: "p1"},
				&Proc{Name: "p2"},
				&Proc{Name: "p3"},
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
		NewTable("t1"),
		NewTable("t2"),
		NewTable("t3"),
	)
	r, err = ExcludeRealm(r, []string{"*.*[type=view]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Views)
	require.Len(t, r.Schemas[0].Tables, 3)

	r, err = ExcludeRealm(r, []string{"*.t[12][type=table]"})
	require.Empty(t, r.Schemas[0].Views)
	require.Len(t, r.Schemas[0].Tables, 1)

	r.Schemas[0].AddViews(
		NewView("v1", "SELECT * FROM t1"),
	)
	r, err = ExcludeRealm(r, []string{"*.*[type=view|table]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Views)
	require.Empty(t, r.Schemas[0].Tables)

	r.Schemas[0].
		AddTables(
			NewTable("t1").AddColumns(NewColumn("c1")).AddIndexes(NewIndex("i1")).AddChecks(NewCheck().SetName("k1")),
		).
		AddViews(
			NewView("v1", "SELECT * FROM t1").AddColumns(NewColumn("c1")),
		)
	r.Schemas[0].Views[0].Triggers = []*Trigger{{Name: "g1"}}
	r.Schemas[0].Tables[0].Triggers = []*Trigger{{Name: "g1"}}
	r, err = ExcludeRealm(r, []string{"*.*[type=table].*1[type=column|check]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables[0].Attrs)
	require.Empty(t, r.Schemas[0].Tables[0].Columns)
	require.Len(t, r.Schemas[0].Tables[0].Indexes, 1)
	require.Len(t, r.Schemas[0].Tables[0].Triggers, 1)
	require.Len(t, r.Schemas[0].Views[0].Columns, 1)
	require.Len(t, r.Schemas[0].Views[0].Triggers, 1)

	r, err = ExcludeRealm(r, []string{"*.*[type=table|view].*1[type=column|check|index|trigger]"})
	require.NoError(t, err)
	require.Empty(t, r.Schemas[0].Tables[0].Indexes)
	require.Empty(t, r.Schemas[0].Views[0].Columns)
	require.Empty(t, r.Schemas[0].Tables[0].Triggers)
	require.Empty(t, r.Schemas[0].Views[0].Triggers)
}

func TestExcludeRealm_Checks(t *testing.T) {
	r := NewRealm(
		New("s1").AddTables(
			NewTable("t1").AddChecks(NewCheck().SetName("c1")),
		),
	)
	r, err := ExcludeRealm(r, []string{"s1.t1.c1"})
	require.NoError(t, err)
	require.Len(t, r.Schemas[0].Tables[0].Attrs, 0)
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
