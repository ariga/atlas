// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func TestDefaultPlan(t *testing.T) {
	changes, err := DefaultPlan.PlanChanges(context.Background(), "plan", []schema.Change{
		&schema.AddTable{
			T: schema.NewTable("t1").
				SetSchema(schema.New("dbo")).
				AddColumns(
					schema.NewIntColumn("a", "int"),
				),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(changes.Changes))
	require.Equal(t, `CREATE TABLE "dbo"."t1" ("a" int NOT NULL)`, changes.Changes[0].Cmd)
}

func TestTable_Rename(t *testing.T) {
	changes, err := DefaultPlan.PlanChanges(context.Background(), "plan", []schema.Change{
		&schema.RenameTable{
			From: schema.NewTable("t1").
				SetSchema(schema.New("dbo")).
				AddColumns(
					schema.NewIntColumn("a", "int"),
				),
			To: schema.NewTable("t2").
				SetSchema(schema.New("dbo")).
				AddColumns(
					schema.NewIntColumn("a", "int"),
				),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(changes.Changes))
	require.Equal(t, `EXEC sp_rename '"dbo"."t1"', '"t2"'`, changes.Changes[0].Cmd)
}
