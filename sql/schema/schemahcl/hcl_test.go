package schemahcl

import (
	"testing"

	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	c := `column "name" {
	type = "int"
	str_attr = "s"
	int_attr = 1
	float_attr = 1.1
	bool_attr = true
	
	feature "x" {
		hello = "world"
	}
}`
	col, err := Parse([]byte(c), "file.hcl")
	require.NoError(t, err)

	require.EqualValues(t, &schemaspec.Column{
		Name:     "name",
		TypeName: "int",
		Attrs: []*schemaspec.Attr{
			{K: "str_attr", V: schemaspec.String("s")},
			{K: "int_attr", V: schemaspec.Number(1)},
			{K: "float_attr", V: schemaspec.Number(1.1)},
			{K: "bool_attr", V: schemaspec.Bool(true)},
		},
	}, col[0])

}
