// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/stretchr/testify/require"
)

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
		Columns: []*schemahcl.Ref{
			{V: "$column.parent_id"},
		},
		RefColumns: []*schemahcl.Ref{
			{V: "$column.id"},
		},
		OnUpdate: &schemahcl.Ref{V: "NO_ACTION"},
		OnDelete: &schemahcl.Ref{V: "CASCADE"},
	}, key)

	fk.OnDelete = ""
	fk.OnUpdate = ""
	key, err = FromForeignKey(fk)
	require.NoError(t, err)
	require.EqualValues(t, &sqlspec.ForeignKey{
		Symbol: "fk",
		Columns: []*schemahcl.Ref{
			{V: "$column.parent_id"},
		},
		RefColumns: []*schemahcl.Ref{
			{V: "$column.id"},
		},
	}, key)
}
