// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"testing"

	"ariga.io/atlas/schemahcl"
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
			name: "change identity attributes",
			from: func() *schema.Table {
				t := &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
					},
				}
				t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
				return t
			}(),
			to: func() *schema.Table {
				t := &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024, Increment: 1}}}},
					},
				}
				t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
				return t
			}(),
			wantChanges: []schema.Change{
				&schema.ModifyColumn{
					From:   &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
					To:     &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024, Increment: 1}}}},
					Change: schema.ChangeAttr,
				},
			},
		},
		{
			name: "drop partition key",
			from: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeRange,
					Parts: []*PartitionPart{{C: schema.NewColumn("c")}},
				}),
			to:      schema.NewTable("logs"),
			wantErr: true,
		},
		{
			name: "add partition key",
			from: schema.NewTable("logs"),
			to: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeRange,
					Parts: []*PartitionPart{{C: schema.NewColumn("c")}},
				}),
			wantErr: true,
		},
		{
			name: "change partition key column",
			from: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeRange,
					Parts: []*PartitionPart{{C: schema.NewColumn("c")}},
				}),
			to: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeRange,
					Parts: []*PartitionPart{{C: schema.NewColumn("d")}},
				}),
			wantErr: true,
		},
		{
			name: "change partition key type",
			from: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeRange,
					Parts: []*PartitionPart{{C: schema.NewColumn("c")}},
				}),
			to: schema.NewTable("logs").
				AddAttrs(&Partition{
					T:     PartitionTypeHash,
					Parts: []*PartitionPart{{C: schema.NewColumn("c")}},
				}),
			wantErr: true,
		},
		{
			name: "add check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
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
			name: "add comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Comment{Text: "t1"},
				},
			},
		},
		{
			name: "drop comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: ""},
				},
			},
		},
		{
			name: "modify comment",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Comment{Text: "t1"}}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Comment{Text: "t1!"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "t1"},
					To:   &schema.Comment{Text: "t1!"},
				},
			},
		},
		func() testcase {
			var (
				s    = schema.New("public")
				from = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int").
							SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
					)
				to = schema.NewTable("t1").
					SetSchema(s).
					AddColumns(
						schema.NewIntColumn("c1", "int"),
					)
			)
			return testcase{
				name: "drop generation expression",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{From: from.Columns[0], To: to.Columns[0], Change: schema.ChangeGenerated},
				},
			}
		}(),
		{
			name: "change generation expression",
			from: schema.NewTable("t1").
				SetSchema(schema.New("public")).
				AddColumns(
					schema.NewIntColumn("c1", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
				),
			to: schema.NewTable("t1").
				SetSchema(schema.New("public")).
				AddColumns(
					schema.NewIntColumn("c1", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "2", Type: "STORED"}),
				),
			wantErr: true,
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
							Attrs:   []schema.Attr{&schema.Comment{Text: "json comment"}},
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
						Change: schema.ChangeNull | schema.ChangeComment | schema.ChangeDefault,
					},
					&schema.DropColumn{C: from.Columns[1]},
					&schema.AddColumn{C: to.Columns[1]},
				},
			}
		}(),
		// Modify enum type or values.
		func() testcase {
			var (
				from = schema.NewTable("users").
					SetSchema(schema.New("public")).
					AddColumns(
						schema.NewEnumColumn("enum1", schema.EnumName("enum1"), schema.EnumValues("a")),
						schema.NewEnumColumn("enum3", schema.EnumName("enum3"), schema.EnumValues("a")),
						schema.NewEnumColumn("enum4", schema.EnumName("enum4"), schema.EnumValues("a"), schema.EnumSchema(schema.New("public"))),
					)
				to = schema.NewTable("users").
					SetSchema(schema.New("public")).
					AddColumns(
						// Change type.
						schema.NewEnumColumn("enum1", schema.EnumName("enum2"), schema.EnumValues("a")),
						// No change as schema is optional.
						schema.NewEnumColumn("enum3", schema.EnumName("enum3"), schema.EnumValues("a"), schema.EnumSchema(schema.New("public"))),
						// Enum type was changed (reside in a different schema).
						schema.NewEnumColumn("enum4", schema.EnumName("enum4"), schema.EnumValues("a"), schema.EnumSchema(schema.New("test"))),
					)
			)
			return testcase{
				name: "enums",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{From: from.Columns[0], To: to.Columns[0], Change: schema.ChangeType},
					&schema.ModifyColumn{From: from.Columns[2], To: to.Columns[2], Change: schema.ChangeType},
				},
			}
		}(),
		// Modify array of type enum.
		func() testcase {
			var (
				from = schema.NewTable("users").
					SetSchema(schema.New("public")).
					AddColumns(
						schema.NewColumn("a1").SetType(&ArrayType{T: "state[]", Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}),
						schema.NewColumn("a3").SetType(&ArrayType{T: "state[]", Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}),
					)
				to = schema.NewTable("users").
					SetSchema(schema.New("public")).
					AddColumns(
						schema.NewColumn("a1").SetType(&ArrayType{T: "status[]", Type: &schema.EnumType{T: "status", Values: []string{"on", "off"}}}),
						schema.NewColumn("a3").SetType(&ArrayType{T: "state[]", Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}),
					)
			)
			return testcase{
				name: "enum arrays",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{From: from.Columns[0], To: to.Columns[0], Change: schema.ChangeType},
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
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}, Default: &schema.RawExpr{X: "'{}'"}},
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
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}, Default: &schema.RawExpr{X: "'{}'::json"}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "int8", Type: &schema.IntegerType{T: "int8"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "c1_index", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c2_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "(c4 <> NULL)"}}},
				{Name: "c4_storage_params", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexStorageParams{PagesPerRange: 4}}},
				{Name: "c5_include_no_change", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_added", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c5_include_dropped", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c6_nulls_not_distinct", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexNullsDistinct{V: true}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "c1_index", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c3_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c3_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c3 <> NULL"}}},
				{Name: "c4_predicate", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexPredicate{P: "c4 <> NULL"}}},
				{Name: "c4_storage_params", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexStorageParams{PagesPerRange: 2}}},
				{Name: "c5_include_no_change", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_added", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexInclude{Columns: from.Columns[:1]}}},
				{Name: "c5_include_dropped", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c6_nulls_not_distinct", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}, Attrs: []schema.Attr{&IndexNullsDistinct{V: false}}},
			}
			return testcase{
				name: "indexes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
					&schema.DropIndex{I: from.Indexes[1]},
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[4], To: to.Indexes[4], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[6], To: to.Indexes[6], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[7], To: to.Indexes[7], Change: schema.ChangeAttr},
					&schema.ModifyIndex{From: from.Indexes[8], To: to.Indexes[8], Change: schema.ChangeAttr},
					&schema.AddIndex{I: to.Indexes[1]},
				},
			}
		}(),
		func() testcase {
			var (
				from = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					AddColumns(schema.NewIntColumn("c1", "int8"))
				to = schema.NewTable("t1").
					SetSchema(schema.New("public")).
					AddColumns(schema.NewIntColumn("c1", "int8"))
			)
			from.Indexes = []*schema.Index{
				schema.NewIndex("idx1").AddParts(schema.NewColumnPart(from.Columns[0])),
				schema.NewIndex("idx2").AddParts(schema.NewColumnPart(from.Columns[0])),
				schema.NewIndex("idx3").AddParts(schema.NewColumnPart(from.Columns[0])),
				schema.NewIndex("idx4").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_ops"})),
				schema.NewIndex("idx5").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_ops"})),
			}
			to.Indexes = []*schema.Index{
				// A default operator class was added.
				schema.NewIndex("idx1").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_ops"})),
				// Unrecognized operator class with explicit default.
				schema.NewIndex("idx2").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_custom", Default: true})),
				// A default operator class but with custom parameters.
				schema.NewIndex("idx3").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_ops", Params: []struct{ N, V string }{{"signlen", "1"}}})),
				// A default operator class was dropped.
				schema.NewIndex("idx4").AddParts(schema.NewColumnPart(from.Columns[0])),
				// Equal operators.
				schema.NewIndex("idx5").AddParts(schema.NewColumnPart(to.Columns[0]).AddAttrs(&IndexOpClass{Name: "int8_ops"})),
			}
			return testcase{
				name: "operator class",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeParts},
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
		mock{m}.version("130000")
		drv, err := Open(db)
		require.NoError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			changes, err := drv.TableDiff(tt.from, tt.to)
			require.Equalf(t, tt.wantErr, err != nil, "got: %v", err)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_RealmDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("130000")
	drv, err := Open(db)
	to := schema.New("public").
		AddTables(
			schema.NewTable("users").AddColumns(schema.NewIntColumn("t2_id", "int")),
		).
		AddObjects(
			&schema.EnumType{T: "e", Values: []string{"b"}},
		)
	changes, err := drv.RealmDiff(schema.NewRealm(), schema.NewRealm(to))
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.AddSchema{S: to},
		&schema.AddObject{O: to.Objects[0]},
		&schema.AddTable{T: to.Tables[0]},
	}, changes)
}

