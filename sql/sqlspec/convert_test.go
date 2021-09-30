package sqlspec

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestRef_GetColumnName(t *testing.T) {
	ref := &schemaspec.Ref{V: "$table.accounts.$column.user_active"}
	c, err := getColumnName(ref)
	require.NoError(t, err)
	require.Equal(t, "user_active", c)
}

func TestRef_GetTableName(t *testing.T) {
	ref := &schemaspec.Ref{V: "$table.accounts.$column.user_active"}
	c, err := getTableName(ref)
	require.NoError(t, err)
	require.Equal(t, "accounts", c)
}
