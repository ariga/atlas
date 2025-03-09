// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package hclsqlspec

import (
	"math/big"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

var dialects = []struct {
	name string
	schemahcl.Marshaler
	Eval func(b []byte, v any, inp map[string]cty.Value) error
}{
	{
		name:      "mysql",
		Marshaler: mysql.MarshalHCL,
		Eval:      mysql.EvalHCLBytes,
	},
	{
		name:      "postgres",
		Marshaler: postgres.MarshalHCL,
		Eval:      postgres.EvalHCLBytes,
	},
	{
		name:      "sqlite",
		Marshaler: sqlite.MarshalHCL,
		Eval:      sqlite.EvalHCLBytes,
	},
}

func TestHCL_SQL(t *testing.T) {
	file, err := decode(`
schema "hi" {

}

table "users" {
	schema = schema.hi
	
	column "id" {
		type = int
		null = false
		default = 123
	}
	column "age" {
		type = int
		null = false
		default = 10
	}
	column "active" {
		type = boolean
		default = true
	}

	column "account_active" {
		type = boolean
		default "DF_expr1" {
			as = true
		}
	}

	primary_key {
		columns = [table.users.column.id, table.users.column.age]
	}
	
	index "age" {
		unique = true
		columns = [table.users.column.age]
	}
	index "active" {
		unique = false
		columns = [table.users.column.active]
	}

	foreign_key "fk" {
		columns = [table.users.column.account_active]
		ref_columns = [table.accounts.column.active]
		on_delete = "SET NULL"
	}
}

table "accounts" {
	schema = schema.hi
	
	column "id" {
		type = int
		null = false
		default = 123
	}
	column "age" {
		type = int
		null = false
		default = 10
	}
	column "active" {
		type = boolean
		default = true
	}

	column "user_active" {
		type = boolean
		default = true
	}

	primary_key {
		columns = [table.accounts.column.id]
	}
	
	index "age" {
		unique = true
		columns = [table.accounts.column.age]
	}
	index "active" {
		unique = false
		columns = [table.accounts.column.active]
	}

	foreign_key "fk" {
		columns = [table.accounts.column.user_active]
		ref_columns = [table.users.column.active]
		on_delete = "SET NULL"
	}
}

`)
	require.NoError(t, err)
	expected := &db{
		Schemas: []*sqlspec.Schema{
			{Name: "hi"},
		},
		Tables: []*sqlspec.Table{
			{
				Name:   "users",
				Schema: &schemahcl.Ref{V: "$schema.hi"},
				Columns: []*sqlspec.Column{
					{
						Name: "id",
						Type: &schemahcl.Type{T: "int"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.NumberVal(big.NewFloat(123).SetPrec(512))},
								},
							},
						},
					},
					{
						Name: "age",
						Type: &schemahcl.Type{T: "int"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.NumberVal(big.NewFloat(10).SetPrec(512))},
								},
							},
						},
					},
					{
						Name: "active",
						Type: &schemahcl.Type{T: "boolean"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.BoolVal(true)},
								},
							},
						},
					},
					{
						Name: "account_active",
						Type: &schemahcl.Type{T: "boolean"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Children: []*schemahcl.Resource{
									{
										Type: "default",
										Name: "DF_expr1",
										Attrs: []*schemahcl.Attr{
											{K: "as", V: cty.BoolVal(true)},
										},
									},
								},
							},
						},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: []*schemahcl.Ref{
						{
							V: "$table.users.$column.id",
						},
						{
							V: "$table.users.$column.age",
						},
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: []*schemahcl.Ref{
							{
								V: "$table.users.$column.age",
							},
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: []*schemahcl.Ref{
							{
								V: "$table.users.$column.active",
							},
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: []*schemahcl.Ref{
							{
								V: "$table.users.$column.account_active",
							},
						},
						RefColumns: []*schemahcl.Ref{
							{
								V: "$table.accounts.$column.active",
							},
						},
						OnDelete: &schemahcl.Ref{V: string(schema.SetNull)},
					},
				},
			},
			{
				Name:   "accounts",
				Schema: &schemahcl.Ref{V: "$schema.hi"},
				Columns: []*sqlspec.Column{
					{
						Name: "id",
						Type: &schemahcl.Type{T: "int"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.NumberVal(big.NewFloat(123).SetPrec(512))},
								},
							},
						},
					},
					{
						Name: "age",
						Type: &schemahcl.Type{T: "int"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.NumberVal(big.NewFloat(10).SetPrec(512))},
								},
							},
						},
					},
					{
						Name: "active",
						Type: &schemahcl.Type{T: "boolean"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.BoolVal(true)},
								},
							},
						},
					},
					{
						Name: "user_active",
						Type: &schemahcl.Type{T: "boolean"},
						Null: false,
						DefaultExtension: schemahcl.DefaultExtension{
							Extra: schemahcl.Resource{
								Attrs: []*schemahcl.Attr{
									{K: "default", V: cty.BoolVal(true)},
								},
							},
						},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: []*schemahcl.Ref{
						{
							V: "$table.accounts.$column.id",
						},
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: []*schemahcl.Ref{
							{
								V: "$table.accounts.$column.age",
							},
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: []*schemahcl.Ref{
							{
								V: "$table.accounts.$column.active",
							},
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: []*schemahcl.Ref{
							{
								V: "$table.accounts.$column.user_active",
							},
						},
						RefColumns: []*schemahcl.Ref{
							{
								V: "$table.users.$column.active",
							},
						},
						OnDelete: &schemahcl.Ref{V: string(schema.SetNull)},
					},
				},
			},
		},
	}
	require.EqualValues(t, expected, file)
}

