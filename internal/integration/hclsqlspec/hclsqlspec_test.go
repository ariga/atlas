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
				},
			},
		},
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
	schemaspec.Extension
}

