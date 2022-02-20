// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

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
			name:    "mismatched names",
			from:    &schema.Table{Name: "users"},
			to:      &schema.Table{Name: "pets"},
			wantErr: true,
		},
		{
			name: "no changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users"},
		},
		{
			name: "no changes",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Columns: []*schema.Column{{Name: "enum", Default: &schema.RawExpr{X: "'A'"}, Type: &schema.ColumnType{Type: &schema.EnumType{Values: []string{"A"}}}}}},
			to:   &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "enum", Default: &schema.RawExpr{X: `"A"`}, Type: &schema.ColumnType{Type: &schema.EnumType{Values: []string{"A"}}}}}},
		},
		{
			name: "change primary key",
			from: func() *schema.Table {
				t := &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}}}}
				t.PrimaryKey = &schema.Index{
					Parts: []*schema.IndexPart{{C: t.Columns[0]}},
				}
				return t
			}(),
			to:      &schema.Table{Name: "users"},
			wantErr: true,
		},
		{
			name: "modify counter",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&AutoIncrement{V: 1}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&AutoIncrement{V: 100}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &AutoIncrement{V: 1},
					To:   &AutoIncrement{V: 100},
				},
			},
		},
		{
			name: "add collation",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Collation{V: "latin1"}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Collation{V: "latin1"},
				},
			},
		},
		{
			name: "drop collation means modify",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public", Attrs: []schema.Attr{&schema.Collation{V: "utf8mb4_0900_ai_ci"}}}, Attrs: []schema.Attr{&schema.Collation{V: "utf8mb4_bin"}}},
			to:   &schema.Table{Name: "users"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Collation{V: "utf8mb4_bin"},
					To:   &schema.Collation{V: "utf8mb4_0900_ai_ci"},
				},
			},
		},
		{
			name: "modify collation",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Collation{V: "utf8"}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Collation{V: "latin1"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Collation{V: "utf8"},
					To:   &schema.Collation{V: "latin1"},
				},
			},
		},
		{
			name: "add charset",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Charset{V: "hebrew"}}},
			wantChanges: []schema.Change{
				&schema.AddAttr{
					A: &schema.Charset{V: "hebrew"},
				},
			},
		},
		{
			name: "drop charset means modify",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public", Attrs: []schema.Attr{&schema.Charset{V: "hebrew"}}}, Attrs: []schema.Attr{&schema.Charset{V: "hebrew_bin"}}},
			to:   &schema.Table{Name: "users"},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Charset{V: "hebrew_bin"},
					To:   &schema.Charset{V: "hebrew"},
				},
			},
		},
		{
			name: "modify charset",
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Charset{V: "hebrew"}}},
			to:   &schema.Table{Name: "users", Attrs: []schema.Attr{&schema.Charset{V: "binary"}}},
			wantChanges: []schema.Change{
				&schema.ModifyAttr{
					From: &schema.Charset{V: "hebrew"},
					To:   &schema.Charset{V: "binary"},
				},
			},
		},
		{
			name: "add check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"}}},
			wantChanges: []schema.Change{
				&schema.AddCheck{
					C: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"},
				},
			},
		},
		{
			name: "drop check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"}}},
			to:   &schema.Table{Name: "t1"},
			wantChanges: []schema.Change{
				&schema.DropCheck{
					C: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')"},
				},
			},
		},
		{
			name: "modify check",
			from: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}, Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: false}}}}},
			to:   &schema.Table{Name: "t1", Attrs: []schema.Attr{&schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: true}}}}},
			wantChanges: []schema.Change{
				&schema.ModifyCheck{
					From: &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: false}}},
					To:   &schema.Check{Name: "users_chk1_c1", Expr: "(`c1` <>_latin1\\'foo\\')", Attrs: []schema.Attr{&Enforced{V: true}}},
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
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
						{Name: "c4", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float"}}, Default: &schema.Literal{V: "0.00"}},
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
						{Name: "c4", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float"}}, Default: &schema.Literal{V: "0.00"}},
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
		func() testcase {
			var (
				from = &schema.Table{
					Name: "t1",
					Schema: &schema.Schema{
						Name: "public",
					},
					Columns: []*schema.Column{
						{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
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
						{Name: "c2", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
						{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "c1_index", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c2_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
				{Name: "c1_prefix", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1], Attrs: []schema.Attr{&SubPart{Len: 50}}}}},
				{Name: "c1_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1]}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "c1_index", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
				{Name: "c3_unique", Unique: true, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[1]}}},
				{Name: "c1_prefix", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0], Attrs: []schema.Attr{&SubPart{Len: 100}}}}},
				{Name: "c1_desc", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[1], Desc: true}}},
			}
			return testcase{
				name: "indexes",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
					&schema.DropIndex{I: from.Indexes[1]},
					&schema.ModifyIndex{From: from.Indexes[2], To: to.Indexes[2], Change: schema.ChangeParts},
					&schema.ModifyIndex{From: from.Indexes[3], To: to.Indexes[3], Change: schema.ChangeParts},
					&schema.AddIndex{I: to.Indexes[1]},
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
		mock{m}.version("8.0.19")
		drv, err := Open(db)
		require.NoError(t, err)
		t.Run(tt.name, func(t *testing.T) {
			changes, err := drv.TableDiff(tt.from, tt.to)
			require.Equal(t, tt.wantErr, err != nil)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_UnsupportedChecks(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("5.6.35")
	drv, err := Open(db)
	require.NoError(t, err)
	s := schema.New("public")
	changes, err := drv.TableDiff(
		schema.NewTable("t").SetSchema(s),
		schema.NewTable("t").SetSchema(s).AddChecks(schema.NewCheck()),
	)
	require.Nil(t, changes)
	require.EqualError(t, err, `version "5.6.35" does not support CHECK constraints`)
}

func TestDiff_SchemaDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("8.0.19")
	drv, err := Open(db)
	require.NoError(t, err)

	changes, err := drv.SchemaDiff(&schema.Schema{Name: "public"}, &schema.Schema{Name: "test"})
	require.Error(t, err)
	require.Nil(t, changes)

	from := &schema.Schema{
		Realm: &schema.Realm{
			Attrs: []schema.Attr{
				&schema.Collation{V: "latin1"},
			},
		},
		Tables: []*schema.Table{
			{Name: "users"},
			{Name: "pets"},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "latin1"},
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
		Attrs: []schema.Attr{
			&schema.Collation{V: "utf8"},
		},
	}
	from.Tables[0].Schema = from
	from.Tables[1].Schema = from
	changes, err = drv.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to, Changes: []schema.Change{&schema.ModifyAttr{From: from.Attrs[0], To: to.Attrs[0]}}},
		&schema.ModifyTable{T: to.Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Tables[0].Columns[0]}}},
		&schema.DropTable{T: from.Tables[1]},
		&schema.AddTable{T: to.Tables[1]},
	}, changes)
}

