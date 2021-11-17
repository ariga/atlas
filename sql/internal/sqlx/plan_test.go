// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

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
	planned, err := DetachCycles(changes)
	require.NoError(t, err)
	require.Equal(t, changes, planned)

	deletion := []schema.Change{&schema.DropTable{T: users}, &schema.DropTable{T: workplaces}}
	planned, err = DetachCycles(deletion)
	require.NoError(t, err)
	require.Equal(t, deletion, planned)

	// Create a circular reference.
	workplaces.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
	}
	planned, err = DetachCycles(changes)
	require.NoError(t, err)
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

	planned, err = DetachCycles(deletion)
	require.NoError(t, err)
	require.Equal(t, &schema.ModifyTable{
		T: users,
		Changes: []schema.Change{
			&schema.DropForeignKey{
				F: &schema.ForeignKey{Symbol: "workplace", Table: users, Columns: users.Columns[1:], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
			},
		},
	}, planned[0])
	require.Equal(t, &schema.ModifyTable{
		T: workplaces,
		Changes: []schema.Change{
			&schema.DropForeignKey{
				F: &schema.ForeignKey{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
			},
		},
	}, planned[1])
	users.ForeignKeys = nil
	workplaces.ForeignKeys = nil
	require.Equal(t, deletion, planned[2:])
}
