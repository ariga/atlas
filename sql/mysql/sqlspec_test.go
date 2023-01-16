// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

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
	schema = schema.schema
	column "col" {
		type = int
		comment = "column comment"
	}
	column "age" {
		type = int
	}
	column "price1" {
		type = int
		auto_increment = false
	}
	column "price2" {
		type = int
		auto_increment = true
	}
	column "account_name" {
		type = varchar(32)
	}
	column "created_at" {
		type    = datetime(4)
		default = sql("now(4)")
	}
	column "updated_at" {
		type      = timestamp(6)
		default   = sql("current_timestamp(6)")
		on_update = sql("current_timestamp(6)")
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
		expr = "price1 > 0"
	}
	check {
		expr     = "price1 <> price2"
		enforced = true
	}
	check {
		expr     = "price2 <> price1"
		enforced = false
	}
	comment = "table comment"
	auto_increment = 1000
}

table "accounts" {
	schema = schema.schema
	column "name" {
		type = varchar(32)
	}
	column "unsigned_float" {
		type     = float(10)
		unsigned = true
	}
	column "unsigned_decimal" {
		type     = decimal(10,2)
		unsigned = true
	}
	primary_key {
		columns = [table.accounts.column.name]
	}
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
							T: TypeInt,
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
							T: TypeInt,
						},
					},
				},
				{
					Name: "price1",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: TypeInt,
						},
					},
				},
				{
					Name: "price2",
					Type: &schema.ColumnType{
						Type: &schema.IntegerType{
							T: TypeInt,
						},
					},
					Attrs: []schema.Attr{&AutoIncrement{}},
				},
				{
					Name: "account_name",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    TypeVarchar,
							Size: 32,
						},
					},
				},
				{
					Name: "created_at",
					Type: &schema.ColumnType{
						Type: typeTime(TypeDateTime, 4),
					},
					Default: &schema.RawExpr{X: "now(4)"},
				},
				{
					Name: "updated_at",
					Type: &schema.ColumnType{
						Type: typeTime(TypeTimestamp, 6),
					},
					Default: &schema.RawExpr{X: "current_timestamp(6)"},
					Attrs:   []schema.Attr{&OnUpdate{A: "current_timestamp(6)"}},
				},
			},
			Attrs: []schema.Attr{
				&schema.Check{
					Name: "positive price",
					Expr: "price1 > 0",
				},
				&schema.Check{
					Expr:  "price1 <> price2",
					Attrs: []schema.Attr{&Enforced{V: true}},
				},
				&schema.Check{
					Expr:  "price2 <> price1",
					Attrs: []schema.Attr{&Enforced{V: false}},
				},
				&schema.Comment{Text: "table comment"},
				&AutoIncrement{V: 1000},
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
							T:    TypeVarchar,
							Size: 32,
						},
					},
				},
				{
					Name: "unsigned_float",
					Type: &schema.ColumnType{
						Type: &schema.FloatType{
							T:         TypeFloat,
							Precision: 10,
							Unsigned:  true,
						},
					},
				},
				{
					Name: "unsigned_decimal",
					Type: &schema.ColumnType{
						Type: &schema.DecimalType{
							T:         TypeDecimal,
							Precision: 10,
							Scale:     2,
							Unsigned:  true,
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
			},
		},
	}
	exp.Tables[0].ForeignKeys = []*schema.ForeignKey{
		{
			Symbol:     "accounts",
			Table:      exp.Tables[0],
			Columns:    []*schema.Column{exp.Tables[0].Columns[4]},
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

func TestMarshalSpec_Charset(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Attrs: []schema.Attr{
			&schema.Charset{V: "utf8mb4"},
			&schema.Collation{V: "utf8mb4_0900_ai_ci"},
		},
		Tables: []*schema.Table{
			{
				Name: "users",
				Attrs: []schema.Attr{
					&schema.Charset{V: "utf8mb4"},
					&schema.Collation{V: "utf8mb4_0900_ai_ci"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "b",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "utf8mb4"},
							&schema.Collation{V: "utf8mb4_0900_ai_ci"},
						},
					},
				},
			},
			{
				Name: "posts",
				Attrs: []schema.Attr{
					&schema.Charset{V: "latin1"},
					&schema.Collation{V: "latin1_swedish_ci"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "b",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Charset{V: "utf8mb4"},
							&schema.Collation{V: "utf8mb4_0900_ai_ci"},
						},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[1].Schema = s
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	// Charset and collate that are identical to their parent elements
	// should not be printed as they are inherited by default from it.
	const expected = `table "users" {
  schema = schema.test
  column "a" {
    null    = false
    type    = text
    charset = "latin1"
    collate = "latin1_swedish_ci"
  }
  column "b" {
    null = false
    type = text
  }
}
table "posts" {
  schema  = schema.test
  charset = "latin1"
  collate = "latin1_swedish_ci"
  column "a" {
    null = false
    type = text
  }
  column "b" {
    null    = false
    type    = text
    charset = "utf8mb4"
    collate = "utf8mb4_0900_ai_ci"
  }
}
schema "test" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
`
	require.EqualValues(t, expected, string(buf))

	var (
		s2    schema.Schema
		latin = []schema.Attr{
			&schema.Charset{V: "latin1"},
			&schema.Collation{V: "latin1_swedish_ci"},
		}
		utf8mb4 = []schema.Attr{
			&schema.Charset{V: "utf8mb4"},
			&schema.Collation{V: "utf8mb4_0900_ai_ci"},
		}
	)
	require.NoError(t, EvalHCLBytes(buf, &s2, nil))
	require.Equal(t, utf8mb4, s2.Attrs)
	posts, ok := s2.Table("posts")
	require.True(t, ok)
	require.Equal(t, latin, posts.Attrs)
	users, ok := s2.Table("users")
	require.True(t, ok)
	require.Empty(t, users.Attrs)
	a, ok := users.Column("a")
	require.True(t, ok)
	require.Equal(t, latin, a.Attrs)
	b, ok := posts.Column("b")
	require.True(t, ok)
	require.Equal(t, utf8mb4, b.Attrs)
}

func TestMarshalSpec_Comment(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Attrs: []schema.Attr{
					&schema.Comment{Text: "table comment"},
				},
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
						Attrs: []schema.Attr{
							&schema.Comment{Text: "column comment"},
						},
					},
				},
			},
			{
				Name: "posts",
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[1].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "index",
			Table:  s.Tables[0],
			Unique: true,
			Parts:  []*schema.IndexPart{{SeqNo: 0, C: s.Tables[0].Columns[0]}},
			Attrs: []schema.Attr{
				&schema.Comment{Text: "index comment"},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	// We expect a zero value comment to not be present in the marshaled HCL.
	const expected = `table "users" {
  schema  = schema.test
  comment = "table comment"
  column "a" {
    null    = false
    type    = text
    comment = "column comment"
  }
  index "index" {
    unique  = true
    columns = [column.a]
    comment = "index comment"
  }
}
table "posts" {
  schema = schema.test
  column "a" {
    null = false
    type = text
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_AutoIncrement(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{
						Name: "id",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
						Attrs: []schema.Attr{
							&AutoIncrement{},
						},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "id" {
    null           = false
    type           = bigint
    auto_increment = true
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_Check(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("products").
				AddColumns(
					schema.NewIntColumn("price1", TypeInt),
					schema.NewIntColumn("price2", TypeInt),
				).
				AddChecks(
					schema.NewCheck().SetName("price1 positive").SetExpr("price1 > 0"),
					schema.NewCheck().SetExpr("price1 <> price2").AddAttrs(&Enforced{}),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "products" {
  schema = schema.test
  column "price1" {
    null = false
    type = int
  }
  column "price2" {
    null = false
    type = int
  }
  check "price1 positive" {
    expr = "price1 > 0"
  }
  check {
    expr     = "price1 <> price2"
    enforced = true
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestUnmarshalSpec_IndexParts(t *testing.T) {
	var (
		s schema.Schema
		f = `
schema "test" {}
table "users" {
	schema = schema.test
	column "name" {
		type = text
	}
	index "idx" {
		on {
			column = table.users.column.name
			desc = true
			prefix = 10
		}
		on {
			expr = "lower(name)"
		}
	}
}
`
	)
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	c := schema.NewStringColumn("name", "text")
	exp := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(c).
				AddIndexes(
					schema.NewIndex("idx").
						AddParts(
							schema.NewColumnPart(c).SetDesc(true).AddAttrs(&SubPart{Len: 10}),
							schema.NewExprPart(&schema.RawExpr{X: "lower(name)"}),
						),
				),
		)
	exp.Tables[0].Columns[0].Indexes = nil
	require.EqualValues(t, exp, &s)
}

func TestMarshalSpec_PrimaryKeyType(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewStringColumn("id", "varchar(255)"),
				),
		)
	s.Tables[0].SetPrimaryKey(
		schema.NewPrimaryKey(s.Tables[0].Columns...).
			AddAttrs(&IndexType{T: IndexTypeHash}),
	)
	buf, err := MarshalHCL(s)
	require.NoError(t, err)
	exp := `table "users" {
  schema = schema.test
  column "id" {
    null = false
    type = sql("varchar(255)")
  }
  primary_key {
    columns = [column.id]
    type    = HASH
  }
}
schema "test" {
}
`
	require.EqualValues(t, exp, string(buf))
}

func TestUnmarshalSpec_PrimaryKeyType(t *testing.T) {
	var s schema.Schema
	err := EvalHCLBytes([]byte(`table "users" {
  schema = schema.test
  column "id" {
    null = false
    type = sql("varchar(255)")
  }
  primary_key {
    columns = [column.id]
    type    = HASH
  }
}
schema "test" {
}`), &s, nil)
	require.NoError(t, err)
	exp := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(&schema.Column{
					Name: "id",
					Type: &schema.ColumnType{
						Type: &schema.StringType{
							T:    "varchar",
							Size: 255,
						},
					},
				}),
		)
	exp.Tables[0].SetPrimaryKey(&schema.Index{
		Parts: []*schema.IndexPart{{C: exp.Tables[0].Columns[0]}},
		Attrs: []schema.Attr{&IndexType{T: IndexTypeHash}},
	})
	require.EqualValues(t, exp, &s)
}

func TestMarshalSpec_IndexParts(t *testing.T) {
	c := schema.NewStringColumn("name", "text")
	c2 := schema.NewStringColumn("Full Name", "text")
	s := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(c, c2).
				AddIndexes(
					schema.NewIndex("idx").
						AddParts(
							schema.NewColumnPart(c).SetDesc(true).AddAttrs(&SubPart{Len: 10}),
							schema.NewExprPart(&schema.RawExpr{X: "lower(name)"}),
						),
					schema.NewIndex("idx2").
						AddParts(
							schema.NewColumnPart(c2).SetDesc(true).AddAttrs(&SubPart{Len: 10}),
						),
					schema.NewIndex("idx3").
						AddParts(schema.NewColumnPart(c2)),
				),
		)
	buf, err := MarshalHCL(s)
	require.NoError(t, err)
	exp := `table "users" {
  schema = schema.test
  column "name" {
    null = false
    type = text
  }
  column "Full Name" {
    null = false
    type = text
  }
  index "idx" {
    on {
      desc   = true
      column = column.name
      prefix = 10
    }
    on {
      expr = "lower(name)"
    }
  }
  index "idx2" {
    on {
      desc   = true
      column = column["Full Name"]
      prefix = 10
    }
  }
  index "idx3" {
    columns = [column["Full Name"]]
  }
}
schema "test" {
}
`
	require.EqualValues(t, exp, string(buf))

	// Columns only.
	s = schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(c, c2).
				AddIndexes(
					schema.NewIndex("idx").
						AddParts(
							schema.NewColumnPart(c).AddAttrs(&SubPart{Len: 10}),
						),
				),
		)
	buf, err = MarshalHCL(s)
	require.NoError(t, err)
	exp = `table "users" {
  schema = schema.test
  column "name" {
    null = false
    type = text
  }
  column "Full Name" {
    null = false
    type = text
  }
  index "idx" {
    on {
      column = column.name
      prefix = 10
    }
  }
}
schema "test" {
}
`
	require.EqualValues(t, exp, string(buf))
}

func TestMarshalSpec_TimePrecision(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("times").
				AddColumns(
					schema.NewTimeColumn("tTimeDef", TypeTime),
					schema.NewTimeColumn("tTime", TypeTime, schema.TimePrecision(1)),
					schema.NewTimeColumn("tDatetime", TypeDateTime, schema.TimePrecision(2)),
					schema.NewTimeColumn("tTimestamp", TypeTimestamp, schema.TimePrecision(3)).
						SetDefault(&schema.RawExpr{X: "current_timestamp(3)"}).
						AddAttrs(&OnUpdate{A: "current_timestamp(3)"}),
					schema.NewTimeColumn("tDate", TypeDate),
					schema.NewTimeColumn("tYear", TypeYear, schema.TimePrecision(2)),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "times" {
  schema = schema.test
  column "tTimeDef" {
    null = false
    type = time
  }
  column "tTime" {
    null = false
    type = time(1)
  }
  column "tDatetime" {
    null = false
    type = datetime(2)
  }
  column "tTimestamp" {
    null      = false
    type      = timestamp(3)
    default   = sql("current_timestamp(3)")
    on_update = sql("current_timestamp(3)")
  }
  column "tDate" {
    null = false
    type = date
  }
  column "tYear" {
    null = false
    type = year(2)
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_GeneratedColumn(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c1", "int"),
					schema.NewIntColumn("c2", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c1 * 2"}),
					schema.NewIntColumn("c3", "int").
						SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c2 * c3", Type: "VIRTUAL"}),
					schema.NewIntColumn("c4", "int").
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
  }
  column "c2" {
    null = false
    type = int
    as {
      expr = "c1 * 2"
      type = VIRTUAL
    }
  }
  column "c3" {
    null = false
    type = int
    as {
      expr = "c2 * c3"
      type = VIRTUAL
    }
  }
  column "c4" {
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
	}
	column "c2" {
		type = int
		as = "c1 * 2"
	}
	column "c3" {
		type = int
		as {
			expr = "c2 * 2"
		}
	}
	column "c4" {
		type = int
		as {
			expr = "c3 * 2"
			type = STORED
		}
	}
}
`
	)
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	exp := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c1", "int"),
					schema.NewIntColumn("c2", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c1 * 2", Type: "VIRTUAL"}),
					schema.NewIntColumn("c3", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c2 * 2", Type: "VIRTUAL"}),
					schema.NewIntColumn("c4", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c3 * 2", Type: "STORED"}),
				),
		)
	require.EqualValues(t, exp, &s)
}

func TestMarshalSpec_FloatUnsigned(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("test").
				AddColumns(
					schema.NewFloatColumn(
						"float_col",
						TypeFloat,
						schema.FloatPrecision(10),
						schema.FloatUnsigned(true),
					),
					schema.NewDecimalColumn(
						"decimal_col",
						TypeDecimal,
						schema.DecimalPrecision(10),
						schema.DecimalScale(2),
						schema.DecimalUnsigned(true),
					),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "test" {
  schema = schema.test
  column "float_col" {
    null     = false
    type     = float(10)
    unsigned = true
  }
  column "decimal_col" {
    null     = false
    type     = decimal(10,2)
    unsigned = true
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestTypes(t *testing.T) {
	p := func(i int) *int { return &i }
	tests := []struct {
		typeExpr  string
		extraAttr string
		expected  schema.Type
	}{
		{
			typeExpr: "varchar(255)",
			expected: &schema.StringType{T: TypeVarchar, Size: 255},
		},
		{
			typeExpr: "char(255)",
			expected: &schema.StringType{T: TypeChar, Size: 255},
		},
		{
			typeExpr: `sql("custom")`,
			expected: &schema.UnsupportedType{T: "custom"},
		},
		{
			typeExpr: "binary(255)",
			expected: &schema.BinaryType{T: TypeBinary, Size: p(255)},
		},
		{
			typeExpr: "varbinary(255)",
			expected: &schema.BinaryType{T: TypeVarBinary, Size: p(255)},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr:  "int",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeInt, Unsigned: true},
		},
		{
			typeExpr: "int",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr: "bigint",
			expected: &schema.IntegerType{T: TypeBigInt},
		},
		{
			typeExpr:  "bigint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeBigInt, Unsigned: true},
		},
		{
			typeExpr: "tinyint",
			expected: &schema.IntegerType{T: TypeTinyInt},
		},
		{
			typeExpr:  "tinyint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeTinyInt, Unsigned: true},
		},
		{
			typeExpr: "smallint",
			expected: &schema.IntegerType{T: TypeSmallInt},
		},
		{
			typeExpr:  "smallint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeSmallInt, Unsigned: true},
		},
		{
			typeExpr: "mediumint",
			expected: &schema.IntegerType{T: TypeMediumInt},
		},
		{
			typeExpr:  "mediumint",
			extraAttr: "unsigned=true",
			expected:  &schema.IntegerType{T: TypeMediumInt, Unsigned: true},
		},
		{
			typeExpr: "tinytext",
			expected: &schema.StringType{T: TypeTinyText},
		},
		{
			typeExpr: "mediumtext",
			expected: &schema.StringType{T: TypeMediumText},
		},
		{
			typeExpr: "longtext",
			expected: &schema.StringType{T: TypeLongText},
		},
		{
			typeExpr: "text",
			expected: &schema.StringType{T: TypeText},
		},
		{
			typeExpr: `enum("on","off")`,
			expected: &schema.EnumType{T: TypeEnum, Values: []string{"on", "off"}},
		},
		{
			typeExpr: "bit",
			expected: &BitType{T: TypeBit},
		},
		{
			typeExpr: "bit(10)",
			expected: &BitType{T: TypeBit, Size: 10},
		},
		{
			typeExpr: "int(10)",
			expected: &schema.IntegerType{T: TypeInt},
		},
		{
			typeExpr: "tinyint(10)",
			expected: &schema.IntegerType{T: TypeTinyInt},
		},
		{
			typeExpr: "smallint(10)",
			expected: &schema.IntegerType{T: TypeSmallInt},
		},
		{
			typeExpr: "mediumint(10)",
			expected: &schema.IntegerType{T: TypeMediumInt},
		},
		{
			typeExpr: "bigint(10)",
			expected: &schema.IntegerType{T: TypeBigInt},
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
			typeExpr:  "decimal(10,2)",
			extraAttr: "unsigned=true",
			expected:  &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2, Unsigned: true},
		},
		{
			typeExpr: "numeric",
			expected: &schema.DecimalType{T: TypeNumeric},
		},
		{
			typeExpr:  "numeric",
			extraAttr: "unsigned=true",
			expected:  &schema.DecimalType{T: TypeNumeric, Unsigned: true},
		},
		{
			typeExpr: "numeric(10)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10},
		},
		{
			typeExpr: "numeric(10,2)",
			expected: &schema.DecimalType{T: TypeNumeric, Precision: 10, Scale: 2},
		},
		{
			typeExpr: "float(10,0)",
			expected: &schema.FloatType{T: TypeFloat, Precision: 10},
		},
		{
			typeExpr:  "float(10)",
			extraAttr: "unsigned=true",
			expected:  &schema.FloatType{T: TypeFloat, Precision: 10, Unsigned: true},
		},
		{
			typeExpr: "double(10,0)",
			expected: &schema.FloatType{T: TypeDouble, Precision: 10},
		},
		{
			typeExpr: "real",
			expected: &schema.FloatType{T: TypeReal},
		},
		{
			typeExpr:  "real",
			extraAttr: "unsigned=true",
			expected:  &schema.FloatType{T: TypeReal, Unsigned: true},
		},
		{
			typeExpr: "timestamp",
			expected: &schema.TimeType{T: TypeTimestamp},
		},
		{
			typeExpr: "timestamp(6)",
			expected: typeTime(TypeTimestamp, 6),
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
			typeExpr: "time(6)",
			expected: typeTime(TypeTime, 6),
		},
		{
			typeExpr: "datetime",
			expected: &schema.TimeType{T: TypeDateTime},
		},
		{
			typeExpr: "datetime(6)",
			expected: typeTime(TypeDateTime, 6),
		},
		{
			typeExpr: "year",
			expected: &schema.TimeType{T: TypeYear},
		},
		{
			typeExpr: "year(2)",
			expected: typeTime(TypeYear, 2),
		},
		{
			typeExpr: "varchar(10)",
			expected: &schema.StringType{T: TypeVarchar, Size: 10},
		},
		{
			typeExpr: "char(25)",
			expected: &schema.StringType{T: TypeChar, Size: 25},
		},
		{
			typeExpr: "varbinary(30)",
			expected: &schema.BinaryType{T: TypeVarBinary, Size: p(30)},
		},
		{
			typeExpr: "binary",
			expected: &schema.BinaryType{T: TypeBinary},
		},
		{
			typeExpr: "binary(5)",
			expected: &schema.BinaryType{T: TypeBinary, Size: p(5)},
		},
		{
			typeExpr: "blob(5)",
			expected: &schema.BinaryType{T: TypeBlob},
		},
		{
			typeExpr: "tinyblob",
			expected: &schema.BinaryType{T: TypeTinyBlob},
		},
		{
			typeExpr: "mediumblob",
			expected: &schema.BinaryType{T: TypeMediumBlob},
		},
		{
			typeExpr: "longblob",
			expected: &schema.BinaryType{T: TypeLongBlob},
		},
		{
			typeExpr: "json",
			expected: &schema.JSONType{T: TypeJSON},
		},
		{
			typeExpr: "text(13)",
			expected: &schema.StringType{T: TypeText},
		},
		{
			typeExpr: "tinytext",
			expected: &schema.StringType{T: TypeTinyText},
		},
		{
			typeExpr: "mediumtext",
			expected: &schema.StringType{T: TypeMediumText},
		},
		{
			typeExpr: "longtext",
			expected: &schema.StringType{T: TypeLongText},
		},
		{
			typeExpr: `set("a","b")`,
			expected: &SetType{Values: []string{"a", "b"}},
		},
		{
			typeExpr: "geometry",
			expected: &schema.SpatialType{T: TypeGeometry},
		},
		{
			typeExpr: "point",
			expected: &schema.SpatialType{T: TypePoint},
		},
		{
			typeExpr: "multipoint",
			expected: &schema.SpatialType{T: TypeMultiPoint},
		},
		{
			typeExpr: "linestring",
			expected: &schema.SpatialType{T: TypeLineString},
		},
		{
			typeExpr: "multilinestring",
			expected: &schema.SpatialType{T: TypeMultiLineString},
		},
		{
			typeExpr: "polygon",
			expected: &schema.SpatialType{T: TypePolygon},
		},
		{
			typeExpr: "multipolygon",
			expected: &schema.SpatialType{T: TypeMultiPolygon},
		},
		{
			typeExpr: "geometrycollection",
			expected: &schema.SpatialType{T: TypeGeometryCollection},
		},
		{
			typeExpr: "tinyint(1)",
			expected: &schema.BoolType{T: TypeBool},
		},
		{
			typeExpr: "bool",
			expected: &schema.BoolType{T: TypeBool},
		},
		{
			typeExpr: "boolean",
			expected: &schema.BoolType{T: TypeBool},
		},
	}
	for _, tt := range tests {
		t.Run(tt.typeExpr, func(t *testing.T) {
			doc := fmt.Sprintf(`table "test" {
	schema = schema.test
	column "test" {
		null = false
		type = %s%s
	}
}
schema "test" {
}
`, tt.typeExpr, lineIfSet(tt.extraAttr))
			var test schema.Schema
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

func TestInputVars(t *testing.T) {
	spectest.TestInputVars(t, EvalHCL)
}

func TestParseType_Decimal(t *testing.T) {
	for _, tt := range []struct {
		input   string
		wantT   *schema.DecimalType
		wantErr bool
	}{
		{
			input: "decimal",
			wantT: &schema.DecimalType{T: TypeDecimal},
		},
		{
			input: "decimal unsigned",
			wantT: &schema.DecimalType{T: TypeDecimal, Unsigned: true},
		},
		{
			input: "decimal(10)",
			wantT: &schema.DecimalType{T: TypeDecimal, Precision: 10},
		},
		{
			input: "decimal(10) unsigned",
			wantT: &schema.DecimalType{T: TypeDecimal, Precision: 10, Unsigned: true},
		},
		{
			input: "decimal(10,2)",
			wantT: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2},
		},
		{
			input: "decimal(10, 2) unsigned",
			wantT: &schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2, Unsigned: true},
		},
	} {
		d, err := ParseType(tt.input)
		require.Equal(t, tt.wantErr, err != nil)
		require.Equal(t, tt.wantT, d)
	}
}

func typeTime(t string, p int) schema.Type {
	return &schema.TimeType{T: t, Precision: &p}
}

func lineIfSet(s string) string {
	if s != "" {
		return "\n" + s
	}
	return s
}

func TestUnmarshalSpec(t *testing.T) {
	s := []byte(`
schema "s1" {}
schema "s2" {}

table "s1" "t1" {
 schema  = schema.s1
 column "id" {
   type = int
 }
}

table "s2" "t1" {
  schema  = schema.s2
  column "id" {
    type = int
  }
}
table "s2" "t2" {
  schema  = schema.s2
  column "oid" {
    type = int
  }
  foreign_key "fk" {
    columns = [column.oid]
    ref_columns = [table.s2.t1.column.id]
  }
}
`)
	var (
		r        schema.Realm
		expected = schema.NewRealm(
			schema.New("s1").AddTables(schema.NewTable("t1").AddColumns(schema.NewIntColumn("id", "int"))),
			schema.New("s2").AddTables(
				schema.NewTable("t1").AddColumns(schema.NewIntColumn("id", "int")),
				schema.NewTable("t2").AddColumns(schema.NewIntColumn("oid", "int")),
			),
		)
	)
	expected.Schemas[1].Tables[1].AddForeignKeys(schema.NewForeignKey("fk").
		AddColumns(expected.Schemas[1].Tables[1].Columns[0]).
		SetRefTable(expected.Schemas[1].Tables[0]).
		AddRefColumns(expected.Schemas[1].Tables[0].Columns[0]))
	require.NoError(t, EvalHCLBytes(s, &r, nil))
}

func TestMarshalRealm(t *testing.T) {
	t1 := schema.NewTable("t1").
		AddColumns(schema.NewIntColumn("id", "int"))
	t2 := schema.NewTable("t2").
		SetComment("Qualified with s1").
		AddColumns(schema.NewIntColumn("oid", "int"))
	t2.AddForeignKeys(schema.NewForeignKey("oid2id").AddColumns(t2.Columns[0]).SetRefTable(t1).AddRefColumns(t1.Columns[0]))

	t3 := schema.NewTable("t3").
		AddColumns(schema.NewIntColumn("id", "int"))
	t4 := schema.NewTable("t2").
		SetComment("Qualified with s2").
		AddColumns(schema.NewIntColumn("oid", "int"))
	t4.AddForeignKeys(schema.NewForeignKey("oid2id").AddColumns(t4.Columns[0]).SetRefTable(t3).AddRefColumns(t3.Columns[0]))
	t5 := schema.NewTable("t5").
		AddColumns(schema.NewIntColumn("oid", "int"))
	t5.AddForeignKeys(schema.NewForeignKey("oid2id1").AddColumns(t5.Columns[0]).SetRefTable(t1).AddRefColumns(t1.Columns[0]))
	// Reference is qualified with s1.
	t5.AddForeignKeys(schema.NewForeignKey("oid2id2").AddColumns(t5.Columns[0]).SetRefTable(t2).AddRefColumns(t2.Columns[0]))

	r := schema.NewRealm(
		schema.New("s1").AddTables(t1, t2),
		schema.New("s2").AddTables(t3, t4, t5),
	)
	got, err := MarshalHCL.MarshalSpec(r)
	require.NoError(t, err)
	require.Equal(
		t,
		`table "t1" {
  schema = schema.s1
  column "id" {
    null = false
    type = int
  }
}
table "s1" "t2" {
  schema  = schema.s1
  comment = "Qualified with s1"
  column "oid" {
    null = false
    type = int
  }
  foreign_key "oid2id" {
    columns     = [column.oid]
    ref_columns = [table.t1.column.id]
  }
}
table "t3" {
  schema = schema.s2
  column "id" {
    null = false
    type = int
  }
}
table "s2" "t2" {
  schema  = schema.s2
  comment = "Qualified with s2"
  column "oid" {
    null = false
    type = int
  }
  foreign_key "oid2id" {
    columns     = [column.oid]
    ref_columns = [table.t3.column.id]
  }
}
table "t5" {
  schema = schema.s2
  column "oid" {
    null = false
    type = int
  }
  foreign_key "oid2id1" {
    columns     = [column.oid]
    ref_columns = [table.t1.column.id]
  }
  foreign_key "oid2id2" {
    columns     = [column.oid]
    ref_columns = [table.s1.t2.column.oid]
  }
}
schema "s1" {
}
schema "s2" {
}
`,
		string(got))
}

func TestUnmarshalSpec_QuotedIdentifiers(t *testing.T) {
	var (
		r schema.Realm
		s = []byte(`
schema "a8m.schema" {}
table "a8m.table" {
  schema = schema["a8m.schema"]
  column "a8m.column" {
    type = int
  }
}

schema "nati.schema" {}
table "nati.schema" "nati.table" {
  schema = schema["nati.schema"]
  column "nati.column" {
    type = int
  }
  foreign_key "nati.fk" {
    columns = [column["nati.column"]]
	ref_columns = [table["a8m.table"].column["a8m.column"]]
  }
}
`)
	)
	require.NoError(t, EvalHCLBytes(s, &r, nil))
	require.Len(t, r.Schemas, 2)
	s1, ok := r.Schema("a8m.schema")
	require.True(t, ok)
	require.Equal(t, "a8m.schema", s1.Name)
	require.Len(t, r.Schemas[0].Tables, 1)
	require.Equal(t, "a8m.table", s1.Tables[0].Name)
	require.Len(t, r.Schemas[0].Tables[0].Columns, 1)
	require.Equal(t, "a8m.column", s1.Tables[0].Columns[0].Name)
	s2, ok := r.Schema("nati.schema")
	require.True(t, ok)
	require.Equal(t, "nati.schema", s2.Name)
	require.Len(t, r.Schemas[1].Tables, 1)
	require.Equal(t, "nati.table", s2.Tables[0].Name)
	require.Len(t, r.Schemas[1].Tables[0].Columns, 1)
	require.Equal(t, "nati.column", s2.Tables[0].Columns[0].Name)
	require.Len(t, r.Schemas[1].Tables[0].ForeignKeys, 1)
	require.Equal(t, "nati.fk", s2.Tables[0].ForeignKeys[0].Symbol)
	require.Len(t, r.Schemas[1].Tables[0].ForeignKeys[0].Columns, 1)
	require.Equal(t, "nati.column", s2.Tables[0].ForeignKeys[0].Columns[0].Name)
	require.Len(t, r.Schemas[1].Tables[0].ForeignKeys[0].RefColumns, 1)
	require.Equal(t, "a8m.column", s2.Tables[0].ForeignKeys[0].RefColumns[0].Name)
}

func TestMarshalSpec_QuotedIdentifiers(t *testing.T) {
	s1 := schema.New("a8m.schema").
		AddTables(schema.NewTable("a8m.table").
			AddColumns(schema.NewIntColumn("a8m.column", "int")))
	s2 := schema.New("nati.schema").
		AddTables(schema.NewTable("nati.table").
			AddColumns(schema.NewIntColumn("nati.column", "int")).
			AddForeignKeys(schema.NewForeignKey("nati.fk").
				AddColumns(s1.Tables[0].Columns[0]).
				SetRefTable(s1.Tables[0]).
				AddRefColumns(s1.Tables[0].Columns[0])))
	r := schema.NewRealm(s1, s2)
	got, err := MarshalHCL.MarshalSpec(r)
	require.NoError(t, err)
	require.Equal(t, `table "a8m.table" {
  schema = schema["a8m.schema"]
  column "a8m.column" {
    null = false
    type = int
  }
}
table "nati.table" {
  schema = schema["nati.schema"]
  column "nati.column" {
    null = false
    type = int
  }
  foreign_key "nati.fk" {
    columns     = [column["a8m.column"]]
    ref_columns = [table["a8m.table"].column["a8m.column"]]
  }
}
schema "a8m.schema" {
}
schema "nati.schema" {
}
`, string(got))
}