func TestDiff_RealmDiff(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("8.0.19")
	drv, err := Open(db)
	require.NoError(t, err)
	from := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
				Tables: []*schema.Table{
					{Name: "users"},
					{Name: "pets"},
				},
				Attrs: []schema.Attr{
					&schema.Collation{V: "latin1"},
				},
			},
			{
				Name: "internal",
				Tables: []*schema.Table{
					{Name: "pets"},
				},
			},
		},
	}
	to := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
				Tables: []*schema.Table{
					{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "t2_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
						},
					},
					{Name: "pets"},
				},
				Attrs: []schema.Attr{
					&schema.Collation{V: "utf8"},
				},
			},
			{
				Name: "test",
				Tables: []*schema.Table{
					{Name: "pets"},
				},
			},
		},
	}
	from.Schemas[0].Realm = from
	from.Schemas[0].Tables[0].Schema = from.Schemas[0]
	from.Schemas[0].Tables[1].Schema = from.Schemas[0]
	to.Schemas[0].Realm = to
	to.Schemas[0].Tables[0].Schema = to.Schemas[0]
	changes, err := drv.RealmDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifySchema{S: to.Schemas[0], Changes: []schema.Change{&schema.ModifyAttr{From: from.Schemas[0].Attrs[0], To: to.Schemas[0].Attrs[0]}}},
		&schema.ModifyTable{T: to.Schemas[0].Tables[0], Changes: []schema.Change{&schema.AddColumn{C: to.Schemas[0].Tables[0].Columns[0]}}},
		&schema.DropSchema{S: from.Schemas[1]},
		&schema.AddSchema{S: to.Schemas[1]},
		&schema.AddTable{T: to.Schemas[1].Tables[0]},
	}, changes)
}
