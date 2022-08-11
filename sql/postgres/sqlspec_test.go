// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"fmt"
	"strconv"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/spectest"
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
	schema = schema.schema
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
		where = "active"
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
	schema = schema.schema
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
	schema = schema.schema
	values = ["private", "business"]
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
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
						Type: typeTime(TypeTimestamp, 4),
					},
					Default: &schema.RawExpr{X: "current_timestamp(4)"},
				},
				{
					Name: "updated_at",
					Type: &schema.ColumnType{
						Type: typeTime(TypeTime, 6),
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
							Schema: exp,
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
				&IndexPredicate{P: "active"},
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
	exp.Realm = schema.NewRealm(exp)
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
		err := EvalHCLBytes([]byte(f), &schema.Schema{}, nil)
		require.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		var (
			s schema.Schema
			f = fmt.Sprintf(f, "HASH")
		)
		err := EvalHCLBytes([]byte(f), &s, nil)
		require.NoError(t, err)
		idx := s.Tables[0].Indexes[0]
		require.Equal(t, IndexTypeHash, idx.Attrs[0].(*IndexType).T)
	})
}

func TestUnmarshalSpec_BRINIndex(t *testing.T) {
	f := `
schema "s" {}
table "t" {
	schema = schema.s
	column "c" {
		type = int
	}
	index "i" {
		type = BRIN
		columns = [column.c]
		page_per_range = 2
	}
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	idx := s.Tables[0].Indexes[0]
	require.Equal(t, IndexTypeBRIN, idx.Attrs[0].(*IndexType).T)
	require.EqualValues(t, 2, idx.Attrs[1].(*IndexStorageParams).PagesPerRange)
}

func TestUnmarshalSpec_Partitioned(t *testing.T) {
	t.Run("Columns", func(t *testing.T) {
		var (
			s = &schema.Schema{}
			f = `
schema "test" {}
table "logs" {
	schema = schema.test
	column "name" {
		type = text
	}
	partition {
		type = HASH
		columns = [
			column.name
		]
	}
}
`
		)
		err := EvalHCLBytes([]byte(f), s, nil)
		require.NoError(t, err)
		c := schema.NewStringColumn("name", "text")
		expected := schema.New("test").
			AddTables(schema.NewTable("logs").AddColumns(c).AddAttrs(&Partition{T: PartitionTypeHash, Parts: []*PartitionPart{{C: c}}}))
		expected.SetRealm(schema.NewRealm(expected))
		require.Equal(t, expected, s)
	})

	t.Run("Parts", func(t *testing.T) {
		var (
			s = &schema.Schema{}
			f = `
schema "test" {}
table "logs" {
	schema = schema.test
	column "name" {
		type = text
	}
	partition {
		type = RANGE
		by {
			column = column.name
		}
		by {
			expr = "lower(name)"
		}
	}
}
`
		)
		err := EvalHCLBytes([]byte(f), s, nil)
		require.NoError(t, err)
		c := schema.NewStringColumn("name", "text")
		expected := schema.New("test").
			AddTables(schema.NewTable("logs").AddColumns(c).AddAttrs(&Partition{T: PartitionTypeRange, Parts: []*PartitionPart{{C: c}, {X: &schema.RawExpr{X: "lower(name)"}}}}))
		expected.SetRealm(schema.NewRealm(expected))
		require.Equal(t, expected, s)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := EvalHCLBytes([]byte(`
			schema "test" {}
			table "logs" {
				schema = schema.test
				column "name" { type = text }
				partition {
					columns = [column.name]
				}
			}
		`), &schema.Schema{}, nil)
		require.EqualError(t, err, "missing attribute logs.partition.type")

		err = EvalHCLBytes([]byte(`
			schema "test" {}
			table "logs" {
				schema = schema.test
				column "name" { type = text }
				partition {
					type = HASH
				}
			}
		`), &schema.Schema{}, nil)
		require.EqualError(t, err, `missing columns or expressions for logs.partition`)

		err = EvalHCLBytes([]byte(`
			schema "test" {}
			table "logs" {
				schema = schema.test
				column "name" { type = text }
				partition {
					type = HASH
					columns = [column.name]
					by { column = column.name }
				}
			}
		`), &schema.Schema{}, nil)
		require.EqualError(t, err, `multiple definitions for logs.partition, use "columns" or "by"`)
	})
}

func TestMarshalSpec_Partitioned(t *testing.T) {
	t.Run("Columns", func(t *testing.T) {
		c := schema.NewStringColumn("name", "text")
		s := schema.New("test").
			AddTables(schema.NewTable("logs").AddColumns(c).AddAttrs(&Partition{T: PartitionTypeHash, Parts: []*PartitionPart{{C: c}}}))
		buf, err := MarshalHCL(s)
		require.NoError(t, err)
		require.Equal(t, `table "logs" {
  schema = schema.test
  column "name" {
    null = false
    type = text
  }
  partition {
    type    = HASH
    columns = [column.name]
  }
}
schema "test" {
}
`, string(buf))
	})

	t.Run("Parts", func(t *testing.T) {
		c := schema.NewStringColumn("name", "text")
		s := schema.New("test").
			AddTables(schema.NewTable("logs").AddColumns(c).AddAttrs(&Partition{T: PartitionTypeHash, Parts: []*PartitionPart{{C: c}, {X: &schema.RawExpr{X: "lower(name)"}}}}))
		buf, err := MarshalHCL(s)
		require.NoError(t, err)
		require.Equal(t, `table "logs" {
  schema = schema.test
  column "name" {
    null = false
    type = text
  }
  partition {
    type = HASH
    by {
      column = column.name
    }
    by {
      expr = "lower(name)"
    }
  }
}
schema "test" {
}
`, string(buf))
	})
}

func TestMarshalSpec_IndexPredicate(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{
						Name: "id",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[0].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0]},
			},
			Attrs: []schema.Attr{
				&IndexPredicate{P: "id <> 0"},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "id" {
    null = false
    type = int
  }
  index "index" {
    unique  = true
    columns = [column.id]
    where   = "id <> 0"
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_BRINIndex(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{
						Name: "id",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[0].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0]},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeBRIN},
				&IndexStorageParams{PagesPerRange: 2},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "id" {
    null = false
    type = int
  }
  index "index" {
    unique         = true
    columns        = [column.id]
    type           = BRIN
    page_per_range = 2
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
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
		err := EvalHCLBytes([]byte(f), &schema.Schema{}, nil)
		require.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		var (
			s schema.Schema
			f = fmt.Sprintf(f, "ALWAYS")
		)
		err := EvalHCLBytes([]byte(f), &s, nil)
		require.NoError(t, err)
		id := s.Tables[0].Columns[0].Attrs[0].(*Identity)
		require.Equal(t, GeneratedTypeAlways, id.Generation)
		require.EqualValues(t, 10, id.Sequence.Start)
		require.Zero(t, id.Sequence.Increment)
	})
}

func TestMarshalSpec_GeneratedColumn(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c1", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c1 * 2"}),
					schema.NewIntColumn("c2", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c3 * c4", Type: "STORED"}),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "c1" {
    null = false
    type = int
    as {
      expr = "c1 * 2"
      type = STORED
    }
  }
  column "c2" {
    null = false
    type = int
    as {
      expr = "c3 * c4"
      type = STORED
    }
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestUnmarshalSpec_GeneratedColumns(t *testing.T) {
	var (
		s schema.Schema
		f = `
schema "test" {}
table "users" {
	schema = schema.test
	column "c1" {
		type = int
		as = "1"
	}
	column "c2" {
		type = int
		as {
			expr = "2"
		}
	}
	column "c3" {
		type = int
		as {
			expr = "3"
			type = STORED
		}
	}
}
`
	)
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	expected := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c1", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
					schema.NewIntColumn("c2", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "2", Type: "STORED"}),
					schema.NewIntColumn("c3", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "3", Type: "STORED"}),
				),
		)
	expected.SetRealm(schema.NewRealm(expected))
	require.EqualValues(t, expected, &s)
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
			schema.NewTable("table2").
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
table "table2" {
  schema = schema.test
  column "account_type" {
    null = false
    type = enum.account_type
  }
}
enum "account_type" {
  schema = schema.test
  values = ["private", "business"]
}
schema "test" {
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
					schema.NewTimeColumn("t_time_with_time_zone", TypeTimeTZ, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_time_without_time_zone", TypeTime, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamp", TypeTimestamp, schema.TimePrecision(2)),
					schema.NewTimeColumn("t_timestamptz", TypeTimestampTZ, schema.TimePrecision(2)),
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
    type = timetz(2)
  }
  column "t_time_without_time_zone" {
    null = false
    type = time(2)
  }
  column "t_timestamp" {
    null = false
    type = timestamp(2)
  }
  column "t_timestamptz" {
    null = false
    type = timestamptz(2)
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestTypes(t *testing.T) {
	p := func(i int) *int { return &i }
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
			expected: typeTime(TypeTime, 6),
		},
		{
			typeExpr: "time(4)",
			expected: typeTime(TypeTime, 4),
		},
		{
			typeExpr: "timetz",
			expected: typeTime(TypeTimeTZ, 6),
		},
		{
			typeExpr: "timestamp",
			expected: typeTime(TypeTimestamp, 6),
		},
		{
			typeExpr: "timestamp(4)",
			expected: typeTime(TypeTimestamp, 4),
		},
		{
			typeExpr: "timestamptz",
			expected: typeTime(TypeTimestampTZ, 6),
		},
		{
			typeExpr: "timestamptz(4)",
			expected: typeTime(TypeTimestampTZ, 4),
		},
		{
			typeExpr: "interval",
			expected: &IntervalType{T: "interval"},
		},
		{
			typeExpr: "interval(1)",
			expected: &IntervalType{T: "interval", Precision: p(1)},
		},
		{
			typeExpr: "second",
			expected: &IntervalType{T: "interval", F: "second"},
		},
		{
			typeExpr: "minute_to_second",
			expected: &IntervalType{T: "interval", F: "minute to second"},
		},
		{
			typeExpr: "minute_to_second(2)",
			expected: &IntervalType{T: "interval", F: "minute to second", Precision: p(2)},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: TypeReal, Precision: 24},
		},
		{
			typeExpr: "float",
			expected: &schema.FloatType{T: TypeFloat},
		},
		{
			typeExpr: "float(1)",
			expected: &schema.FloatType{T: TypeFloat, Precision: 1},
		},
		{
			typeExpr: "float(25)",
			expected: &schema.FloatType{T: TypeFloat, Precision: 25},
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
			typeExpr: "numeric(10)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10},
		},
		{
			typeExpr: "numeric(10, 2)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10, Scale: 2},
		},
		{
			typeExpr: "decimal",
			expected: &schema.DecimalType{T: TypeDecimal},
		},
		{
			typeExpr: "decimal(10)",
			expected: &schema.DecimalType{T: TypeDecimal, Precision: 10},
		},
		{
			typeExpr: "decimal(10,2)",
			expected: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2},
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
			expected: &ArrayType{Type: &schema.IntegerType{T: "int"}, T: "int[]"},
		},
		{
			typeExpr: `sql("int[2]")`,
			expected: &ArrayType{Type: &schema.IntegerType{T: "int"}, T: "int[]"},
		},
		{
			typeExpr: `sql("text[][]")`,
			expected: &ArrayType{Type: &schema.StringType{T: "text"}, T: "text[]"},
		},
		{
			typeExpr: `sql("integer [3][3]")`,
			expected: &ArrayType{Type: &schema.IntegerType{T: "integer"}, T: "integer[]"},
		},
		{
			typeExpr: `sql("integer  ARRAY[4]")`,
			expected: &ArrayType{Type: &schema.IntegerType{T: "integer"}, T: "integer[]"},
		},
		{
			typeExpr: `sql("integer ARRAY")`,
			expected: &ArrayType{Type: &schema.IntegerType{T: "integer"}, T: "integer[]"},
		},
		{
			typeExpr: `sql("character varying(255) [1][2]")`,
			expected: &ArrayType{Type: &schema.StringType{T: "character varying", Size: 255}, T: "character varying(255)[]"},
		},
		{
			typeExpr: `sql("character varying ARRAY[2]")`,
			expected: &ArrayType{Type: &schema.StringType{T: "character varying"}, T: "character varying[]"},
		},
		{
			typeExpr: `sql("varchar(2) [ 2 ] [  ]")`,
			expected: &ArrayType{Type: &schema.StringType{T: "varchar", Size: 2}, T: "varchar(2)[]"},
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
			err := EvalHCLBytes([]byte(doc), &test, nil)
			require.NoError(t, err)
			colspec := test.Tables[0].Columns[0]
			require.EqualValues(t, tt.expected, colspec.Type.Type)
			spec, err := MarshalHCL(&test)
			require.NoError(t, err)
			var after schema.Schema
			err = EvalHCLBytes(spec, &after, nil)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, after.Tables[0].Columns[0].Type.Type)
		})
	}
}

func typeTime(t string, p int) schema.Type {
	return &schema.TimeType{T: t, Precision: &p}
}

func TestParseType_Time(t *testing.T) {
	for _, tt := range []struct {
		typ      string
		expected schema.Type
	}{
		{
			typ:      "timestamptz",
			expected: typeTime(TypeTimestampTZ, 6),
		},
		{
			typ:      "timestamptz(0)",
			expected: typeTime(TypeTimestampTZ, 0),
		},
		{
			typ:      "timestamptz(6)",
			expected: typeTime(TypeTimestampTZ, 6),
		},
		{
			typ:      "timestamp with time zone",
			expected: typeTime(TypeTimestampTZ, 6),
		},
		{
			typ:      "timestamp(1) with time zone",
			expected: typeTime(TypeTimestampTZ, 1),
		},
		{
			typ:      "timestamp",
			expected: typeTime(TypeTimestamp, 6),
		},
		{
			typ:      "timestamp(0)",
			expected: typeTime(TypeTimestamp, 0),
		},
		{
			typ:      "timestamp(6)",
			expected: typeTime(TypeTimestamp, 6),
		},
		{
			typ:      "timestamp without time zone",
			expected: typeTime(TypeTimestamp, 6),
		},
		{
			typ:      "timestamp(1) without time zone",
			expected: typeTime(TypeTimestamp, 1),
		},
		{
			typ:      "time",
			expected: typeTime(TypeTime, 6),
		},
		{
			typ:      "time(3)",
			expected: typeTime(TypeTime, 3),
		},
		{
			typ:      "time without time zone",
			expected: typeTime(TypeTime, 6),
		},
		{
			typ:      "time(3) without time zone",
			expected: typeTime(TypeTime, 3),
		},
		{
			typ:      "timetz",
			expected: typeTime(TypeTimeTZ, 6),
		},
		{
			typ:      "timetz(4)",
			expected: typeTime(TypeTimeTZ, 4),
		},
		{
			typ:      "time with time zone",
			expected: typeTime(TypeTimeTZ, 6),
		},
		{
			typ:      "time(4) with time zone",
			expected: typeTime(TypeTimeTZ, 4),
		},
	} {
		t.Run(tt.typ, func(t *testing.T) {
			typ, err := ParseType(tt.typ)
			require.NoError(t, err)
			require.Equal(t, tt.expected, typ)
		})
	}
}

func TestFormatType_Interval(t *testing.T) {
	p := func(i int) *int { return &i }
	for i, tt := range []struct {
		typ *IntervalType
		fmt string
	}{
		{
			typ: &IntervalType{T: "interval"},
			fmt: "interval",
		},
		{
			typ: &IntervalType{T: "interval", Precision: p(6)},
			fmt: "interval",
		},
		{
			typ: &IntervalType{T: "interval", Precision: p(3)},
			fmt: "interval(3)",
		},
		{
			typ: &IntervalType{T: "interval", F: "DAY"},
			fmt: "interval day",
		},
		{
			typ: &IntervalType{T: "interval", F: "HOUR TO SECOND"},
			fmt: "interval hour to second",
		},
		{
			typ: &IntervalType{T: "interval", F: "HOUR TO SECOND", Precision: p(2)},
			fmt: "interval hour to second(2)",
		},
		{
			typ: &IntervalType{T: "interval", F: "DAY TO HOUR", Precision: p(6)},
			fmt: "interval day to hour",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := FormatType(tt.typ)
			require.NoError(t, err)
			require.Equal(t, tt.fmt, f)
		})
	}
}
func TestParseType_Interval(t *testing.T) {
	p := func(i int) *int { return &i }
	for i, tt := range []struct {
		typ    string
		parsed *IntervalType
	}{
		{
			typ:    "interval",
			parsed: &IntervalType{T: "interval", Precision: p(6)},
		},
		{
			typ:    "interval(2)",
			parsed: &IntervalType{T: "interval", Precision: p(2)},
		},
		{
			typ:    "interval day",
			parsed: &IntervalType{T: "interval", F: "day", Precision: p(6)},
		},
		{
			typ:    "interval day to second(2)",
			parsed: &IntervalType{T: "interval", F: "day to second", Precision: p(2)},
		},
		{
			typ:    "interval day to second (2)",
			parsed: &IntervalType{T: "interval", F: "day to second", Precision: p(2)},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			p, err := ParseType(tt.typ)
			require.NoError(t, err)
			require.Equal(t, tt.parsed, p)
		})
	}
}

func TestRegistrySanity(t *testing.T) {
	spectest.RegistrySanityTest(t, TypeRegistry, []string{"enum"})
}

func TestInputVars(t *testing.T) {
	spectest.TestInputVars(t, EvalHCL)
}

func TestQualifyReferencedTables(t *testing.T) {
	var (
		col       = schema.NewIntColumn("col", "integer")
		targetCol = schema.NewIntColumn("target_column", "integer")
		targetTbl = schema.NewTable("target_table").AddColumns(targetCol)
		realm     = schema.NewRealm(
			schema.New("target_schema").AddTables(targetTbl),
			schema.New("sch").
				AddTables(
					schema.NewTable("tbl").
						AddColumns(col).
						AddForeignKeys(
							schema.NewForeignKey("col_fk").
								SetRefTable(targetTbl).
								AddColumns(col).
								AddRefColumns(targetCol),
						),
				),
		)
		tables = []*sqlspec.Table{
			{
				Name:   "tbl",
				Schema: &schemahcl.Ref{V: "$schema.sch"},
			},
			{
				Name:   "target_table",
				Schema: &schemahcl.Ref{V: "$schema.target_schema"},
			},
		}
	)
	require.NoError(t, specutil.QualifyReferencedTables(tables, realm))
	require.Zero(t, tables[0].Qualifier)
	require.Equal(t, "target_schema", tables[1].Qualifier)
}
