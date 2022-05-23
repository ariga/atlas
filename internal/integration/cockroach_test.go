// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type crdbTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
}

var crdbTests = struct {
	drivers map[string]*crdbTest
	ports   map[string]int
}{
	drivers: make(map[string]*crdbTest),
	ports: map[string]int{
		"cockroach": 26257,
	},
}

func crdbInit(d string) []io.Closer {
	var cs []io.Closer
	if d != "" {
		p, ok := crdbTests.ports[d]
		if ok {
			crdbTests.ports = map[string]int{d: p}
		} else {
			crdbTests.ports = make(map[string]int)
		}
	}
	for version, port := range crdbTests.ports {
		db, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=root dbname=defaultdb password=pass sslmode=disable", port))
		if err != nil {
			log.Fatalln(err)
		}
		cs = append(cs, db)

		drv, err := postgres.OpenCRDB(db)
		if err != nil {
			log.Fatalln(err)
		}
		crdbTests.drivers[version] = &crdbTest{db: db, drv: drv, version: version, port: port, rrw: &rrw{}}
	}
	return cs
}

func crdbRun(t *testing.T, fn func(*crdbTest)) {
	for version, tt := range crdbTests.drivers {
		t.Run(version, func(t *testing.T) {
			tt := &crdbTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
			fn(tt)
		})
	}
}

func TestCockroach_AddDropTable(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testAddDrop(t)
	})
}

func (t *crdbTest) dsn() string {
	return fmt.Sprintf("postgres://postgres:pass@localhost:%d/test?sslmode=disable", t.port)
}

func (t *crdbTest) driver() migrate.Driver {
	return t.drv
}

func (t *crdbTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *crdbTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := postgres.UnmarshalHCL([]byte(spec), &desired)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *crdbTest) valueByVersion(values map[string]string, defaults string) string {
	if v, ok := values[t.version]; ok {
		return v
	}
	return defaults
}

func (t *crdbTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"public"},
	})
	require.NoError(t, err)
	return r
}

func (t *crdbTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *crdbTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *crdbTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *crdbTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
			{
				Name: "x",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
			},
		},
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (t *crdbTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
			{
				Name:    "author_id",
				Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
				Default: &schema.Literal{V: "10"},
			},
			{
				Name: "ctime",
				Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}},
				Default: &schema.RawExpr{
					X: "CURRENT_TIMESTAMP",
				},
			},
		},
		Attrs: []schema.Attr{
			&schema.Comment{Text: "posts comment"},
		},
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

func (t *crdbTest) revisions() *schema.Table {
	versionsT := &schema.Table{
		Name: "atlas_schema_revisions",
		Columns: []*schema.Column{
			{Name: "version", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "description", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "execution_state", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "executed_at", Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone"}}},
			{Name: "execution_time", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			{Name: "hash", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "operator_version", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "meta", Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}, Raw: "jsonb"}},
		},
	}
	versionsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: versionsT.Columns[0]}}}
	return versionsT
}

func (t *crdbTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
			},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "C.UTF-8"},
			&postgres.CType{V: "C.UTF-8"},
		},
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *crdbTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *crdbTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *crdbTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *crdbTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP SCHEMA IF EXISTS " + strings.Join(names, ", ") + " CASCADE")
		require.NoError(t.T, err, "drop schema %q", names)
	})
}
func (t *crdbTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := postgres.UnmarshalHCL([]byte(spec), &desired)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}
