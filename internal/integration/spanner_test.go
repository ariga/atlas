// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/spanner"
	_ "github.com/googleapis/go-sql-spanner"
	"github.com/stretchr/testify/require"
)

type spannerTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
	once    sync.Once
}

var spannerTests = map[string]*spannerTest{
	"spanner-emulator": {port: 9010},
}

func stRun(t *testing.T, fn func(*spannerTest)) {
	for version, tt := range spannerTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("spanner", "projects/atlas-dev/instances/instance-1/databases/db-1")
					if err != nil {
						t.Fatal(err)
					}
					dbs = append(dbs, tt.db) // close connection after all tests have been run
					tt.drv, err = spanner.Open(tt.db)
					if err != nil {
						t.Fatal(err)
					}
				})
				tt := &spannerTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
}

func TestSpanner_AddDropTable(t *testing.T) {
	stRun(t, func(t *spannerTest) {
		usersT := t.users()
		postsT := t.posts()
		petsT := &schema.Table{
			Name:   "pets",
			Schema: usersT.Schema,
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int64"}}},
				{Name: "fk_pets_users_owner_id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int64"}, Null: true}},
			},
		}
		petsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
		petsT.ForeignKeys = []*schema.ForeignKey{
			{Symbol: "fk_pets_users_owner_id", Table: petsT, Columns: petsT.Columns[1:], RefTable: usersT, RefColumns: usersT.Columns[:1]},
		}

		t.dropTables(postsT.Name, usersT.Name, petsT.Name)
		t.dropIndexes("idx_author_id", "idx_id_author_id_unique")
		t.dropConstraints("pets.fk_pets_users_owner_id")

		t.migrate(
			&schema.AddTable{T: petsT},
			&schema.AddTable{T: postsT},
			&schema.AddTable{T: usersT},
		)
		ensureNoChange(t, usersT, postsT, petsT)
		t.migrate(
			&schema.DropForeignKey{F: &schema.ForeignKey{
				Symbol: "fk_posts_users_author_id",
				Table:  postsT,
			}},
			&schema.DropForeignKey{F: &schema.ForeignKey{
				Symbol: "fk_pets_users_owner_id",
				Table:  petsT,
			}},
			&schema.DropIndex{I: &schema.Index{
				Table: postsT,
				Name:  "idx_author_id",
			}},
			&schema.DropIndex{I: &schema.Index{
				Table: postsT,
				Name:  "idx_id_author_id_unique",
			}},
			&schema.DropTable{T: usersT},
			&schema.DropTable{T: postsT},
			&schema.DropTable{T: petsT},
		)
		// Ensure the realm is empty.
		require.EqualValues(t, t.realm(), t.loadRealm())
	})
}

func TestSpanner_AddColumns(t *testing.T) {
	stRun(t, func(t *spannerTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Type: &schema.BinaryType{T: spanner.TypeBytes}, Null: true}},
			&schema.Column{Name: "b", Type: &schema.ColumnType{Type: &schema.BinaryType{T: spanner.TypeBytes}, Null: true}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 2)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestSpanner_ColumnInt(t *testing.T) {
	ctx := context.Background()
	run := func(t *testing.T, change func(*schema.Column)) {
		stRun(t, func(t *spannerTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "INT64"}}},
					{Name: "b", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "BYTES"}, Null: true}},
				},
			}
			usersT.PrimaryKey = &schema.Index{
				Name:   "PRIMARY_KEY_USERS",
				Unique: true,
				Table:  usersT,
				Parts:  []*schema.IndexPart{{C: usersT.Columns[0]}},
			}
			usersT.Columns[0].Indexes = []*schema.Index{usersT.PrimaryKey}

			err := t.drv.ApplyChanges(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			change(usersT.Columns[1])
			changes := t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 1)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)
		})
	}

	t.Run("ChangeNull", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Type.Null = false
		})
	})

	t.Run("ChangeType", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Type.Type = &schema.StringType{T: "STRING", Size: 41}
		})
	})
}

func TestSpanner_ColumnArray(t *testing.T) {
	stRun(t, func(t *spannerTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})

		// Add column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "ARRAY<INT64>", Type: &spanner.ArrayType{Type: &schema.IntegerType{T: "INT64"}, T: "ARRAY<INT64>"}, Null: true}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestSpanner_HCL(t *testing.T) {
	full := `
schema "default" {
}
table "users" {
	schema = schema.default
	column "id" {
		type = INT64
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
table "posts" {
	schema = schema.default
	column "id" {
		type = INT64
	}
	column "tags" {
		type = STRING(42)
	}
	column "author_id" {
		type = INT64
	}
	foreign_key "author" {
		columns = [
			table.posts.column.author_id,
		]
		ref_columns = [
			table.users.column.id,
		]
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
`
	empty := `
schema "default" {
}
`
	stRun(t, func(t *spannerTest) {
		t.applyHcl(full)
		users := t.loadUsers()
		posts := t.loadPosts()
		t.dropTables(users.Name, posts.Name)
		t.dropIndexes("idx_author_id", "idx_id_author_id_unique")
		t.dropConstraints("posts.fk_posts_users_author_id")
		column, ok := users.Column("id")
		require.True(t, ok, "expected id column")
		require.Equal(t, "users", users.Name)
		column, ok = posts.Column("author_id")
		require.Equal(t, "author_id", column.Name)
		t.applyHcl(empty)
		require.Empty(t, t.realm().Schemas[0].Tables)
	})
}

