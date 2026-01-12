// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestPlanChanges_AddTable(t *testing.T) {
	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
		wantErr  bool
	}{
		{
			name: "simple table with primary key",
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").
						AddColumns(
							schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
							schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
						).
						SetPrimaryKey(schema.NewPrimaryKey(
							schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
						)),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `users` (`id` int64 NOT NULL, `name` utf8 NOT NULL, PRIMARY KEY (`id`))",
						Reverse: "DROP TABLE `users`",
						Comment: `create "users" table`,
					},
				},
			},
		},
		{
			name: "table with nullable column",
			changes: []schema.Change{
				&schema.AddTable{
					T: func() *schema.Table {
						t := schema.NewTable("users").
							AddColumns(
								schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
								schema.NewNullColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
						return t
					}(),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `users` (`id` int64 NOT NULL, `email` utf8, PRIMARY KEY (`id`))",
						Reverse: "DROP TABLE `users`",
						Comment: `create "users" table`,
					},
				},
			},
		},
		{
			name: "table with secondary index",
			changes: []schema.Change{
				&schema.AddTable{
					T: func() *schema.Table {
						t := schema.NewTable("users").
							AddColumns(
								schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
								schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
						t.AddIndexes(schema.NewIndex("idx_name").AddColumns(t.Columns[1]))
						return t
					}(),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `users` (`id` int64 NOT NULL, `name` utf8 NOT NULL, PRIMARY KEY (`id`), INDEX `idx_name` GLOBAL ON (`name`))",
						Reverse: "DROP TABLE `users`",
						Comment: `create "users" table`,
					},
				},
			},
		},
		{
			name: "table with composite primary key",
			changes: []schema.Change{
				&schema.AddTable{
					T: func() *schema.Table {
						t := schema.NewTable("order_items").
							AddColumns(
								schema.NewColumn("order_id").SetType(&schema.IntegerType{T: TypeInt64}),
								schema.NewColumn("item_id").SetType(&schema.IntegerType{T: TypeInt64}),
								schema.NewColumn("quantity").SetType(&schema.IntegerType{T: TypeInt32}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0], t.Columns[1]))
						return t
					}(),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `order_items` (`order_id` int64 NOT NULL, `item_id` int64 NOT NULL, `quantity` int32 NOT NULL, PRIMARY KEY (`order_id`, `item_id`))",
						Reverse: "DROP TABLE `order_items`",
						Comment: `create "order_items" table`,
					},
				},
			},
		},
		{
			name: "table without primary key - error",
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").
						AddColumns(
							schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
						),
				},
			},
			wantErr: true,
		},
		{
			name: "table with various types",
			changes: []schema.Change{
				&schema.AddTable{
					T: func() *schema.Table {
						t := schema.NewTable("data").
							AddColumns(
								schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
								schema.NewColumn("flag").SetType(&schema.BoolType{T: TypeBool}),
								schema.NewColumn("price").SetType(&schema.DecimalType{T: TypeDecimal, Precision: 10, Scale: 2}),
								schema.NewColumn("timestamp").SetType(&schema.TimeType{T: TypeTimestamp}),
								schema.NewColumn("data").SetType(&schema.JSONType{T: TypeJSON}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
						return t
					}(),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `data` (`id` int64 NOT NULL, `flag` bool NOT NULL, `price` decimal(10,2) NOT NULL, `timestamp` timestamp NOT NULL, `data` json NOT NULL, PRIMARY KEY (`id`))",
						Reverse: "DROP TABLE `data`",
						Comment: `create "data" table`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantPlan.Transactional, plan.Transactional)
			require.Len(t, plan.Changes, len(tt.wantPlan.Changes))
			for i, c := range plan.Changes {
				require.Equal(t, tt.wantPlan.Changes[i].Cmd, c.Cmd)
				require.Equal(t, tt.wantPlan.Changes[i].Reverse, c.Reverse)
				require.Equal(t, tt.wantPlan.Changes[i].Comment, c.Comment)
			}
		})
	}
}

func TestPlanChanges_DropTable(t *testing.T) {
	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
		wantErr  bool
	}{
		{
			name: "drop table",
			changes: []schema.Change{
				&schema.DropTable{
					T: func() *schema.Table {
						t := schema.NewTable("users").
							AddColumns(
								schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
						return t
					}(),
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "DROP TABLE `users`",
						Reverse: "CREATE TABLE `users` (`id` int64 NOT NULL, PRIMARY KEY (`id`))",
						Comment: `drop "users" table`,
					},
				},
			},
		},
		{
			name: "drop table if exists",
			changes: []schema.Change{
				&schema.DropTable{
					T: func() *schema.Table {
						t := schema.NewTable("users").
							AddColumns(
								schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
							)
						t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
						return t
					}(),
					Extra: []schema.Clause{&schema.IfExists{}},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "DROP TABLE IF EXISTS `users`",
						Reverse: "CREATE TABLE `users` (`id` int64 NOT NULL, PRIMARY KEY (`id`))",
						Comment: `drop "users" table`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantPlan.Transactional, plan.Transactional)
			require.Len(t, plan.Changes, len(tt.wantPlan.Changes))
			for i, c := range plan.Changes {
				require.Equal(t, tt.wantPlan.Changes[i].Cmd, c.Cmd)
				require.Equal(t, tt.wantPlan.Changes[i].Reverse, c.Reverse)
				require.Equal(t, tt.wantPlan.Changes[i].Comment, c.Comment)
			}
		})
	}
}

func TestPlanChanges_UnsupportedChange(t *testing.T) {
	// Test that unsupported changes return an error
	_, err := DefaultPlan.PlanChanges(context.Background(), "test", []schema.Change{
		&schema.ModifyTable{T: schema.NewTable("users")},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported change type")
}

func TestPlanChanges_MultipleTables(t *testing.T) {
	changes := []schema.Change{
		&schema.AddTable{
			T: func() *schema.Table {
				t := schema.NewTable("users").
					AddColumns(
						schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
					)
				t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
				return t
			}(),
		},
		&schema.AddTable{
			T: func() *schema.Table {
				t := schema.NewTable("posts").
					AddColumns(
						schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
						schema.NewColumn("user_id").SetType(&schema.IntegerType{T: TypeInt64}),
					)
				t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
				return t
			}(),
		},
	}

	plan, err := DefaultPlan.PlanChanges(context.Background(), "test", changes)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 2)
	require.Contains(t, plan.Changes[0].Cmd, "CREATE TABLE `users`")
	require.Contains(t, plan.Changes[1].Cmd, "CREATE TABLE `posts`")
}
