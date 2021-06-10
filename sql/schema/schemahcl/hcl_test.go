package schemahcl

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestReEncode(t *testing.T) {
	tbl := &schema.TableSpec{
		Name: "users",
		Columns: []*schema.ColumnSpec{
			{
				Name: "user_id",
				Type: "int64",
				Attrs: []*schema.SpecAttr{
					{K: "hello", V: &schema.SpecLiteral{V: `"world"`}},
				},
				Children: []*schema.ResourceSpec{
					{
						Type: "resource",
						Name: "super_index",
						Attrs: []*schema.SpecAttr{
							{K: "enabled", V: &schema.SpecLiteral{V: `true`}},
						},
					},
				},
			},
		},
	}
	config, err := Encode(tbl)
	require.NoError(t, err)
	require.Equal(t, `table "users" {
  column "user_id" {
    type  = "int64"
    hello = "world"
    resource "super_index" {
      enabled = true
    }
  }
}
`, string(config))
	tgt := &schema.SchemaSpec{}
	err = Decode(config, tgt)
	require.NoError(t, err)
	require.EqualValues(t, tbl, tgt.Tables[0])
}

func TestSchemaRef(t *testing.T) {
	f := `schema "s1" {
}

table "users" {
	schema = schema.s1
	column "name" {
		type = "string"
	}
}`
	tgt := &schema.SchemaSpec{}
	err := Decode([]byte(f), tgt)
	require.NoError(t, err)
	require.Equal(t, "s1", tgt.Tables[0].SchemaName)
}

func TestPrimaryKey(t *testing.T) {
	f := `
table "users" {
	column "name" {
		type = "string"
	}
	column "age" {
		type = "int"
	}
	primary_key {
		columns = [
			table.users.column.name
		]
	}
}
`
	tgt := &schema.SchemaSpec{}
	err := Decode([]byte(f), tgt)
	require.NoError(t, err)
	require.Equal(t, &schema.PrimaryKeySpec{
		Columns: []*schema.ColumnRef{
			{
				Table: "users",
				Name:  "name",
			},
		},
	}, tgt.Tables[0].PrimaryKey)
}

func TestIndex(t *testing.T) {
	f := `
table "users" {
	column "name" {
		type = "string"
	}
	column "age" {
		type = "int"
	}
	column "txn_id" {
		type = "int"
	}
	index "txn_id" {
		columns = [
			table.users.column.txn_id
		]
		unique = true
	}
}
`
	tgt := &schema.SchemaSpec{}
	err := Decode([]byte(f), tgt)
	require.NoError(t, err)
	require.Equal(t, &schema.IndexSpec{
		Name:   "txn_id",
		Unique: true,
		Columns: []*schema.ColumnRef{
			{
				Name:  "txn_id",
				Table: "users",
			},
		},
	}, tgt.Tables[0].Indexes[0])
}

func TestForeignKey(t *testing.T) {
	f := `
table "users" {
	column "name" {
		type = "string"
	}
	column "age" {
		type = "int"
	}
	primary_key {
		columns = [
			table.users.column.name
		]
	}
}

table "user_messages" {
	column "text" {
		type = "string"
	}
	column "user_name" {
		type = "string"
	}
	foreign_key "user_name_ref" {
		columns = [
			table.user_messages.column.user_name
		]
		references =  [
			table.users.column.name
		]
	}
}
`
	tgt := &schema.SchemaSpec{}
	err := Decode([]byte(f), tgt)
	require.NoError(t, err)
	require.Equal(t, &schema.ForeignKeySpec{
		Symbol: "user_name_ref",
		Columns: []*schema.ColumnRef{
			{Table: "user_messages", Name: "user_name"},
		},
		RefColumns: []*schema.ColumnRef{
			{Table: "users", Name: "name"},
		},
	}, tgt.Tables[1].ForeignKeys[0])
}
