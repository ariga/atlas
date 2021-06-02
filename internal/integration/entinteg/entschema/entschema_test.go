package entschema

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	graph, err := entc.LoadGraph("../ent/schema", &gen.Config{})
	require.NoError(t, err)
	sch, err := Convert(graph)
	require.NoError(t, err)
	tbl, ok := sch.Table("users")
	require.True(t, ok, "expected users table to exist")
	require.EqualValues(t, "users", tbl.Name)
	for _, tt := range []struct {
		fld      string
		expected *schema.ColumnType
	}{
		{
			fld: "name",
			expected: &schema.ColumnType{
				Type: &schema.StringType{
					T: "string",
				},
			},
		},
		{
			fld: "optional",
			expected: &schema.ColumnType{
				Type: &schema.StringType{
					T: "string",
				},
				Null: true,
			},
		},
		{
			fld: "int",
			expected: &schema.ColumnType{
				Type: &schema.IntegerType{
					T: "integer",
				},
			},
		},
		{
			fld: "uint",
			expected: &schema.ColumnType{
				Type: &schema.IntegerType{
					T:        "integer",
					Unsigned: true,
				},
			},
		},
		{
			fld: "time",
			expected: &schema.ColumnType{
				Type: &schema.TimeType{T: "time"},
			},
		},
		{
			fld: "bool",
			expected: &schema.ColumnType{
				Type: &schema.BoolType{T: "boolean"},
			},
		},
		{
			fld: "enum",
			expected: &schema.ColumnType{
				Type: &schema.EnumType{Values: []string{"1", "2", "3"}},
			},
		},
		{
			fld: "enum_2",
			expected: &schema.ColumnType{
				Type: &schema.EnumType{Values: []string{"1", "2", "3"}},
			},
		},
		{
			fld: "uuid",
			expected: &schema.ColumnType{
				Type: &schema.BinaryType{
					T:    "binary",
					Size: 16,
				},
			},
		},
		{
			fld: "bytes",
			expected: &schema.ColumnType{
				Type: &schema.BinaryType{
					T: "binary",
				},
			},
		},
	} {
		t.Run(tt.fld, func(t *testing.T) {
			column, ok := tbl.Column(tt.fld)
			require.True(t, ok, "expected column to exist")
			require.EqualValues(t, tt.expected, column.Type)
			//out, err := convertColumn(tt.fld)
			//require.NoError(t, err)
			//require.EqualValues(t, tt.expected, out)
		})
	}
}
