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
	"spanner-emulator": {port: 9020},
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

		t.dropTables(postsT.Name, usersT.Name)
		t.dropIndexes("idx_author_id", "idx_id_author_id_unique")

		t.migrate(
			&schema.AddTable{T: usersT},
			&schema.AddTable{T: postsT},
		)
		ensureNoChange(t, usersT, postsT)
		t.migrate(
			&schema.DropForeignKey{F: &schema.ForeignKey{
				Symbol: "fk_posts_users_author_id",
				Table:  postsT,
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
		)
		// Ensure the realm is empty.
		require.EqualValues(t, t.realm(), t.loadRealm())
	})
}

func (t *spannerTest) driver() migrate.Driver {
	return t.drv
}

func (t *spannerTest) applyHcl(spec string) {
	// not implemented
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
			_, err := t.db.Exec("DROP INDEX " + idx)
			// TODO(tmc): Add more check conditions
			if err != nil {
				if !strings.Contains(err.Error(), fmt.Sprintf("Index not found: %v", idx)) {
					require.NoError(t.T, err, "drop index %q", idx)
				}
			}
		}
	})
}

func (t *spannerTest) dropTables(names ...string) {
	t.Cleanup(func() {
		for _, tbl := range names {
			_, err := t.db.Exec("DROP TABLE " + tbl)
			if err != nil {
				// TODO(tmc): Add more check conditions
				if !strings.Contains(err.Error(), fmt.Sprintf("Table not found: %v", tbl)) {
					require.NoError(t.T, err, "drop table %q", tbl)
				}
			}
		}
	})
}

func (t *spannerTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP SCHEMA " + strings.Join(names, ", ") + " CASCADE")
		require.NoError(t.T, err, "drop schema %q", names)
	})
}
