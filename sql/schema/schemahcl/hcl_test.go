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
					{K: "hello", V: &schema.LiteralValue{V: `"world"`}},
				},
				Children: []*schema.ResourceSpec{
					{
						Type: "resource",
						Name: "super_index",
						Attrs: []*schema.SpecAttr{
							{K: "enabled", V: &schema.LiteralValue{V: `true`}},
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
