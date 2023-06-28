// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mssql"
	"ariga.io/atlas/sql/schema"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"
)

type msTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
	once    sync.Once
}

var msTests = map[string]*myTest{
	"sqlserver-2022": {port: 1433},
}

func msRun(t *testing.T, fn func(*msTest)) {
	for version, tt := range msTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					q := url.Values{}
					q.Add("database", "db2")
					q.Add("connection timeout", "30")
					u := &url.URL{
						Scheme:   "sqlserver",
						User:     url.UserPassword("sa", "Passw0rd!995"),
						Host:     fmt.Sprintf("localhost:%d", tt.port),
						RawQuery: q.Encode(),
					}
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("sqlserver", u.String())
					require.NoError(t, err)
					// Close connection after all tests have been run.
					dbs = append(dbs, tt.db)
					tt.drv, err = mssql.Open(tt.db)
					require.NoError(t, err)
				})
				tt := &msTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
}

func TestMSSQL_Executor(t *testing.T) {
	msRun(t, func(t *msTest) {
		testExecutor(t)
	})
}

func (t *msTest) url(dbname string) string {
	q := url.Values{}
	q.Add("database", dbname)
	q.Add("connection timeout", "30")
	u := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword("sa", "Passw0rd!995"),
		Host:     fmt.Sprintf("localhost:%d", t.port),
		RawQuery: q.Encode(),
	}
	return u.String()
}

func (t *msTest) driver() migrate.Driver {
	return t.drv
}

func (t *msTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *msTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name:  "dbo",
				Attrs: t.defaultAttrs(),
			},
		},
		Attrs: t.defaultAttrs(),
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *msTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"dbo"},
	})
	require.NoError(t, err)
	return r
}

func (t *msTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&mssql.Identity{}},
			},
			{
				Name: "x",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
			},
		},
		Attrs: t.defaultAttrs(),
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (t *msTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *msTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&mssql.Identity{}},
			},
			{
				Name:    "author_id",
				Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
				Default: &schema.RawExpr{X: "10"},
			},
			{
				Name: "ctime",
				Type: &schema.ColumnType{Raw: mssql.TypeDateTime2, Type: &schema.TimeType{T: mssql.TypeDateTime2}},
				Default: &schema.RawExpr{
					X: "CURRENT_TIMESTAMP",
				},
				Attrs: []schema.Attr{
					// &mssql.OnUpdate{
					// 	A: "CURRENT_TIMESTAMP",
					// },
				},
			},
		},
		Attrs: t.defaultAttrs(),
	}
	postsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: postsT.Columns[0]}}}
	postsT.Indexes = []*schema.Index{
		{Name: "author_id", Parts: []*schema.IndexPart{{C: postsT.Columns[1]}}},
		{Name: "id_author_id_unique", Unique: true, Parts: []*schema.IndexPart{{C: postsT.Columns[1]}, {C: postsT.Columns[0]}}},
	}
	postsT.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "author_id", Table: postsT, Columns: postsT.Columns[1:2], RefTable: usersT, RefColumns: usersT.Columns[:1], OnDelete: schema.NoAction},
	}
	return postsT
}

func (t *msTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *msTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *msTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *msTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		for _, n := range names {
			_, err := t.db.Exec("DROP DATABASE IF EXISTS " + n)
			require.NoError(t.T, err, "drop db %q", names)
		}
	})
}

func (t *msTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *msTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *msTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := mssql.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	require.NoError(t, err)
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *msTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := mssql.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

// defaultConfig returns the default charset and
// collation configuration based on the MySQL version.
func (t *msTest) defaultAttrs() []schema.Attr {
	var (
		charset   = "latin1"
		collation = "latin1_swedish_ci"
	)
	switch {
	case strings.Contains(t.version, "tidb"):
		charset = "utf8mb4"
		collation = "utf8mb4_bin"
	case t.version == "mysql8":
		charset = "utf8mb4"
		collation = "utf8mb4_0900_ai_ci"
	case t.version == "maria107":
		charset = "utf8mb4"
		collation = "utf8mb4_general_ci"
	}
	return []schema.Attr{
		&schema.Charset{
			V: charset,
		},
		&schema.Collation{
			V: collation,
		},
	}
}
