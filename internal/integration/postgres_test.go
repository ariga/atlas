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
	s.NoError(err)
	s.Empty(changes)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.DropTable{T: usersT},
	})
	s.NoError(err)
	s.TestEmptyRealm()
}

func (s *pgSuite) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
			},
		},
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
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
