// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package pgparse_test

import (
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/pgparse"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestFixChange_RenameColumns(t *testing.T) {
	_, err := pgparse.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		nil,
	)
	require.Error(t, err)

	_, err = pgparse.FixChange(
		nil,
		"ALTER TABLE t RENAME COLUMN c1 TO c2",
		schema.Changes{&schema.AddTable{}},
	)
	require.Error(t, err)

	changes, err := pgparse.FixChange(
		nil,
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
}

func TestFixChange_RenameTable(t *testing.T) {
	changes, err := pgparse.FixChange(
		nil,
		"ALTER TABLE t1 RENAME TO t2",
		schema.Changes{
			&schema.DropTable{T: schema.NewTable("t1")},
			&schema.AddTable{T: schema.NewTable("t2")},
			&schema.AddTable{T: schema.NewTable("t3")},
		},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		schema.Changes{
			&schema.RenameTable{From: schema.NewTable("t1"), To: schema.NewTable("t2")},
			&schema.AddTable{T: schema.NewTable("t3")},
		},
		changes,
	)
}
