// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
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
				&schema.AddSchema{S: schema.New("test"), Extra: []schema.Clause{&schema.IfNotExists{}}},
				&schema.DropSchema{S: schema.New("test"), Extra: []schema.Clause{&schema.IfExists{}}},
				&schema.DropSchema{S: schema.New("test"), Extra: []schema.Clause{&Cascade{}}},
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE SCHEMA IF NOT EXISTS "test"`,
						Reverse: `DROP SCHEMA "test"`,
					},
					{
						Cmd: `DROP SCHEMA IF EXISTS "test"`,
					},
					{
						Cmd: `DROP SCHEMA "test" CASCADE`,
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
						Symbol:     "pets_user_id_fkey",
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
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "pets" DROP CONSTRAINT "pets_user_id_fkey"`,
						Reverse: `ALTER TABLE "pets" ADD CONSTRAINT "pets_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE`,
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
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `DROP INDEX "name_index"`,
						Reverse: `CREATE INDEX "name_index" ON "users" ("name")`,
					},
				},
			},
		},
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
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&Identity{}, &schema.Comment{}}},
							{Name: "text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
						},
						Attrs: []schema.Attr{
							&schema.Comment{},
							&schema.Check{Name: "id_nonzero", Expr: `("id" > 0)`},
							&schema.Check{Name: "text_len", Expr: `(length("text") > 0)`, Attrs: []schema.Attr{&NoInherit{}}},
							&schema.Check{Name: "a_in_b", Expr: `(a) in (b)`},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY, "text" text NULL, CONSTRAINT "id_nonzero" CHECK ("id" > 0), CONSTRAINT "text_len" CHECK (length("text") > 0) NO INHERIT, CONSTRAINT "a_in_b" CHECK ((a) in (b)))`, Reverse: `DROP TABLE "posts"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024}}}},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY (START WITH 1024))`, Reverse: `DROP TABLE "posts"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Increment: 2}}}},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY (INCREMENT BY 2))`, Reverse: `DROP TABLE "posts"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 100, Increment: 2}}}},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY (START WITH 100 INCREMENT BY 2))`, Reverse: `DROP TABLE "posts"`}},
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
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}, Attrs: []schema.Attr{&schema.Comment{Text: "foo"}}},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "id_key",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0], Desc: true},
									},
									Attrs: []schema.Attr{
										&schema.Comment{Text: "comment"},
									},
								},
							},
							&schema.AddCheck{
								C: &schema.Check{Name: "name_not_empty", Expr: `("name" <> '')`},
							},
							&schema.DropCheck{
								C: &schema.Check{Name: "id_nonzero", Expr: `("id" <> 0)`},
							},
							&schema.ModifyCheck{
								From: &schema.Check{Name: "id_iseven", Expr: `("id" % 2 = 0)`},
								To:   &schema.Check{Name: "id_iseven", Expr: `(("id") % 2 = 0)`},
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" ADD COLUMN "name" character varying(255) NOT NULL, ADD CONSTRAINT "name_not_empty" CHECK ("name" <> ''), DROP CONSTRAINT "id_nonzero", DROP CONSTRAINT "id_iseven", ADD CONSTRAINT "id_iseven" CHECK (("id") % 2 = 0)`,
						Reverse: `ALTER TABLE "users" DROP COLUMN "name", DROP CONSTRAINT "name_not_empty", ADD CONSTRAINT "id_nonzero" CHECK ("id" <> 0), DROP CONSTRAINT "id_iseven", ADD CONSTRAINT "id_iseven" CHECK ("id" % 2 = 0)`,
					},
					{
						Cmd:     `CREATE INDEX "id_key" ON "users" ("id" DESC)`,
						Reverse: `DROP INDEX "id_key"`,
					},
					{
						Cmd:     `COMMENT ON COLUMN "users" ."name" IS 'foo'`,
						Reverse: `COMMENT ON COLUMN "users" ."name" IS ''`,
					},
					{
						Cmd:     `COMMENT ON INDEX "id_key" IS 'comment'`,
						Reverse: `COMMENT ON INDEX "id_key" IS ''`,
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
							&schema.DropColumn{
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}},
							},
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{}}},
								To:     &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024}}}},
								Change: schema.ChangeAttr,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TABLE "users" DROP COLUMN "name", ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1024 SET INCREMENT BY 1 RESTART`},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
					Changes: []schema.Change{
						&schema.AddAttr{
							A: &schema.Comment{Text: "foo"},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `COMMENT ON TABLE "public"."users" IS 'foo'`, Reverse: `COMMENT ON TABLE "public"."users" IS ''`},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: &schema.Table{Name: "users", Schema: &schema.Schema{Name: "public"}},
					Changes: []schema.Change{
						&schema.ModifyAttr{
							To:   &schema.Comment{Text: "foo"},
							From: &schema.Comment{Text: "bar"},
						},
					},
				},
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `COMMENT ON TABLE "public"."users" IS 'foo'`, Reverse: `COMMENT ON TABLE "public"."users" IS 'bar'`},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}},
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
								Change: schema.ChangeType,
							},
						},
					}
				}(),
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type WHERE typname = $1 AND typtype = 'e'")).
					WithArgs("state").WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `CREATE TYPE "state" AS ENUM ('on', 'off')`, Reverse: `DROP TYPE "state"`},
					{Cmd: `ALTER TABLE "users" ALTER COLUMN "state" TYPE state`, Reverse: `ALTER TABLE "users" ALTER COLUMN "state" TYPE text`},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off", "unknown"}}}},
								Change: schema.ChangeType,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TYPE "state" ADD VALUE 'unknown'`},
				},
			},
		},
		// Modify column type and drop comment.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}, Attrs: []schema.Attr{&schema.Comment{Text: "foo"}}},
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off", "unknown"}}}},
								Change: schema.ChangeType | schema.ChangeComment,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TYPE "state" ADD VALUE 'unknown'`},
					{Cmd: `COMMENT ON COLUMN "users" ."state" IS ''`, Reverse: `COMMENT ON COLUMN "users" ."state" IS 'foo'`},
				},
			},
		},
		// Modify column type and add comment.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off", "unknown"}}}, Attrs: []schema.Attr{&schema.Comment{Text: "foo"}}},
								Change: schema.ChangeType | schema.ChangeComment,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TYPE "state" ADD VALUE 'unknown'`},
					{Cmd: `COMMENT ON COLUMN "users" ."state" IS 'foo'`, Reverse: `COMMENT ON COLUMN "users" ."state" IS ''`},
				},
			},
		},
		// Modify column comment.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "state", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Attrs: []schema.Attr{&schema.Comment{Text: "bar"}}},
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Attrs: []schema.Attr{&schema.Comment{Text: "foo"}}},
								Change: schema.ChangeComment,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `COMMENT ON COLUMN "users" ."state" IS 'foo'`, Reverse: `COMMENT ON COLUMN "users" ."state" IS 'bar'`},
				},
			},
		},
		// Modify index comment.
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
							&schema.ModifyIndex{
								From: schema.NewIndex("id_key").
									AddColumns(users.Columns[0]).
									SetComment("foo"),
								To: schema.NewIndex("id_key").
									AddColumns(users.Columns[0]).
									SetComment("bar"),
								Change: schema.ChangeComment,
							},
						},
					}
				}(),
			},
			plan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `COMMENT ON INDEX "id_key" IS 'bar'`, Reverse: `COMMENT ON INDEX "id_key" IS 'foo'`},
				},
			},
		},
	}
	for _, tt := range tests {
		db, mk, err := sqlmock.New()
		require.NoError(t, err)
		m := mock{mk}
		m.version("130000")
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
