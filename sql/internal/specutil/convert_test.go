package specutil

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
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

func TestFromSpec_SchemaName(t *testing.T) {
	sc := &schema.Schema{
		Name: "schema",
		Tables: []*schema.Table{
			{},
		},
	}
	sc.Tables[0].Schema = sc
	s, ta, err := FromSchema(sc, func(table *schema.Table) (*sqlspec.Table, error) {
		return &sqlspec.Table{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, sc.Name, s.Name)
	require.Equal(t, sc.Name, ta[0].SchemaName)
}
