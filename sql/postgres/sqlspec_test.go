package postgres

import (
	"fmt"
	"log"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

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
				T:    "varchar",
				Size: 10_485_761,
			},
		},
		{
			spec: schemautil.ColSpec("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("decimal(10, 2)", "decimal(10, 2)"),
			expected: &schema.DecimalType{
				T:         "decimal",
				Scale:     2,
				Precision: 10,
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
			expected: &schema.FloatType{T: "real", Precision: 10},
		},
		{
			spec:     schemautil.ColSpec("float", "float", schemautil.LitAttr("precision", "25")),
			expected: &schema.FloatType{T: "double precision", Precision: 25},
		},
		{
			spec:     schemautil.ColSpec("cidr", "cidr"),
			expected: &NetworkType{T: "cidr"},
		},
		{
			spec:     schemautil.ColSpec("money", "money"),
			expected: &CurrencyType{T: "money"},
		},
		{
			spec:     schemautil.ColSpec("bit", "bit"),
			expected: &BitType{T: "bit", Len: 1},
		},
		{
			spec:     schemautil.ColSpec("bitvar", "bit varying"),
			expected: &BitType{T: "bit varying"},
		},
		{
			spec:     schemautil.ColSpec("bitvar8", "bit varying(8)"),
			expected: &BitType{T: "bit varying", Len: 8},
		},
		{
			spec:     schemautil.ColSpec("bit8", "bit(8)"),
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

// hcl returns an Atlas HCL document containing the column spec.
func hcl(c *sqlspec.Column) []byte {
	mm, err := schemahcl.Marshal(c)
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
	body := fmt.Sprintf(tmpl, string(mm))
	return []byte(body)
}

//{
//	spec:        schemautil.ColSpec("uint", "uint"),
//	expectedErr: "postgres: unsigned integers currently not supported",
//},
//{
//	spec:        schemautil.ColSpec("int8", "int8"),
//	expectedErr: "postgres: 8-bit integers not supported",
//},

//{
//	spec:        schemautil.ColSpec("uint64", "uint64"),
//	expectedErr: "postgres: unsigned integers currently not supported",
//},
