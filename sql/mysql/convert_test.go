// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestConvertSchema(t *testing.T) {
	spec := &schemaspec.Schema{
		Name: "schema",
	}
	tables := []*schemaspec.Table{
		{
			Name: "table",
			Columns: []*schemaspec.Column{
				{
					Name: "col",
					Type: "int",
				},
				{
					Name: "age",
					Type: "int",
				},
				{
					Name: "account_name",
					Type: "varchar(32)",
				},
			},
			PrimaryKey: &schemaspec.PrimaryKey{
				Columns: []*schemaspec.ColumnRef{{Table: "table", Name: "col"}},
			},
			ForeignKeys: []*schemaspec.ForeignKey{
				{
					Symbol: "accounts",
					Columns: []*schemaspec.ColumnRef{
						{Table: "table", Name: "account_name"},
					},
					RefColumns: []*schemaspec.ColumnRef{
						{Table: "accounts", Name: "name"},
					},
					OnDelete: string(schema.SetNull),
				},
			},
			Indexes: []*schemaspec.Index{
				{
					Name:   "index",
					Unique: true,
					Columns: []*schemaspec.ColumnRef{
						{Table: "table", Name: "col"},
						{Table: "table", Name: "age"},
					},
				},
			},
		},
		{
			Name: "accounts",
			Columns: []*schemaspec.Column{
				{
					Name: "name",
					Type: "varchar(32)",
				},
			},
		},
	}
	d := &Driver{}
	sch, err := d.ConvertSchema(spec, tables)
	require.NoError(t, err)
	exp := &schema.Schema{
		Name: "schema",
		Spec: spec,
	}
	exp.Tables = []*schema.Table{
		{
			Name:   "table",
			Schema: exp,
			Spec:   tables[0],
			Columns: []*schema.Column{
				{
					Name: "col",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: tInt,
						},
					},
					Spec: tables[0].Columns[0],
				},
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: tInt,
						},
					},
					Spec: tables[0].Columns[1],
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    tVarchar,
							Size: 32,
						},
					},
					Spec: tables[0].Columns[2],
				},
			},
		},
		{
			Name:   "accounts",
			Spec:   tables[1],
			Schema: exp,
			Columns: []*schema.Column{
				{
					Name: "name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    tVarchar,
							Size: 32,
						},
					},
					Spec: tables[1].Columns[0],
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
	require.EqualValues(t, exp, sch)
}

func TestConvertColumnType(t *testing.T) {
	for _, tt := range []struct {
		spec     *schemaspec.Column
		expected schema.Type
	}{
		{
			spec: schemautil.ColSpec("int", "int"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
			},
		},
		{
			spec: schemautil.ColSpec("uint", "uint"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: true,
			},
		},
		{
			spec: schemautil.ColSpec("int8", "int8"),
			expected: &schema.IntegerType{
				T:        tTinyInt,
				Unsigned: false,
			},
		},
		{
			spec: schemautil.ColSpec("int64", "int64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
		},
		{
			spec: schemautil.ColSpec("uint64", "uint64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: true,
			},
		},
		{
			spec: schemautil.ColSpec("string_varchar", "string", schemautil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("string_mediumtext", "string", schemautil.LitAttr("size", "100000")),
			expected: &schema.StringType{
				T:    tMediumText,
				Size: 100_000,
			},
		},
		{
			spec: schemautil.ColSpec("string_longtext", "string", schemautil.LitAttr("size", "17000000")),
			expected: &schema.StringType{
				T:    tLongText,
				Size: 17_000_000,
			},
		},
		{
			spec: schemautil.ColSpec("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("decimal(10, 2) unsigned", "decimal(10, 2) unsigned"),
			expected: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec: schemautil.ColSpec("blob", "binary"),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: schemautil.ColSpec("tinyblob", "binary", schemautil.LitAttr("size", "16")),
			expected: &schema.BinaryType{
				T:    tTinyBlob,
				Size: 16,
			},
		},
		{
			spec: schemautil.ColSpec("mediumblob", "binary", schemautil.LitAttr("size", "100000")),
			expected: &schema.BinaryType{
				T:    tMediumBlob,
				Size: 100_000,
			},
		},
		{
			spec: schemautil.ColSpec("longblob", "binary", schemautil.LitAttr("size", "20000000")),
			expected: &schema.BinaryType{
				T:    tLongBlob,
				Size: 20_000_000,
			},
		},
		{
			spec:     schemautil.ColSpec("enum", "enum", schemautil.ListAttr("values", "a", "b", "c")),
			expected: &schema.EnumType{Values: []string{"a", "b", "c"}},
		},
		{
			spec:     schemautil.ColSpec("bool", "boolean"),
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     schemautil.ColSpec("decimal", "decimal", schemautil.LitAttr("precision", "10"), schemautil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: "float", Precision: 10},
		},
		{
			spec:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: "double", Precision: 25},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			columnType, err := ConvertColumnType(tt.spec)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, columnType)
		})
	}
}

