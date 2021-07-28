package sqlx

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestDetachCycles(t *testing.T) {
	users := &schema.Table{
		Name: "users",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
			{Name: "workplace_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
		},
	}
	workplaces := &schema.Table{
		Name: "workplaces",
		Columns: []*schema.Column{
			{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
			{Name: "owner_id", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
		},
	}
	users.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "workplace", Table: users, Columns: users.Columns[1:], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
	}
	changes := []schema.Change{&schema.AddTable{T: workplaces}, &schema.AddTable{T: users}}
	planned := DetachCycles(changes)
	require.Equal(t, changes, planned)

	// Create a circular reference.
	workplaces.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
	}
	planned = DetachCycles(changes)
	require.Len(t, planned, 4)
	require.Empty(t, planned[0].(*schema.AddTable).T.ForeignKeys)
	require.Empty(t, planned[1].(*schema.AddTable).T.ForeignKeys)
	require.Equal(t, &schema.ModifyTable{
		T: workplaces,
		Changes: []schema.Change{
			&schema.AddForeignKey{
				F: &schema.ForeignKey{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
			},
		},
	}, planned[2])
	require.Equal(t, &schema.ModifyTable{
		T: users,
		Changes: []schema.Change{
			&schema.AddForeignKey{
				F: &schema.ForeignKey{Symbol: "workplace", Table: users, Columns: users.Columns[1:], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
			},
		},
	}, planned[3])
}
