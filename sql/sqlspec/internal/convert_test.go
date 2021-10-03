package internal

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestRef_ColumnName(t *testing.T) {
	ref := &schemaspec.Ref{V: "$table.accounts.$column.user_active"}
	c, err := columnName(ref)
	require.NoError(t, err)
	require.Equal(t, "user_active", c)
}

func TestRef_TableName(t *testing.T) {
	ref := &schemaspec.Ref{V: "$table.accounts.$column.user_active"}
	c, err := tableName(ref)
	require.NoError(t, err)
	require.Equal(t, "accounts", c)
}
