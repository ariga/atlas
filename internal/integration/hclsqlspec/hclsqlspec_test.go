package hclsqlspec

import (
	"testing"

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
	}
}
`)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.File{
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
					},
				},
			},
		},
	}, file)
}

func decode(f string) (*sqlspec.File, error) {
	res, err := schemahcl.Decode([]byte(f))
	if err != nil {
		return nil, err
	}
	s := sqlspec.File{}
	if err := res.As(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
