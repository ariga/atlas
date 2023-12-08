// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"fmt"
	"strconv"
	"testing"

	"ariga.io/atlas/sql/internal/spectest"
	"ariga.io/atlas/sql/internal/sqlx"
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
	exp := schema.New("schema")
	exp.AddObjects(&schema.EnumType{T: "account_type", Values: []string{"private", "business"}, Schema: exp})
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

func TestMarshalSpec_Schema(t *testing.T) {
	s := schema.New("public").SetComment("schema comment")
	buf, err := MarshalHCL(s)
	require.NoError(t, err)
	require.Equal(t, `schema "public" {
  comment = "schema comment"
}
`, string(buf))

	r := schema.NewRealm(
		schema.New("s1").SetComment("c1"),
		schema.New("s2").SetComment("c2"),
		schema.New("s3"),
	)
	buf, err = MarshalHCL(r)
	require.NoError(t, err)
	require.Equal(t, `schema "s1" {
  comment = "c1"
}
schema "s2" {
  comment = "c2"
}
schema "s3" {
}
`, string(buf))
}

func TestUnmarshalSpec_Schema(t *testing.T) {
	var (
		s schema.Schema
		f = `
schema "public" {
  comment = "schema comment"
}
`
	)
	require.NoError(t, EvalHCLBytes([]byte(f), &s, nil))
	require.Equal(t, "public", s.Name)
	require.Len(t, s.Attrs, 1)
	require.Equal(t, "schema comment", s.Attrs[0].(*schema.Comment).Text)
}

func TestMarshalViews(t *testing.T) {
	s := schema.New("public").
		AddTables(
			schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("id", "int"),
				),
		).
		AddViews(
			schema.NewView("v1", "SELECT 1").
				SetCheckOption(schema.ViewCheckOptionLocal),
			schema.NewView("v2", "SELECT * FROM t2\n\tWHERE id IS NOT NULL"),
			schema.NewView("v3", "SELECT * FROM t3\n\tWHERE id IS NOT NULL\n\tORDER BY id").
				AddColumns(
					schema.NewIntColumn("id", "id"),
				).
				SetComment("view comment"),
			schema.NewMaterializedView("m1", "SELECT * FROM t1"),
		)
	s.AddViews(
		schema.NewView("v4", "SELECT * FROM v2 JOIN t1 USING (id)").
			AddDeps(
				s.Views[1],
				s.Tables[0],
			),
		schema.NewMaterializedView("m2", "SELECT * FROM t1").
			AddDeps(
				s.Views[1],
				s.Views[3],
				s.Tables[0],
			),
	)
	buf, err := MarshalHCL(s)
	require.NoError(t, err)
	f := `table "t1" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
}
view "v1" {
  schema       = schema.public
  as           = "SELECT 1"
  check_option = LOCAL
}
view "v2" {
  schema = schema.public
  as     = <<-SQL
  SELECT * FROM t2
  	WHERE id IS NOT NULL
  SQL
}
view "v3" {
  schema = schema.public
  column "id" {
    null = false
    type = sql("id")
  }
  as      = <<-SQL
  SELECT * FROM t3
  	WHERE id IS NOT NULL
  	ORDER BY id
  SQL
  comment = "view comment"
}
view "v4" {
  schema     = schema.public
  as         = "SELECT * FROM v2 JOIN t1 USING (id)"
  depends_on = [view.v2, table.t1]
}
materialized "m1" {
  schema = schema.public
  as     = "SELECT * FROM t1"
}
materialized "m2" {
  schema     = schema.public
  as         = "SELECT * FROM t1"
  depends_on = [view.v2, materialized.m1, table.t1]
}
schema "public" {
}
`
	require.Equal(t, f, string(buf))
}

