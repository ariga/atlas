// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
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
		options []migrate.PlanOption
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
				&schema.DropTable{T: schema.NewTable("posts").AddColumns(schema.NewIntColumn("id", "integer"))},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{
						Cmd:     "DROP TABLE `posts`",
						Reverse: "CREATE TABLE `posts` (`id` integer NOT NULL)",
					},
					{Cmd: "PRAGMA foreign_keys = on"},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{
					T: func() *schema.Table {
						id := schema.NewIntColumn("id", "int")
						return schema.NewTable("posts").
							SetComment("a8m's posts").
							AddColumns(id).
							AddIndexes(
								schema.NewIndex("idx").AddColumns(id).SetComment("a8m's index"),
							)
					}(),
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{
						Cmd: "DROP TABLE `posts`",
						Reverse: []string{
							"CREATE TABLE `posts` (`id` int NOT NULL)",
							"CREATE INDEX `idx` ON `posts` (`id`)",
						},
					},
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
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "ALTER TABLE `users` ADD COLUMN `name` varchar(255) NOT NULL", Reverse: "ALTER TABLE `users` DROP COLUMN `name`"},
					{Cmd: "CREATE INDEX `id_key` ON `users` (`id`)", Reverse: "DROP INDEX `id_key`"},
				},
			},
		},
		// Add VIRTUAL column.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(schema.NewIntColumn("id", "bigint"))
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.AddColumn{
								C: schema.NewIntColumn("nid", "bigint").
									SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "ALTER TABLE `users` ADD COLUMN `nid` bigint NOT NULL AS (1)", Reverse: "ALTER TABLE `users` DROP COLUMN `nid`"},
				},
			},
		},
		// Add STORED column.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(
							schema.NewIntColumn("id", "bigint"),
							schema.NewIntColumn("nid", "bigint").
								SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						)
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.AddColumn{
								C: users.Columns[1],
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{Cmd: "CREATE TABLE `new_users` (`id` bigint NOT NULL, `nid` bigint NOT NULL AS (1) STORED)", Reverse: "DROP TABLE `new_users`"},
					{Cmd: "INSERT INTO `new_users` (`id`) SELECT `id` FROM `users`"},
					{Cmd: "DROP TABLE `users`"},
					{Cmd: "ALTER TABLE `new_users` RENAME TO `users`"},
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
							schema.NewIntColumn("id", "bigint"),
							schema.NewIntColumn("rank", "int").SetDefault(&schema.Literal{V: "1"}),
							schema.NewStringColumn("nick", "text").SetDefault(&schema.Literal{V: "a8m"}),
						},
						Attrs: []schema.Attr{
							&schema.Check{Expr: "(id <> 0)"},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   schema.NewNullIntColumn("id", "bigint"),
								To:     users.Columns[1],
								Change: schema.ChangeNull | schema.ChangeDefault,
							},
							&schema.ModifyColumn{
								From:   schema.NewNullStringColumn("nick", "text"),
								To:     users.Columns[2],
								Change: schema.ChangeNull | schema.ChangeDefault,
							},
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
					{Cmd: "CREATE TABLE `new_users` (`id` bigint NOT NULL, `rank` int NOT NULL DEFAULT 1, `nick` text NOT NULL DEFAULT 'a8m', CHECK (id <> 0))", Reverse: "DROP TABLE `new_users`"},
					{Cmd: "INSERT INTO `new_users` (`id`, `rank`, `nick`) SELECT `id`, IFNULL(`rank`, 1) AS `rank`, IFNULL(`nick`, 'a8m') AS `nick` FROM `users`"},
					{Cmd: "DROP TABLE `users`"},
					{Cmd: "ALTER TABLE `new_users` RENAME TO `users`"},
					{Cmd: "PRAGMA foreign_keys = on"},
				},
			},
		},
		// Nothing to INSERT.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "c2", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropColumn{
								C: &schema.Column{Name: "c1", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}},
							},
							&schema.AddColumn{
								C: &schema.Column{Name: "c2", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: "PRAGMA foreign_keys = off"},
					{Cmd: "CREATE TABLE `new_users` (`c2` bigint NOT NULL)", Reverse: "DROP TABLE `new_users`"},
					/* Nothing to INSERT from `users` as `c1` was dropped. */
					{Cmd: "DROP TABLE `users`"},
					{Cmd: "ALTER TABLE `new_users` RENAME TO `users`"},
					{Cmd: "PRAGMA foreign_keys = on"},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.RenameTable{
					From: schema.NewTable("t1"),
					To:   schema.NewTable("t2"),
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `t1` RENAME TO `t2`",
						Reverse: "ALTER TABLE `t2` RENAME TO `t1`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1"),
					Changes: []schema.Change{
						&schema.RenameColumn{
							From: schema.NewColumn("a"),
							To:   schema.NewColumn("b"),
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `t1` RENAME COLUMN `a` TO `b`",
						Reverse: "ALTER TABLE `t1` RENAME COLUMN `b` TO `a`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1"),
					Changes: []schema.Change{
						&schema.RenameIndex{
							From: schema.NewIndex("a").AddColumns(schema.NewColumn("a")),
							To:   schema.NewIndex("b").AddColumns(schema.NewColumn("a")),
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE INDEX `b` ON `t1` (`a`)",
						Reverse: "DROP INDEX `b`",
					},
					{
						Cmd:     "DROP INDEX `a`",
						Reverse: "CREATE INDEX `a` ON `t1` (`a`)",
					},
				},
			},
		},
		// The default is no qualifier.
		{
			changes: []schema.Change{
				&schema.AddTable{T: schema.NewTable("t").SetSchema(schema.New("main")).AddColumns(schema.NewIntColumn("a", "int"))},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `t` (`a` int NOT NULL)",
						Reverse: "DROP TABLE `t`",
					},
				},
			},
		},
		// Custom qualifier.
		{
			changes: []schema.Change{
				&schema.AddTable{T: schema.NewTable("t").SetSchema(schema.New("d")).AddColumns(schema.NewIntColumn("a", "int"))},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) {
					s := "other"
					o.SchemaQualifier = &s
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `other`.`t` (`a` int NOT NULL)",
						Reverse: "DROP TABLE `other`.`t`",
					},
				},
			},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			db, mk, err := sqlmock.New()
			require.NoError(t, err)
			m := mock{mk}
			m.systemVars("3.36.0")
			if tt.mock != nil {
				tt.mock(m)
			}
			drv, err := Open(db)
			require.NoError(t, err)
			plan, err := drv.PlanChanges(context.Background(), "plan", tt.changes, tt.options...)
			require.NoError(t, err)
			require.Equal(t, tt.plan.Reversible, plan.Reversible)
			require.Equal(t, tt.plan.Transactional, plan.Transactional)
			for i, c := range plan.Changes {
				require.Equal(t, tt.plan.Changes[i].Cmd, c.Cmd)
				require.Equal(t, tt.plan.Changes[i].Reverse, c.Reverse)
			}
		})
	}
}

