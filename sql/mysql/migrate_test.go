// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"strconv"
	"strings"
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
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `users` (`id` bigint NOT NULL)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("DROP TABLE IF EXISTS `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE IF NOT EXISTS `public`.`pets` (`a` int NOT NULL DEFAULT (int(rand())), `b` bigint NOT NULL DEFAULT 1, `c` bigint NULL, PRIMARY KEY (`a`, `b`), UNIQUE INDEX `b_c_unique` (`b`, `c`) COMMENT \"comment\")")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` DROP INDEX `id_spouse_id`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("ALTER TABLE `users` ADD CONSTRAINT `spouse` FOREIGN KEY (`spouse_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, ADD INDEX `id_spouse_id` (`spouse_id`, `id` DESC) COMMENT \"comment\"")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `posts` (`id` bigint NOT NULL, `author_id` bigint NULL, CONSTRAINT `author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(sqltest.Escape("CREATE TABLE `comments` (`id` bigint NOT NULL, `post_id` bigint NULL, CONSTRAINT `comment` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	ns := &schema.Schema{Name: "public"}
	err = migrate.ApplyChanges(context.Background(), []schema.Change{
		&schema.AddSchema{S: &schema.Schema{Name: "test", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
		&schema.DropSchema{S: &schema.Schema{Name: "atlas", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
		&schema.DropTable{
			T: &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				},
			},
		},
		&schema.AddTable{
			T: &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				},
			},
		},
		&schema.DropTable{
			T: &schema.Table{
				Name:   "pets",
				Schema: ns,
				Columns: []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
				},
			},
			Extra: []schema.Clause{&schema.IfExists{}},
		},
		&schema.AddTable{
			T: func() *schema.Table {
				t := &schema.Table{
					Name:   "pets",
					Schema: ns,
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
		version  string
		changes  []schema.Change
		options  []migrate.PlanOption
		wantPlan *migrate.Plan
		wantErr  bool
	}{
		{
			changes: []schema.Change{
				&schema.AddTable{T: schema.NewTable("users")},
			},
			// Table "users" has no columns.
			wantErr: true,
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{T: schema.NewTable("users")},
			},
			// Table "users" has no columns; drop the table instead.
			wantErr: true,
		},
		{
			changes: []schema.Change{
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
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewIntColumn("id", "int")).
						AddAttrs(&Engine{V: EngineInnoDB, Default: true}),
				},
				&schema.AddTable{
					T: schema.NewTable("pets").
						AddColumns(schema.NewIntColumn("id", "int")).
						AddAttrs(&Engine{V: EngineMyISAM, Default: false}),
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `users` (`id` int NOT NULL)",
						Reverse: "DROP TABLE `users`",
					},
					{
						Cmd:     "CREATE TABLE `pets` (`id` int NOT NULL) ENGINE MyISAM",
						Reverse: "DROP TABLE `pets`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(
							schema.NewIntColumn("id", "bigint"),
							schema.NewStringColumn("name", "varchar(255)"),
						).
						AddIndexes(
							schema.NewIndex("name_index").
								AddParts(schema.NewColumnPart(schema.NewColumn("name"))),
						)
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropIndex{
								I: users.Indexes[0],
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
		// Drop a primary key.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						SetSchema(schema.New("test")).
						AddColumns(
							schema.NewIntColumn("id", "bigint"),
						)
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropPrimaryKey{
								P: schema.NewPrimaryKey(users.Columns...),
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP PRIMARY KEY",
						Reverse: "ALTER TABLE `users` ADD PRIMARY KEY (`id`)",
					},
				},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
		},
		// Add a primary key.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						SetSchema(schema.New("test")).
						AddColumns(
							schema.NewIntColumn("id", "bigint"),
						)
					users.SetPrimaryKey(schema.NewPrimaryKey(users.Columns...))
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.AddPrimaryKey{
								P: users.PrimaryKey,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `dev`.`users` ADD PRIMARY KEY (`id`)",
						Reverse: "ALTER TABLE `dev`.`users` DROP PRIMARY KEY",
					},
				},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) {
					dev := "dev"
					o.SchemaQualifier = &dev
				},
			},
		},
		// Modify a primary key.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(
							schema.NewStringColumn("id", "varchar(255)"),
						)
					users.SetPrimaryKey(schema.NewPrimaryKey(users.Columns...))
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyPrimaryKey{
								From: schema.NewPrimaryKey(users.Columns...).
									AddAttrs(&IndexType{T: IndexTypeHash}),
								To: users.PrimaryKey,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` DROP PRIMARY KEY, ADD PRIMARY KEY (`id`)",
						Reverse: "ALTER TABLE `users` DROP PRIMARY KEY, ADD PRIMARY KEY USING HASH (`id`)",
					},
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
			changes: []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: "test", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE DATABASE `test` CHARSET latin", Reverse: "DROP DATABASE `test`"}},
			},
		},
		// Default database charset can be omitted.
		{
			changes: []schema.Change{
				&schema.AddSchema{S: schema.New("test").SetCharset("utf8")},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "CREATE DATABASE `test`", Reverse: "DROP DATABASE `test`"}},
			},
		},
		// Add the default database charset on modify can be omitted.
		{
			changes: []schema.Change{
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
			changes: []schema.Change{
				&schema.ModifySchema{
					S: schema.New("test").SetCharset("latin1"),
					Changes: []schema.Change{
						&schema.AddAttr{A: &schema.Charset{V: "latin1"}},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "ALTER DATABASE `test` CHARSET latin1", Reverse: "ALTER DATABASE `test` CHARSET utf8"}},
			},
		},
		// Modify charset.
		{
			changes: []schema.Change{
				&schema.ModifySchema{
					S: schema.New("test").SetCharset("utf8"),
					Changes: []schema.Change{
						&schema.ModifyAttr{From: &schema.Charset{V: "latin1"}, To: &schema.Charset{V: "utf8"}},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes:    []*migrate.Change{{Cmd: "ALTER DATABASE `test` CHARSET utf8", Reverse: "ALTER DATABASE `test` CHARSET latin1"}},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropSchema{S: &schema.Schema{Name: "atlas", Attrs: []schema.Attr{&schema.Charset{V: "latin"}}}},
			},
			wantPlan: &migrate.Plan{
				Changes: []*migrate.Change{{Cmd: "DROP DATABASE `atlas`"}},
			},
		},
		{
			changes: []schema.Change{
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
					t.AddIndexes(
						schema.NewIndex("text").
							AddParts(&schema.IndexPart{C: t.Columns[1]}).
							AddAttrs(&IndexType{T: "FULLTEXT"}, &schema.Comment{Text: "text index"}, &IndexParser{P: "ngram"}),
					)
					return &schema.AddTable{T: t}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `posts` (`id` bigint NOT NULL AUTO_INCREMENT, `text` text NULL, `uuid` char(36) CHARSET utf8mb4 NULL COLLATE utf8mb4_bin, PRIMARY KEY (`id`), FULLTEXT INDEX `text` (`text`) COMMENT \"text index\" WITH PARSER `ngram`)",
						Reverse: "DROP TABLE `posts`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
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
			changes: []schema.Change{
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
			changes: []schema.Change{
				&schema.DropTable{T: schema.NewTable("posts").AddColumns(schema.NewIntColumn("id", "bigint"))},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "DROP TABLE `posts`",
						Reverse: "CREATE TABLE `posts` (`id` bigint NOT NULL)",
					},
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
						Reverse: "ALTER TABLE `users` AUTO_INCREMENT 1, DROP CONSTRAINT `id_nonzero`, DROP INDEX `id_key`, DROP COLUMN `name`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(schema.NewIntColumn("id", "int"))
					posts := schema.NewTable("posts").
						AddColumns(
							schema.NewIntColumn("id", "int"),
							schema.NewIntColumn("author_id", "int"),
						)
					posts.AddForeignKeys(
						schema.NewForeignKey("author").
							AddColumns(posts.Columns[1]).
							SetRefTable(users).
							AddRefColumns(users.Columns[0]),
					)
					return &schema.ModifyTable{
						T: posts,
						Changes: []schema.Change{
							&schema.AddColumn{C: posts.Columns[1]},
							&schema.AddForeignKey{F: posts.ForeignKeys[0]},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `posts` ADD COLUMN `author_id` int NOT NULL, ADD CONSTRAINT `author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`)",
						Reverse: "ALTER TABLE `posts` DROP FOREIGN KEY `author`, DROP COLUMN `author_id`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						AddColumns(schema.NewIntColumn("c1", "int"))
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.AddColumn{
								C: schema.NewIntColumn("c2", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c1*2"}),
							},
							&schema.AddColumn{
								C: schema.NewIntColumn("c3", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "c1*c2", Type: "STORED"}),
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` ADD COLUMN `c2` int AS (c1*2) NOT NULL, ADD COLUMN `c3` int AS (c1*c2) STORED NOT NULL",
						Reverse: "ALTER TABLE `users` DROP COLUMN `c3`, DROP COLUMN `c2`",
					},
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
						Reverse: "ALTER TABLE `users` DROP CHECK `check3`, ADD CONSTRAINT `check3` CHECK (id > 0), ALTER CHECK `check2` ENFORCED, ALTER CHECK `check1` NOT ENFORCED",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "bigint").SetCharset("utf8mb4")),
				},
			},
			wantErr: true,
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "bigint").SetCollation("utf8mb4_general_ci")),
				},
			},
			wantErr: true,
		},
		// Changing a regular column to a VIRTUAL generated column is not allowed.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"})),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From:   schema.NewColumn("c"),
							To:     schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1"}),
						},
					},
				},
			},
			wantErr: true,
		},
		// Changing a VIRTUAL generated column to a regular column is not allowed.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewColumn("c")),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From:   schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "VIRTUAL"}),
							To:     schema.NewColumn("c"),
						},
					},
				},
			},
			wantErr: true,
		},
		// Changing the storage type of generated column is not allowed.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"})),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From:   schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "VIRTUAL"}),
							To:     schema.NewColumn("c").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						},
					},
				},
			},
			wantErr: true,
		},
		// Changing a STORED generated column to a regular column.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewIntColumn("c", "int")),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From:   schema.NewIntColumn("c", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
							To:     schema.NewIntColumn("c", "int"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` MODIFY COLUMN `c` int NOT NULL",
						Reverse: "ALTER TABLE `users` MODIFY COLUMN `c` int AS (1) STORED NOT NULL",
					},
				},
			},
		},
		// Changing a regular column to a STORED generated column.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("users").
						AddColumns(schema.NewIntColumn("c", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"})),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From:   schema.NewIntColumn("c", "int"),
							To:     schema.NewIntColumn("c", "int").SetGeneratedExpr(&schema.GeneratedExpr{Expr: "1", Type: "STORED"}),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `users` MODIFY COLUMN `c` int AS (1) STORED NOT NULL",
						Reverse: "ALTER TABLE `users` MODIFY COLUMN `c` int NOT NULL",
					},
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
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "RENAME TABLE `t1` TO `t2`",
						Reverse: "RENAME TABLE `t2` TO `t1`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.RenameTable{
					From: schema.NewTable("t1").SetSchema(schema.New("s1")),
					To:   schema.NewTable("t2").SetSchema(schema.New("s2")),
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "RENAME TABLE `s1`.`t1` TO `s2`.`t2`",
						Reverse: "RENAME TABLE `s2`.`t2` TO `s1`.`t1`",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").
						SetSchema(schema.New("s1")).
						AddColumns(schema.NewColumn("b")),
					Changes: []schema.Change{
						&schema.RenameColumn{
							From: schema.NewColumn("a"),
							To:   schema.NewColumn("b"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `s1`.`t1` RENAME COLUMN `a` TO `b`",
						Reverse: "ALTER TABLE `s1`.`t1` RENAME COLUMN `b` TO `a`",
					},
				},
			},
		},
		{
			version: "5.6",
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").
						SetSchema(schema.New("s1")).
						AddColumns(schema.NewIntColumn("b", "int")),
					Changes: []schema.Change{
						&schema.RenameColumn{
							From: schema.NewIntColumn("a", "int"),
							To:   schema.NewIntColumn("b", "int"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `s1`.`t1` CHANGE COLUMN `a` `b` int NOT NULL",
						Reverse: "ALTER TABLE `s1`.`t1` CHANGE COLUMN `b` `a` int NOT NULL",
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").
						SetSchema(schema.New("s1")).
						AddColumns(schema.NewIntColumn("b", "int")),
					Changes: []schema.Change{
						&schema.RenameIndex{
							From: schema.NewIndex("a"),
							To:   schema.NewIndex("b"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "ALTER TABLE `s1`.`t1` RENAME INDEX `a` TO `b`",
						Reverse: "ALTER TABLE `s1`.`t1` RENAME INDEX `b` TO `a`",
					},
				},
			},
		},
		// Empty qualifier.
		{
			changes: []schema.Change{
				&schema.AddTable{T: schema.NewTable("t").SetSchema(schema.New("d")).AddColumns(schema.NewIntColumn("a", "int"))},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			wantPlan: &migrate.Plan{
				Reversible: true,
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
			wantPlan: &migrate.Plan{
				Reversible: true,
				Changes: []*migrate.Change{
					{
						Cmd:     "CREATE TABLE `other`.`t` (`a` int NOT NULL)",
						Reverse: "DROP TABLE `other`.`t`",
					},
				},
			},
		},
		// Empty qualifier in multi-schema mode should fail.
		{
			changes: []schema.Change{
				&schema.AddTable{T: schema.NewTable("t1").SetSchema(schema.New("s1")).AddColumns(schema.NewIntColumn("a", "int"))},
				&schema.AddTable{T: schema.NewTable("t2").SetSchema(schema.New("s2")).AddColumns(schema.NewIntColumn("a", "int"))},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			wantErr: true,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.version == "" {
				tt.version = "8.0.16"
			}
			db, _, err := newMigrate(tt.version)
			require.NoError(t, err)
			plan, err := db.PlanChanges(context.Background(), "wantPlan", tt.changes, tt.options...)
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
		})
	}
}

func TestDefaultPlan(t *testing.T) {
	changes, err := DefaultPlan.PlanChanges(context.Background(), "plan", []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").SetSchema(schema.New("s1")).AddColumns(schema.NewIntColumn("a", "int"))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(changes.Changes))
	require.Equal(t, "CREATE TABLE `s1`.`t1` (`a` int NOT NULL)", changes.Changes[0].Cmd)

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
				SetSchema(schema.New("s1")).
				AddColumns(schema.NewIntColumn("a", "int")),
			Cmd: join(
				"CREATE TABLE `s1`.`t1` (",
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
				AddIndexes(
					schema.NewIndex("idx").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  INDEX `idx` (`a`, `b`)",
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
				AddIndexes(
					schema.NewIndex("idx").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  `id` int NOT NULL,",
				"  PRIMARY KEY (`id`),",
				"  INDEX `idx` (`a`, `b`)",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				).
				AddIndexes(
					schema.NewIndex("idx1").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
					schema.NewIndex("idx2").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
				),
			Cmd: join(
				"CREATE TABLE `t1` (",
				"  `a` int NOT NULL,",
				"  `b` int NOT NULL,",
				"  INDEX `idx1` (`a`, `b`),",
				"  INDEX `idx2` (`a`, `b`)",
				")",
			),
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				).
				AddIndexes(
					schema.NewIndex("idx1").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
					schema.NewIndex("idx2").
						AddColumns(
							schema.NewIntColumn("a", "int"),
							schema.NewIntColumn("b", "int"),
						),
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
				"  INDEX `idx1` (`a`, `b`),",
				"  INDEX `idx2` (`a`, `b`),",
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
				).
				SetComment("Comment"),
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
				`) COMMENT "Comment"`,
			),
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			db, _, err := newMigrate("8.0.16")
			require.NoError(t, err)
			plan, err := db.PlanChanges(context.Background(), "wantPlan", []schema.Change{&schema.AddTable{T: tt.T}}, func(opts *migrate.PlanOptions) {
				opts.Indent = "  "
			})
			require.NoError(t, err)
			require.Len(t, plan.Changes, 1)
			require.Equal(t, tt.Cmd, plan.Changes[0].Cmd)
		})
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

func join(lines ...string) string { return strings.Join(lines, "\n") }
