// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDiff_TableDiff(t *testing.T) {
	type testcase struct {
		name        string
		from, to    *schema.Table
		wantChanges []schema.Change
		wantErr     bool
	}
	tests := []testcase{
		{
			name: "no changes",
			from: schema.NewTable("users").SetSchema(&schema.Schema{Name: "dbo"}),
			to:   schema.NewTable("users"),
		},
		func() testcase {
			from := func() *schema.Table {
				t := schema.NewTable("users").
					SetSchema(&schema.Schema{Name: "dbo"}).
					AddColumns(schema.NewIntColumn("id", "int"))
				t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
				return t
			}()
			to := func() *schema.Table {
				t := schema.NewTable("users").
					SetSchema(&schema.Schema{Name: "dbo"}).
					AddColumns(schema.NewIntColumn("id", "int").
						AddAttrs(&Identity{Seed: 1024, Increment: 1}),
					)
				t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
				return t
			}()
			return testcase{
				name: "change identity attributes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeAttr,
					},
				},
			}
		}(),
		{
			name: "add check",
			from: schema.NewTable("t1").SetSchema(&schema.Schema{Name: "dbo"}),
			to: schema.NewTable("t1").AddChecks(schema.NewCheck().
				SetName("t1_c1_check").
				SetExpr("(c1 > 1)")),
			wantChanges: []schema.Change{
				&schema.AddCheck{
					C: schema.NewCheck().SetName("t1_c1_check").SetExpr("(c1 > 1)"),
				},
			},
		},
		{
			name: "drop check",
			from: schema.NewTable("t1").
				SetSchema(&schema.Schema{Name: "dbo"}).
				AddChecks(schema.NewCheck().
					SetName("t1_c1_check").
					SetExpr("(c1 > 1)")),
			to: schema.NewTable("t1"),
			wantChanges: []schema.Change{
				&schema.DropCheck{
					C: schema.NewCheck().SetName("t1_c1_check").SetExpr("(c1 > 1)"),
				},
			},
		},
		{
			name: "add comment",
			from: schema.NewTable("t1").SetSchema(&schema.Schema{Name: "dbo"}),
			to:   schema.NewTable("t1").AddAttrs(&schema.Comment{Text: "t1"}),
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Comment{Text: "t1"},
				},
			},
		},
		{
			name: "drop comment",
			from: schema.NewTable("t1").
				SetSchema(&schema.Schema{Name: "dbo"}).
				AddAttrs(&schema.Comment{Text: "t1"}),
			to: schema.NewTable("t1"),
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: ""},
				},
			},
		},
		{
			name: "modify comment",
			from: schema.NewTable("t1").
				SetSchema(&schema.Schema{Name: "dbo"}).
				AddAttrs(&schema.Comment{Text: "t1"}),
			to: schema.NewTable("t1").
				AddAttrs(&schema.Comment{Text: "t1!"}),
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: "t1!"},
				},
			},
		},
		func() testcase {
			var (
				s    = schema.New("dbo")
				from = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: computedPersisted}),
					)
				to = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int"),
					)
			)
			return testcase{
				name:    "drop generation expression",
				from:    from,
				to:      to,
				wantErr: true,
			}
		}(),
		{
			name: "change generation expression",
			from: schema.NewTable("t1").
				SetSchema(schema.New("dbo")).
				AddColumns(
					schema.NewIntColumn("c1", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: computedPersisted}),
				),
			to: schema.NewTable("t1").
				SetSchema(schema.New("dbo")).
				AddColumns(
					schema.NewIntColumn("c1", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "2", Type: computedPersisted}),
				),
			wantErr: true,
		},
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("dbo")).
					AddColumns(
						schema.NewStringColumn("c1", "varchar", schema.StringSize(7)),
						schema.NewIntColumn("c2", "int"),
					)
				to = schema.NewTable("t1").
					AddColumns(
						schema.NewNullStringColumn("c1", "varchar", schema.StringSize(8)).
							SetDefault(&schema.Literal{V: "'ARIGA'"}).
							AddAttrs(&schema.Comment{Text: "Ariga comment"}),
						schema.NewIntColumn("c3", "int"),
					)
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeNull | schema.ChangeComment | schema.ChangeDefault | schema.ChangeType,
					},
					&schema.DropColumn{C: from.Columns[1]},
					&schema.AddColumn{C: to.Columns[1]},
				},
			}
		}(),
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("dbo")).
					AddColumns(
						schema.NewStringColumn("c1", "varchar", schema.StringSize(7)).
							SetDefault(&schema.Literal{V: "'ARIGA'"}),
						schema.NewIntColumn("c2", "int"),
						schema.NewIntColumn("c3", "int"),
					)
				to = schema.NewTable("t1").
					AddColumns(
						schema.NewStringColumn("c1", "varchar", schema.StringSize(7)).
							SetDefault(&schema.Literal{V: "'ARIGA'"}),
						schema.NewIntColumn("c2", "int"),
						schema.NewIntColumn("c3", "int"),
					)
			)
			from.Indexes = []*schema.Index{
				{Name: "c1_index", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c2_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "(c4 <> NULL)"}}},
				{Name: "c5_include_no_change", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_added", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c5_include_dropped", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "c1_index", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c3_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c3 <> NULL"}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c4 <> NULL"}}},
				{Name: "c5_include_no_change", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_added", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_dropped", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
			}
			return testcase{
				name: "indexes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
					&schema.DropIndex{I: from.Indexes[1]},
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[5], To: to.Indexes[5], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[6], To: to.Indexes[6], Change: schema.ChangeAttr},
					&schema.AddIndex{I: to.Indexes[1]},
				},
			}
		}(),
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("test")).
					AddColumns(
						schema.NewStringColumn("id", "varchar"),
						schema.NewColumn("active").SetType(&BitType{T: "bit"}),
					)
				to = schema.NewTable("t1").
					SetSchema(schema.New("test")).
					AddColumns(
						schema.NewStringColumn("id", "varchar"),
						schema.NewColumn("active").SetType(&BitType{T: "bit"}),
					)
			)
			from.SetPrimaryKey(schema.NewPrimaryKey(from.Columns...))
			to.SetPrimaryKey(
				schema.NewPrimaryKey(from.Columns...).
					AddAttrs(&IndexPredicate{P: "active"}),
			)
			return testcase{
				name: "modify primary-key",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyPrimaryKey{
						From:   from.PrimaryKey,
						To:     to.PrimaryKey,
						Change: schema.ChangeAttr,
					},
				},
			}
		}(),
		func() testcase {
			var (
				s   = &schema.Schema{Name: "dbo"}
				ref = schema.NewTable("t2").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("id", "int"),
						schema.NewIntColumn("ref_id", "int"),
					)
				from = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(schema.NewIntColumn("t2_id", "int"))
				to = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(schema.NewIntColumn("t2_id", "int"))
			)
			from.ForeignKeys = []*schema.ForeignKey{
				{Table: from, Columns: from.Columns, RefTable: ref, RefColumns: ref.Columns[:1]},
			}
			to.ForeignKeys = []*schema.ForeignKey{
				{Table: to, Columns: to.Columns, RefTable: ref, RefColumns: ref.Columns[1:]},
			}
			return testcase{
				name: "foreign-keys",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyForeignKey{
						From:   from.ForeignKeys[0],
						To:     to.ForeignKeys[0],
						Change: schema.ChangeRefColumn,
					},
				},
			}
		}(),
	}
	for _, tt := range tests {
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		mock{m}.version("16.0.4035.4")
		drv, err := Open(db)
		require.NoError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			changes, err := drv.TableDiff(tt.from, tt.to)
			require.Equalf(t, tt.wantErr, err != nil, "got: %v", err)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_SchemaDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("16.0.4035.4")
	drv, err := Open(db)
	require.NoError(t, err)
	from := &schema.Schema{
		Tables: []*schema.Table{
			schema.NewTable("users"),
			schema.NewTable("pets"),
		},
	}
	to := &schema.Schema{
		Tables: []*schema.Table{
			schema.NewTable("users").AddColumns(schema.NewIntColumn("t2_id", "int")),
			schema.NewTable("groups"),
		},
	}
	from.Tables[0].Schema = from
	from.Tables[1].Schema = from
	changes, err := drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifyTable{T: to.Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Tables[0].Columns[0]}}},
		&schema.DropTable{T: from.Tables[1]},
		&schema.AddTable{T: to.Tables[1]},
	}, changes)
}

func TestDefaultDiff(t *testing.T) {
	changes, err := DefaultDiff.SchemaDiff(
		schema.New("dbo").
			AddTables(
				schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
			),
		schema.New("dbo"),
	)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.IsType(t, &schema.DropTable{}, changes[0])
}
