// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"fmt"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/stretchr/testify/require"
)

var hclState = schemahcl.New(schemahcl.WithTypes(TypeRegistry.Specs()))

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
			RefTable:   exp.Tables[1],
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

func TestUnmarshalSpecColumnTypes(t *testing.T) {
	for _, tt := range []struct {
		spec     *sqlspec.Column
		expected schema.Type
	}{
		{
			spec: specutil.NewCol("int", "int"),
			expected: &schema.IntegerType{
				T:        tInteger,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("int8", "int8"),
			expected: &schema.IntegerType{
				T:        tInteger,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("int64", "int64"),
			expected: &schema.IntegerType{
				T:        tInteger,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("string_varchar", "string", specutil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    tText,
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("string_mediumtext", "string", specutil.LitAttr("size", "100000")),
			expected: &schema.StringType{
				T:    tText,
				Size: 100_000,
			},
		},
		{
			spec: specutil.NewCol("string_longtext", "string", specutil.LitAttr("size", "17000000")),
			expected: &schema.StringType{
				T:    tText,
				Size: 17_000_000,
			},
		},
		{
			spec: specutil.NewCol("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("decimal(10, 2) unsigned", "decimal(10, 2) unsigned"),
			expected: &schema.DecimalType{
				T:         "decimal",
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec: specutil.NewCol("blob", "binary"),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: specutil.NewCol("tinyblob", "binary", specutil.LitAttr("size", "16")),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: specutil.NewCol("mediumblob", "binary", specutil.LitAttr("size", "100000")),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: specutil.NewCol("longblob", "binary", specutil.LitAttr("size", "20000000")),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec:     specutil.NewCol("enum", "enum", specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
			expected: &schema.StringType{T: tText},
		},
		{
			spec:     specutil.NewCol("bool", "boolean"),
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     specutil.NewCol("decimal", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: tReal, Precision: 10},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: tReal, Precision: 25},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			var s schema.Schema
			err := UnmarshalSpec(hcl(t, tt.spec), schemahcl.Unmarshal, &s)
			require.NoError(t, err)
			tbl, ok := s.Table("table")
			require.True(t, ok)
			col, ok := tbl.Column(tt.spec.Name)
			require.True(t, ok)
			require.EqualValues(t, tt.expected, col.Type.Type)
		})
	}
}

func TestNotSupportedUnmarshalSpecColumnTypes(t *testing.T) {
	var s schema.Schema
	err := UnmarshalSpec(hcl(t, specutil.NewCol("uint64", "uint64")), schemahcl.Unmarshal, &s)
	require.Error(t, err)
}

// hcl returns an Atlas HCL document containing the column spec.
func hcl(t *testing.T, c *sqlspec.Column) []byte {
	buf, err := schemahcl.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := `
schema "default" {
}
table "table" {
	schema = schema.default
	%s
}
`
	body := fmt.Sprintf(tmpl, buf)
	return []byte(body)
}

func TestMarshalSpecColumnType(t *testing.T) {
	for _, tt := range []struct {
		schem    schema.Type
		expected *sqlspec.Column
	}{
		{
			schem:    &schema.IntegerType{T: tInteger},
			expected: specutil.NewCol("column", "int"),
		},
		{
			schem:    &schema.StringType{T: tText, Size: 17_000_000},
			expected: specutil.NewCol("column", "string", specutil.LitAttr("size", "17000000")),
		},
		{
			schem:    &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
			expected: specutil.NewCol("column", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
		},
		{
			schem:    &schema.EnumType{T: "enum", Values: []string{"a", "b", "c"}},
			expected: specutil.NewCol("column", "enum", specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
		},
		{
			schem:    &schema.BoolType{T: "boolean"},
			expected: specutil.NewCol("column", "boolean"),
		},
		{
			schem:    &schema.FloatType{T: "float", Precision: 10},
			expected: specutil.NewCol("column", "float", specutil.LitAttr("precision", "10")),
		},
	} {
		t.Run(tt.expected.Type, func(t *testing.T) {
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
			require.EqualValues(t, tt.expected.Type, test.Table.Columns[0].Type)
			require.ElementsMatch(t, tt.expected.Extra.Attrs, test.Table.Columns[0].Extra.Attrs)
		})
	}
}

func TestNotSupportedMarshalSpecColumnType(t *testing.T) {
	for _, tt := range []struct {
		schem    schema.Type
		expected *sqlspec.Column
	}{
		{
			schem:    &schema.IntegerType{T: tInteger, Unsigned: true},
			expected: specutil.NewCol("column", "int"),
		},
	} {
		t.Run(tt.expected.Type, func(t *testing.T) {
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
			_, err := MarshalSpec(&s, schemahcl.Marshal)
			require.Error(t, err)
		})
	}
}

func TestTypes(t *testing.T) {
	for _, tt := range []struct {
		typeExpr string
		expected schema.Type
	}{
		{
			typeExpr: "integer(10)",
			expected: &schema.IntegerType{T: "integer"},
		},
		{
			typeExpr: "int(10)",
			expected: &schema.IntegerType{T: "int"},
		},
		{
			typeExpr: "tinyint(10)",
			expected: &schema.IntegerType{T: "tinyint"},
		},
		{
			typeExpr: "smallint(10)",
			expected: &schema.IntegerType{T: "smallint"},
		},
		{
			typeExpr: "mediumint(10)",
			expected: &schema.IntegerType{T: "mediumint"},
		},
		{
			typeExpr: "bigint(10)",
			expected: &schema.IntegerType{T: "bigint"},
		},
		{
			typeExpr: "unsigned_big_int(10)",
			expected: &schema.IntegerType{T: "unsigned big int"},
		},
		{
			typeExpr: "int2(10)",
			expected: &schema.IntegerType{T: "int2"},
		},
		{
			typeExpr: "int8(10)",
			expected: &schema.IntegerType{T: "int8"},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: "real"},
		},
		{
			typeExpr: "double",
			expected: &schema.FloatType{T: "double"},
		},
		{
			typeExpr: "double_precision",
			expected: &schema.FloatType{T: "double_precision"},
		},
		{
			typeExpr: "float(10)",
			expected: &schema.IntegerType{T: "float"},
		},
		{
			typeExpr: "text(10)",
			expected: &schema.IntegerType{T: "text"},
		},
		{
			typeExpr: "character(10)",
			expected: &schema.IntegerType{T: "character"},
		},
		{
			typeExpr: "varchar(10)",
			expected: &schema.IntegerType{T: "varchar"},
		},
		{
			typeExpr: "varying",
			expected: &schema.IntegerType{T: "varying"},
		},
		{
			typeExpr: "nchar(10)",
			expected: &schema.IntegerType{T: "nchar"},
		},
		{
			typeExpr: "native",
			expected: &schema.IntegerType{T: "native"},
		},
		{
			typeExpr: "nvarchar(10)",
			expected: &schema.IntegerType{T: "nvarchar"},
		},
		{
			typeExpr: "clob(10)",
			expected: &schema.IntegerType{T: "clob"},
		},
		{
			typeExpr: "blob(10)",
			expected: &schema.IntegerType{T: "blob"},
		},
		{
			typeExpr: "numeric(10)",
			expected: &schema.IntegerType{T: "numeric"},
		},
		{
			typeExpr: "decimal(10,5)",
			expected: &schema.IntegerType{T: "decimal"},
		},
		{
			typeExpr: "boolean",
			expected: &schema.IntegerType{T: "boolean"},
		},
		{
			typeExpr: "date",
			expected: &schema.IntegerType{T: "date"},
		},
		{
			typeExpr: "datetime",
			expected: &schema.IntegerType{T: "datetime"},
		},
	} {
		t.Run(tt.typeExpr, func(t *testing.T) {
			// simulates sqlspec.Column until we change its Type field.
			type col struct {
				Type *schemaspec.Type `spec:"type"`
				schemaspec.DefaultExtension
			}
			var test struct {
				Columns []*col `spec:"column"`
			}
			doc := fmt.Sprintf(`column {
	type = %s
}
`, tt.typeExpr)
			err := hclState.UnmarshalSpec([]byte(doc), &test)
			require.NoError(t, err)
			column := test.Columns[0]
			typ, err := TypeRegistry.Type(column.Type, column.Extra.Attrs, parseRawType)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, typ)
			spec, err := hclState.MarshalSpec(&test)
			require.NoError(t, err)
			require.EqualValues(t, string(hclwrite.Format([]byte(doc))), string(hclwrite.Format(spec)))
		})
	}
}
