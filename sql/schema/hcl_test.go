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
	um := &Unmarshaler{}
	tables, err := um.UnmarshalHCL(bytes, filename)
	require.NoError(t, err)
	require.EqualValues(t, tables[0].Name, "users")
	require.EqualValues(t, tables[0].Columns[0].Name, "id")
	require.EqualValues(t, tables[0].Columns[1].Name, "name")
	require.EqualValues(t, tables[1].Name, "roles")
	require.EqualValues(t, tables[1].Columns[0].Name, "id")
	require.EqualValues(t, tables[1].Columns[1].Name, "name")
}