func TestDiff_SchemaDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)
	from := schema.New("public").
		AddTables(schema.NewTable("users"), schema.NewTable("pets")).
		AddObjects(&schema.EnumType{T: "dropped"}, &schema.EnumType{T: "modified", Values: []string{"a"}}, &schema.EnumType{T: "unchanged"})
	to := schema.New("public").AddTables(schema.NewTable("users").AddColumns(schema.NewIntColumn("t2_id", "int")), schema.NewTable("groups")).
		AddObjects(&schema.EnumType{T: "modified", Values: []string{"b"}}, &schema.EnumType{T: "unchanged"}, &schema.EnumType{T: "added"})
	changes, err := drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.DropObject{O: from.Objects[0]},
		&schema.ModifyObject{From: from.Objects[1], To: to.Objects[0]},
		&schema.AddObject{O: to.Objects[2]},
		&schema.ModifyTable{T: to.Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Tables[0].Columns[0]}}},
		&schema.DropTable{T: from.Tables[1]},
		&schema.AddTable{T: to.Tables[1]},
	}, changes)

	// Add comment.
	from, to = schema.New("public"), schema.New("public").SetComment("comment")
	changes, err = drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to, Changes: schema.Changes{&schema.AddAttr{A: &schema.Comment{Text: "comment"}}}},
	}, changes)
	// Modify comment.
	from.SetComment("boring")
	changes, err = drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to, Changes: schema.Changes{
			&schema.ModifyAttr{
				From: &schema.Comment{Text: "boring"},
				To:   &schema.Comment{Text: "comment"},
			},
		}},
	}, changes)

	t.Run("DefaultComment", func(t *testing.T) {
		from, to := schema.New("public").SetComment("standard public schema"), schema.New("public")
		changes, err = drv.SchemaDiff(from, to)
		require.NoError(t, err)
		require.Empty(t, changes)

		changes, err = drv.SchemaDiff(to, from)
		require.NoError(t, err)
		require.Empty(t, changes)

		from.Name, to.Name = "", ""
		changes, err = drv.SchemaDiff(from, to)
		require.NoError(t, err)
		require.Empty(t, changes)

		from.Name, to.Name = "other", "other"
		changes, err = drv.SchemaDiff(from, to)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		require.EqualValues(t, []schema.Change{
			&schema.ModifySchema{S: to, Changes: schema.Changes{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "standard public schema"},
					To:   &schema.Comment{Text: ""},
				},
			}},
		}, changes)

		from.Name, to.Name = "public", "public"
		from.SetComment("custom public schema")
		changes, err = drv.SchemaDiff(from, to)
		require.NoError(t, err)
		require.EqualValues(t, []schema.Change{
			&schema.ModifySchema{S: to, Changes: schema.Changes{
				&schema.ModifyAttr{
					From: &schema.Comment{Text: "custom public schema"},
					To:   &schema.Comment{Text: ""},
				},
			}},
		}, changes)
	})
}

