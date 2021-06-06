package schemahcl

import (
	"fmt"
	"testing"

	"ariga.io/atlas/sql/schema/schemacodec"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	c := `
table "table" {
	column "name" {
		type = "int"
		str_attr = "s"
		int_attr = 1
		float_attr = 1.1
		bool_attr = true
		
		feature "x" {
			hello = "world"
		}
	}
}
`
	tgt := &schemacodec.SchemaSpec{}
	err := Decode([]byte(c), tgt)
	require.NoError(t, err)

	expected := &schemacodec.SchemaSpec{
		Tables: []*schemacodec.TableSpec{
			{
				Name: "table",
				Columns: []*schemacodec.ColumnSpec{
					{
						Name:     "name",
						TypeName: "int",
						Attrs: []*schemacodec.Attr{
							{K: "str_attr", V: schemacodec.String("s")},
							{K: "int_attr", V: schemacodec.Number(1)},
							{K: "float_attr", V: schemacodec.Number(1.1)},
							{K: "bool_attr", V: schemacodec.Bool(true)},
						},
						Blocks: []*schemacodec.Block{
							{
								Type:   "feature",
								Labels: []string{"x"},
								Attrs: []*schemacodec.Attr{
									{K: "hello", V: schemacodec.String("world")},
								},
							},
						},
					},
				},
			},
		},
	}
	require.EqualValues(t, expected, tgt)
}

func TestEncode(t *testing.T) {
	in := &schemacodec.TableSpec{
		Name: "table",
		Columns: []*schemacodec.ColumnSpec{
			{
				Name:     "column",
				TypeName: "int",
				Null:     true,
			},
		},
	}
	bytes, err := Encode(in)
	fmt.Println(string(bytes), err)
}
