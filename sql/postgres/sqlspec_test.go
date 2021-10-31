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
	//index "index" {
	//	unique = true
	//	columns = [
	//		table.table.column.col,
	//		table.table.column.age,
	//	]
	//}
	//foreign_key "accounts" {
	//	columns = [
	//		table.table.column.account_name,
	//	]
	//	ref_columns = [
	//		table.accounts.column.name,
	//	]
	//	on_delete = "SET NULL"
	//}
}

//table "accounts" {
//	column "name" {
//		type = "string"
//		size = 32
//	}
//	primary_key {
//		columns = [table.accounts.column.name]
//	}
//}
`
	s := schema.Schema{}
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
		//{
		//	Name:   "accounts",
		//	Schema: exp,
		//	Columns: []*schema.Column{
		//		{
		//			Name: "name",
		//			Type: &schema.ColumnType{
		//				Type: &schema.StringType{
		//					T:    "varchar",
		//					Size: 32,
		//				},
		//			},
		//		},
		//	},
		//},
	}
	exp.Tables[0].PrimaryKey = &schema.Index{
		Table: exp.Tables[0],
		Parts: []*schema.IndexPart{
			{SeqNo: 0, C: exp.Tables[0].Columns[0]},
		},
	}
	//exp.Tables[0].Indexes = []*schema.Index{
	//	{
	//		Name:   "index",
	//		Table:  exp.Tables[0],
	//		Unique: true,
	//		Parts: []*schema.IndexPart{
	//			{SeqNo: 0, C: exp.Tables[0].Columns[0]},
	//			{SeqNo: 1, C: exp.Tables[0].Columns[1]},
	//		},
	//	},
	//}
	//exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
	//	{
	//		Symbol:     "accounts",
	//		Table:      exp.Tables[0],
	//		Columns:    []*schema.Column{exp.Tables[0].Columns[2]},
	//		RefColumns: []*schema.Column{exp.Tables[1].Columns[0]},
	//		OnDelete:   schema.SetNull,
	//	},
	//}

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
				T:        "bigint",
				Unsigned: false,
			},
		},
		{
			spec: specutil.NewCol("string_varchar", "string", specutil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: specutil.NewCol("string_test", "string", specutil.LitAttr("size", "10485761")),
			expected: &schema.StringType{
				T:    "text",
				Size: 10_485_761,
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
			spec: specutil.NewCol("decimal(10, 2)", "decimal(10, 2)"),
			expected: &schema.DecimalType{
				T:         "decimal",
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
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     specutil.NewCol("decimal", "decimal", specutil.LitAttr("precision", "10"), specutil.LitAttr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "10")),
			expected: &schema.FloatType{T: "real", Precision: 10},
		},
		{
			spec:     specutil.NewCol("float", "float", specutil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: "double precision", Precision: 25},
		},
		{
			spec:     specutil.NewCol("cidr", "cidr"),
			expected: &NetworkType{T: "cidr"},
		},
		{
			spec:     specutil.NewCol("money", "money"),
			expected: &CurrencyType{T: "money"},
		},
		{
			spec:     specutil.NewCol("bit", "bit"),
			expected: &BitType{T: "bit", Len: 1},
		},
		{
			spec:     specutil.NewCol("bitvar", "bit varying"),
			expected: &BitType{T: "bit varying"},
		},
		{
			spec:     specutil.NewCol("bitvar8", "bit varying(8)"),
			expected: &BitType{T: "bit varying", Len: 8},
		},
		{
			spec:     specutil.NewCol("bit8", "bit(8)"),
			expected: &BitType{T: "bit", Len: 8},
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
	body := fmt.Sprintf(tmpl, string(buf))
	return []byte(body)
}
