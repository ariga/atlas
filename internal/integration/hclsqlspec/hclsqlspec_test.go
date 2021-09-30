package hclsqlspec

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/stretchr/testify/require"
)

func TestHCL_SQL(t *testing.T) {
	file, err := decode(`
schema "hi" {

}

table "users" {
	schema = "hi"
	
	column "id" {
		type = "uint"
		null = false
		default = 123
	}
	column "age" {
		type = "int"
		null = false
		default = 10
	}
	column "active" {
		type = "bool"
		default = true
	}

	column "account_active" {
		type = "bool"
		default = true
	}

	primary_key {
		columns = table.users.column.id
	}
	
	index "age" {
		unique = true
		columns = table.users.column.age
	}
	index "active" {
		unique = false
		columns = table.users.column.active
	}

	foreign_key "fk" {
		columns = table.users.column.account_active
		ref_columns = table.accounts.column.active
		on_delete = "SET NULL"
	}
}

table "accounts" {
	schema = "hi"
	
	column "id" {
		type = "uint"
		null = false
		default = 123
	}
	column "age" {
		type = "int"
		null = false
		default = 10
	}
	column "active" {
		type = "bool"
		default = true
	}

	column "user_active" {
		type = "bool"
		default = true
	}

	primary_key {
		columns = table.accounts.column.id
	}
	
	index "age" {
		unique = true
		columns = table.accounts.column.age
	}
	index "active" {
		unique = false
		columns = table.accounts.column.active
	}

	foreign_key "fk" {
		columns = table.accounts.column.user_active
		ref_columns = table.users.column.active
		on_delete = "SET NULL"
	}
}

`)
	require.NoError(t, err)
	require.EqualValues(t, &db{
		Schemas: []*sqlspec.Schema{
			{Name: "hi"},
		},
		Tables: []*sqlspec.Table{
			{
				Name:       "users",
				SchemaName: "hi",
				Columns: []*sqlspec.Column{
					{
						Name:     "id",
						TypeName: "uint",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "123"},
					},
					{
						Name:     "age",
						TypeName: "int",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "10"},
					},
					{
						Name:     "active",
						TypeName: "bool",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "true"},
					},
					{
						Name:     "account_active",
						TypeName: "bool",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "true"},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: &schemaspec.Ref{
						V: "$table.users.$column.id",
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: &schemaspec.Ref{
							V: "$table.users.$column.age",
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: &schemaspec.Ref{
							V: "$table.users.$column.active",
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: &schemaspec.Ref{
							V: "$table.users.$column.account_active",
						},
						RefColumns: &schemaspec.Ref{
							V: "$table.accounts.$column.active",
						},
						OnDelete: sqlspec.SetNull,
					},
				},
			},
			{
				Name:       "accounts",
				SchemaName: "hi",
				Columns: []*sqlspec.Column{
					{
						Name:     "id",
						TypeName: "uint",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "123"},
					},
					{
						Name:     "age",
						TypeName: "int",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "10"},
					},
					{
						Name:     "active",
						TypeName: "bool",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "true"},
					},
					{
						Name:     "user_active",
						TypeName: "bool",
						Null:     false,
						Default:  &schemaspec.LiteralValue{V: "true"},
					},
				},
				PrimaryKey: &sqlspec.PrimaryKey{
					Columns: &schemaspec.Ref{
						V: "$table.accounts.$column.id",
					},
				},
				Indexes: []*sqlspec.Index{
					{
						Name:   "age",
						Unique: true,
						Columns: &schemaspec.Ref{
							V: "$table.accounts.$column.age",
						},
					},
					{
						Name:   "active",
						Unique: false,
						Columns: &schemaspec.Ref{
							V: "$table.accounts.$column.active",
						},
					},
				},
				ForeignKeys: []*sqlspec.ForeignKey{
					{
						Symbol: "fk",
						Columns: &schemaspec.Ref{
							V: "$table.accounts.$column.user_active",
						},
						RefColumns: &schemaspec.Ref{
							V: "$table.users.$column.active",
						},
						OnDelete: sqlspec.SetNull,
					},
				},
			},
		},
	}, file)
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
				DefaultExtension: schemaspec.DefaultExtension{
					Extra: schemaspec.Resource{
						Attrs: []*schemaspec.Attr{
							{K: "x", V: &schemaspec.LiteralValue{V: "1"}},
						},
					},
				},
			},
		},
		Tables: []*sqlspec.Table{},
	}, file)
}

func decode(f string) (*db, error) {
	res, err := schemahcl.Decode([]byte(f))
	if err != nil {
		return nil, err
	}
	s := db{}
	if err := res.As(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

type db struct {
	Schemas []*sqlspec.Schema `spec:"schema"`
	Tables  []*sqlspec.Table  `spec:"table"`
}
