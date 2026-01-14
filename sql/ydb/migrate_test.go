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

func TestPlanChanges_RenameTable(t *testing.T) {
	changes := []schema.Change{
		&schema.RenameTable{
			From: func() *schema.Table {
				t := schema.NewTable("old_users").
					AddColumns(
						schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
						schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
					)
				t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
				return t
			}(),
			To: func() *schema.Table {
				t := schema.NewTable("new_users").
					AddColumns(
						schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
						schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
					)
				t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
				return t
			}(),
		},
	}

	plan, err := DefaultPlan.PlanChanges(context.Background(), "test", changes)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	require.Equal(t, "ALTER TABLE `old_users` RENAME TO `new_users`", plan.Changes[0].Cmd)
	require.Equal(t, "ALTER TABLE `new_users` RENAME TO `old_users`", plan.Changes[0].Reverse)
	require.Equal(t, `rename a table from "old_users" to "new_users"`, plan.Changes[0].Comment)
}

func TestPlanChanges_AddColumn(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "add single column",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddColumn{
							C: schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD COLUMN `email` utf8 NOT NULL",
						Reverse: "ALTER TABLE `users` DROP COLUMN `email`",
						Comment: `modify "users" table`,
					},
				},
			},
		},
		{
			name: "add nullable column",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddColumn{
							C: schema.NewNullColumn("bio").SetType(&schema.StringType{T: TypeUtf8}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD COLUMN `bio` utf8",
						Reverse: "ALTER TABLE `users` DROP COLUMN `bio`",
						Comment: `modify "users" table`,
					},
				},
			},
		},
		{
			name: "add multiple columns",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddColumn{
							C: schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
						},
						&schema.AddColumn{
							C: schema.NewColumn("age").SetType(&schema.IntegerType{T: TypeInt32}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD COLUMN `email` utf8 NOT NULL, ADD COLUMN `age` int32 NOT NULL",
						Reverse: "ALTER TABLE `users` DROP COLUMN `age`, DROP COLUMN `email`",
						Comment: `modify "users" table`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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

func TestPlanChanges_DropColumn(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
				schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
				schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "drop single column",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.DropColumn{
							C: schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP COLUMN `email`",
						Reverse: "ALTER TABLE `users` ADD COLUMN `email` utf8 NOT NULL",
						Comment: `modify "users" table`,
					},
				},
			},
		},
		{
			name: "drop multiple columns",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.DropColumn{
							C: schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
						},
						&schema.DropColumn{
							C: schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP COLUMN `name`, DROP COLUMN `email`",
						Reverse: "ALTER TABLE `users` ADD COLUMN `email` utf8 NOT NULL, ADD COLUMN `name` utf8 NOT NULL",
						Comment: `modify "users" table`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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

func TestPlanChanges_AddIndex(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
				schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
				schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "add single column index",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddIndex{
							I: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`)",
						Reverse: "ALTER TABLE `users` DROP INDEX `idx_name`",
						Comment: `create index "idx_name" to table: "users"`,
					},
				},
			},
		},
		{
			name: "add composite index",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddIndex{
							I: schema.NewIndex("idx_name_email").AddColumns(usersTable.Columns[1], usersTable.Columns[2]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD INDEX `idx_name_email` GLOBAL SYNC ON (`name`, `email`)",
						Reverse: "ALTER TABLE `users` DROP INDEX `idx_name_email`",
						Comment: `create index "idx_name_email" to table: "users"`,
					},
				},
			},
		},
		{
			name: "add multiple indexes",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.AddIndex{
							I: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
						},
						&schema.AddIndex{
							I: schema.NewIndex("idx_email").AddColumns(usersTable.Columns[2]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`)",
						Reverse: "ALTER TABLE `users` DROP INDEX `idx_name`",
						Comment: `create index "idx_name" to table: "users"`,
					},
					{
						Cmd:     "ALTER TABLE `users` ADD INDEX `idx_email` GLOBAL SYNC ON (`email`)",
						Reverse: "ALTER TABLE `users` DROP INDEX `idx_email`",
						Comment: `create index "idx_email" to table: "users"`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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

func TestPlanChanges_DropIndex(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
				schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
				schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "drop single index",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.DropIndex{
							I: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP INDEX `idx_name`",
						Reverse: "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`)",
						Comment: `drop index "idx_name" from table: "users"`,
					},
				},
			},
		},
		{
			name: "drop multiple indexes",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.DropIndex{
							I: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
						},
						&schema.DropIndex{
							I: schema.NewIndex("idx_email").AddColumns(usersTable.Columns[2]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP INDEX `idx_name`",
						Reverse: "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`)",
						Comment: `drop index "idx_name" from table: "users"`,
					},
					{
						Cmd:     "ALTER TABLE `users` DROP INDEX `idx_email`",
						Reverse: "ALTER TABLE `users` ADD INDEX `idx_email` GLOBAL SYNC ON (`email`)",
						Comment: `drop index "idx_email" from table: "users"`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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

func TestPlanChanges_ModifyIndex(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
				schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
				schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "modify index (drop and recreate)",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.ModifyIndex{
							From: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
							To:   schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1], usersTable.Columns[2]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP INDEX `idx_name`",
						Reverse: "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`)",
						Comment: `drop index "idx_name" from table: "users"`,
					},
					{
						Cmd:     "ALTER TABLE `users` ADD INDEX `idx_name` GLOBAL SYNC ON (`name`, `email`)",
						Reverse: "ALTER TABLE `users` DROP INDEX `idx_name`",
						Comment: `create index "idx_name" to table: "users"`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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

func TestPlanChanges_RenameIndex(t *testing.T) {
	usersTable := func() *schema.Table {
		t := schema.NewTable("users").
			AddColumns(
				schema.NewColumn("id").SetType(&schema.IntegerType{T: TypeInt64}),
				schema.NewColumn("name").SetType(&schema.StringType{T: TypeUtf8}),
				schema.NewColumn("email").SetType(&schema.StringType{T: TypeUtf8}),
			)
		t.SetPrimaryKey(schema.NewPrimaryKey(t.Columns[0]))
		return t
	}()

	tests := []struct {
		name     string
		changes  []schema.Change
		wantPlan *migrate.Plan
	}{
		{
			name: "rename index",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: usersTable,
					Changes: []schema.Change{
						&schema.RenameIndex{
							From: schema.NewIndex("idx_name").AddColumns(usersTable.Columns[1]),
							To:   schema.NewIndex("idx_user_name").AddColumns(usersTable.Columns[1]),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` RENAME INDEX `idx_name` TO `idx_user_name`",
						Reverse: "ALTER TABLE `users` RENAME INDEX `idx_user_name` TO `idx_name`",
						Comment: `rename an index from "idx_name" to "idx_user_name"`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := DefaultPlan.PlanChanges(context.Background(), "test", tt.changes)
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
