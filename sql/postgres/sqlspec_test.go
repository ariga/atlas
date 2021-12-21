package postgres

import (
	"fmt"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/spectest"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

var hclState = schemahcl.New(schemahcl.WithTypes(TypeRegistry.Specs()))

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
	err := UnmarshalSpec([]byte(f), hclState, &s)
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
			expected: &BitType{T: tBit, Len: 10},
		},
		{
			typeExpr: "bit_varying(10)",
			expected: &BitType{T: tBitVar, Len: 10},
		},
		{
			typeExpr: "boolean",
			expected: &schema.BoolType{T: tBoolean},
		},
		{
			typeExpr: "bool",
			expected: &schema.BoolType{T: tBool},
		},
		{
			typeExpr: "bytea",
			expected: &schema.BinaryType{T: tBytea},
		},
		{
			typeExpr: "varchar(255)",
			expected: &schema.StringType{T: tVarChar, Size: 255},
		},
		{
			typeExpr: "char(255)",
			expected: &schema.StringType{T: tChar, Size: 255},
		},
		{
			typeExpr: "character(255)",
			expected: &schema.StringType{T: tCharacter, Size: 255},
		},
		{
			typeExpr: "text",
			expected: &schema.StringType{T: tText},
		},
		{
			typeExpr: "smallint",
			expected: &schema.IntegerType{T: tSmallInt},
		},
		{
			typeExpr: "integer",
			expected: &schema.IntegerType{T: tInteger},
		},
		{
			typeExpr: "bigint",
			expected: &schema.IntegerType{T: tBigInt},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: tInt},
		},
		{
			typeExpr: "int2",
			expected: &schema.IntegerType{T: tInt2},
		},
		{
			typeExpr: "int4",
			expected: &schema.IntegerType{T: tInt4},
		},
		{
			typeExpr: "int8",
			expected: &schema.IntegerType{T: tInt8},
		},
		{
			typeExpr: "cidr",
			expected: &NetworkType{T: tCIDR},
		},
		{
			typeExpr: "inet",
			expected: &NetworkType{T: tInet},
		},
		{
			typeExpr: "macaddr",
			expected: &NetworkType{T: tMACAddr},
		},
		{
			typeExpr: "macaddr8",
			expected: &NetworkType{T: tMACAddr8},
		},
		{
			typeExpr: "circle",
			expected: &schema.SpatialType{T: tCircle},
		},
		{
			typeExpr: "line",
			expected: &schema.SpatialType{T: tLine},
		},
		{
			typeExpr: "lseg",
			expected: &schema.SpatialType{T: tLseg},
		},
		{
			typeExpr: "box",
			expected: &schema.SpatialType{T: tBox},
		},
		{
			typeExpr: "path",
			expected: &schema.SpatialType{T: tPath},
		},
		{
			typeExpr: "point",
			expected: &schema.SpatialType{T: tPoint},
		},

		{
			typeExpr: "date",
			expected: &schema.TimeType{T: tDate},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: tTime},
		},
		{
			typeExpr: "time_with_time_zone",
			expected: &schema.TimeType{T: tTimeWTZ},
		},
		{
			typeExpr: "time_without_time_zone",
			expected: &schema.TimeType{T: tTimeWOTZ},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: tTimestamp},
		},
		{
			typeExpr: "timestamp_with_time_zone",
			expected: &schema.TimeType{T: tTimestampWTZ},
		},
		{
			typeExpr: "timestamp_without_time_zone",
			expected: &schema.TimeType{T: tTimestampWOTZ},
		},
		{
			typeExpr: "time",
			expected: &schema.TimeType{T: tTime},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: tReal, Precision: 24},
		},
		{
			typeExpr: "float8",
			expected: &schema.FloatType{T: tFloat8, Precision: 53},
		},
		{
			typeExpr: "float4",
			expected: &schema.FloatType{T: tFloat4, Precision: 24},
		},
		{
			typeExpr: "numeric",
			expected: &schema.DecimalType{T: tNumeric},
		},
		{
			typeExpr: "decimal",
			expected: &schema.DecimalType{T: tDecimal},
		},
		{
			typeExpr: "smallserial",
			expected: &SerialType{T: tSmallSerial},
		},
		{
			typeExpr: "serial",
			expected: &SerialType{T: tSerial},
		},
		{
			typeExpr: "bigserial",
			expected: &SerialType{T: tBigSerial},
		},
		{
			typeExpr: "serial2",
			expected: &SerialType{T: tSerial2},
		},
		{
			typeExpr: "serial4",
			expected: &SerialType{T: tSerial4},
		},
		{
			typeExpr: "serial8",
			expected: &SerialType{T: tSerial8},
		},

		{
			typeExpr: "xml",
			expected: &XMLType{T: tXML},
		},
		{
			typeExpr: "json",
			expected: &schema.JSONType{T: tJSON},
		},
		{
			typeExpr: "jsonb",
			expected: &schema.JSONType{T: tJSONB},
		},
		{
			typeExpr: "uuid",
			expected: &UUIDType{T: tUUID},
		},
		{
			typeExpr: "money",
			expected: &CurrencyType{T: tMoney},
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
			err := UnmarshalSpec([]byte(doc), hclState, &test)
			require.NoError(t, err)
			colspec := test.Tables[0].Columns[0]
			require.EqualValues(t, tt.expected, colspec.Type.Type)
			spec, err := MarshalSpec(&test, hclState)
			require.NoError(t, err)
			var after schema.Schema
			err = UnmarshalSpec(spec, hclState, &after)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, after.Tables[0].Columns[0].Type.Type)
		})
	}
}

func TestRegistrySanity(t *testing.T) {
	spectest.RegistrySanityTest(t, TypeRegistry, []string{"enum"})
}
