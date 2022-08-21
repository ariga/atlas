// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package myparse_test

import (
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/myparse"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestFixChange_RenameColumns(t *testing.T) {
	_, err := myparse.FixChange(
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		nil,
	)
	require.Error(t, err)

	_, err = myparse.FixChange(
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{&schema.AddTable{}},
	)
	require.Error(t, err)

	changes, err := myparse.FixChange(
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.DropColumn{C: schema.NewColumn("c1")},
					&schema.AddColumn{C: schema.NewColumn("c2")},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.RenameColumn{From: schema.NewColumn("c1"), To: schema.NewColumn("c2")},
				},
			},
		},
		changes,
	)

	changes, err = myparse.FixChange(
		"ALTER TABLE t ADD INDEX i(id), RENAME COLUMN c1 TO c2, ADD COLUMN c3 int, DROP COLUMN c4",
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i").AddColumns(schema.NewColumn("id"))},
					&schema.DropColumn{C: schema.NewColumn("c1")},
					&schema.AddColumn{C: schema.NewColumn("c2")},
					&schema.AddColumn{C: schema.NewColumn("c3")},
					&schema.AddColumn{C: schema.NewColumn("c4")},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.ModifyTable{
				Changes: schema.Changes{
					&schema.AddIndex{I: schema.NewIndex("i").AddColumns(schema.NewColumn("id"))},
					&schema.RenameColumn{From: schema.NewColumn("c1"), To: schema.NewColumn("c2")},
					&schema.AddColumn{C: schema.NewColumn("c3")},
					&schema.AddColumn{C: schema.NewColumn("c4")},
				},
			},
		},
		changes,
	)
}

func TestFixChange_RenameTable(t *testing.T) {
	changes, err := myparse.FixChange(
		"RENAME TABLE t1 TO t2",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
		},
		changes,
	)
	changes, err = myparse.FixChange(
		"RENAME TABLE t1 TO t2, t3 TO t4",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
			&schema.DropTable{T: schema.NewTable("t3")},
			&schema.AddTable{T: schema.NewTable("t4")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
			&schema.RenameTable{From: schema.NewTable("t3"), To: schema.NewTable("t4")},
		},
		changes,
	)
}
