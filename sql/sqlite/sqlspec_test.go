// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestSQLSpec(t *testing.T) {
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
							T:    "varchar",
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
							T:    "varchar",
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
	require.EqualValues(t, exp, sch)
}

//func TestUnmarshalSpecColumnTypes(t *testing.T) {
//	for _, tt := range []struct {
//		spec        *schemaspec.Column
//		expected    schema.Type
//		expectedErr string
//	}{
//		{
//			spec: schemautil.ColSpec("int", "int"),
//			expected: &schema.IntegerType{
//				T:        "integer",
//				Unsigned: false,
//			},
//		},
//		{
//			spec:        schemautil.ColSpec("uint", "uint"),
//			expectedErr: "sqlite: unsigned integers currently not supported",
//		},
//		{
//			spec: schemautil.ColSpec("int64", "int64"),
//			expected: &schema.IntegerType{
//				T:        "integer",
//				Unsigned: false,
//			},
//		},
//		{
//			spec:        schemautil.ColSpec("uint64", "uint64"),
//			expectedErr: "sqlite: unsigned integers currently not supported",
//		},
//		{
//			spec: schemautil.ColSpec("string_varchar", "string", schemautil.LitAttr("size", "255")),
//			expected: &schema.StringType{
//				T:    "text",
//				Size: 255,
//			},
//		},
//		{
//			spec: schemautil.ColSpec("string_test", "string", schemautil.LitAttr("size", "10485761")),
//			expected: &schema.StringType{
//				T:    "text",
//				Size: 10_485_761,
//			},
//		},
//		{
//			spec: schemautil.ColSpec("varchar(255)", "varchar(255)"),
//			expected: &schema.StringType{
//				T:    "varchar",
//				Size: 255,
//			},
//		},
//		{
//			spec: schemautil.ColSpec("decimal(10, 2)", "decimal(10, 2)"),
//			expected: &schema.DecimalType{
//				T:         "decimal",
//				Scale:     2,
//				Precision: 10,
//			},
//		},
//		{
//			spec:     schemautil.ColSpec("enum", "enum", schemautil.ListAttr("values", "a", "b", "c")),
//			expected: &schema.StringType{T: "text"},
//		},
//		{
//			spec:     schemautil.ColSpec("bool", "boolean"),
//			expected: &schema.BoolType{T: "boolean"},
//		},
//		{
//			spec:     schemautil.ColSpec("decimal", "decimal", schemautil.LitAttr("precision", "10"), schemautil.LitAttr("scale", "2")),
//			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
//		},
//		{
//			spec:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "10")),
//			expected: &schema.FloatType{T: "real", Precision: 10},
//		},
//	} {
//		t.Run(tt.spec.Name, func(t *testing.T) {
//			columnType, err := ConvertColumnType(tt.spec)
//			if tt.expectedErr != "" && err != nil {
//				require.Equal(t, tt.expectedErr, err.Error())
//				return
//			}
//			require.NoError(t, err)
//			require.EqualValues(t, tt.expected, columnType)
//		})
//	}
//}

//func TestMarshalSpecColumnType(t *testing.T) {
//	for _, tt := range []struct {
//		schem    schema.Type
//		expected *sqlspec.Column
//	}{
//		{
//			schem: &schema.IntegerType{
//				T:        tInt,
//				Unsigned: false,
//			},
//			expected: specutil.NewCol("column", "int"),
//		},
//		{
//			schem: &schema.IntegerType{
//				T:        tInt,
//				Unsigned: true,
//			},
//			expected: specutil.NewCol("column", "uint"),
//		},
//		{
//			schem: &schema.IntegerType{
//				T:        tTinyInt,
//				Unsigned: false,
//			},
//			expected: specutil.NewCol("column", "int8"),
//		},
//		{
//			schem: &schema.IntegerType{
//				T:        tBigInt,
//				Unsigned: false,
//			},
//			expected: specutil.NewCol("column", "int64"),
//		},
//		{
//			schem: &schema.IntegerType{
//				T:        tBigInt,
//				Unsigned: true,
//			},
//			expected: specutil.NewCol("column", "uint64"),
//		},
//		{
//			schem: &schema.StringType{
//				T:    tVarchar,
//				Size: 255,
//			},
//			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "255")),
//		},
//		{
//			schem: &schema.StringType{
//				T:    tMediumText,
//				Size: 100_000,
//			},
//			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "100000")),
//		},
//		{
//			schem: &schema.StringType{
//				T:    tLongText,
//				Size: 17_000_000,
//			},
//			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "17000000")),
//		},
//		{
//			schem:    &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
//			expected: specutil.NewCol("column", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
//		},
//		{
//			schem: &schema.BinaryType{
//				T: tBlob,
//			},
//			expected: specutil.NewCol("column", "binary"),
//		},
//		{
//			schem: &schema.BinaryType{
//				T:    tTinyBlob,
//				Size: 16,
//			},
//			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "16")),
//		},
//		{
//			schem: &schema.BinaryType{
//				T:    tMediumBlob,
//				Size: 100_000,
//			},
//			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "100000")),
//		},
//		{
//			schem: &schema.BinaryType{
//				T:    tLongBlob,
//				Size: 20_000_000,
//			},
//			expected: specutil.NewCol("column", "binary", specutil.LitAttr("size", "20000000")),
//		},
//		{
//			schem:    &schema.EnumType{Values: []string{"a", "b", "c"}},
//			expected: specutil.NewCol("column", "enum", specutil.ListAttr("values", `a`, `b`, `c`)),
//		},
//		{
//			schem:    &schema.BoolType{T: "boolean"},
//			expected: specutil.NewCol("column", "boolean"),
//		},
//		{
//			schem:    &schema.FloatType{T: "float", Precision: 10},
//			expected: specutil.NewCol("column", "float", specutil.LitAttr("precision", "10")),
//		},
//		{
//			schem:    &schema.FloatType{T: "double", Precision: 25},
//			expected: specutil.NewCol("column", "float", specutil.LitAttr("precision", "25")),
//		},
//		{
//			schem:    &schema.TimeType{T: "date"},
//			expected: specutil.NewCol("column", "date"),
//		},
//		{
//			schem:    &schema.TimeType{T: "datetime"},
//			expected: specutil.NewCol("column", "datetime"),
//		},
//		{
//			schem:    &schema.TimeType{T: "time"},
//			expected: specutil.NewCol("column", "time"),
//		},
//		{
//			schem:    &schema.TimeType{T: "timestamp"},
//			expected: specutil.NewCol("column", "timestamp"),
//		},
//		{
//			schem:    &schema.TimeType{T: "year"},
//			expected: specutil.NewCol("column", "year"),
//		},
//		{
//			schem:    &schema.TimeType{T: "year(4)"},
//			expected: specutil.NewCol("column", "year(4)"),
//		},
//	} {
//		t.Run(tt.expected.TypeName, func(t *testing.T) {
//			s := schema.Schema{
//				Tables: []*schema.Table{
//					{
//						Name: "table",
//						Columns: []*schema.Column{
//							{
//								Name: "column",
//								Type: &schema.ColumnType{Type: tt.schem},
//							},
//						},
//					},
//				},
//			}
//			s.Tables[0].Schema = &s
//			ddl, err := MarshalSpec(&s, schemahcl.Marshal)
//			require.NoError(t, err)
//			var test struct {
//				Table *sqlspec.Table `spec:"table"`
//			}
//			err = schemahcl.Unmarshal(ddl, &test)
//			require.NoError(t, err)
//			require.EqualValues(t, tt.expected, test.Table.Columns[0])
//		})
//	}
//}
//
