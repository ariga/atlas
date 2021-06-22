package postgres

import (
	"strconv"
	"testing"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestConvertColumnType(t *testing.T) {
	for _, tt := range []struct {
		spec        *schemaspec.Column
		expected    schema.Type
		expectedErr string
	}{
		{
			spec: colspec("int", "int"),
			expected: &schema.IntegerType{
				T:        "integer",
				Unsigned: false,
				Size:     4,
			},
		},
		{
			spec:        colspec("uint", "uint"),
			expectedErr: "postgres: unsigned integers currently not supported",
		},
		{
			spec:        colspec("int8", "int8"),
			expectedErr: "postgres: 8-bit integers not supported",
		},
		{
			spec: colspec("int64", "int64"),
			expected: &schema.IntegerType{
				T:        "bigint",
				Unsigned: false,
				Size:     8,
			},
		},
		{
			spec:        colspec("uint64", "uint64"),
			expectedErr: "postgres: unsigned integers currently not supported",
		},
		{
			spec: colspec("string_varchar", "string", attr("size", "255")),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: colspec("string_test", "string", attr("size", "10485761")),
			expected: &schema.StringType{
				T:    "text",
				Size: 10_485_761,
			},
		},
		{
			spec: colspec("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: colspec("decimal(10, 2)", "decimal(10, 2)"),
			expected: &schema.DecimalType{
				T:         "decimal",
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec:     colspec("enum", "enum", listattr("values", "a", "b", "c")),
			expected: &schema.EnumType{Values: []string{"a", "b", "c"}},
		},
		{
			spec:     colspec("bool", "boolean"),
			expected: &schema.BoolType{T: "boolean"},
		},
		{
			spec:     colspec("decimal", "decimal", attr("precision", "10"), attr("scale", "2")),
			expected: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
		},
		{
			spec:     colspec("float", "float", attr("precision", "10")),
			expected: &schema.FloatType{T: "real", Precision: 10},
		},
		{
			spec:     colspec("float", "float", attr("precision", "25")),
			expected: &schema.FloatType{T: "double precision", Precision: 25},
		},
		{
			spec:     colspec("cidr", "cidr"),
			expected: &NetworkType{T: "cidr"},
		},
		{
			spec:     colspec("money", "money"),
			expected: &CurrencyType{T: "money"},
		},
		{
			spec:     colspec("bit", "bit"),
			expected: &BitType{T: "bit", Len: 1},
		},
		{
			spec:     colspec("bitvar", "bit varying"),
			expected: &BitType{T: "bit varying"},
		},
		{
			spec:     colspec("bitvar8", "bit varying(8)"),
			expected: &BitType{T: "bit varying", Len: 8},
		},
		{
			spec:     colspec("bit8", "bit(8)"),
			expected: &BitType{T: "bit", Len: 8},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			columnType, err := ConvertColumnType(tt.spec)
			if tt.expectedErr != "" && err != nil {
				require.Equal(t, tt.expectedErr, err.Error())
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, columnType)
		})
	}
}

func colspec(name, coltype string, attrs ...*schemaspec.Attr) *schemaspec.Column {
	return &schemaspec.Column{
		Name:  name,
		Type:  coltype,
		Attrs: attrs,
	}
}

func attr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: v},
	}
}

func listattr(k string, values ...string) *schemaspec.Attr {
	for i, v := range values {
		values[i] = strconv.Quote(v)
	}
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.ListValue{V: values},
	}
}