func TestDefaultPlan(t *testing.T) {
	changes, err := DefaultPlan.PlanChanges(context.Background(), "plan", []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("a", "int"))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(changes.Changes))
	require.Equal(t, "CREATE TABLE `t1` (`a` int NOT NULL)", changes.Changes[0].Cmd)

	err = DefaultPlan.ApplyChanges(context.Background(), []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("a", "int"))},
	})
	require.EqualError(t, err, `create "t1" table: cannot execute statements without a database connection. use Open to create a new Driver`)
}

func TestIndentedPlan(t *testing.T) {
	tests := []struct {
		T   *schema.Table
		Cmd string
	}{
		{
			T: schema.NewTable("t1").
				AddColumns(schema.NewIntColumn("a", "int")),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				).
				SetPrimaryKey(
					schema.NewPrimaryKey(schema.NewIntColumn("id", "int")),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  `id` int NOT NULL,",
				"  PRIMARY KEY (`id`)",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				).
				AddForeignKeys(
					schema.NewForeignKey("fk1").
						AddColumns(schema.NewIntColumn("a", "int")).
						SetRefTable(schema.NewTable("t2")).
						AddRefColumns(schema.NewIntColumn("a", "int")),
					schema.NewForeignKey("fk2").
						AddColumns(schema.NewIntColumn("a", "int")).
						SetRefTable(schema.NewTable("t2")).
						AddRefColumns(schema.NewIntColumn("a", "int")),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  CONSTRAINT `fk1` FOREIGN KEY (`a`) REFERENCES `t2` (`a`),",
				"  CONSTRAINT `fk2` FOREIGN KEY (`a`) REFERENCES `t2` (`a`)",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				).
				SetPrimaryKey(
					schema.NewPrimaryKey(schema.NewIntColumn("id", "int")),
				).
				AddForeignKeys(
					schema.NewForeignKey("fk1").
						AddColumns(schema.NewIntColumn("a", "int")).
						SetRefTable(schema.NewTable("t2")).
						AddRefColumns(schema.NewIntColumn("a", "int")),
					schema.NewForeignKey("fk2").
						AddColumns(schema.NewIntColumn("a", "int")).
						SetRefTable(schema.NewTable("t2")).
						AddRefColumns(schema.NewIntColumn("a", "int")),
				).
				AddChecks(
					schema.NewCheck().SetName("ck1").SetExpr("a > 0"),
					schema.NewCheck().SetName("ck2").SetExpr("a > 0"),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  `id` int NOT NULL,",
				"  PRIMARY KEY (`id`),",
				"  CONSTRAINT `fk1` FOREIGN KEY (`a`) REFERENCES `t2` (`a`),",
				"  CONSTRAINT `fk2` FOREIGN KEY (`a`) REFERENCES `t2` (`a`),",
				"  CONSTRAINT `ck1` CHECK (a > 0),",
				"  CONSTRAINT `ck2` CHECK (a > 0)",
				`)`,
			),
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			db, mk, err := sqlmock.New()
			require.NoError(t, err)
			mock{mk}.systemVars("3.36.0")
			drv, err := Open(db)
			require.NoError(t, err)
			plan, err := drv.PlanChanges(context.Background(), "wantPlan", []schema.Change{&schema.AddTable{T: tt.T}}, func(opts *migrate.PlanOptions) {
				opts.Indent = "  "
			})
			require.NoError(t, err)
			require.Len(t, plan.Changes, 1)
			require.Equal(t, tt.Cmd, plan.Changes[0].Cmd)
		})
	}
}

func join(lines ...string) string { return strings.Join(lines, "\n") }
