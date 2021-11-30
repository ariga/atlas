package hclsqlspec

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	f := `
modify_table "users" {
	add_column {
		column "id" {
			type = "int"
		}
	}
}

add_table {
	table "products" {
		column "sku" {
			type = "string"
		}
		index "xyz" {
			unique = true
		}
	}
}

drop_table "products" {}
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
					Name: "id",
					Null: false,
					Type: "int",
				},
			},
		},
	}, test.Changes[0])
	require.EqualValues(t, &sqlspec.AddTable{
		Table: &sqlspec.Table{
			Name: "products",
			Columns: []*sqlspec.Column{
				{Name: "sku", Type: "string"},
			},
			Indexes: []*sqlspec.Index{
				{Name: "xyz", Unique: true},
			},
			ForeignKeys: []*sqlspec.ForeignKey{},
		},
	}, test.Changes[1])
	require.EqualValues(t, &sqlspec.DropTable{
		Table: "products",
	}, test.Changes[2])
}
