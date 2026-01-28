// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"testing"

	"ariga.io/atlas/sql/schema"

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
			from: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "local"}},
			to:   &schema.Table{Name: "users"},
		},
		{
			name: "add column",
			from: &schema.Table{
				Name:   "users",
				Schema: &schema.Schema{Name: "local"},
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
				},
			},
			to: &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}},
				},
			},
			wantChanges: []schema.Change{
				&schema.AddColumn{C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}}},
			},
		},
		{
			name: "drop column",
			from: &schema.Table{
				Name:   "users",
				Schema: &schema.Schema{Name: "local"},
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}},
				},
			},
			to: &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
				},
			},
			wantChanges: []schema.Change{
				&schema.DropColumn{C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}}},
			},
		},
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt64}}},
					},
				}
			)
			return testcase{
				name: "modify column type",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeType,
					},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}, Null: false}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}, Null: true}},
					},
				}
			)
			return testcase{
				name: "modify column null",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeNull,
					},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}, Default: &schema.RawExpr{X: "1"}},
					},
				}
			)
			return testcase{
				name: "modify column default",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyColumn{
						From:   from.Columns[0],
						To:     to.Columns[0],
						Change: schema.ChangeDefault,
					},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "idx_id", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
			}
			return testcase{
				name: "drop index",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.DropIndex{I: from.Indexes[0]},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
			)
			to.Indexes = []*schema.Index{
				{Name: "idx_id", Table: to, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[0]}}},
			}
			return testcase{
				name: "add index",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.AddIndex{I: to.Indexes[0]},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "idx_id", Unique: false, Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "idx_id", Unique: true, Table: to, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[0]}}},
			}
			return testcase{
				name: "modify index unique",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeUnique},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "idx_id", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}, Attrs: []schema.Attr{&IndexAttributes{Async: false}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "idx_id", Table: to, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[0]}}, Attrs: []schema.Attr{&IndexAttributes{Async: true}}},
			}
			return testcase{
				name: "modify index async",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeAttr},
				},
			}
		}(),
		func() testcase {
			var (
				from = &schema.Table{
					Name:   "users",
					Schema: &schema.Schema{Name: "local"},
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
						{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}},
					},
				}
				to = &schema.Table{
					Name: "users",
					Columns: []*schema.Column{
						{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
						{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}},
					},
				}
			)
			from.Indexes = []*schema.Index{
				{Name: "idx_id", Table: from, Parts: []*schema.IndexPart{{SeqNo: 1, C: from.Columns[0]}}},
			}
			to.Indexes = []*schema.Index{
				{Name: "idx_id", Table: to, Parts: []*schema.IndexPart{{SeqNo: 1, C: to.Columns[0]}}, Attrs: []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{to.Columns[1]}}}},
			}
			return testcase{
				name: "modify index add cover columns",
				from: from,
				to:   to,
				wantChanges: []schema.Change{
					&schema.ModifyIndex{From: from.Indexes[0], To: to.Indexes[0], Change: schema.ChangeAttr},
				},
			}
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := DefaultDiff.TableDiff(tt.from, tt.to)
			require.Equalf(t, tt.wantErr, err != nil, "error: %v", err)
			require.EqualValues(t, tt.wantChanges, changes)
		})
	}
}

func TestDiff_SchemaDiff(t *testing.T) {
	from := schema.New("local").
		AddTables(schema.NewTable("users"), schema.NewTable("pets"))
	to := schema.New("local").
		AddTables(
			schema.NewTable("users").AddColumns(schema.NewIntColumn("id", TypeInt32)),
			schema.NewTable("groups"),
		)

	changes, err := DefaultDiff.SchemaDiff(from, to)
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.ModifyTable{
			T: to.Tables[0],
			Changes: []schema.Change{
				&schema.AddColumn{C: to.Tables[0].Columns[0]},
			},
		},
		&schema.DropTable{T: from.Tables[1]},
		&schema.AddTable{T: to.Tables[1]},
	}, changes)
}

func TestDiff_RealmDiff(t *testing.T) {
	to := schema.New("local").
		AddTables(
			schema.NewTable("users").AddColumns(schema.NewIntColumn("id", TypeInt32)),
		)

	changes, err := DefaultDiff.RealmDiff(schema.NewRealm(), schema.NewRealm(to))
	require.NoError(t, err)
	require.EqualValues(t, []schema.Change{
		&schema.AddSchema{S: to},
		&schema.AddTable{T: to.Tables[0]},
	}, changes)
}

