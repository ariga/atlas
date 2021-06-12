package mysql

import (
	"strconv"
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
					{
						Name: "age",
						Type: "int",
					},
					{
						Name: "account_name",
						Type: "varchar(32)",
					},
				},
				PrimaryKey: &schema.PrimaryKeySpec{
					Columns: []*schema.ColumnRef{{Table: "table", Name: "col"}},
				},
				ForeignKeys: []*schema.ForeignKeySpec{
					{
						Symbol: "accounts",
						Columns: []*schema.ColumnRef{
							{Table: "table", Name: "account_name"},
						},
						RefColumns: []*schema.ColumnRef{
							{Table: "accounts", Name: "name"},
						},
						OnDelete: string(schema.SetNull),
					},
				},
				Indexes: []*schema.IndexSpec{
					{
						Name:   "index",
						Unique: true,
						Columns: []*schema.ColumnRef{
							{Table: "table", Name: "col"},
							{Table: "table", Name: "age"},
						},
					},
				},
			},
			{
				Name: "accounts",
				Columns: []*schema.ColumnSpec{
					{
						Name: "name",
						Type: "varchar(32)",
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
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T:    tInt,
							Size: 4,
						},
					},
					Spec: spec.Tables[0].Columns[1],
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    tVarchar,
							Size: 32,
						},
					},
					Spec: spec.Tables[0].Columns[2],
				},
			},
		},
		{
			Name:   "accounts",
			Spec:   spec.Tables[1],
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
					Spec: spec.Tables[1].Columns[0],
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
			expected: &schema.FloatType{T: "float", Precision: 10},
		},
		{
			spec:     colspec("float", "float", attr("precision", "25")),
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
		V: &schema.LiteralValue{V: v},
	}
}

func listattr(k string, values ...string) *schema.SpecAttr {
	for i, v := range values {
		values[i] = strconv.Quote(v)
	}
	return &schema.SpecAttr{
		K: k,
		V: &schema.ListValue{V: values},
	}
}
