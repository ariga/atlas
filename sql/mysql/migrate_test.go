// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMigrate_ApplyChanges(t *testing.T) {
	migrate, mk, err := newMigrate("8.0.13")
	require.NoError(t, err)
	mk.ExpectExec(sqltest.Escape("CREATE DATABASE `test` CHARSET latin")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mk.ExpectExec(sqltest.Escape("DROP DATABASE `atlas`")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mk.ExpectExec(sqltest.Escape("DROP TABLE `users`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("DROP TABLE IF EXISTS `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE IF NOT EXISTS `pets` (`a` int NOT NULL DEFAULT (int(rand())), `b` bigint NOT NULL DEFAULT 1, `c` bigint NULL, PRIMARY KEY (`a`, `b`), UNIQUE INDEX `b_c_unique` (`b`, `c`) COMMENT \"comment\")")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` DROP INDEX `id_spouse_id`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` ADD CONSTRAINT `spouse` FOREIGN KEY (`spouse_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, ADD INDEX `id_spouse_id` (`spouse_id`, `id` DESC) COMMENT \"comment\"")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `posts` (`id` bigint NOT NULL, `author_id` bigint NULL, CONSTRAINT `author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `comments` (`id` bigint NOT NULL, `post_id` bigint NULL, CONSTRAINT `comment` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = migrate.ApplyChanges(context.Background(), []schema.Change{
		&schema.AddSchema{S: &schema.Schema{Name: "test", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
		&schema.DropSchema{S: &schema.Schema{Name: "atlas", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
		&schema.DropTable{T: &schema.Table{Name: "users"}},
		&schema.DropTable{T: &schema.Table{Name: "pets", Schema: &schema.Schema{Name: "public"}}, Extra: []schema.Clause{&schema.IfExists{}}},
		&schema.AddTable{
			T: func() *schema.Table {
				t := &schema.Table{
					Name: "pets",
					Columns: []*schema.Column{
						{Name: "a", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Default: &schema.RawExpr{X: "(int(rand()))"}},
						{Name: "b", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}, Default: &schema.Literal{V: "1"}},
						{Name: "c", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
					},
				}
				t.PrimaryKey = &schema.Index{
					Parts: []*schema.IndexPart{{C: t.Columns[0]}, {C: t.Columns[1]}},
				}
				t.Indexes = []*schema.Index{
					{Name: "b_c_unique", Unique: true, Parts: []*schema.IndexPart{{C: t.Columns[1]}, {C: t.Columns[2]}}, Attrs: []schema.Attr{&schema.Comment{Text: "comment"}}},
				}
				return t
			}(),
			Extra: []schema.Clause{
				&schema.IfNotExists{},
			},
		},
	})
	require.NoError(t, err)
	err = migrate.ApplyChanges(context.Background(), func() []schema.Change {
		users := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				{Name: "spouse_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
			},
		}
		posts := &schema.Table{
			Name: "posts",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				{Name: "author_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
			},
		}
		posts.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "author", Table: posts, Columns: posts.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
		}
		comments := &schema.Table{
			Name: "comments",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				{Name: "post_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
			},
		}
		comments.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "comment", Table: comments, Columns: comments.Columns[1:], RefTable: posts, RefColumns: posts.Columns[:1]},
		}
		return []schema.Change{
			&schema.AddTable{T: posts},
			&schema.AddTable{T: comments},
			&schema.ModifyTable{
				T: users,
				Changes: []schema.Change{
					&schema.AddForeignKey{
						F: &schema.ForeignKey{
							Symbol:     "spouse",
							Table:      users,
							Columns:    users.Columns[1:],
							RefTable:   users,
							RefColumns: users.Columns[:1],
							OnDelete:   "SET NULL",
						},
					},
					&schema.ModifyIndex{
						From: &schema.Index{Name: "id_spouse_id", Parts: []*schema.IndexPart{{C: users.Columns[0]}, {C: users.Columns[1]}}},
						To: &schema.Index{
							Name: "id_spouse_id",
							Parts: []*schema.IndexPart{
								{C: users.Columns[1]},
								{C: users.Columns[0], Desc: true},
							},
							Attrs: []schema.Attr{
								&schema.Comment{Text: "comment"},
							},
						},
					},
				},
			},
		}
	}())
	require.NoError(t, err)
}

func TestMigrate_DetachCycles(t *testing.T) {
	migrate, mk, err := newMigrate("8.0.13")
	require.NoError(t, err)
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `users` (`id` bigint NOT NULL, `workplace_id` bigint NULL)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `workplaces` (`id` bigint NOT NULL, `owner_id` bigint NULL)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` ADD CONSTRAINT `workplace` FOREIGN KEY (`workplace_id`) REFERENCES `workplaces` (`id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `workplaces` ADD CONSTRAINT `owner` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = migrate.ApplyChanges(context.Background(), func() []schema.Change {
		users := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				{Name: "workplace_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
			},
		}
		workplaces := &schema.Table{
			Name: "workplaces",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				{Name: "owner_id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true}},
			},
		}
		users.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "workplace", Table: users, Columns: users.Columns[1:], RefTable: workplaces, RefColumns: workplaces.Columns[:1]},
		}
		workplaces.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "owner", Table: workplaces, Columns: workplaces.Columns[1:], RefTable: users, RefColumns: users.Columns[:1]},
		}
		return []schema.Change{
			&schema.AddTable{T: users},
			&schema.AddTable{T: workplaces},
		}
	}())
	require.NoError(t, err)
}

func TestPlanChanges(t *testing.T) {
	tests := []struct {
		input    []schema.Change
		wantPlan *migrate.Plan
		wantErr  bool
	}{
		{
			input: []schema.Change{
				&schema.AddSchema{S: schema.New("test").SetCharset("utf8mb4"), Extra: []schema.Clause{&schema.IfNotExists{}}},
				&schema.DropSchema{S: schema.New("test").SetCharset("utf8mb4"), Extra: []schema.Clause{&schema.IfExists{}}},
			},
			wantPlan: &migrate.Plan{
				Reversible: false,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE DATABASE IF NOT EXISTS `test` CHARSET utf8mb4",
						Reverse: "DROP DATABASE `test`",
					},
					{
						Cmd: "DROP DATABASE IF EXISTS `test`",
					},
				},
			},
		},
		{
			input: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							{
								Name: "name",
								Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}},
								Indexes: []*schema.Index{
									schema.NewIndex("name_index").
										AddParts(schema.NewColumnPart(schema.NewColumn("name"))),
								},
							}},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropIndex{
								I: schema.NewIndex("name_index").
									AddParts(schema.NewColumnPart(schema.NewColumn("name"))),
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP INDEX `name_index`",
						Reverse: "ALTER TABLE `users` ADD INDEX `name_index` (`name`)",
					},
				},
			},
		},
		{
			input: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
						},
					}
					pets := &schema.Table{
						Name: "pets",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							{Name: "user_id",
								Type: &schema.ColumnType{
									Type: &schema.IntegerType{T: "bigint"},
								},
							},
						},
					}
					fk := &schema.ForeignKey{
						Symbol:     "user_id",
						Table:      pets,
						OnUpdate:   schema.NoAction,
						OnDelete:   schema.Cascade,
						RefTable:   users,
						Columns:    []*schema.Column{pets.Columns[1]},
						RefColumns: []*schema.Column{users.Columns[0]},
					}
					pets.ForeignKeys = []*schema.ForeignKey{fk}
					return &schema.ModifyTable{
						T: pets,
						Changes: []schema.Change{
							&schema.DropForeignKey{
								F: fk,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `pets` DROP FOREIGN KEY `user_id`",
						Reverse: "ALTER TABLE `pets` ADD CONSTRAINT `user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE",
					},
				},
			},
		},
		{
			input: []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: "test", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE DATABASE `test` CHARSET latin", Reverse: "DROP DATABASE `test`"}},
			},
		},
		// Default database charset can be omitted.
		{
			input: []schema.Change{
				&schema.AddSchema{S: schema.New("test").SetCharset("utf8")},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE DATABASE `test`", Reverse: "DROP DATABASE `test`"}},
			},
		},
		// Add the default database charset on modify can be omitted.
		{
			input: []schema.Change{
				&schema.ModifySchema{
					S: schema.New("test").SetCharset("utf8"),
					Changes: []schema.Change{
						&schema.AddAttr{A: &schema.Charset{V: "utf8"}},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
			},
		},
		// Add custom charset.
		{
			input: []schema.Change{
				&schema.ModifySchema{
					S: schema.New("test").SetCharset("latin1"),
					Changes: []schema.Change{
						&schema.AddAttr{A: &schema.Charset{V: "latin1"}},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "ALTER DATABASE `test` CHARSET latin1", Reverse: "ALTER DATABASE `test`CHARSET utf8"}},
			},
		},
		// Modify charset.
		{
			input: []schema.Change{
				&schema.ModifySchema{
					S: schema.New("test").SetCharset("utf8"),
					Changes: []schema.Change{
						&schema.ModifyAttr{From: &schema.Charset{V: "latin1"}, To: &schema.Charset{V: "utf8"}},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "ALTER DATABASE `test` CHARSET utf8", Reverse: "ALTER DATABASE `test`CHARSET latin1"}},
			},
		},
		{
			input: []schema.Change{
				&schema.DropSchema{S: &schema.Schema{Name: "atlas", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
			},
			wantPlan: &migrate.Plan{
				Changes: []*migrate.Change{{Cmd: "DROP DATABASE `atlas`"}},
			},
		},
		{
			input: []schema.Change{
				func() *schema.AddTable {
					t := &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoIncrement{}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
							{Name: "uuid", Type: &schema.ColumnType{Type: &schema.StringType{T: "char", Size: 36}, Null: true}, Attrs: []schema.Attr{&schema.Charset{V: "utf8mb4"}, &schema.Collation{V: "utf8mb4_bin"}}},
						},
					}
					t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
					return &schema.AddTable{T: t}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE TABLE `posts` (`id` bigint NOT NULL AUTO_INCREMENT, `text` text NULL, `uuid` char(36) NULL CHARSET utf8mb4 COLLATE utf8mb4_bin, PRIMARY KEY (`id`))", Reverse: "DROP TABLE `posts`"}},
			},
		},
		{
			input: []schema.Change{
				func() *schema.AddTable {
					t := &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoIncrement{V: 100}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
							{Name: "ch", Type: &schema.ColumnType{Type: &schema.StringType{T: "char"}}},
						},
						Attrs: []schema.Attr{
							&schema.Charset{V: "utf8mb4"},
							&schema.Collation{V: "utf8mb4_bin"},
							&schema.Comment{Text: "posts comment"},
							&schema.Check{Name: "id_nonzero", Expr: "(`id` > 0)"},
							&CreateOptions{V: `COMPRESSION="ZLIB"`},
						},
						Indexes: []*schema.Index{
							{
								Name: "text_prefix",
								Parts: []*schema.IndexPart{
									{Desc: true, Attrs: []schema.Attr{&SubPart{Len: 100}}},
								},
							},
						},
					}
					t.Indexes[0].Parts[0].C = t.Columns[1]
					t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
					return &schema.AddTable{T: t}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE TABLE `posts` (`id` bigint NOT NULL AUTO_INCREMENT, `text` text NULL, `ch` char NOT NULL, PRIMARY KEY (`id`), INDEX `text_prefix` (`text` (100) DESC), CONSTRAINT `id_nonzero` CHECK (`id` > 0)) CHARSET utf8mb4 COLLATE utf8mb4_bin COMMENT \"posts comment\" COMPRESSION=\"ZLIB\" AUTO_INCREMENT 100", Reverse: "DROP TABLE `posts`"}},
			},
		},
		{
			input: []schema.Change{
				func() *schema.AddTable {
					t := &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoIncrement{}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
						},
						Attrs: []schema.Attr{&AutoIncrement{V: 10}},
					}
					t.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: t.Columns[0]}}}
					return &schema.AddTable{T: t}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE TABLE `posts` (`id` bigint NOT NULL AUTO_INCREMENT, `text` text NULL, PRIMARY KEY (`id`)) AUTO_INCREMENT 10", Reverse: "DROP TABLE `posts`"}},
			},
		},
		{
			input: []schema.Change{
				&schema.DropTable{T: &schema.Table{Name: "posts"}},
			},
			wantPlan: &migrate.Plan{
				Changes: []*migrate.Change{{Cmd: "DROP TABLE `posts`"}},
			},
		},
		{
			input: []schema.Change{
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
										&IndexType{T: IndexTypeHash},
									},
								},
							},
							&schema.AddCheck{
								C: &schema.Check{
									Name:  "id_nonzero",
									Expr:  "(id > 0)",
									Attrs: []schema.Attr{&Enforced{}},
								},
							},
							&schema.ModifyAttr{
								From: &AutoIncrement{V: 1},
								To:   &AutoIncrement{V: 1000},
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD COLUMN `name` varchar(255) NOT NULL, ADD INDEX `id_key` USING HASH (`id`) COMMENT \"comment\", ADD CONSTRAINT `id_nonzero` CHECK (id > 0) ENFORCED, AUTO_INCREMENT 1000",
						Reverse: "ALTER TABLE `users` DROP COLUMN `name`, DROP INDEX `id_key`, DROP CONSTRAINT `id_nonzero`, AUTO_INCREMENT 1",
					},
				},
			},
		},
		{
			input: []schema.Change{
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
							&schema.DropCheck{
								C: &schema.Check{
									Name:  "id_nonzero",
									Expr:  "(id > 0)",
									Attrs: []schema.Attr{&Enforced{}},
								},
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP CONSTRAINT `id_nonzero`",
						Reverse: "ALTER TABLE `users` ADD CONSTRAINT `id_nonzero` CHECK (id > 0) ENFORCED",
					},
				},
			},
		},
		{
			input: []schema.Change{
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
							&schema.ModifyCheck{
								From: &schema.Check{
									Name:  "check1",
									Expr:  "(id > 0)",
									Attrs: []schema.Attr{&Enforced{}},
								},
								To: &schema.Check{
									Name: "check1",
									Expr: "(id > 0)",
								},
							},
							&schema.ModifyCheck{
								From: &schema.Check{
									Name: "check2",
									Expr: "(id > 0)",
								},
								To: &schema.Check{
									Name:  "check2",
									Expr:  "(id > 0)",
									Attrs: []schema.Attr{&Enforced{}},
								},
							},
							&schema.ModifyCheck{
								From: &schema.Check{
									Name: "check3",
									Expr: "(id > 0)",
								},
								To: &schema.Check{
									Name: "check3",
									Expr: "(id >= 0)",
								},
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ALTER CHECK `check1` ENFORCED, ALTER CHECK `check2` NOT ENFORCED, DROP CHECK `check3`, ADD CONSTRAINT `check3` CHECK (id >= 0)",
						Reverse: "ALTER TABLE `users` ALTER CHECK `check1` NOT ENFORCED, ALTER CHECK `check2` ENFORCED, DROP CHECK `check3`, ADD CONSTRAINT `check3` CHECK (id > 0)",
					},
				},
			},
		},
		{
			input: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "bigint").SetCharset("utf8mb4")),
				},
			},
			wantErr: true,
		},
		{
			input: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "bigint").SetCollation("utf8mb4_general_ci")),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		db, _, err := newMigrate("8.0.16")
		require.NoError(t, err)
		plan, err := db.PlanChanges(context.Background(), "wantPlan", tt.input)
		if tt.wantErr {
			require.Error(t, err, "expect plan to fail")
			return
		}
		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Equal(t, tt.wantPlan.Reversible, plan.Reversible)
		require.Equal(t, tt.wantPlan.Transactional, plan.Transactional)
		require.Equal(t, len(tt.wantPlan.Changes), len(plan.Changes))
		for i, c := range plan.Changes {
			require.Equal(t, tt.wantPlan.Changes[i].Cmd, c.Cmd)
			require.Equal(t, tt.wantPlan.Changes[i].Reverse, c.Reverse)
		}
	}
}

func newMigrate(version string) (migrate.PlanApplier, *mock, error) {
	db, m, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	mk := &mock{m}
	mk.version(version)
	drv, err := Open(db)
	if err != nil {
		return nil, nil, err
	}
	return drv, mk, nil
}
