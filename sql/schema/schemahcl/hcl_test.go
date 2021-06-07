package schemahcl

import (
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	tbl := &schema.TableSpec{
		Name: "users",
		Columns: []*schema.ColumnSpec{
			{
				Name:     "user_id",
				TypeName: "int64",
				Attrs: []*schema.SpecAttr{
					{K: "hello", V: schema.String("world")},
				},
				Children: []*schema.ResourceSpec{
					{
						Type: "resource",
						Name: "super_index",
						Attrs: []*schema.SpecAttr{
							{K: "enabled", V: schema.Bool(true)},
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
    type = "int64"
    resource "super_index" {
      enabled = true
    }
  }
  hello = "world"
}
`, string(config))
}
