// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

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
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users"},
		},
		{
			name: "add attrs",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&WithoutRowID{}, &Strict{}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &WithoutRowID{},
				},
				&schema.AddAttr{
					A: &Strict{},
				},
			},
		},
		{
			name: "drop attrs",
			from: &schema.Table{Name: "t1", Attrs: []schema.Attr{&WithoutRowID{}, &Strict{}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.DropAttr{
					A: &WithoutRowID{},
				},
				&schema.DropAttr{
					A: &Strict{},
				},
			},
		},
		{
			name: "add check",
			from: &schema.Table{Name: "t1"},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "t1_c1_check", Expr: "(c1 > 1)"}}},
			wantChanges: []schema.Change{
				&schema.AddCheck{
					C: &schema.Check{Name: "t1_c1_check", Expr: "(c1 > 1)"},
				},
			},
		},
		{
			name: "drop check",
			from: &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "t1_c1_check", Expr: "(c1 > 1)"}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.DropCheck{
					C: &schema.Check{Name: "t1_c1_check", Expr: "(c1 > 1)"},
				},
			},
		},
		{
			name: "find check by expr",
			from: &schema.Table{
				Name: "t1",
				Attrs: []schema.Attr{
					&schema.Check{Name: "t1_c1_check", Expr: "(c1 > 1)"},
					&schema.Check{Expr: "(d1 > 1)"},
				},
			},
			to: &schema.Table{
				Name: "t1",
				Attrs: []schema.Attr{
					&schema.Check{Expr: "(c1 > 1)"},
					&schema.Check{Name: "add_name_to_check", Expr: "(d1 > 1)"},
				},
			},
		},
		func() testcase {
			var (
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "int8", Type: &schema.IntegerType{T: "int8"}}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Columns: []*schema.Column{
						{
							Name:    "c1",
							Type:    &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}, Null: true},
							Default: &schema.RawExpr{X: "{}"},
						},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			return testcase{
				name: "columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeNull | schema.ChangeDefault,
					},
					&schema.DropColumn{C: from.Columns[1]},
					&schema.AddColumn{C: to.Columns[1]},
				},
			}
		}(),
		func() testcase {
			var (
				s    = schema.New("public")
				from = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int"),
						schema.NewIntColumn("c2", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						schema.NewIntColumn("c3", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
						schema.NewIntColumn("c4", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "VIRTUAL"}),
					)
				to = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						// Add generated expression.
						schema.NewIntColumn("c1", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						// Drop generated expression.
						schema.NewIntColumn("c2", "int"),
						// Modify generated expression.
						schema.NewIntColumn("c3", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "2"}),
						// No change.
						schema.NewIntColumn("c4", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
					)
			)
			return testcase{
				name: "modify column generated",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{From: from.Columns[0], To: to.Columns[0], Change: schema.ChangeGenerated},
					&schema.ModifyColumn{From: from.Columns[1], To: to.Columns[1], Change: schema.ChangeGenerated},
					&schema.ModifyColumn{From: from.Columns[2], To: to.Columns[2], Change: schema.ChangeGenerated},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "int8", Type: &schema.IntegerType{T: "int8"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "int8", Type: &schema.IntegerType{T: "int8"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "c1_index", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c2_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c3_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "(c4 <> NULL)"}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "c1_index", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c3_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c3 <> NULL"}}},
				{Name: "c3_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, Desc: true, C: to.Columns[1]}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c4 <> NULL"}}},
			}
			return testcase{
				name: "indexes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
					&schema.DropIndex{I: from.Indexes[1]},
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[3], To: to.Indexes[3], Change: schema.ChangeParts},
					&schema.AddIndex{I: to.Indexes[1]},
				},
			}
		}(),
		func() testcase {
			from := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewStringColumn("id", "varchar"), schema.NewBoolColumn("active", "bool"))
			from.SetPrimaryKey(schema.NewPrimaryKey(from.Columns...))
			to := schema.NewTable("t1").
				SetSchema(schema.New("test")).
				AddColumns(schema.NewStringColumn("id", "varchar"), schema.NewBoolColumn("active", "bool"))
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
				ref = &schema.Table{
					Name: "t2",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						{Name: "ref_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
				to = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			from.ForeignKeys = []*schema.ForeignKey{
				{Symbol: "fk1", Table: to, Columns: to.Columns, RefTable: ref, RefColumns: ref.Columns[1:]},
				{Symbol: "1", Table: from, Columns: from.Columns, RefTable: ref, RefColumns: ref.Columns[:1]},
			}
			to.ForeignKeys = []*schema.ForeignKey{
				{Symbol: "fk1", Table: to, Columns: to.Columns, RefTable: ref, RefColumns: ref.Columns[:1]},
				// The below "constraint" is identical to "0" above, therefore, the differ does not report a change.
				{Symbol: "constraint", Table: from, Columns: from.Columns, RefTable: ref, RefColumns: ref.Columns[:1]},
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
		mock{m}.systemVars("3.36.0")
		drv, err := Open(db)
		require.NoError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			changes, err := drv.TableDiff(tt.from, tt.to)
			require.Equal(t, tt.wantErr, err != nil, err)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_SchemaDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.systemVars("3.36.0")
	drv, err := Open(db)
	require.NoError(t, err)
	from := &schema.Schema{
		Tables: []*schema.Table{
			{Name: "users"},
			{Name: "pets"},
		},
	}
	to := &schema.Schema{
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
				},
			},
			{Name: "groups"},
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
		schema.New("main").
			AddTables(
				schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
			),
		schema.New("main"),
	)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.IsType(t, &schema.DropTable{}, changes[0])
}
