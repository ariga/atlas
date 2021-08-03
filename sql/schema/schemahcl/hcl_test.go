// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestReEncode(t *testing.T) {
	tbl := &schemaspec.Table{
		Name: "users",
		Columns: []*schemaspec.Column{
			{
				Name: "user_id",
				Type: "int64",
				Resource: schemaspec.Resource{
					Attrs: []*schemaspec.Attr{
						{K: "hello", V: &schemaspec.LiteralValue{V: `"world"`}},
					},
					Children: []*schemaspec.Resource{
						{
							Type: "resource",
							Name: "super_index",
							Attrs: []*schemaspec.Attr{
								{K: "enabled", V: &schemaspec.LiteralValue{V: `true`}},
							},
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
	s := &schemaspec.Schema{}
	err = Decode(config, s)
	require.NoError(t, err)
	require.EqualValues(t, tbl, s.Tables[0])
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
	s := &schemaspec.Schema{}
	err := Decode([]byte(f), s)
	require.NoError(t, err)
	require.Equal(t, "s1", s.Tables[0].SchemaName)
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
	s := &schemaspec.Schema{}
	err := Decode([]byte(f), s)
	require.NoError(t, err)
	require.Equal(t, &schemaspec.PrimaryKey{
		Columns: []*schemaspec.ColumnRef{
			{
				Table: "users",
				Name:  "name",
			},
		},
	}, s.Tables[0].PrimaryKey)
	encode, err := Encode(s)
	require.NoError(t, err)
	generated := &schemaspec.Schema{}
	err = Decode(encode, generated)
	require.NoError(t, err)
	require.EqualValues(t, s, generated)
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
	s := &schemaspec.Schema{}
	err := Decode([]byte(f), s)
	require.NoError(t, err)
	require.Equal(t, &schemaspec.Index{
		Name:   "txn_id",
		Unique: true,
		Columns: []*schemaspec.ColumnRef{
			{
				Name:  "txn_id",
				Table: "users",
			},
		},
	}, s.Tables[0].Indexes[0])
	encode, err := Encode(s)
	require.NoError(t, err)
	generated := &schemaspec.Schema{}
	err = Decode(encode, generated)
	require.NoError(t, err)
	require.EqualValues(t, s, generated)
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
		on_delete = reference_option.no_action
	}
}
`
	s := &schemaspec.Schema{}
	err := Decode([]byte(f), s)
	require.NoError(t, err)
	require.Equal(t, &schemaspec.ForeignKey{
		Symbol: "user_name_ref",
		Columns: []*schemaspec.ColumnRef{
			{Table: "user_messages", Name: "user_name"},
		},
		RefColumns: []*schemaspec.ColumnRef{
			{Table: "users", Name: "name"},
		},
		OnDelete: string(schema.NoAction),
	}, s.Tables[1].ForeignKeys[0])
	encode, err := Encode(s)
	require.NoError(t, err)
	generated := &schemaspec.Schema{}
	err = Decode(encode, generated)
	require.NoError(t, err)
	require.EqualValues(t, s, generated)
}

func TestRewriteHCL(t *testing.T) {
	dir, err := ioutil.ReadDir("testdata")
	require.NoError(t, err)
	for _, tt := range dir {
		if tt.IsDir() {
			continue
		}
		filename := filepath.Join("testdata", tt.Name())
		t.Run(filename, func(t *testing.T) {
			fb, err := ioutil.ReadFile(filename)
			require.NoError(t, err)
			decoded := &schemaspec.Schema{}
			err = Decode(fb, decoded)
			require.NoError(t, err)
			encode, err := Encode(decoded)
			require.NoError(t, err)
			generated := &schemaspec.Schema{}
			err = Decode(encode, generated)
			require.NoError(t, err)
			require.EqualValues(t, decoded, generated)
		})
	}
}

func TestColumnOverride(t *testing.T) {
	h := `schema "todo" {

}

table "user" {
  schema = schema.todo
  column "name" {
    type = "string"
    dialect "mysql" {
      type = "varchar(255)"
    }
    dialect "mysql" {
      version = "10"
      type = "text"
    }
  }
}`
	decoded := &schemaspec.Schema{}
	err := Decode([]byte(h), decoded)
	require.NoError(t, err)
	ut, ok := decoded.Table("user")
	require.True(t, ok)
	name, ok := ut.Column("name")
	require.True(t, ok)
	mo := name.Override("mysql")
	require.NotNil(t, mo)
	attr, ok := mo.Attr("type")
	require.True(t, ok)
	typ, err := attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "varchar(255)", typ)

	mo = name.Override("mysql", "mysql 10")
	require.NotNil(t, mo)
	attr, ok = mo.Attr("type")
	require.True(t, ok)
	typ, err = attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "text", typ)
}
