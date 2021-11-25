package hclsqlspec

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	f := `
modify_table {
	table = "users"
	add_column {
		column "id" {
			type = "int"
		}
	}
}
`
	var test struct {
		Changes []sqlspec.Change `spec:""`
	}
	err := schemahcl.Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ModifyTable{
		Table: "users",
		Changes: []sqlspec.Change{
			&sqlspec.AddColumn{
				Column: &sqlspec.Column{
					Name:     "id",
					Null:     false,
					TypeName: "int",
				},
			},
		},
	}, test.Changes[0])
}