func TestWithRemain(t *testing.T) {
	file, err := decode(`
schema "hi" {
	x = 1
}`)
	require.NoError(t, err)
	require.EqualValues(t, &db{
		Schemas: []*sqlspec.Schema{
			{
				Name: "hi",
				DefaultExtension: schemahcl.DefaultExtension{
					Extra: schemahcl.Resource{
						Attrs: []*schemahcl.Attr{
							schemahcl.IntAttr("x", 1),
						},
					},
				},
			},
		},
	}, file)
}

func TestMultiTable(t *testing.T) {
	_, err := decode(`
schema "hi" {

}

table "users" {
	schema = schema.hi
	column "id" {
		type = int
		unsigned = true
		null = false
		default = 123
	}
}

table "accounts" {
	schema = schema.hi
	column "id" {
		type = varchar(255)
	}
	index "name" {
		unique = true
	}
}

`)
	require.NoError(t, err)
}

var hcl = schemahcl.New(schemahcl.WithTypes("table.column.type", postgres.TypeRegistry.Specs()))

func TestMarshalTopLevel(t *testing.T) {
	c := &sqlspec.Schema{
		Name: "schema",
	}
	h, err := hcl.MarshalSpec(c)
	require.NoError(t, err)
	require.EqualValues(t, `schema "schema" {
}
`, string(h))
}

func TestRealm(t *testing.T) {
	f := `schema "account_a" {
}
table "t1" {
	schema = schema.account_a
}
schema "account_b" {
}
table "t2" {
	schema = schema.account_b
}
`

	for _, tt := range dialects {
		t.Run(tt.name, func(t *testing.T) {
			var r schema.Realm
			err := tt.Eval([]byte(f), &r, nil)
			require.NoError(t, err)
			exp := &schema.Realm{
				Schemas: []*schema.Schema{
					{
						Name:  "account_a",
						Realm: &r,
						Tables: []*schema.Table{
							{Name: "t1"},
						},
					},
					{
						Name:  "account_b",
						Realm: &r,
						Tables: []*schema.Table{
							{Name: "t2"},
						},
					},
				},
			}
			exp.Schemas[0].Tables[0].Schema = exp.Schemas[0]
			exp.Schemas[1].Tables[0].Schema = exp.Schemas[1]
			require.EqualValues(t, exp, &r)
			hcl, err := tt.MarshalSpec(&r)
			require.NoError(t, err)
			var after schema.Realm
			err = tt.Eval(hcl, &after, nil)
			require.NoError(t, err)
			require.EqualValues(t, exp, &after)
		})
	}
}

func TestUnsignedImmutability(t *testing.T) {
	f := `table "users" {
	schema = schema.test
	column "id" {
		type = bigint
		unsigned = true
	}
	column "shouldnt" {
		type = bigint
	}
}
schema "test" {
}`
	var s schema.Schema
	err := mysql.EvalHCLBytes([]byte(f), &s, nil)
	require.NoError(t, err)
	tbl := s.Tables[0]
	require.EqualValues(t, &schema.IntegerType{T: "bigint", Unsigned: true}, tbl.Columns[0].Type.Type)
	require.EqualValues(t, &schema.IntegerType{T: "bigint", Unsigned: false}, tbl.Columns[1].Type.Type)
}

func TestTablesWithQualifiers(t *testing.T) {
	h := `
schema "a" {}
schema "b" {}

table "a" "users" {
	schema = schema.a
	column "id" {
		type = int
	}
	column "friend_id" {
		type = int
	}
	foreign_key "friend_b" {
		columns = [column.friend_id]
		ref_columns = [table.b.users.column.id]
	}
}

table "b" "users" {
	schema = schema.b
	column "id" {
		type = int
	}
	column "friend_id" {
		type = int
	}
	foreign_key "friend_a" {
		columns = [column.friend_id]
		ref_columns = [table.a.users.column.id]
	}
}
`
	var r schema.Realm
	err := mysql.EvalHCLBytes([]byte(h), &r, nil)
	require.NoError(t, err)

	require.EqualValues(t, r.Schemas[0].Tables[0].Columns[0], r.Schemas[1].Tables[0].ForeignKeys[0].RefColumns[0])
	require.EqualValues(t, "b", r.Schemas[0].Tables[0].ForeignKeys[0].RefTable.Schema.Name)
}

func TestQualifyMarshal(t *testing.T) {
	for _, tt := range dialects {
		t.Run(tt.name, func(t *testing.T) {
			r := schema.NewRealm(
				schema.New("a").
					AddTables(
						schema.NewTable("users"),
						schema.NewTable("tbl_a"),
					),
				schema.New("b").
					AddTables(
						schema.NewTable("users"),
						schema.NewTable("tbl_b"),
					),
				schema.New("c").
					AddTables(
						schema.NewTable("users"),
						schema.NewTable("tbl_c"),
					),
			)
			h, err := tt.Marshaler.MarshalSpec(r)
			require.NoError(t, err)
			expected := `table "a" "users" {
  schema = schema.a
}
table "tbl_a" {
  schema = schema.a
}
table "b" "users" {
  schema = schema.b
}
table "tbl_b" {
  schema = schema.b
}
table "c" "users" {
  schema = schema.c
}
table "tbl_c" {
  schema = schema.c
}
schema "a" {
}
schema "b" {
}
schema "c" {
}
`
			require.EqualValues(t, expected, string(h))
		})
	}

}

func decode(f string) (*db, error) {
	d := &db{}
	if err := hcl.EvalBytes([]byte(f), d, nil); err != nil {
		return nil, err
	}
	return d, nil
}

type db struct {
	Schemas []*sqlspec.Schema `spec:"schema"`
	Tables  []*sqlspec.Table  `spec:"table"`
}
