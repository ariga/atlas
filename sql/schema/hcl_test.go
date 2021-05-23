package schema

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicSchemaUnmarshal(t *testing.T) {
	filename := "testdata/basic_schema.hcl"
	bytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	schemas, err := UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	require.EqualValues(t, schemas[0].Name, "todo")
	tables := schemas[0].Tables
	require.EqualValues(t, tables[0].Name, "users")
	require.EqualValues(t, tables[0].Columns[0].Name, "id")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &IntegerType{
			T:        "integer",
			Size:     1,
			Unsigned: true,
		},
	}, tables[0].Columns[0].Type)
	require.EqualValues(t, tables[0].Columns[1].Name, "name")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &IntegerType{
			T:    "integer",
			Size: 0,
		},
	}, tables[1].Columns[0].Type)
	require.EqualValues(t, tables[1].Name, "roles")
	require.EqualValues(t, tables[1].Columns[0].Name, "id")
	require.EqualValues(t, tables[1].Columns[1].Name, "name")
	require.EqualValues(t, &ColumnType{
		Null: false,
		Type: &StringType{
			T:    "string",
			Size: 0,
		},
	}, tables[1].Columns[1].Type)
	require.EqualValues(t, tables[2].Name, "todos")
	require.EqualValues(t, &EnumType{
		Values: []string{"pending", "in_progress", "done"},
	}, tables[2].Columns[2].Type.Type)
}
