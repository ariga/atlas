package hclsqlspec

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/stretchr/testify/require"
)

var hcl = schemahcl.New(schemahcl.WithTypes(postgres.TypeRegistry.Specs()))

func TestMigrate(t *testing.T) {
	f := `
modify_table {
	table = "users"
	add_column {
		column "id" {
			type = int
		}
	}
}
`
	var test struct {
		Changes []sqlspec.Change `spec:""`
	}
	err := hcl.UnmarshalSpec([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ModifyTable{
		Table: "users",
		Changes: []sqlspec.Change{
			&sqlspec.AddColumn{
				Column: &sqlspec.Column{
					Name: "id",
					Null: false,
					Type: &schemaspec.Type{T: "int"},
				},
			},
		},
	}, test.Changes[0])
}
