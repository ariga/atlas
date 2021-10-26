// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = "int"
	}
	column "age" {
		type = "int"
	}
	column "account_name" {
		type = "string"
		size = 32
	}
	primary_key {
		columns = [table.table.column.col]
	}
	index "index" {
		unique = true
		columns = [
			table.table.column.col,
			table.table.column.age,
		]
	}
	foreign_key "accounts" {
		columns = [
			table.table.column.account_name,
		]
		ref_columns = [
			table.accounts.column.name,
		]
		on_delete = "SET NULL"
	}
}

table "accounts" {
	column "name" {
		type = "string"
		size = 32
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
}
`
	exp := &schema.Schema{
		Name: "schema",
	}
	exp.Tables = []*schema.Table{
		{
			Name:   "table",
			Schema: exp,
			Columns: []*schema.Column{
				{
					Name: "col",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: "integer",
						},
					},
				},
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: "integer",
						},
					},
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "text",
							Size: 32,
						},
					},
				},
			},
		},
		{
			Name:   "accounts",
			Schema: exp,
			Columns: []*schema.Column{
				{
					Name: "name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "text",
							Size: 32,
						},
					},
				},
			},
		},
	}
	exp.Tables[0].PrimaryKey = &schema.Index{
		Table: exp.Tables[0],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[0].Columns[0]},
		},
	}
	exp.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  exp.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: exp.Tables[0].Columns[0]},
				{SeqNo: 1, C: exp.Tables[0].Columns[1]},
			},
		},
	}
	exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
		{
			Symbol:     "accounts",
			Table:      exp.Tables[0],
			Columns:    []*schema.Column{exp.Tables[0].Columns[2]},
			RefColumns: []*schema.Column{exp.Tables[1].Columns[0]},
			OnDelete:   schema.SetNull,
		},
	}
	exp.Tables[1].PrimaryKey = &schema.Index{
		Table: exp.Tables[1],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[1].Columns[0]},
		},
	}

	var s schema.Schema
	err := UnmarshalSpec([]byte(f), schemahcl.Unmarshal, &s)
	require.NoError(t, err)
	require.EqualValues(t, exp, &s)
}

func TestMarshalSpecColumnType(t *testing.T) {
	for _, tt := range []struct {
		schem       schema.Type
		expected    *sqlspec.Column
	}{
		{
			schem: &schema.IntegerType{
				T: tInteger,
			},
			expected:    specutil.NewCol("column", "int"),
		},
		//{
		//	schem: &schema.IntegerType{
		//		T: tInteger,
		//		Unsigned: true,
		//	},
		//	expected:    nil,
		//},
		//{
		{
			schem: &schema.StringType{
				T:    tText,
				Size: 17_000_000,
			},
			expected:   specutil.NewCol("column", "string", specutil.LitAttr("size", "17000000")),
		},
		//{
		//	schem: schemautil.ColSpec("decimal(10, 2)", "decimal(10, 2)"),
		//	expected: &schema.DecimalType{
		//		T:         "decimal",
		//		Scale:     2,
		//		Precision: 10,
		//	},
		//},
		//{
		//	schem:     schemautil.ColSpec("enum", "enum", schemautil.ListAttr("values", "a", "b", "c")),
		//	expected: &schema.StringType{T: "text"},
		//},
		//{
		//	schem:     schemautil.ColSpec("bool", "boolean"),
		//	expected: &schema.BoolType{T: "boolean"},
		//},
		//{
		//	schem:     schemautil.ColSpec("decimal", "decimal", schemautil.LitAttr("precision", "10"), schemautil.LitAttr("scale", "2")),
		//	expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		//},
		//{
		//	schem:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "10")),
		//	expected: &schema.FloatType{T: "real", Precision: 10},
		//},
	} {
		t.Run(tt.expected.TypeName, func(t *testing.T) {
			s := schema.Schema{
				Tables: []*schema.Table{
					{
						Name: "table",
						Columns: []*schema.Column{
							{
								Name: "column",
								Type: &schema.ColumnType{Type: tt.schem},
							},
						},
					},
				},
			}
			s.Tables[0].Schema = &s
			ddl, err := MarshalSpec(&s, schemahcl.Marshal)
			require.NoError(t, err)
			var test struct {
				Table *sqlspec.Table `spec:"table"`
			}
			err = schemahcl.Unmarshal(ddl, &test)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, test.Table.Columns[0])
		})
	}
}
