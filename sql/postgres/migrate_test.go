// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"strconv"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPlanChanges(t *testing.T) {
	tests := []struct {
		changes  []schema.Change
		options  []migrate.PlanOption
		mock     func(mock)
		wantPlan *migrate.Plan
		wantErr  bool
	}{
		{
			changes: []schema.Change{
				&schema.AddSchema{S: schema.New("public")},
				&schema.AddSchema{S: schema.New("test"), Extra: []schema.Clause{&schema.IfNotExists{}}},
				&schema.DropSchema{S: schema.New("test"), Extra: []schema.Clause{&schema.IfExists{}}},
				&schema.DropSchema{S: schema.New("test"), Extra: []schema.Clause{}},
			},
			wantPlan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE SCHEMA IF NOT EXISTS "public"`,
						Reverse: `DROP SCHEMA "public" CASCADE`,
					},
					{
						Cmd:     `CREATE SCHEMA IF NOT EXISTS "test"`,
						Reverse: `DROP SCHEMA "test" CASCADE`,
					},
					{
						Cmd: `DROP SCHEMA IF EXISTS "test" CASCADE`,
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
			wantPlan: &migrate.Plan{
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
							{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}}},
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
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							{Name: "nickname", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}}},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropIndex{
								I: schema.NewUniqueIndex("unique_nickname").
									AddColumns(schema.NewColumn("nickname")).
									AddAttrs(&Constraint{T: "u"}),
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" DROP CONSTRAINT "unique_nickname"`,
						Reverse: `ALTER TABLE "users" ADD CONSTRAINT "unique_nickname" UNIQUE ("nickname")`,
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: "test"}},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE SCHEMA "test"`, Reverse: `DROP SCHEMA "test" CASCADE`}}},
		},
		{
			changes: []schema.Change{
				&schema.DropSchema{S: &schema.Schema{Name: "atlas"}},
			},
			wantPlan: &migrate.Plan{
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `DROP SCHEMA "atlas" CASCADE`}},
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
							{Name: "directions", Type: &schema.ColumnType{Type: &ArrayType{T: "direction[]", Type: &schema.EnumType{T: "direction", Values: []string{"NORTH", "SOUTH"}, Schema: schema.New("public")}}}},
							{Name: "states", Type: &schema.ColumnType{Type: &ArrayType{T: "state[]", Type: &schema.EnumType{T: "state", Values: []string{"ON", "OFF"}}}}},
						},
						Attrs: []schema.Attr{
							&schema.Comment{},
							&schema.Check{Name: "id_nonzero", Expr: `("id" > 0)`},
							&schema.Check{Name: "text_len", Expr: `(length("text") > 0)`, Attrs: []schema.Attr{&NoInherit{}}},
							&schema.Check{Name: "a_in_b", Expr: `(a) in (b)`},
							&Partition{T: "HASH", Parts: []*PartitionPart{{C: schema.NewColumn("text")}}},
						},
					},
				},
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2")).
					WithArgs("direction", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e'")).
					WithArgs("state").
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("state"))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `CREATE TYPE "public"."direction" AS ENUM ('NORTH', 'SOUTH')`, Reverse: `DROP TYPE "public"."direction"`},
					{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY, "text" text NULL, "directions" "public"."direction"[] NOT NULL, "states" "state"[] NOT NULL, CONSTRAINT "id_nonzero" CHECK ("id" > 0), CONSTRAINT "text_len" CHECK (length("text") > 0) NO INHERIT, CONSTRAINT "a_in_b" CHECK ((a) in (b))) PARTITION BY HASH ("text")`, Reverse: `DROP TABLE "posts"`},
				},
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
			wantPlan: &migrate.Plan{
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
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}},
							{Name: "nid", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&schema.GeneratedExpr{Expr: "id+1"}}},
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE TABLE "posts" ("id" integer NOT NULL, "nid" integer NOT NULL GENERATED ALWAYS AS (id+1) STORED)`,
						Reverse: `DROP TABLE "posts"`,
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						AddColumns(
							schema.NewIntColumn("c1", "int").
								SetGeneratedExpr(&schema.GeneratedExpr{Expr: "id+1"}),
						),
					Changes: []schema.Change{
						&schema.ModifyColumn{
							Change: schema.ChangeGenerated,
							From: schema.NewIntColumn("c1", "int").
								SetGeneratedExpr(&schema.GeneratedExpr{Expr: "id+1"}),
							To: schema.NewIntColumn("c1", "int"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd: `ALTER TABLE "posts" ALTER COLUMN "c1" DROP EXPRESSION`,
					},
				},
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
			wantPlan: &migrate.Plan{
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
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes:       []*migrate.Change{{Cmd: `CREATE TABLE "posts" ("id" integer NOT NULL GENERATED BY DEFAULT AS IDENTITY (START WITH 100 INCREMENT BY 2))`, Reverse: `DROP TABLE "posts"`}},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{T: schema.NewTable("posts").AddColumns(schema.NewIntColumn("id", "int"))},
				&schema.DropTable{T: schema.NewTable("users").AddColumns(schema.NewIntColumn("id", "int")), Extra: []schema.Clause{&Cascade{}}},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `DROP TABLE "posts"`,
						Reverse: `CREATE TABLE "posts" ("id" integer NOT NULL)`,
					},
					{
						Cmd:     `DROP TABLE "users" CASCADE`,
						Reverse: `CREATE TABLE "users" ("id" integer NOT NULL)`,
					},
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
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd: `DROP TABLE "posts"`,
						Reverse: []string{
							`CREATE TABLE "posts" ("id" integer NOT NULL)`,
							`CREATE INDEX "idx" ON "posts" ("id")`,
							`COMMENT ON TABLE "posts" IS 'a8m''s posts'`,
							`COMMENT ON INDEX "idx" IS 'a8m''s index'`,
						},
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
							schema.NewIntColumn("id", "int"),
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
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "test"."users" DROP CONSTRAINT "users_pkey"`,
						Reverse: `ALTER TABLE "test"."users" ADD PRIMARY KEY ("id")`,
					},
				},
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
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "test"."users" ADD PRIMARY KEY ("id")`,
						Reverse: `ALTER TABLE "test"."users" DROP CONSTRAINT "users_pkey"`,
					},
				},
			},
		},
		// Modify a primary key.
		{
			changes: []schema.Change{
				func() schema.Change {
					dropC, addC := schema.NewStringColumn("email", "text"), schema.NewStringColumn("name", "text")
					users := schema.NewTable("users").
						AddColumns(schema.NewStringColumn("id", "text"), addC, schema.NewStringColumn("last", "text"))
					users.SetPrimaryKey(schema.NewPrimaryKey(addC))
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.DropColumn{C: dropC},
							&schema.AddColumn{C: addC},
							&schema.ModifyPrimaryKey{
								From: schema.NewPrimaryKey(dropC).
									AddAttrs(&IndexInclude{Columns: users.Columns[2:]}),
								To: users.PrimaryKey,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" DROP CONSTRAINT "users_pkey", DROP COLUMN "email", ADD COLUMN "name" text NOT NULL, ADD PRIMARY KEY ("name")`,
						Reverse: `ALTER TABLE "users" DROP CONSTRAINT "users_pkey", DROP COLUMN "name", ADD COLUMN "email" text NOT NULL, ADD PRIMARY KEY ("email") INCLUDE ("last")`,
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
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}, Attrs: []schema.Attr{&schema.Comment{Text: "foo"}}, Default: &schema.Literal{V: "'logged_in'"}},
							},
							&schema.AddColumn{
								C: &schema.Column{Name: "last", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}, Attrs: []schema.Attr{&schema.Comment{Text: "bar"}}, Default: &schema.RawExpr{X: "'logged_in'"}},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "id_key",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0], Desc: true},
									},
									Attrs: []schema.Attr{
										&schema.Comment{Text: "comment"},
										&IndexPredicate{P: "success"},
									},
								},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "id_brin",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0], Desc: true},
									},
									Attrs: []schema.Attr{
										&IndexType{T: IndexTypeBRIN},
										&IndexStorageParams{PagesPerRange: 2},
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
							&schema.AddIndex{
								I: &schema.Index{
									Name: "include_key",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0]},
									},
									Attrs: []schema.Attr{
										&IndexInclude{Columns: []*schema.Column{schema.NewColumn("a"), schema.NewColumn("b")}},
									},
								},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "add_con",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0]},
									},
								},
								Extra: []schema.Clause{
									&Concurrently{},
								},
							},
							&schema.DropIndex{
								I: &schema.Index{
									Name: "drop_con",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0]},
									},
								},
								Extra: []schema.Clause{
									&Concurrently{},
								},
							},
							&schema.AddIndex{
								I: &schema.Index{
									Name: "operator_class",
									Parts: []*schema.IndexPart{
										{C: users.Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "int8_bloom_ops"}}},
										{C: users.Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "int8_minmax_ops"}}},
										{C: users.Columns[0], Attrs: []schema.Attr{&IndexOpClass{Name: "int8_minmax_multi_ops", Params: []struct{ N, V string }{{"values_per_range", "8"}}}}},
									},
									Attrs: []schema.Attr{
										&IndexType{T: IndexTypeBRIN},
									},
								},
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `DROP INDEX CONCURRENTLY "drop_con"`,
						Reverse: `CREATE INDEX CONCURRENTLY "drop_con" ON "users" ("id")`,
					},
					{
						Cmd:     `ALTER TABLE "users" DROP CONSTRAINT "id_nonzero", ADD COLUMN "name" character varying(255) NOT NULL DEFAULT 'logged_in', ADD COLUMN "last" character varying(255) NOT NULL DEFAULT 'logged_in', ADD CONSTRAINT "name_not_empty" CHECK ("name" <> ''), DROP CONSTRAINT "id_iseven", ADD CONSTRAINT "id_iseven" CHECK (("id") % 2 = 0)`,
						Reverse: `ALTER TABLE "users" DROP CONSTRAINT "id_iseven", ADD CONSTRAINT "id_iseven" CHECK ("id" % 2 = 0), DROP CONSTRAINT "name_not_empty", DROP COLUMN "last", DROP COLUMN "name", ADD CONSTRAINT "id_nonzero" CHECK ("id" <> 0)`,
					},
					{
						Cmd:     `CREATE INDEX "id_key" ON "users" ("id" DESC) WHERE success`,
						Reverse: `DROP INDEX "id_key"`,
					},
					{
						Cmd:     `CREATE INDEX "id_brin" ON "users" USING BRIN ("id" DESC) WITH (pages_per_range = 2)`,
						Reverse: `DROP INDEX "id_brin"`,
					},
					{
						Cmd:     `CREATE INDEX "include_key" ON "users" ("id") INCLUDE ("a", "b")`,
						Reverse: `DROP INDEX "include_key"`,
					},
					{
						Cmd:     `CREATE INDEX CONCURRENTLY "add_con" ON "users" ("id")`,
						Reverse: `DROP INDEX CONCURRENTLY "add_con"`,
					},
					{
						Cmd:     `CREATE INDEX "operator_class" ON "users" USING BRIN ("id" int8_bloom_ops, "id", "id" int8_minmax_multi_ops(values_per_range=8))`,
						Reverse: `DROP INDEX "operator_class"`,
					},
					{
						Cmd:     `COMMENT ON COLUMN "users" ."name" IS 'foo'`,
						Reverse: `COMMENT ON COLUMN "users" ."name" IS ''`,
					},
					{
						Cmd:     `COMMENT ON COLUMN "users" ."last" IS 'bar'`,
						Reverse: `COMMENT ON COLUMN "users" ."last" IS ''`,
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
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar"}}},
							},
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{}, &schema.Comment{Text: "comment"}}},
								To:     &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024}}}},
								Change: schema.ChangeAttr | schema.ChangeComment,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" DROP COLUMN "name", ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1024 SET INCREMENT BY 1 RESTART`,
						Reverse: `ALTER TABLE "users" ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1 SET INCREMENT BY 1 RESTART, ADD COLUMN "name" character varying NOT NULL`,
					},
					{
						Cmd:     `COMMENT ON COLUMN "users" ."id" IS ''`,
						Reverse: `COMMENT ON COLUMN "users" ."id" IS 'comment'`,
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
								C: &schema.Column{Name: "name", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar"}}},
							},
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 0, Last: 1025}}}},
								To:     &schema.Column{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{Sequence: &Sequence{Start: 1024}}}},
								Change: schema.ChangeAttr,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" DROP COLUMN "name", ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1024 SET INCREMENT BY 1`,
						Reverse: `ALTER TABLE "users" ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1 SET INCREMENT BY 1, ADD COLUMN "name" character varying NOT NULL`,
					},
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
			wantPlan: &migrate.Plan{
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
			wantPlan: &migrate.Plan{
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
					users := schema.NewTable("users").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
							schema.NewEnumColumn("status", schema.EnumName("status"), schema.EnumValues("a", "b"), schema.EnumSchema(schema.New("test"))),
						)
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}},
								To:     users.Columns[0],
								Change: schema.ChangeType,
							},
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "status", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}},
								To:     users.Columns[1],
								Change: schema.ChangeType,
							},
							&schema.DropColumn{
								C: schema.NewEnumColumn("dc1", schema.EnumName("de"), schema.EnumValues("on")),
							},
							&schema.DropColumn{
								C: schema.NewEnumColumn("dc2", schema.EnumName("de"), schema.EnumValues("on")),
							},
						},
					}
				}(),
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("state", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("status", "test").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `CREATE TYPE "public"."state" AS ENUM ('on', 'off')`, Reverse: `DROP TYPE "public"."state"`},
					{Cmd: `CREATE TYPE "test"."status" AS ENUM ('a', 'b')`, Reverse: `DROP TYPE "test"."status"`},
					{Cmd: `ALTER TABLE "public"."users" ALTER COLUMN "state" TYPE "public"."state", ALTER COLUMN "status" TYPE "test"."status", DROP COLUMN "dc1", DROP COLUMN "dc2"`, Reverse: `ALTER TABLE "public"."users" ADD COLUMN "dc2" "public"."de" NOT NULL, ADD COLUMN "dc1" "public"."de" NOT NULL, ALTER COLUMN "status" TYPE text, ALTER COLUMN "state" TYPE text`},
					{Cmd: `DROP TYPE "public"."de"`, Reverse: `CREATE TYPE "public"."de" AS ENUM ('on')`},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{
					T: schema.NewTable("users").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
							schema.NewEnumColumn("status", schema.EnumName("status"), schema.EnumValues("on", "off")),
						),
				},
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("state", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("status", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `DROP TABLE "public"."users"`, Reverse: `CREATE TABLE "public"."users" ("state" "public"."state" NOT NULL, "status" "public"."status" NOT NULL)`},
					{Cmd: `DROP TYPE "public"."state"`, Reverse: `CREATE TYPE "public"."state" AS ENUM ('on', 'off')`},
					{Cmd: `DROP TYPE "public"."status"`, Reverse: `CREATE TYPE "public"."status" AS ENUM ('on', 'off')`},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.DropTable{
					T: schema.NewTable("users").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
							schema.NewEnumColumn("status", schema.EnumName("status"), schema.EnumValues("on", "off")),
						),
				},
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("state", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("status", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `DROP TABLE "public"."users"`, Reverse: `CREATE TABLE "public"."users" ("state" "public"."state" NOT NULL, "status" "public"."status" NOT NULL)`},
					{Cmd: `DROP TYPE "public"."state"`, Reverse: `CREATE TYPE "public"."state" AS ENUM ('on', 'off')`},
					{Cmd: `DROP TYPE "public"."status"`, Reverse: `CREATE TYPE "public"."status" AS ENUM ('on', 'off')`},
				},
			},
		},
		{
			changes: func() []schema.Change {
				s := schema.New("public").
					AddTables(
						schema.NewTable("t1").
							AddColumns(
								schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
							),
						schema.NewTable("t2").
							AddColumns(
								schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
							),
					)
				return []schema.Change{
					&schema.DropTable{
						T: s.Tables[0],
					},
					&schema.DropTable{
						T: s.Tables[1],
					},
				}
			}(),
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("state", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("state"))
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e' AND n.nspname = $2 ")).
					WithArgs("state", "public").
					WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("state"))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `DROP TABLE "public"."t1"`,
						Reverse: `CREATE TABLE "public"."t1" ("state" "public"."state" NOT NULL)`,
					},
					{
						Cmd:     `DROP TABLE "public"."t2"`,
						Reverse: `CREATE TABLE "public"."t2" ("state" "public"."state" NOT NULL)`,
					},
					{Cmd: `DROP TYPE "public"."state"`, Reverse: `CREATE TYPE "public"."state" AS ENUM ('on', 'off')`},
				},
			},
		},
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
						)
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   users.Columns[0],
								To:     &schema.Column{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off", "unknown"}}}},
								Change: schema.ChangeType,
							},
						},
					}
				}(),
			},
			wantPlan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TYPE "public"."state" ADD VALUE 'unknown'`},
				},
			},
		},
		// Modify column type and drop comment.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := schema.NewTable("users").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewEnumColumn("state", schema.EnumName("state"), schema.EnumValues("on", "off")),
						)
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
			wantPlan: &migrate.Plan{
				Reversible:    false,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `ALTER TYPE "public"."state" ADD VALUE 'unknown'`},
					{Cmd: `COMMENT ON COLUMN "public"."users" ."state" IS ''`, Reverse: `COMMENT ON COLUMN "public"."users" ."state" IS 'foo'`},
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
			wantPlan: &migrate.Plan{
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
			wantPlan: &migrate.Plan{
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
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{Cmd: `COMMENT ON INDEX "id_key" IS 'bar'`, Reverse: `COMMENT ON INDEX "id_key" IS 'foo'`},
				},
			},
		},
		// Modify default values.
		{
			changes: []schema.Change{
				func() schema.Change {
					users := &schema.Table{
						Name: "users",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							{Name: "one", Default: &schema.Literal{V: "'one'"}, Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
							{Name: "two", Default: &schema.Literal{V: "'two'"}, Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
						},
					}
					return &schema.ModifyTable{
						T: users,
						Changes: []schema.Change{
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "one", Default: &schema.Literal{V: "'one'"}, Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
								To:     &schema.Column{Name: "one", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
								Change: schema.ChangeDefault,
							},
							&schema.ModifyColumn{
								From:   &schema.Column{Name: "two", Default: &schema.Literal{V: "'two'"}, Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
								To:     &schema.Column{Name: "two", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
								Change: schema.ChangeDefault,
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
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "users" ALTER COLUMN "one" DROP DEFAULT, ALTER COLUMN "two" DROP DEFAULT, ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1024 SET INCREMENT BY 1 RESTART`,
						Reverse: `ALTER TABLE "users" ALTER COLUMN "id" SET GENERATED BY DEFAULT SET START WITH 1 SET INCREMENT BY 1 RESTART, ALTER COLUMN "two" SET DEFAULT 'two', ALTER COLUMN "one" SET DEFAULT 'one'`,
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
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "s1"."t1" RENAME TO "s2"."t2"`,
						Reverse: `ALTER TABLE "s2"."t2" RENAME TO "s1"."t1"`,
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").SetSchema(schema.New("s1")),
					Changes: []schema.Change{
						&schema.RenameColumn{
							From: schema.NewColumn("a"),
							To:   schema.NewColumn("b"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "s1"."t1" RENAME COLUMN "a" TO "b"`,
						Reverse: `ALTER TABLE "s1"."t1" RENAME COLUMN "b" TO "a"`,
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").SetSchema(schema.New("s1")),
					Changes: []schema.Change{
						&schema.RenameColumn{
							From: schema.NewColumn("a"),
							To:   schema.NewColumn("b"),
						},
						&schema.AddColumn{
							C: schema.NewIntColumn("c", "int"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "s1"."t1" ADD COLUMN "c" integer NOT NULL`,
						Reverse: `ALTER TABLE "s1"."t1" DROP COLUMN "c"`,
					},
					{
						Cmd:     `ALTER TABLE "s1"."t1" RENAME COLUMN "a" TO "b"`,
						Reverse: `ALTER TABLE "s1"."t1" RENAME COLUMN "b" TO "a"`,
					},
				},
			},
		},
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("t1").SetSchema(schema.New("s1")),
					Changes: []schema.Change{
						&schema.RenameIndex{
							From: schema.NewIndex("a"),
							To:   schema.NewIndex("b"),
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER INDEX "a" RENAME TO "b"`,
						Reverse: `ALTER INDEX "b" RENAME TO "a"`,
					},
				},
			},
		},
		// Invalid serial type.
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: &schema.Table{
						Name: "posts",
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{Type: &SerialType{T: "serial"}, Null: true}},
						},
					},
				},
			},
			wantErr: true,
		},
		// Drop serial sequence.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewIntColumn("c1", "integer"),
							schema.NewIntColumn("c2", "integer"),
						),
					Changes: schema.Changes{
						&schema.ModifyColumn{
							From:   schema.NewColumn("c1").SetType(&SerialType{T: "smallserial"}),
							To:     schema.NewIntColumn("c1", "integer"),
							Change: schema.ChangeType,
						},
						&schema.ModifyColumn{
							From:   schema.NewColumn("c2").SetType(&SerialType{T: "serial", SequenceName: "previous_name"}),
							To:     schema.NewIntColumn("c2", "integer"),
							Change: schema.ChangeType,
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "public"."posts" ALTER COLUMN "c1" DROP DEFAULT, ALTER COLUMN "c1" TYPE integer, ALTER COLUMN "c2" DROP DEFAULT`,
						Reverse: `ALTER TABLE "public"."posts" ALTER COLUMN "c2" SET DEFAULT nextval('"public"."previous_name"'), ALTER COLUMN "c1" SET DEFAULT nextval('"public"."posts_c1_seq"'), ALTER COLUMN "c1" TYPE smallint`,
					},
					{
						Cmd:     `DROP SEQUENCE IF EXISTS "public"."posts_c1_seq"`,
						Reverse: `CREATE SEQUENCE IF NOT EXISTS "public"."posts_c1_seq" OWNED BY "public"."posts"."c1"`,
					},
					{
						Cmd:     `DROP SEQUENCE IF EXISTS "public"."previous_name"`,
						Reverse: `CREATE SEQUENCE IF NOT EXISTS "public"."previous_name" OWNED BY "public"."posts"."c2"`,
					},
				},
			},
		},
		// Add serial sequence.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
							schema.NewColumn("c2").SetType(&SerialType{T: "bigserial"}),
						),
					Changes: schema.Changes{
						&schema.ModifyColumn{
							From:   schema.NewIntColumn("c1", "integer"),
							To:     schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
							Change: schema.ChangeType,
						},
						&schema.ModifyColumn{
							From:   schema.NewIntColumn("c2", "integer"),
							To:     schema.NewColumn("c2").SetType(&SerialType{T: "bigserial"}),
							Change: schema.ChangeType,
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE SEQUENCE IF NOT EXISTS "public"."posts_c1_seq" OWNED BY "public"."posts"."c1"`,
						Reverse: `DROP SEQUENCE IF EXISTS "public"."posts_c1_seq"`,
					},
					{
						Cmd:     `CREATE SEQUENCE IF NOT EXISTS "public"."posts_c2_seq" OWNED BY "public"."posts"."c2"`,
						Reverse: `DROP SEQUENCE IF EXISTS "public"."posts_c2_seq"`,
					},
					{
						Cmd:     `ALTER TABLE "public"."posts" ALTER COLUMN "c1" SET DEFAULT nextval('"public"."posts_c1_seq"'), ALTER COLUMN "c2" SET DEFAULT nextval('"public"."posts_c2_seq"'), ALTER COLUMN "c2" TYPE bigint`,
						Reverse: `ALTER TABLE "public"."posts" ALTER COLUMN "c2" DROP DEFAULT, ALTER COLUMN "c2" TYPE integer, ALTER COLUMN "c1" DROP DEFAULT`,
					},
				},
			},
		},
		// Change underlying sequence type.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
							schema.NewColumn("c2").SetType(&SerialType{T: "bigserial"}),
						),
					Changes: schema.Changes{
						&schema.ModifyColumn{
							From:   schema.NewColumn("c1").SetType(&SerialType{T: "smallserial"}),
							To:     schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
							Change: schema.ChangeType,
						},
						&schema.ModifyColumn{
							From:   schema.NewColumn("c2").SetType(&SerialType{T: "serial"}),
							To:     schema.NewColumn("c2").SetType(&SerialType{T: "bigserial"}),
							Change: schema.ChangeType,
						},
					},
				},
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "public"."posts" ALTER COLUMN "c1" TYPE integer, ALTER COLUMN "c2" TYPE bigint`,
						Reverse: `ALTER TABLE "public"."posts" ALTER COLUMN "c2" TYPE integer, ALTER COLUMN "c1" TYPE smallint`,
					},
				},
			},
		},
		// Empty qualifier.
		{
			changes: []schema.Change{
				&schema.AddTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("test1")).
						AddColumns(
							schema.NewEnumColumn("c1", schema.EnumName("enum"), schema.EnumValues("a"), schema.EnumSchema(schema.New("test2"))),
						),
				},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			mock: func(m mock) {
				m.ExpectQuery(sqltest.Escape("SELECT * FROM pg_type t JOIN pg_namespace n on t.typnamespace = n.oid WHERE t.typname = $1 AND t.typtype = 'e'")).
					WithArgs("enum").
					WillReturnRows(sqlmock.NewRows([]string{"name"}))
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE TYPE "enum" AS ENUM ('a')`,
						Reverse: `DROP TYPE "enum"`,
					},
					{
						Cmd:     `CREATE TABLE "posts" ("c1" "enum" NOT NULL)`,
						Reverse: `DROP TABLE "posts"`,
					},
				},
			},
		},
		// Empty sequence qualifier.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
						),
					Changes: schema.Changes{
						&schema.ModifyColumn{
							From:   schema.NewIntColumn("c1", "integer"),
							To:     schema.NewColumn("c1").SetType(&SerialType{T: "serial"}),
							Change: schema.ChangeType,
						},
					},
				},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE SEQUENCE IF NOT EXISTS "posts_c1_seq" OWNED BY "posts"."c1"`,
						Reverse: `DROP SEQUENCE IF EXISTS "posts_c1_seq"`,
					},
					{
						Cmd:     `ALTER TABLE "posts" ALTER COLUMN "c1" SET DEFAULT nextval('"posts_c1_seq"')`,
						Reverse: `ALTER TABLE "posts" ALTER COLUMN "c1" DROP DEFAULT`,
					},
				},
			},
		},
		// Empty index qualifier.
		{
			changes: []schema.Change{
				&schema.ModifyTable{
					T: schema.NewTable("posts").
						SetSchema(schema.New("public")).
						AddColumns(
							schema.NewIntColumn("c", "int"),
						),
					Changes: schema.Changes{
						&schema.AddIndex{
							I: schema.NewIndex("i").AddColumns(schema.NewIntColumn("c", "int")),
						},
					},
				},
			},
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `CREATE INDEX "i" ON "posts" ("c")`,
						Reverse: `DROP INDEX "i"`,
					},
				},
			},
		},
		// Foreign keys should be dropped before the tables they reference.
		{
			changes: func() []schema.Change {
				usersT := schema.NewTable("users").SetSchema(schema.New("public")).AddColumns(schema.NewIntColumn("id", "int"))
				postsT := schema.NewTable("posts").SetSchema(schema.New("public")).
					AddColumns(schema.NewIntColumn("id", "int"), schema.NewIntColumn("author_id", "int"))
				postsT.AddForeignKeys(schema.NewForeignKey("author").AddColumns(postsT.Columns[1]).SetRefTable(usersT).AddRefColumns(usersT.Columns...))
				return []schema.Change{
					&schema.DropTable{T: usersT},
					&schema.ModifyTable{T: postsT, Changes: []schema.Change{&schema.DropForeignKey{F: postsT.ForeignKeys[0]}}},
				}
			}(),
			options: []migrate.PlanOption{
				func(o *migrate.PlanOptions) { o.SchemaQualifier = new(string) },
			},
			wantPlan: &migrate.Plan{
				Reversible:    true,
				Transactional: true,
				Changes: []*migrate.Change{
					{
						Cmd:     `ALTER TABLE "posts" DROP CONSTRAINT "author"`,
						Reverse: `ALTER TABLE "posts" ADD CONSTRAINT "author" FOREIGN KEY ("author_id") REFERENCES "users" ("id")`,
					},
					{
						Cmd:     `DROP TABLE "users"`,
						Reverse: `CREATE TABLE "users" ("id" integer NOT NULL)`,
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
			db, mk, err := sqlmock.New()
			require.NoError(t, err)
			m := mock{mk}
			m.version("130000")
			if tt.mock != nil {
				tt.mock(m)
			}
			drv, err := Open(db)
			require.NoError(t, err)
			plan, err := drv.PlanChanges(context.Background(), "wantPlan", tt.changes, tt.options...)
			if tt.wantErr {
				require.Error(t, err, "expect plan to fail")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantPlan.Reversible, plan.Reversible)
			require.Equal(t, tt.wantPlan.Transactional, plan.Transactional)
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
	require.Equal(t, `CREATE TABLE "s1"."t1" ("a" integer NOT NULL)`, changes.Changes[0].Cmd)

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
			Cmd: `CREATE TABLE "t1" (
  "a" integer NOT NULL
)`,
		},
		{
			T: schema.NewTable("t1").
				AddColumns(
					schema.NewIntColumn("a", "int"),
					schema.NewIntColumn("b", "int"),
				),
			Cmd: `CREATE TABLE "t1" (
  "a" integer NOT NULL,
  "b" integer NOT NULL
)`,
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
			Cmd: `CREATE TABLE "t1" (
  "a" integer NOT NULL,
  "b" integer NOT NULL,
  "id" integer NOT NULL,
  PRIMARY KEY ("id")
)`,
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
			Cmd: `CREATE TABLE "t1" (
  "a" integer NOT NULL,
  "b" integer NOT NULL,
  CONSTRAINT "fk1" FOREIGN KEY ("a") REFERENCES "t2" ("a"),
  CONSTRAINT "fk2" FOREIGN KEY ("a") REFERENCES "t2" ("a")
)`,
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
			Cmd: `CREATE TABLE "t1" (
  "a" integer NOT NULL,
  "b" integer NOT NULL,
  "id" integer NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk1" FOREIGN KEY ("a") REFERENCES "t2" ("a"),
  CONSTRAINT "fk2" FOREIGN KEY ("a") REFERENCES "t2" ("a"),
  CONSTRAINT "ck1" CHECK (a > 0),
  CONSTRAINT "ck2" CHECK (a > 0)
)`,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			db, mk, err := sqlmock.New()
			require.NoError(t, err)
			mock{mk}.version("130000")
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