func TestDiff_TypeChanged(t *testing.T) {
	d := &diff{conn: &conn{}}

	tests := []struct {
		name    string
		from    *schema.Column
		to      *schema.Column
		changed bool
		wantErr bool
	}{
		{
			name: "same type",
			from: &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
			to:   &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
		},
		{
			name:    "different integer types",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt64}}},
			changed: true,
		},
		{
			name:    "different type kinds",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.StringType{T: TypeUtf8}}},
			changed: true,
		},
		{
			name:    "decimal precision changed",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 20, Scale: 2}}},
			changed: true,
		},
		{
			name:    "decimal scale changed",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 5}}},
			changed: true,
		},
		{
			name: "decimal same",
			from: &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}}},
			to:   &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}}},
		},
		{
			name:    "nil from type",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: nil}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
			wantErr: true,
		},
		{
			name:    "nil to type",
			from:    &schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: TypeInt32}}},
			to:      &schema.Column{Name: "c", Type: &schema.ColumnType{Type: nil}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed, err := d.typeChanged(tt.from, tt.to)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.changed, changed)
			}
		})
	}
}

func TestDiff_DefaultChanged(t *testing.T) {
	d := &diff{conn: &conn{}}

	tests := []struct {
		name    string
		from    *schema.Column
		to      *schema.Column
		changed bool
	}{
		{
			name: "no default to no default",
			from: &schema.Column{Name: "c"},
			to:   &schema.Column{Name: "c"},
		},
		{
			name:    "no default to default",
			from:    &schema.Column{Name: "c"},
			to:      &schema.Column{Name: "c", Default: &schema.RawExpr{X: "1"}},
			changed: true,
		},
		{
			name:    "default to no default",
			from:    &schema.Column{Name: "c", Default: &schema.RawExpr{X: "1"}},
			to:      &schema.Column{Name: "c"},
			changed: true,
		},
		{
			name: "same default",
			from: &schema.Column{Name: "c", Default: &schema.RawExpr{X: "1"}},
			to:   &schema.Column{Name: "c", Default: &schema.RawExpr{X: "1"}},
		},
		{
			name:    "different default",
			from:    &schema.Column{Name: "c", Default: &schema.RawExpr{X: "1"}},
			to:      &schema.Column{Name: "c", Default: &schema.RawExpr{X: "2"}},
			changed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := d.defaultChanged(tt.from, tt.to)
			require.Equal(t, tt.changed, changed)
		})
	}
}

func TestDiff_IndexAttrChanged(t *testing.T) {
	d := &diff{conn: &conn{}}

	col1 := &schema.Column{Name: "name"}
	col2 := &schema.Column{Name: "email"}

	tests := []struct {
		name    string
		from    []schema.Attr
		to      []schema.Attr
		changed bool
	}{
		{
			name: "no attributes",
			from: nil,
			to:   nil,
		},
		{
			name:    "add async attribute",
			from:    nil,
			to:      []schema.Attr{&IndexAttributes{Async: true}},
			changed: true,
		},
		{
			name:    "remove async attribute",
			from:    []schema.Attr{&IndexAttributes{Async: true}},
			to:      nil,
			changed: true,
		},
		{
			name: "same async false",
			from: []schema.Attr{&IndexAttributes{Async: false}},
			to:   []schema.Attr{&IndexAttributes{Async: false}},
		},
		{
			name: "same async true",
			from: []schema.Attr{&IndexAttributes{Async: true}},
			to:   []schema.Attr{&IndexAttributes{Async: true}},
		},
		{
			name:    "change async false to true",
			from:    []schema.Attr{&IndexAttributes{Async: false}},
			to:      []schema.Attr{&IndexAttributes{Async: true}},
			changed: true,
		},
		{
			name:    "change async true to false",
			from:    []schema.Attr{&IndexAttributes{Async: true}},
			to:      []schema.Attr{&IndexAttributes{Async: false}},
			changed: true,
		},
		{
			name: "same cover columns",
			from: []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
			to:   []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
		},
		{
			name:    "add cover columns",
			from:    []schema.Attr{&IndexAttributes{}},
			to:      []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
			changed: true,
		},
		{
			name:    "remove cover columns",
			from:    []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
			to:      []schema.Attr{&IndexAttributes{}},
			changed: true,
		},
		{
			name:    "different cover columns count",
			from:    []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
			to:      []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1, col2}}},
			changed: true,
		},
		{
			name:    "different cover column names",
			from:    []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col1}}},
			to:      []schema.Attr{&IndexAttributes{CoverColumns: []*schema.Column{col2}}},
			changed: true,
		},
		{
			name: "same async and cover columns",
			from: []schema.Attr{&IndexAttributes{Async: true, CoverColumns: []*schema.Column{col1, col2}}},
			to:   []schema.Attr{&IndexAttributes{Async: true, CoverColumns: []*schema.Column{col1, col2}}},
		},
		{
			name:    "same cover columns different async",
			from:    []schema.Attr{&IndexAttributes{Async: false, CoverColumns: []*schema.Column{col1}}},
			to:      []schema.Attr{&IndexAttributes{Async: true, CoverColumns: []*schema.Column{col1}}},
			changed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := d.IndexAttrChanged(tt.from, tt.to)
			require.Equal(t, tt.changed, changed)
		})
	}
}
