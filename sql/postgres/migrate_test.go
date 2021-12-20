// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPlanChanges(t *testing.T) {
	tests := []struct {
		changes []schema.Change
		plan    *migrate.Plan
	}{
		{
			changes: []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: "test"}},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE SCHEMA "test"`, Reverse: `DROP SCHEMA "test"`}}},
		},
		{
			changes: []schema.Change{
				&schema.DropSchema{S: &schema.Schema{Name: "atlas"}},
			},
			plan: &migrate.Plan{
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `DROP SCHEMA "atlas"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL, "text" text NULL)`, Reverse: `DROP TABLE "posts"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{T: &schema.Table{Name: "posts"}},
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `DROP TABLE "posts"`},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.AddColumn{
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "id_key",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0]},
									},
									Attrs: []schema.Attr{
										&schema.Comment{Text: "comment"},
									},
								},
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TABLE "users" ADD COLUMN "name" character varying(255) NOT NULL`, Reverse: `ALTER TABLE "users" DROP COLUMN "name"`},
					{Cmd: `CREATE INDEX "id_key" ON "users" ("id")`, Reverse: `DROP INDEX "id_key"`},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropColumn{
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}},
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TABLE "users" DROP COLUMN "name"`},
				},
			},
		},
	}
	for _, tt := range tests {
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		mock{m}.version("130000")
		drv, err := Open(db)
		require.NoError(t, err)
		plan, err := drv.PlanChanges(context.Background(), "plan", tt.changes)
		require.NoError(t, err)
		require.Equal(t, tt.plan.Reversible, plan.Reversible)
		require.Equal(t, tt.plan.Transactional, plan.Transactional)
		for i, c := range plan.Changes {
			require.Equal(t, tt.plan.Changes[i].Cmd, c.Cmd)
			require.Equal(t, tt.plan.Changes[i].Reverse, c.Reverse)
		}
	}
}
