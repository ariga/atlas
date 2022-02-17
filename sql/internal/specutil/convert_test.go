package specutil

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

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
	require.Equal(t, "$schema."+sc.Name, ta[0].Schema.V)
}

func TestFromForeignKey(t *testing.T) {
	tbl := &schema.Table{
		Name: "users",
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{
					Type: &schema.IntegerType{
						T: "int",
					},
				},
			},
			{
				Name: "parent_id",
				Type: &schema.ColumnType{
					Type: &schema.IntegerType{
						T: "int",
					},
				},
			},
		},
	}
	fk := &schema.ForeignKey{
		Symbol:     "fk",
		Table:      tbl,
		Columns:    tbl.Columns[1:],
		RefTable:   tbl,
		RefColumns: tbl.Columns[:1],
		OnUpdate:   schema.NoAction,
		OnDelete:   schema.Cascade,
	}
	key, err := FromForeignKey(fk)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ForeignKey{
		Symbol: "fk",
		Columns: []*schemaspec.Ref{
			{V: "$column.parent_id"},
		},
		RefColumns: []*schemaspec.Ref{
			{V: "$column.id"},
		},
		OnUpdate: &schemaspec.Ref{V: "NO_ACTION"},
		OnDelete: &schemaspec.Ref{V: "CASCADE"},
	}, key)
}