func TestUnmarshalViews(t *testing.T) {
	f := `table "t1" {
  schema = schema.public
  column "id" {
    null = false
    type = int
  }
}
view "v1" {
  schema = schema.public
  as     = "SELECT * FROM t2 WHERE id IS NOT NULL"
}
materialized "m1" {
  schema = schema.public
  as     = "SELECT * FROM t2 WHERE id IS NOT NULL"
}
materialized "m2" {
  schema     = schema.public
  as         = "SELECT * FROM multi"
  depends_on = [view.v1, materialized.m1, table.t1]
}
view "v2" {
 schema = schema.public
 column "id" {
   null = false
   type = int
 }
 as      = "SELECT * FROM t3 WHERE id IS NOT NULL ORDER BY id"
 comment = "view comment"
}
view "v3" {
 schema       = schema.public
 as           = "SELECT * FROM v2 JOIN t1 USING (id)"
 check_option = LOCAL
 depends_on   = [view.v1, table.t1]
}

table "public" "t2" {
  schema = schema.public
  column "id" {
    type = int
  }
}

table "other" "t2" {
  schema = schema.other
  column "id" {
    type = int
  }
}

view "public" "v4" {
  schema = schema.public
  as     = "SELECT * FROM public.t2"
  depends_on = [table.public.t2]
}

view "other" "v4" {
  schema = schema.other
  as     = "SELECT * FROM other.t2"
  depends_on = [table.other.t2]
}

schema "public" {}
schema "other" {}
`
	var (
		r      schema.Realm
		got    schema.Realm
		public = schema.New("public").
			AddTables(
				schema.NewTable("t1").
					AddColumns(
						schema.NewIntColumn("id", "int"),
					),
				schema.NewTable("t2").
					AddColumns(
						schema.NewIntColumn("id", "int"),
					),
			).
			AddViews(
				schema.NewView("v1", "SELECT * FROM t2 WHERE id IS NOT NULL"),
				schema.NewView("v2", "SELECT * FROM t3 WHERE id IS NOT NULL ORDER BY id").
					AddColumns(
						schema.NewIntColumn("id", "int"),
					).
					SetComment("view comment"),
			)
		other = schema.New("other").
			AddTables(
				schema.NewTable("t2").
					AddColumns(
						schema.NewIntColumn("id", "int"),
					),
			)
	)
	public.AddViews(
		schema.NewView("v3", "SELECT * FROM v2 JOIN t1 USING (id)").
			SetCheckOption(schema.ViewCheckOptionLocal).
			AddDeps(public.Views[0], public.Tables[0]),
		schema.NewView("v4", "SELECT * FROM public.t2").
			AddDeps(public.Tables[1]),
		schema.NewMaterializedView("m1", "SELECT * FROM t2 WHERE id IS NOT NULL"),
	)
	m1, _ := public.Materialized("m1")
	public.AddViews(
		schema.NewMaterializedView("m2", "SELECT * FROM multi").
			AddDeps(public.Views[0], m1, public.Tables[0]),
	)
	other.AddViews(
		schema.NewView("v4", "SELECT * FROM other.t2").
			AddDeps(other.Tables[0]),
	)
	r.AddSchemas(public, other)
	require.NoError(t, EvalHCLBytes([]byte(f), &got, nil))
	require.EqualValues(t, r, got)
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
			r = fmt.Sprintf(f, "HASH")
		)
		require.NoError(t, EvalHCLBytes([]byte(r), &s, nil))
		idx := s.Tables[0].Indexes[0]
		require.Equal(t, IndexTypeHash, idx.Attrs[0].(*IndexType).T)

		s = schema.Schema{}
		r = fmt.Sprintf(f, "GiST")
		require.NoError(t, EvalHCLBytes([]byte(r), &s, nil))
		idx = s.Tables[0].Indexes[0]
		require.Equal(t, IndexTypeGiST, idx.Attrs[0].(*IndexType).T)
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

