package mysql

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestConvertSchema(t *testing.T) {
	spec := &schema.SchemaSpec{
		Name: "schema",
		Tables: []*schema.TableSpec{
			{
				Name: "table",
				Columns: []*schema.ColumnSpec{
					{
						Name: "col",
						Type: "int",
					},
				},
			},
		},
	}
	sch, err := ConvertSchema(spec)
	require.NoError(t, err)
	exp := &schema.Schema{
		Name: "schema",
		Spec: spec,
	}
	exp.Tables = []*schema.Table{
		{
			Name:   "table",
			Schema: exp,
			Spec:   spec.Tables[0],
			Columns: []*schema.Column{
				{
					Name: "col",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T:    tInt,
							Size: 4,
						},
					},
					Spec: spec.Tables[0].Columns[0],
				},
			},
		},
	}
	require.EqualValues(t, exp, sch)
}

func TestConverter(t *testing.T) {
	for _, tt := range []struct {
		spec     *schema.ColumnSpec
		expected schema.Type
	}{
		{
			spec: colspec("int", "int"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: false,
				Size:     4,
			},
		},
		{
			spec: colspec("uint", "uint"),
			expected: &schema.IntegerType{
				T:        tInt,
				Unsigned: true,
				Size:     4,
			},
		},
		{
			spec: colspec("int8", "int8"),
			expected: &schema.IntegerType{
				T:        tTinyInt,
				Unsigned: false,
				Size:     1,
			},
		},
		{
			spec: colspec("int64", "int64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: false,
				Size:     8,
			},
		},
		{
			spec: colspec("uint64", "uint64"),
			expected: &schema.IntegerType{
				T:        tBigInt,
				Unsigned: true,
				Size:     8,
			},
		},
		{
			spec: colspec("string_varchar", "string", attr("size", "255")),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: colspec("string_mediumtext", "string", attr("size", "100000")),
			expected: &schema.StringType{
				T:    tMediumText,
				Size: 100_000,
			},
		},
		{
			spec: colspec("string_longtext", "string", attr("size", "17000000")),
			expected: &schema.StringType{
				T:    tLongText,
				Size: 17_000_000,
			},
		},
		{
			spec: colspec("varchar(255)", "varchar(255)"),
			expected: &schema.StringType{
				T:    tVarchar,
				Size: 255,
			},
		},
		{
			spec: colspec("decimal(10, 2) unsigned", "decimal(10, 2) unsigned"),
			expected: &schema.DecimalType{
				T:         tDecimal,
				Scale:     2,
				Precision: 10,
			},
		},
		{
			spec: colspec("blob", "binary"),
			expected: &schema.BinaryType{
				T: tBlob,
			},
		},
		{
			spec: colspec("tinyblob", "binary", attr("size", "16")),
			expected: &schema.BinaryType{
				T:    tTinyBlob,
				Size: 16,
			},
		},
		{
			spec: colspec("mediumblob", "binary", attr("size", "100000")),
			expected: &schema.BinaryType{
				T:    tMediumBlob,
				Size: 100_000,
			},
		},
		{
			spec: colspec("longblob", "binary", attr("size", "20000000")),
			expected: &schema.BinaryType{
				T:    tLongBlob,
				Size: 20_000_000,
			},
		},
	} {
		t.Run(tt.spec.Name, func(t *testing.T) {
			columnType, err := ConvertColumnType(tt.spec)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, columnType)
		})
	}
}

func colspec(name, coltype string, attrs ...*schema.SpecAttr) *schema.ColumnSpec {
	return &schema.ColumnSpec{
		Name:  name,
		Type:  coltype,
		Attrs: attrs,
	}
}

func attr(k, v string) *schema.SpecAttr {
	return &schema.SpecAttr{
		K: k,
		V: &schema.SpecLiteral{V: v},
	}
}
