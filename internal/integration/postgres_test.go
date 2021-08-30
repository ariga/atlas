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
	s.ensureNoChange(usersT)
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
	s.ensureNoChange(postsT, usersT)
}

func (s *pgSuite) TestAddIndexedColumns() {
	ctx := context.Background()
	usersT := s.users()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.AddTable{T: usersT},
	})
	s.Require().NoError(err)
	usersT.Columns = append(usersT.Columns, &schema.Column{
		Name:    "a",
		Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
		Default: &schema.RawExpr{X: "10"},
	}, &schema.Column{
		Name:    "b",
		Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
		Default: &schema.RawExpr{X: "10"},
	})
	usersT.Indexes = append(usersT.Indexes, &schema.Index{
		Unique: true,
		Name:   "a_b_unique",
		Parts:  []*schema.IndexPart{{C: usersT.Columns[1]}, {C: usersT.Columns[2]}},
	})
	realm := s.loadRealm()
	changes, err := s.drv.Diff().TableDiff(realm.Schemas[0].Tables[0], usersT)
	s.NoError(err)
	s.NotEmpty(changes, "usersT contains 2 new columns and 1 new index")

	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.NoError(err)
	s.ensureNoChange(usersT)
}

func (s *pgSuite) TestChangeColumn() {
	ctx := context.Background()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: s.users()}})
	s.Require().NoError(err)
	usersT := s.users()
	usersT.Columns[1].Type = &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}, Null: true}
	usersT.Columns[1].Default = &schema.RawExpr{X: "0"}
	changes, err := s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 1)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.NoError(err)
	s.ensureNoChange(usersT)
}

func (s *pgSuite) TestAddColumns() {
	ctx := context.Background()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: s.users()}})
	s.Require().NoError(err)
	usersT := s.users()
	usersT.Columns = append(
		usersT.Columns,
		&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "bytea", Type: &schema.BinaryType{T: "bytea"}}},
		&schema.Column{Name: "b", Type: &schema.ColumnType{Raw: "double precision", Type: &schema.FloatType{T: "double precision", Precision: 10}}, Default: &schema.RawExpr{X: "10.1"}},
		&schema.Column{Name: "c", Type: &schema.ColumnType{Raw: "character", Type: &schema.StringType{T: "character"}}, Default: &schema.RawExpr{X: "'y'"}},
		&schema.Column{Name: "d", Type: &schema.ColumnType{Raw: "numeric(10,2)", Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}}, Default: &schema.RawExpr{X: "0.99"}},
		&schema.Column{Name: "e", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}, Default: &schema.RawExpr{X: "'{}'"}},
		&schema.Column{Name: "f", Type: &schema.ColumnType{Raw: "jsonb", Type: &schema.JSONType{T: "jsonb"}}, Default: &schema.RawExpr{X: "'1'"}},
		&schema.Column{Name: "g", Type: &schema.ColumnType{Raw: "float(10)", Type: &schema.FloatType{T: "float", Precision: 10}}, Default: &schema.RawExpr{X: "'1'"}},
		&schema.Column{Name: "h", Type: &schema.ColumnType{Raw: "float(30)", Type: &schema.FloatType{T: "float", Precision: 30}}, Default: &schema.RawExpr{X: "'1'"}},
		&schema.Column{Name: "i", Type: &schema.ColumnType{Raw: "float(53)", Type: &schema.FloatType{T: "float", Precision: 53}}, Default: &schema.RawExpr{X: "1"}},
		&schema.Column{Name: "j", Type: &schema.ColumnType{Raw: "serial", Type: &postgres.SerialType{T: "serial"}}},
		&schema.Column{Name: "k", Type: &schema.ColumnType{Raw: "money", Type: &postgres.CurrencyType{T: "money"}}, Default: &schema.RawExpr{X: "100"}},
		&schema.Column{Name: "l", Type: &schema.ColumnType{Raw: "money", Type: &postgres.CurrencyType{T: "money"}, Null: true}, Default: &schema.RawExpr{X: "'52093.89'::money"}},
		&schema.Column{Name: "m", Type: &schema.ColumnType{Raw: "boolean", Type: &schema.BoolType{T: "boolean"}, Null: true}, Default: &schema.RawExpr{X: "false"}},
		&schema.Column{Name: "n", Type: &schema.ColumnType{Raw: "point", Type: &schema.SpatialType{T: "point"}, Null: true}, Default: &schema.RawExpr{X: "'(1,2)'"}},
		&schema.Column{Name: "o", Type: &schema.ColumnType{Raw: "line", Type: &schema.SpatialType{T: "line"}, Null: true}, Default: &schema.RawExpr{X: "'{1,2,3}'"}},
	)
	changes, err := s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 15)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.Require().NoError(err)
	s.ensureNoChange(usersT)
}

func (s *pgSuite) TestEnums() {
	ctx := context.Background()
	_, err := s.drv.ExecContext(ctx, "DROP TYPE IF EXISTS state, day")
	s.Require().NoError(err)
	usersT := &schema.Table{
		Name:   "users",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
		},
	}
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
	s.Require().NoError(err, "create a new table with an enum column")
	s.ensureNoChange(usersT)

	usersT.Columns = append(
		usersT.Columns,
		&schema.Column{Name: "day", Type: &schema.ColumnType{Type: &schema.EnumType{T: "day", Values: []string{"sunday", "monday"}}}},
	)
	changes, err := s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 1)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.Require().NoError(err, "add a new enum column to existing table")
	s.ensureNoChange(usersT)

	e := usersT.Columns[1].Type.Type.(*schema.EnumType)
	e.Values = append(e.Values, "tuesday")
	changes, err = s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 1)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.Require().NoError(err, "append a value to existing enum")
	s.ensureNoChange(usersT)

	e = usersT.Columns[1].Type.Type.(*schema.EnumType)
	e.Values = append(e.Values, "wednesday", "thursday", "friday", "saturday")
	changes, err = s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 1)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.Require().NoError(err, "append multiple values to existing enum")
	s.ensureNoChange(usersT)
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
			{
				Name: "x",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
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
		{Symbol: "author_id", Table: postsT, Columns: postsT.Columns[1:2], RefTable: usersT, RefColumns: usersT.Columns[:1], OnDelete: schema.NoAction},
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

func (s *pgSuite) ensureNoChange(tables ...*schema.Table) {
	realm := s.loadRealm()
	s.Require().Equal(len(realm.Schemas[0].Tables), len(tables))
	for i, t := range tables {
		changes, err := s.drv.Diff().TableDiff(realm.Schemas[0].Tables[i], t)
		s.Require().NoError(err)
		s.Empty(changes)
	}
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