func TestUnmarshalSpec_IndexOpClass(t *testing.T) {
	const f = `table "users" {
  schema = schema.test
  column "a" {
    null = false
    type = text
  }
  column "b" {
    null = false
    type = tsvector
  }
  index "idx0" {
    unique  = true
    columns = [column.a, column.b]
  }
  index "idx1" {
    unique = true
    on {
      column = column.a
      ops    = text_pattern_ops
    }
    on {
      column = column.b
    }
  }
  index "idx2" {
    unique  = true
    columns = [column.a]
  }
  index "idx3" {
    unique = true
    on {
      column = column.a
      ops    = text_pattern_ops
    }
  }
  index "idx4" {
    unique = true
    type   = GIST
    on {
      column = column.b
      ops    = sql("tsvector_ops(siglen=1)")
    }
  }
}
schema "test" {
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	// idx0
	idx := s.Tables[0].Indexes[0]
	require.Len(t, idx.Parts, 2)
	require.Equal(t, "a", idx.Parts[0].C.Name)
	require.Empty(t, idx.Parts[0].Attrs)
	require.Equal(t, "b", idx.Parts[1].C.Name)
	require.Empty(t, idx.Parts[1].Attrs)
	// idx1
	idx = s.Tables[0].Indexes[1]
	require.Len(t, idx.Parts, 2)
	require.Equal(t, "a", idx.Parts[0].C.Name)
	require.Len(t, idx.Parts[0].Attrs, 1)
	require.Equal(t, "text_pattern_ops", idx.Parts[0].Attrs[0].(*IndexOpClass).Name)
	require.Equal(t, "b", idx.Parts[1].C.Name)
	require.Empty(t, idx.Parts[1].Attrs)
	// idx2
	idx = s.Tables[0].Indexes[2]
	require.Len(t, idx.Parts, 1)
	require.Equal(t, "a", idx.Parts[0].C.Name)
	require.Empty(t, idx.Parts[0].Attrs)
	// idx3
	idx = s.Tables[0].Indexes[3]
	require.Len(t, idx.Parts, 1)
	require.Equal(t, "a", idx.Parts[0].C.Name)
	require.Len(t, idx.Parts[0].Attrs, 1)
	require.Equal(t, "text_pattern_ops", idx.Parts[0].Attrs[0].(*IndexOpClass).Name)
	// idx4
	idx = s.Tables[0].Indexes[4]
	require.Len(t, idx.Parts, 1)
	require.Equal(t, "b", idx.Parts[0].C.Name)
	require.Len(t, idx.Parts[0].Attrs, 1)
	opc := idx.Parts[0].Attrs[0].(*IndexOpClass)
	require.Equal(t, "tsvector_ops", opc.Name)
	require.Len(t, opc.Params, 1)
	require.Equal(t, "siglen", opc.Params[0].N)
	require.Equal(t, "1", opc.Params[0].V)
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
		require.Error(t, err, "missing partition type")

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
		require.EqualError(t, err, `cannot convert table "logs": missing columns or expressions for logs.partition`)

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
		require.EqualError(t, err, `cannot convert table "logs": multiple definitions for logs.partition, use "columns" or "by"`)
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

func TestMarshalSpec_IndexNullsDistinct(t *testing.T) {
	s := schema.New("public").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c", "int"),
				).
				AddIndexes(
					// Default behavior.
					schema.NewUniqueIndex("without_attribute").
						AddColumns(schema.NewColumn("c")),
					schema.NewUniqueIndex("with_nulls_distinct").
						AddColumns(schema.NewColumn("c")).
						AddAttrs(&IndexNullsDistinct{V: true}),
					// Explicitly disable (NULLS NOT DISTINCT).
					schema.NewUniqueIndex("with_nulls_not_distinct").
						AddColumns(schema.NewColumn("c")).
						AddAttrs(&IndexNullsDistinct{V: false}),
				),
		)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.public
  column "c" {
    null = false
    type = int
  }
  index "without_attribute" {
    unique  = true
    columns = [column.c]
  }
  index "with_nulls_distinct" {
    unique  = true
    columns = [column.c]
  }
  index "with_nulls_not_distinct" {
    unique         = true
    columns        = [column.c]
    nulls_distinct = false
  }
}
schema "public" {
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

func TestMarshalSpec_IndexOpClass(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{
						Name: "a",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}},
					},
					{
						Name: "b",
						Type: &schema.ColumnType{Type: &TextSearchType{T: "tsvector"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[0].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:   "idx0",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "text_ops"}}},
				{SeqNo: 1, C: s.Tables[0].Columns[1], Attrs: []schema.Attr{&IndexOpClass{Name: "tsvector_ops"}}},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeBTree},
			},
		},
		{
			Name:   "idx1",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "text_pattern_ops"}}},
				{SeqNo: 1, C: s.Tables[0].Columns[1], Attrs: []schema.Attr{&IndexOpClass{Name: "tsvector_ops"}}},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeBTree},
			},
		},
		{
			Name:   "idx2",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "text_ops"}}},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeBTree},
			},
		},
		{
			Name:   "idx3",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "text_pattern_ops"}}},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeBTree},
			},
		},
		{
			Name:   "idx4",
			Table:  s.Tables[0],
			Unique: true,
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[1], Attrs: []schema.Attr{&IndexOpClass{Name: "tsvector_ops", Params: []struct{ N, V string }{{"siglen", "1"}}}}},
			},
			Attrs: []schema.Attr{
				&IndexType{T: IndexTypeGiST},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "a" {
    null = false
    type = text
  }
  column "b" {
    null = false
    type = tsvector
  }
  index "idx0" {
    unique  = true
    columns = [column.a, column.b]
  }
  index "idx1" {
    unique = true
    on {
      column = column.a
      ops    = text_pattern_ops
    }
    on {
      column = column.b
    }
  }
  index "idx2" {
    unique  = true
    columns = [column.a]
  }
  index "idx3" {
    unique = true
    on {
      column = column.a
      ops    = text_pattern_ops
    }
  }
  index "idx4" {
    unique = true
    type   = GIST
    on {
      column = column.b
      ops    = sql("tsvector_ops(siglen=1)")
    }
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

func TestUnmarshalSpec_IndexInclude(t *testing.T) {
	f := `
schema "s" {}
table "t" {
	schema = schema.s
	column "c" {
		type = int
	}
	column "d" {
		type = int
	}
	index "c" {
		columns = [
			column.c,
		]
		include = [
			column.d,
		]
	}
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	require.Len(t, s.Tables[0].Columns, 2)
	require.Len(t, s.Tables[0].Indexes, 1)
	idx, ok := s.Tables[0].Index("c")
	require.True(t, ok)
	require.Len(t, idx.Parts, 1)
	require.Len(t, idx.Attrs, 1)
	var include IndexInclude
	require.True(t, sqlx.Has(idx.Attrs, &include))
	require.Len(t, include.Columns, 1)
	require.Equal(t, "d", include.Columns[0].Name)
}

func TestMarshalSpec_IndexInclude(t *testing.T) {
	s := &schema.Schema{
		Name: "test",
		Tables: []*schema.Table{
			{
				Name: "users",
				Columns: []*schema.Column{
					{
						Name: "c",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
					},
					{
						Name: "d",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}},
					},
				},
			},
		},
	}
	s.Tables[0].Schema = s
	s.Tables[0].Indexes = []*schema.Index{
		{
			Name:  "index",
			Table: s.Tables[0],
			Parts: []*schema.IndexPart{
				{SeqNo: 0, C: s.Tables[0].Columns[0]},
			},
			Attrs: []schema.Attr{
				&IndexInclude{Columns: s.Tables[0].Columns[1:]},
			},
		},
	}
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "c" {
    null = false
    type = int
  }
  column "d" {
    null = false
    type = int
  }
  index "index" {
    columns = [column.c]
    include = [column.d]
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestMarshalSpec_PrimaryKey(t *testing.T) {
	s := schema.New("test").
		AddTables(
			schema.NewTable("users").
				AddColumns(
					schema.NewIntColumn("c", "int"),
					schema.NewIntColumn("d", "int"),
				),
		)
	s.Tables[0].SetPrimaryKey(
		schema.NewPrimaryKey(s.Tables[0].Columns[:1]...).
			AddAttrs(&IndexInclude{Columns: s.Tables[0].Columns[1:]}),
	)
	buf, err := MarshalSpec(s, hclState)
	require.NoError(t, err)
	const expected = `table "users" {
  schema = schema.test
  column "c" {
    null = false
    type = int
  }
  column "d" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.c]
    include = [column.d]
  }
}
schema "test" {
}
`
	require.EqualValues(t, expected, string(buf))
}

func TestUnmarshalSpec_PrimaryKey(t *testing.T) {
	f := `
schema "s" {}
table "t" {
	schema = schema.s
	column "c" {
		type = int
	}
	column "d" {
		type = int
	}
	primary_key {
		columns = [
			column.c,
		]
		include = [
			column.d,
		]
	}
}
`
	var s schema.Schema
	err := EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	require.Len(t, s.Tables[0].Columns, 2)
	pk := s.Tables[0].PrimaryKey
	require.NotNil(t, pk)
	require.Len(t, pk.Parts, 1)
	require.Len(t, pk.Attrs, 1)
	var include IndexInclude
	require.True(t, sqlx.Has(pk.Attrs, &include))
	require.Len(t, include.Columns, 1)
	require.Equal(t, "d", include.Columns[0].Name)
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
	stateE := &schema.EnumType{
		T:      "state",
		Values: []string{"on", "off"},
	}
	typeE := &schema.EnumType{
		T:      "account_type",
		Values: []string{"private", "business"},
	}
	s := schema.New("test").
		AddObjects(
			typeE, stateE,
		).
		AddTables(
			schema.NewTable("account").
				AddColumns(
					schema.NewEnumColumn("account_type",
						schema.EnumName("account_type"),
						schema.EnumValues("private", "business"),
					),
					schema.NewColumn("account_states").
						SetType(&ArrayType{
							T:    "states[]",
							Type: stateE,
						}),
				),
			schema.NewTable("table2").
				AddColumns(
					schema.NewColumn("account_type").
						SetType(typeE),
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
  column "account_states" {
    null = false
    type = sql("states[]")
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
enum "state" {
  schema = schema.test
  values = ["on", "off"]
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
			expected: &schema.UUIDType{T: TypeUUID},
		},
		{
			typeExpr: "money",
			expected: &CurrencyType{T: TypeMoney},
		},
		{
			typeExpr: "int4range",
			expected: &RangeType{T: TypeInt4Range},
		},
		{
			typeExpr: "int4multirange",
			expected: &RangeType{T: TypeInt4MultiRange},
		},
		{
			typeExpr: "int8range",
			expected: &RangeType{T: TypeInt8Range},
		},
		{
			typeExpr: "int8multirange",
			expected: &RangeType{T: TypeInt8MultiRange},
		},
		{
			typeExpr: "numrange",
			expected: &RangeType{T: TypeNumRange},
		},
		{
			typeExpr: "nummultirange",
			expected: &RangeType{T: TypeNumMultiRange},
		},
		{
			typeExpr: "tsrange",
			expected: &RangeType{T: TypeTSRange},
		},
		{
			typeExpr: "tsmultirange",
			expected: &RangeType{T: TypeTSMultiRange},
		},
		{
			typeExpr: "tstzrange",
			expected: &RangeType{T: TypeTSTZRange},
		},
		{
			typeExpr: "tstzmultirange",
			expected: &RangeType{T: TypeTSTZMultiRange},
		},
		{
			typeExpr: "daterange",
			expected: &RangeType{T: TypeDateRange},
		},
		{
			typeExpr: "datemultirange",
			expected: &RangeType{T: TypeDateMultiRange},
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
		{
			typeExpr: "oid",
			expected: &OIDType{T: typeOID},
		},
		{
			typeExpr: "regclass",
			expected: &OIDType{T: typeRegClass},
		},
		{
			typeExpr: "name",
			expected: &schema.StringType{T: typeName},
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

	// Two views with the same name resided in different schemas.
	v2 := schema.NewView("v2", "SELECT oid FROM s1.t2").
		AddColumns(schema.NewIntColumn("oid", "int")).
		AddDeps(t2)
	v4 := schema.NewView("v2", "SELECT oid FROM s2.t2").
		AddColumns(schema.NewIntColumn("oid", "int")).
		AddDeps(t4)

	r := schema.NewRealm(
		schema.New("s1").AddTables(t1, t2).AddViews(v2),
		schema.New("s2").AddTables(t3, t4, t5).AddViews(v4),
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
view "s1" "v2" {
  schema = schema.s1
  column "oid" {
    null = false
    type = int
  }
  as         = "SELECT oid FROM s1.t2"
  depends_on = [table.s1.t2]
}
view "s2" "v2" {
  schema = schema.s2
  column "oid" {
    null = false
    type = int
  }
  as         = "SELECT oid FROM s2.t2"
  depends_on = [table.s2.t2]
}
schema "s1" {
}
schema "s2" {
}
`,
		string(got))
}

func TestMarshalSkipQualifiers(t *testing.T) {
	buf, err := MarshalHCL.MarshalSpec(
		schema.New("s1").
			AddTables(
				schema.NewTable("t1").AddColumns(schema.NewIntColumn("id1", "int")),
				schema.NewTable("t1").AddColumns(schema.NewIntColumn("id2", "int")),
			),
	)
	require.NoError(t, err)
	require.Equal(t, `table "t1" {
  schema = schema.s1
  column "id1" {
    null = false
    type = int
  }
}
table "t1" {
  schema = schema.s1
  column "id2" {
    null = false
    type = int
  }
}
schema "s1" {
}
`, string(buf), "qualifiers are skipped if objects belong to the same schema (repeatable blocks)")
}

func TestMarshalQualifiers(t *testing.T) {
	var (
		s1 = schema.New("s1").
			AddTables(
				schema.NewTable("t1").AddColumns(schema.NewIntColumn("id", "int")),
			)
		s2 = schema.New("s2").
			AddTables(
				schema.NewTable("t1").AddColumns(schema.NewIntColumn("id", "int")),
			).
			AddViews(
				schema.NewMaterializedView("m1", "SELECT id FROM s2.t1"),
			)
		s3 = schema.New("s3").
			AddTables(
				schema.NewTable("s1").AddColumns(schema.NewIntColumn("id", "int")),
			).
			AddViews(
				schema.NewMaterializedView("m1", "SELECT id FROM s3.t1"),
			)
	)
	s2.Views[0].AddDeps(s2.Tables[0])
	s3.Views[0].AddDeps(s3.Tables[0])
	buf, err := MarshalHCL.MarshalSpec(schema.NewRealm(s1, s2, s3))
	require.NoError(t, err)
	require.Equal(t, `table "s1" "t1" {
  schema = schema.s1
  column "id" {
    null = false
    type = int
  }
}
table "s2" "t1" {
  schema = schema.s2
  column "id" {
    null = false
    type = int
  }
}
table "s3" "s1" {
  schema = schema.s3
  column "id" {
    null = false
    type = int
  }
}
materialized "s2" "m1" {
  schema     = schema.s2
  as         = "SELECT id FROM s2.t1"
  depends_on = [table.s2.t1]
}
materialized "s3" "m1" {
  schema     = schema.s3
  as         = "SELECT id FROM s3.t1"
  depends_on = [table.s1]
}
schema "s1" {
}
schema "s2" {
}
schema "s3" {
}
`, string(buf))
}
