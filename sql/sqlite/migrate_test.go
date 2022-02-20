// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPlanChanges(t *testing.T) {
	tests := []struct {
		changes []schema.Change
		mock    func(mock)
		plan    *migrate.Plan
	}{
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
						},
						Attrs: []schema.Attr{
							&schema.Check{Expr: "(text <> '')"},
							&schema.Check{Name: "positive_id", Expr: "(id <> 0)"},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: "CREATE TABLE `posts` (`id` integer NOT NULL, `text` text NULL, CHECK (text <> ''), CONSTRAINT `positive_id` CHECK (id <> 0))", Reverse: "DROP TABLE `posts`"}},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&AutoIncrement{Seq: 1024}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
						},
						Attrs: []schema.Attr{
							&schema.Check{Expr: "(text <> '')"},
							&schema.Check{Name: "positive_id", Expr: "(id <> 0)"},
						},
					},
				},
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT seq FROM sqlite_sequence WHERE name = ?")).
					WithArgs("posts").
					WillReturnError(sql.ErrNoRows)
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "CREATE TABLE `posts` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `text` text NULL, CHECK (text <> ''), CONSTRAINT `positive_id` CHECK (id <> 0))", Reverse: "DROP TABLE `posts`"},
					{Cmd: `INSERT INTO sqlite_sequence (name, seq) VALUES ("posts", 1024)`, Reverse: `UPDATE sqlite_sequence SET seq = 0 WHERE name = "posts"`},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{T: &schema.Table{Name: "posts"}},
			},
			plan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{Cmd: "DROP TABLE `posts`"},
					{Cmd: "PRAGMA foreign_keys = on"},
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
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}},
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
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "ALTER TABLE `users` ADD COLUMN `name` varchar(255) NOT NULL"},
					{Cmd: "CREATE INDEX `id_key` ON `users` (`id`)", Reverse: "DROP INDEX `id_key`"},
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
						Attrs: []schema.Attr{
							&schema.Check{Expr: "(id <> 0)"},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropColumn{
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}},
							},
							&schema.AddCheck{
								C: &schema.Check{Expr: "(id <> 0)"},
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{Cmd: "CREATE TABLE `new_users` (`id` bigint NOT NULL, CHECK (id <> 0))", Reverse: "DROP TABLE `new_users`"},
					{Cmd: "INSERT INTO `new_users` (`id`) SELECT `id` FROM `users`"},
					{Cmd: "DROP TABLE `users`"},
					{Cmd: "ALTER TABLE `new_users` RENAME TO `users`"},
					{Cmd: "PRAGMA foreign_keys = on"},
				},
			},
		},
	}
	for _, tt := range tests {
		db, mk, err := sqlmock.New()
		require.NoError(t, err)
		m := mock{mk}
		m.systemVars("3.36.0")
		if tt.mock != nil {
			tt.mock(m)
		}
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