func TestConvertFromColumnType(t *testing.T) {
	for _, tt := range []struct {
		schem    schema.Type
		expected *schemaspec.Column
	}{
		{
			schem: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
			},
			expected: schemautil.ColSpec("", "int"),
		},
		{
			schem: &schema.IntegerType{
				T:        tInt,
				Unsigned: true,
			},
			expected: schemautil.ColSpec("", "uint"),
		},
		{
			schem: &schema.IntegerType{
				T:        tTinyInt,
				Unsigned: false,
			},
			expected: schemautil.ColSpec("", "int8"),
		},
		{
			schem: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
			expected: schemautil.ColSpec("", "int64"),
		},
		{
			schem: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: true,
			},
			expected: schemautil.ColSpec("", "uint64"),
		},
		{
			schem: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
			expected: schemautil.ColSpec("", "string", schemautil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tMediumText,
				Size: 100_000,
			},
			expected: schemautil.ColSpec("", "string", schemautil.LitAttr("size", "100000")),
		},
		{
			schem: &schema.StringType{
				T:    tLongText,
				Size: 17_000_000,
			},
			expected: schemautil.ColSpec("", "string", schemautil.LitAttr("size", "17000000")),
		},
		//{
		//	schem: &schema.StringType{
		//		T:    tVarchar,
		//		Size: 255,
		//	},
		//	expected: schemautil.ColSpec("", "varchar(255)"),
		//},  DISCUSS
		{
			schem: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
			expected: schemautil.ColSpec("", "decimal(10, 2) unsigned"),
		},
		{
			schem: &schema.BinaryType{
				T: tBlob,
			},
			expected: schemautil.ColSpec("", "binary"),
		},
		//{
		//	expected: schemautil.ColSpec("tinyblob", "binary", schemautil.LitAttr("size", "16")),
		//	schem: &schema.BinaryType{
		//		T:    tTinyBlob,
		//		Size: 16,
		//	},
		//},
		//{
		//	expected: schemautil.ColSpec("mediumblob", "binary", schemautil.LitAttr("size", "100000")),
		//	schem: &schema.BinaryType{
		//		T:    tMediumBlob,
		//		Size: 100_000,
		//	},
		//},
		//{
		//	expected: schemautil.ColSpec("longblob", "binary", schemautil.LitAttr("size", "20000000")),
		//	schem: &schema.BinaryType{
		//		T:    tLongBlob,
		//		Size: 20_000_000,
		//	},
		//},
		{
			schem:    &schema.EnumType{Values: []string{"a", "b", "c"}},
			expected: schemautil.ColSpec("", "enum", schemautil.ListAttr("values", "a", "b", "c")),
		},
		//{
		//	expected:  schemautil.ColSpec("bool", "boolean"),
		//	schem: &schema.BoolType{T: "boolean"},
		//},
		//{
		//	expected:  schemautil.ColSpec("decimal", "decimal", schemautil.LitAttr("precision", "10"), schemautil.LitAttr("scale", "2")),
		//	schem: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		//},
		//{
		//	expected:  schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "10")),
		//	schem: &schema.FloatType{T: "float", Precision: 10},
		//},
		//{
		//	expected:  schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "25")),
		//	schem: &schema.FloatType{T: "double", Precision: 25},
		//},
	} {
		t.Run(tt.expected.Name, func(t *testing.T) {
			columnType, err := ColumnTypeSpec(tt.schem)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, columnType)
		})
	}
}

func TestOverride(t *testing.T) {
	s := schemautil.ColSpec("int", "int")
	s.Overrides = []*schemaspec.Override{
		{
			Dialect: "mysql",
			Version: "8",
			Resource: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					schemautil.StrLitAttr("type", "bigint"),
				},
			},
		},
	}
	d := &Driver{version: "8.11"}
	c, err := d.ConvertColumn(s, nil)
	it, ok := c.Type.Type.(*schema.IntegerType)
	require.True(t, ok)
	require.NoError(t, err)
	require.Equal(t, "bigint", it.T)
}
