package postgres

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
		Tables: []*schemaspec.Table{
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
							T:    "integer",
							Size: 4,
						},
					},
					Spec: spec.Tables[0].Columns[0],
				},
				{
					Name: "age",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T:    "integer",
							Size: 4,
						},
					},
					Spec: spec.Tables[0].Columns[1],
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "varchar",
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
							T:    "varchar",
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
		spec        *schemaspec.Column
		expected    schema.Type
		expectedErr string
	}{
		{
			spec: schemautil.ColSpec("int", "int"),
			expected: &schema.IntegerType{
				T:        "integer",
				Unsigned: false,
				Size:     4,
			},
		},
		{
			spec:        schemautil.ColSpec("uint", "uint"),
			expectedErr: "postgres: unsigned integers currently not supported",
		},
		{
			spec:        schemautil.ColSpec("int8", "int8"),
			expectedErr: "postgres: 8-bit integers not supported",
		},
		{
			spec: schemautil.ColSpec("int64", "int64"),
			expected: &schema.IntegerType{
				T:        "bigint",
				Unsigned: false,
				Size:     8,
			},
		},
		{
			spec:        schemautil.ColSpec("uint64", "uint64"),
			expectedErr: "postgres: unsigned integers currently not supported",
		},
		{
			spec: schemautil.ColSpec("string_varchar", "string", schemautil.LitAttr("size", "255")),
			expected: &schema.StringType{
				T:    "varchar",
				Size: 255,
			},
		},
		{
			spec: schemautil.ColSpec("string_test", "string", schemautil.LitAttr("size", "10485761")),
			expected: &schema.StringType{
				T:    "text",
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
