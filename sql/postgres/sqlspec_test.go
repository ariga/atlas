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
		comment = "column comment"
	}
	column "age" {
		type = integer
	}
	column "price" {
		type = int
	}
	column "account_name" {
		type = varchar(32)
	}
	column "varchar_length_is_not_required" {
		type = varchar
	}
	column "character_varying_length_is_not_required" {
		type = character_varying
	}
	column "tags" {
		type = hstore
	}
	column "created_at" {
		type    = timestamp(4)
		default = sql("current_timestamp(4)")
	}
	column "updated_at" {
		type    = time
		default = sql("current_time")
	}
	primary_key {
		columns = [table.table.column.col]
	}
	index "index" {
		type = HASH
		unique = true
		columns = [
			table.table.column.col,
			table.table.column.age,
		]
		comment = "index comment"
	}
	foreign_key "accounts" {
		columns = [
			table.table.column.account_name,
		]
		ref_columns = [
			table.accounts.column.name,
		]
		on_delete = SET_NULL
	}
	check "positive price" {
		expr = "price > 0"
	}
	comment = "table comment"
}

table "accounts" {
	column "name" {
		type = varchar(32)
	}
	column "type" {
		type = enum.account_type
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
}

enum "account_type" {
	values = ["private", "business"]
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
					Attrs: []schema.Attr{
						&schema.Comment{Text: "column comment"},
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
					Name: "price",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: TypeInt,
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
					Name: "varchar_length_is_not_required",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "varchar",
							Size: 0,
						},
					},
				},
				{
					Name: "character_varying_length_is_not_required",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "character varying",
							Size: 0,
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
				{
					Name: "created_at",
					Type: &schema.ColumnType{
						Type: &schema.TimeType{
							T:         TypeTimestamp,
							Precision: 4,
						},
					},
					Default: &schema.RawExpr{X: "current_timestamp(4)"},
				},
				{
					Name: "updated_at",
					Type: &schema.ColumnType{
						Type: &schema.TimeType{
							T:         TypeTime,
							Precision: 6,
						},
					},
					Default: &schema.RawExpr{X: "current_time"},
				},
			},
			Attrs: []schema.Attr{
				&schema.Check{
					Name: "positive price",
					Expr: "price > 0",
				},
				&schema.Comment{Text: "table comment"},
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
				{
					Name: "type",
					Type: &schema.ColumnType{
						Type: &schema.EnumType{
							T:      "account_type",
							Values: []string{"private", "business"},
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
			Attrs: []schema.Attr{
				&schema.Comment{Text: "index comment"},
				&IndexType{T: IndexTypeHash},
			},
		},
	}
	exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
		{
			Symbol:     "accounts",
			Table:      exp.Tables[0],
			Columns:    []*schema.Column{exp.Tables[0].Columns[3]},
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

func TestUnmarshalSpec_IndexType(t *testing.T) {
	f := `
schema "s" {}
table "t" {
	schema = schema.s
	column "c" {
		type = int
	}
	index "i" {
		type = %s
		columns = [column.c]
	}
}
`
	t.Run("Invalid", func(t *testing.T) {
		f := fmt.Sprintf(f, "UNK")
		err := UnmarshalHCL([]byte(f), &schema.Schema{})
		require.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		var (
			s schema.Schema
			f = fmt.Sprintf(f, "HASH")
		)
		err := UnmarshalHCL([]byte(f), &s)
		require.NoError(t, err)
		idx := s.Tables[0].Indexes[0]
		require.Equal(t, IndexTypeHash, idx.Attrs[0].(*IndexType).T)
	})
}

func TestUnmarshalSpec_Identity(t *testing.T) {
	f := `
schema "s" {}
table "t" {
	schema = schema.s
	column "c" {
		type = int
		identity {
			generated = %s
			start = 10
		}
	}
}
`
	t.Run("Invalid", func(t *testing.T) {
		f := fmt.Sprintf(f, "UNK")
		err := UnmarshalHCL([]byte(f), &schema.Schema{})
		require.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		var (
			s schema.Schema
			f = fmt.Sprintf(f, "ALWAYS")
		)
		err := UnmarshalHCL([]byte(f), &s)
		require.NoError(t, err)
		id := s.Tables[0].Columns[0].Attrs[0].(*Identity)
		require.Equal(t, GeneratedTypeAlways, id.Generation)
		require.EqualValues(t, 10, id.Sequence.Start)
		require.Zero(t, id.Sequence.Increment)
	})
}

func TestMarshalSpec_Enum(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("account").
				AddColumns(
					schema.NewEnumColumn("account_type",
						schema.EnumName("account_type"),
						schema.EnumValues("private", "business"),
					),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "account" {
  schema = schema.test
  column "account_type" {
    null = false
    type = enum.account_type
  }
}
schema "test" {
}
enum "account_type" {
  schema = schema.test
  values = ["private", "business"]
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_TimePrecision(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("times").
				AddColumns(
					schema.NewTimeColumn("t_time_def", TypeTime),
					schema.NewTimeColumn("t_time_with_time_zone", TypeTimeWTZ, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_time_without_time_zone", TypeTimeWOTZ, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamp", TypeTimestamp, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamptz", TypeTimestampTZ, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamp_with_time_zone", TypeTimestampWTZ, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamp_without_time_zone", TypeTimestampWOTZ, schema.TimePrecision(2)),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "times" {
  schema = schema.test
  column "t_time_def" {
    null = false
    type = time
  }
  column "t_time_with_time_zone" {
    null = false
    type = time_with_time_zone(2)
  }
  column "t_time_without_time_zone" {
    null = false
    type = time_without_time_zone(2)
  }
  column "t_timestamp" {
    null = false
    type = timestamp(2)
  }
  column "t_timestamptz" {
    null = false
    type = timestamptz(2)
  }
  column "t_timestamp_with_time_zone" {
    null = false
    type = timestamp_with_time_zone(2)
  }
  column "t_timestamp_without_time_zone" {
    null = false
    type = timestamp_without_time_zone(2)
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestTypes(t *testing.T) {
	// TODO(rotemtam) interval
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
			expected: &schema.TimeType{T: TypeTime, Precision: 6},
		},
		{
			typeExpr: "time(4)",
			expected: &schema.TimeType{T: TypeTime, Precision: 4},
		},
		{
			typeExpr: "time_with_time_zone",
			expected: &schema.TimeType{T: TypeTimeWTZ, Precision: 6},
		},
		{
			typeExpr: "time_with_time_zone(4)",
			expected: &schema.TimeType{T: TypeTimeWTZ, Precision: 4},
		},
		{
			typeExpr: "time_without_time_zone",
			expected: &schema.TimeType{T: TypeTimeWOTZ, Precision: 6},
		},
		{
			typeExpr: "time_without_time_zone(4)",
			expected: &schema.TimeType{T: TypeTimeWOTZ, Precision: 4},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: TypeTimestamp, Precision: 6},
		},
		{
			typeExpr: "timestamp(4)",
			expected: &schema.TimeType{T: TypeTimestamp, Precision: 4},
		},
		{
			typeExpr: "timestamptz",
			expected: &schema.TimeType{T: TypeTimestampTZ, Precision: 6},
		},
		{
			typeExpr: "timestamptz(4)",
			expected: &schema.TimeType{T: TypeTimestampTZ, Precision: 4},
		},
		{
			typeExpr: "timestamp_with_time_zone",
			expected: &schema.TimeType{T: TypeTimestampWTZ, Precision: 6},
		},
		{
			typeExpr: "timestamp_with_time_zone(4)",
			expected: &schema.TimeType{T: TypeTimestampWTZ, Precision: 4},
		},
		{
			typeExpr: "timestamp_without_time_zone",
			expected: &schema.TimeType{T: TypeTimestampWOTZ, Precision: 6},
		},
		{
			typeExpr: "timestamp_without_time_zone(4)",
			expected: &schema.TimeType{T: TypeTimestampWOTZ, Precision: 4},
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
		{
			typeExpr: `sql("int[2]")`,
			expected: &ArrayType{T: "int[]"},
		},
		{
			typeExpr: `sql("text[][]")`,
			expected: &ArrayType{T: "text[]"},
		},
		{
			typeExpr: `sql("integer [3][3]")`,
			expected: &ArrayType{T: "integer[]"},
		},
		{
			typeExpr: `sql("integer ARRAY[4]")`,
			expected: &ArrayType{T: "integer[]"},
		},
		{
			typeExpr: `sql("integer ARRAY")`,
			expected: &ArrayType{T: "integer[]"},
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