func (t *spannerTest) driver() migrate.Driver {
	return t.drv
}

func (t *spannerTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := spanner.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *spannerTest) applyRealmHcl(spec string) {
	// not implemented
}

func (t *spannerTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *spannerTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{""},
	})
	require.NoError(t, err)
	return r
}

func (t *spannerTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *spannerTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *spannerTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *spannerTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{Raw: "INT64", Type: &schema.IntegerType{T: "INT64"}},
			},
			{
				Name: "x",
				Type: &schema.ColumnType{Raw: "INT64", Type: &schema.IntegerType{T: "INT64"}},
			},
		},
	}
	usersT.PrimaryKey = &schema.Index{
		Name:   "PRIMARY_KEY_USERS",
		Unique: true,
		Table:  usersT,
		Parts:  []*schema.IndexPart{{C: usersT.Columns[0]}},
	}
	usersT.Columns[0].Indexes = []*schema.Index{usersT.PrimaryKey}
	return usersT
}

func (t *spannerTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{Raw: "INT64", Type: &schema.IntegerType{T: "INT64"}},
			},
			{
				Name: "author_id",
				Type: &schema.ColumnType{Raw: "INT64", Type: &schema.IntegerType{T: "INT64"}, Null: true},
			},
			{
				Name: "ctime",
				Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}},
			},
		},
		Attrs: []schema.Attr{
			&schema.Comment{Text: "posts comment"},
		},
	}
	postsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
	postsT.Indexes = []*schema.Index{
		{Name: "idx_author_id", Parts: []*schema.IndexPart{{C: postsT.Columns[1]}}},
		{Name: "idx_id_author_id_unique", Unique: true, Parts: []*schema.IndexPart{{C: postsT.Columns[1]}, {C: postsT.Columns[0]}}},
	}
	postsT.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "fk_posts_users_author_id", Table: postsT, Columns: postsT.Columns[1:2], RefTable: usersT, RefColumns: usersT.Columns[:1], OnDelete: schema.NoAction},
	}
	return postsT
}

func (t *spannerTest) url(s string) string {
	return s
}
func (t *spannerTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "default",
			},
		},
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *spannerTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *spannerTest) migrate(changes ...schema.Change) {
	t.Helper()
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *spannerTest) dropIndexes(names ...string) {
	t.Cleanup(func() {
		for _, idx := range names {
			_, err := t.db.ExecContext(context.Background(), "DROP INDEX "+idx)
			// TODO(tmc): Add more check conditions
			if err != nil {
				if !strings.Contains(err.Error(), fmt.Sprintf("Index not found: %v", idx)) {
					require.NoError(t.T, err, "drop index %q", idx)
				}
			}
		}
	})
}

// dropConstraints drops foreign keys in the forms of refs. A ref is a period-separated "${table}.${constraint}"
func (t *spannerTest) dropConstraints(refs ...string) {
	t.Cleanup(func() {
		for _, ref := range refs {
			parts := strings.Split(ref, ".")
			table, cstraint := parts[0], parts[1]
			query := fmt.Sprintf("ALTER TABLE `%s` DROP CONSTRAINT `%s`", table, cstraint)
			_, err := t.db.ExecContext(context.Background(), query)
			if err != nil {
				if !strings.Contains(err.Error(), fmt.Sprintf("Table not found: %v", table)) {
					require.NoError(t.T, err, "drop constraint %q", ref)
				}
			}
		}
	})
}

func (t *spannerTest) dropTables(names ...string) {
	t.Cleanup(func() {
		for _, tbl := range names {
			_, err := t.db.ExecContext(context.Background(), "DROP TABLE "+tbl)
			if err != nil {
				if !strings.Contains(err.Error(), fmt.Sprintf("Table not found: %v", tbl)) {
					require.NoError(t.T, err, "drop table %q", tbl)
				}
			}
		}
	})
}

func (t *spannerTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.ExecContext(context.Background(), "DROP SCHEMA "+strings.Join(names, ", ")+" CASCADE")
		require.NoError(t.T, err, "drop schema %q", names)
	})
}