func TestDefaultDiff(t *testing.T) {
	changes, err := DefaultDiff.SchemaDiff(
		schema.New("public").
			AddTables(
				schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")),
			),
		schema.New("public"),
	)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.IsType(t, &schema.DropTable{}, changes[0])
}

func TestDiff_AnnotateChanges(t *testing.T) {
	var cfg struct {
		schemahcl.DefaultExtension
	}
	// language=hcl
	err := schemahcl.New().EvalBytes([]byte(`
concurrent_index {
  add  = true
  drop = true
}
`), &cfg, nil)
	require.NoError(t, err)
	from := schema.New("public").AddTables(schema.NewTable("users").AddIndexes(schema.NewIndex("users_pkey_old").AddColumns(schema.NewIntColumn("id", "int"))))
	to := schema.New("public").AddTables(schema.NewTable("users").AddIndexes(schema.NewIndex("users_pkey_new").AddColumns(schema.NewIntColumn("id", "int"))))
	changes, err := DefaultDiff.SchemaDiff(from, to, func(opts *schema.DiffOptions) { opts.Extra = cfg.DefaultExtension })
	require.NoError(t, err)
	require.Len(t, changes, 1)
	plan, err := DefaultPlan.PlanChanges(context.Background(), "changes", changes)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 2)
	require.Equal(t, `DROP INDEX CONCURRENTLY "public"."users_pkey_old"`, plan.Changes[0].Cmd)
	require.Equal(t, `CREATE INDEX CONCURRENTLY "users_pkey_old" ON "public"."users" ("id")`, plan.Changes[0].Reverse)
	require.Equal(t, `CREATE INDEX CONCURRENTLY "users_pkey_new" ON "public"."users" ("id")`, plan.Changes[1].Cmd)
	require.Equal(t, `DROP INDEX CONCURRENTLY "public"."users_pkey_new"`, plan.Changes[1].Reverse)
}
