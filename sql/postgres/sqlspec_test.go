package postgres

import (
	"fmt"
	"testing"

	"ariga.io/atlas/sql/internal/spectest"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestSQLSpec(t *testing.T) {
	f := `
schema "schema" {
}

table "table" {
	column "col" {
		type = integer
	}
	column "age" {
		type = integer
	}
	column "account_name" {
		type = varchar(32)
	}
	column "tags" {
		type = hstore
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
		type = varchar(32)
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
}
`
	var s schema.Schema
	err := UnmarshalHCL([]byte(f), &s)
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
				{
					Name: "tags",
					Type: &schema.ColumnType{
						Type: &UserDefinedType{
							T: "hstore",
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
	require.EqualValues(t, exp, &s)
}

func TestTypes(t *testing.T) {
	// TODO(rotemtam): enum, timestamptz, interval
	for _, tt := range []struct {
		typeExpr string
		expected schema.Type
	}{
		{
			typeExpr: "bit(10)",
			expected: &BitType{T: TypeBit, Len: 10},
		},
		{
			typeExpr: `hstore`,
			expected: &UserDefinedType{T: "hstore"},
		},
		{
			typeExpr: "bit_varying(10)",
			expected: &BitType{T: TypeBitVar, Len: 10},
		},
		{
			typeExpr: "boolean",
			expected: &schema.BoolType{T: TypeBoolean},
		},
		{
			typeExpr: "bool",
			expected: &schema.BoolType{T: TypeBool},
		},
		{
			typeExpr: "bytea",
			expected: &schema.BinaryType{T: TypeBytea},
		},
		{
			typeExpr: "varchar(255)",
			expected: &schema.StringType{T: TypeVarChar, Size: 255},
		},
		{
			typeExpr: "char(255)",
			expected: &schema.StringType{T: TypeChar, Size: 255},
		},
		{
			typeExpr: "character(255)",
			expected: &schema.StringType{T: TypeCharacter, Size: 255},
		},
		{
			typeExpr: "text",
			expected: &schema.StringType{T: TypeText},
		},
		{
			typeExpr: "smallint",
			expected: &schema.IntegerType{T: TypeSmallInt},
		},
		{
			typeExpr: "integer",
			expected: &schema.IntegerType{T: TypeInteger},
		},
		{
			typeExpr: "bigint",
			expected: &schema.IntegerType{T: TypeBigInt},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr: "int2",
			expected: &schema.IntegerType{T: TypeInt2},
		},
		{
			typeExpr: "int4",
			expected: &schema.IntegerType{T: TypeInt4},
		},
		{
			typeExpr: "int8",
			expected: &schema.IntegerType{T: TypeInt8},
		},
		{
			typeExpr: "cidr",
			expected: &NetworkType{T: TypeCIDR},
		},
		{
			typeExpr: "inet",
			expected: &NetworkType{T: TypeInet},
		},
		{
			typeExpr: "macaddr",
			expected: &NetworkType{T: TypeMACAddr},
		},
		{
			typeExpr: "macaddr8",
			expected: &NetworkType{T: TypeMACAddr8},
		},
		{
			typeExpr: "circle",
			expected: &schema.SpatialType{T: TypeCircle},
		},
		{
			typeExpr: "line",
			expected: &schema.SpatialType{T: TypeLine},
		},
		{
			typeExpr: "lseg",
			expected: &schema.SpatialType{T: TypeLseg},
		},
		{
			typeExpr: "box",
			expected: &schema.SpatialType{T: TypeBox},
		},
		{
			typeExpr: "path",
			expected: &schema.SpatialType{T: TypePath},
		},
		{
			typeExpr: "point",
			expected: &schema.SpatialType{T: TypePoint},
		},

		{
			typeExpr: "date",
			expected: &schema.TimeType{T: TypeDate},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: TypeTime},
		},
		{
			typeExpr: "time_with_time_zone",
			expected: &schema.TimeType{T: TypeTimeWTZ},
		},
		{
			typeExpr: "time_without_time_zone",
			expected: &schema.TimeType{T: TypeTimeWOTZ},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: TypeTimestamp},
		},
		{
			typeExpr: "timestamp_with_time_zone",
			expected: &schema.TimeType{T: TypeTimestampWTZ},
		},
		{
			typeExpr: "timestamp_without_time_zone",
			expected: &schema.TimeType{T: TypeTimestampWOTZ},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: TypeTime},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: TypeReal, Precision: 24},
		},
		{
			typeExpr: "float8",
			expected: &schema.FloatType{T: TypeFloat8, Precision: 53},
		},
		{
			typeExpr: "float4",
			expected: &schema.FloatType{T: TypeFloat4, Precision: 24},
		},
		{
			typeExpr: "numeric",
			expected: &schema.DecimalType{T: TypeNumeric},
		},
		{
			typeExpr: "decimal",
			expected: &schema.DecimalType{T: TypeDecimal},
		},
		{
			typeExpr: "smallserial",
			expected: &SerialType{T: TypeSmallSerial},
		},
		{
			typeExpr: "serial",
			expected: &SerialType{T: TypeSerial},
		},
		{
			typeExpr: "bigserial",
			expected: &SerialType{T: TypeBigSerial},
		},
		{
			typeExpr: "serial2",
			expected: &SerialType{T: TypeSerial2},
		},
		{
			typeExpr: "serial4",
			expected: &SerialType{T: TypeSerial4},
		},
		{
			typeExpr: "serial8",
			expected: &SerialType{T: TypeSerial8},
		},

		{
			typeExpr: "xml",
			expected: &XMLType{T: TypeXML},
		},
		{
			typeExpr: "json",
			expected: &schema.JSONType{T: TypeJSON},
		},
		{
			typeExpr: "jsonb",
			expected: &schema.JSONType{T: TypeJSONB},
		},
		{
			typeExpr: "uuid",
			expected: &UUIDType{T: TypeUUID},
		},
		{
			typeExpr: "money",
			expected: &CurrencyType{T: TypeMoney},
		},
		{
			typeExpr: `sql("int[]")`,
			expected: &ArrayType{T: "int[]"},
		},
	} {
		t.Run(tt.typeExpr, func(t *testing.T) {
			var test schema.Schema
			doc := fmt.Sprintf(`table "test" {
	schema = schema.test
	column "test" {
		null = false
		type = %s
	}
}
schema "test" {
}
`, tt.typeExpr)
			err := UnmarshalHCL([]byte(doc), &test)
			require.NoError(t, err)
			colspec := test.Tables[0].Columns[0]
			require.EqualValues(t, tt.expected, colspec.Type.Type)
			spec, err := MarshalHCL(&test)
			require.NoError(t, err)
			var after schema.Schema
			err = UnmarshalHCL(spec, &after)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, after.Tables[0].Columns[0].Type.Type)
		})
	}
}

func TestRegistrySanity(t *testing.T) {
	spectest.RegistrySanityTest(t, TypeRegistry, []string{"enum"})
}
