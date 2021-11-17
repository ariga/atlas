package postgres

import (
	"fmt"
	"log"
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
	var s schema.Schema
	err := UnmarshalSpec([]byte(f), schemahcl.Unmarshal, &s)
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
	exp.Tables[1].PrimaryKey = &schema.Index{
		Table: exp.Tables[1],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[1].Columns[0]},
		},
	}
	require.EqualValues(t, exp, &s)
}

func TestUnmarshalSpecColumnTypes(t *testing.T) {
	for _, tt := range []struct {
		spec     *sqlspec.Column
		expected schema.Type
	}{
		{
			spec: specutil.NewCol("int64", "int64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("string_varchar", "string", specutil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    tVarChar,
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("string_test", "string", specutil.LitAttr("size", "10485761")),
			expected: &schema.StringType{
				T:    tText,
				Size: 10_485_761,
			},
		},
		{
			spec: specutil.NewCol("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    tVarChar,
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("decimal(10, 2)", "decimal(10, 2)"),
			expected: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec:     specutil.NewCol("enum", "enum", specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
			expected: &schema.EnumType{Values: []string{"a", "b", "c"}},
		},
		{
			spec:     specutil.NewCol("bool", "boolean"),
			expected: &schema.BoolType{T: tBoolean},
		},
		{
			spec:     specutil.NewCol("decimal", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: tDecimal, Precision: 10, Scale: 2},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: tReal, Precision: 10},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: tDouble, Precision: 25},
		},
		{
			spec:     specutil.NewCol("cidr", "cidr"),
			expected: &NetworkType{T: tCIDR},
		},
		{
			spec:     specutil.NewCol("money", "money"),
			expected: &CurrencyType{T: tMoney},
		},
		{
			spec:     specutil.NewCol("bit", "bit"),
			expected: &BitType{T: tBit, Len: 1},
		},
		{
			spec:     specutil.NewCol("bitvar", "bit varying"),
			expected: &BitType{T: tBitVar},
		},
		{
			spec:     specutil.NewCol("bitvar8", "bit varying(8)"),
			expected: &BitType{T: tBitVar, Len: 8},
		},
		{
			spec:     specutil.NewCol("bit8", "bit(8)"),
			expected: &BitType{T: tBit, Len: 8},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			var s schema.Schema
			err := UnmarshalSpec(hcl(tt.spec), schemahcl.Unmarshal, &s)
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
	for _, tt := range []struct {
		spec        *sqlspec.Column
		expectedErr string
	}{
		{
			spec:        specutil.NewCol("uint", "uint"),
			expectedErr: "postgres: failed converting to *schema.Schema: unsigned integers currently not supported",
		},
		{
			spec:        specutil.NewCol("int8", "int8"),
			expectedErr: "postgres: failed converting to *schema.Schema: 8-bit integers not supported",
		},

		{
			spec:        specutil.NewCol("uint64", "uint64"),
			expectedErr: "postgres: failed converting to *schema.Schema: unsigned integers currently not supported",
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			var s schema.Schema
			err := UnmarshalSpec(hcl(tt.spec), schemahcl.Unmarshal, &s)
			require.Equal(t, tt.expectedErr, err.Error())
		})
	}
}

// hcl returns an Atlas HCL document containing the column spec.
func hcl(c *sqlspec.Column) []byte {
	buf, err := schemahcl.Marshal(c)
	if err != nil {
		log.Fatalln(err)
	}
	tmpl := `
schema "default" {
}
table "table" {
	schema = "default"
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
			schem: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("int", "int"),
		},
		{
			schem: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
			},
			expected: specutil.NewCol("int64", "int64"),
		},
		{
			schem: &schema.IntegerType{
				T:        tInteger,
				Unsigned: false,
			},
			expected: specutil.NewCol("integer", "int"),
		},
		{
			schem: &schema.IntegerType{
				T:        tInt2,
				Unsigned: false,
			},
			expected: specutil.NewCol("int2", tInt2),
		},
		{
			schem: &schema.IntegerType{
				T:        tInt4,
				Unsigned: false,
			},
			expected: specutil.NewCol("int4", tInt4),
		},
		{
			schem: &schema.IntegerType{
				T:        tInt8,
				Unsigned: false,
			},
			expected: specutil.NewCol("int8", tInt8),
		},
		{
			schem: &schema.StringType{
				T:    tVarChar,
				Size: 255,
			},
			expected: specutil.NewCol("string_varchar", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tChar,
				Size: 255,
			},
			expected: specutil.NewCol("string_tchar", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tCharacter,
				Size: 255,
			},
			expected: specutil.NewCol("string_Character", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tCharVar,
				Size: 255,
			},
			expected: specutil.NewCol("string_CharVar", "string", specutil.LitAttr("size", "255")),
		},
		{
			schem: &schema.StringType{
				T:    tText,
				Size: 10_485_761,
			},
			expected: specutil.NewCol("string_text", "string", specutil.LitAttr("size", "10485761")),
		},
		{
			schem: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
			expected: specutil.NewCol("decimal", "decimal",
				specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
		},
		{
			schem: &schema.EnumType{
				Values: []string{"a", "b", "c"},
			},
			expected: specutil.NewCol("enum", "enum",
				specutil.ListAttr("values", `"a"`, `"b"`, `"c"`)),
		},
		{
			schem: &schema.BoolType{
				T: tBoolean,
			},
			expected: specutil.NewCol("boolean", "boolean"),
		},
		{
			schem: &schema.FloatType{
				T:         tReal,
				Precision: 10,
			},
			expected: specutil.NewCol("float_real", "float", specutil.LitAttr("precision", "10")),
		},
		{
			schem: &schema.FloatType{
				T:         tDouble,
				Precision: 25,
			},
			expected: specutil.NewCol("float_double", "float", specutil.LitAttr("precision", "25")),
		},
		{
			schem: &NetworkType{
				T: tCIDR,
			},
			expected: specutil.NewCol("network", "cidr"),
		},
		{
			schem: &CurrencyType{
				T: tMoney,
			},
			expected: specutil.NewCol("money", "money"),
		},
		{
			schem: &BitType{
				T:   tBit,
				Len: 1,
			},
			expected: specutil.NewCol("bit", "bit"),
		},
		{
			schem: &BitType{
				T: tBitVar,
			},
			expected: specutil.NewCol("bitvar", "bit varying"),
		},
		{
			schem: &BitType{
				T:   tBit,
				Len: 8,
			},
			expected: specutil.NewCol("bit8", "bit(8)"),
		},
		{
			schem: &BitType{
				T:   tBitVar,
				Len: 8,
			},
			expected: specutil.NewCol("bitvar8", "bit varying(8)"),
		},
	} {
		t.Run(tt.expected.Name, func(t *testing.T) {
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
			require.EqualValues(t, tt.expected.TypeName, test.Table.Columns[0].TypeName)
			require.ElementsMatch(t, tt.expected.Extra.Attrs, test.Table.Columns[0].Extra.Attrs)
		})
	}
}
