// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPostgres(t *testing.T) {
	for version, port := range map[string]int{"13": 5434} {
		t.Run(version, func(t *testing.T) {
			db, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=postgres dbname=test password=pass sslmode=disable", port))
			require.NoError(t, err)
			drv, err := postgres.Open(db)
			require.NoError(t, err)
			suite.Run(t, &pgSuite{
				db:      db,
				drv:     drv,
				version: version,
			})
		})
	}
}

type pgSuite struct {
	suite.Suite
	db      *sql.DB
	drv     *postgres.Driver
	version string
}

func (s *pgSuite) SetupTest() {
	_, err := s.db.Exec(`DROP TABLE IF EXISTS "posts", "users"`)
	s.NoError(err, "truncate database")
}

func (s *pgSuite) TestEmptyRealm() {
	realm := s.loadRealm()
	s.EqualValues(s.realm(), realm)
}

func (s *pgSuite) TestAddDropTable() {
	ctx := context.Background()
	usersT := s.users()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.AddTable{T: usersT},
	})
	s.Require().NoError(err)
	realm := s.loadRealm()
	changes, err := s.drv.Diff().TableDiff(realm.Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Empty(changes)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.DropTable{T: usersT},
	})
	s.NoError(err)
	s.TestEmptyRealm()
}

func (s *pgSuite) TestRelation() {
	ctx := context.Background()
	usersT, postsT := s.users(), s.posts()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.AddTable{T: usersT},
		&schema.AddTable{T: postsT},
	})
	s.Require().NoError(err)
	realm := s.loadRealm()
	changes, err := s.drv.Diff().TableDiff(realm.Schemas[0].Tables[0], postsT)
	s.Require().NoError(err)
	s.Empty(changes)
	changes, err = s.drv.Diff().TableDiff(realm.Schemas[0].Tables[1], usersT)
	s.NoError(err)
	s.Empty(changes)
}

func (s *pgSuite) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
		},
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (s *pgSuite) posts() *schema.Table {
	usersT := s.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
			{
				Name:    "author_id",
				Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
				Default: &schema.RawExpr{X: "10"},
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
		{Symbol: "author_id", Table: postsT, Columns: postsT.Columns[1:2], RefTable: usersT, RefColumns: usersT.Columns, OnDelete: schema.NoAction},
	}
	return postsT
}

func (s *pgSuite) loadRealm() *schema.Realm {
	r, err := s.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"public"},
	})
	s.Require().NoError(err)
	return r
}

func (s *pgSuite) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
			},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "en_US.utf8"},
			&postgres.CType{V: "en_US.utf8"},
		},
	}
	r.Schemas[0].Realm = r
	return r
}
