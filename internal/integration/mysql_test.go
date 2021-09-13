// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMySQL(t *testing.T) {
	for version, port := range map[string]int{"56": 3306, "57": 3307, "8": 3308} {
		t.Run(version, func(t *testing.T) {
			db, err := sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/test?parseTime=True", port))
			require.NoError(t, err)
			drv, err := mysql.Open(db)
			require.NoError(t, err)
			suite.Run(t, &mysqlSuite{
				db:      db,
				drv:     drv,
				version: version,
			})
		})
	}
}

type mysqlSuite struct {
	suite.Suite
	db      *sql.DB
	drv     *mysql.Driver
	version string
}

func (s *mysqlSuite) SetupTest() {
	_, err := s.db.Exec("DROP TABLE IF EXISTS `posts`, `users`")
	s.NoError(err, "truncate database")
}

func (s *mysqlSuite) TestEmptyRealm() {
	ctx := context.Background()
	realm, err := s.drv.InspectRealm(ctx, &schema.InspectRealmOption{
		Schemas: []string{"test"},
	})
	s.NoError(err)
	s.EqualValues(s.realm(), realm)
}

func (s *mysqlSuite) TestAddDropTable() {
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

func (s *mysqlSuite) TestRelation() {
	ctx := context.Background()
	usersT, postsT := s.users(), s.posts()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{
		&schema.AddTable{T: usersT},
		&schema.AddTable{T: postsT},
	})
	s.NoError(err)
	s.ensureNoChange(postsT, usersT)
}

func (s *mysqlSuite) TestAddIndexedColumns() {
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

func (s *mysqlSuite) TestChangeColumn() {
	ctx := context.Background()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: s.users()}})
	s.Require().NoError(err)
	usersT := s.users()
	usersT.Columns[1].Type = &schema.ColumnType{Raw: "mediumint", Type: &schema.IntegerType{T: "mediumint"}, Null: true}
	usersT.Columns[1].Default = &schema.RawExpr{X: "0"}
	changes, err := s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 1)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.NoError(err)
	s.ensureNoChange(usersT)
}

func (s *mysqlSuite) TestAddColumns() {
	ctx := context.Background()
	err := s.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: s.users()}})
	s.Require().NoError(err)
	usersT := s.users()
	usersT.Columns = append(
		usersT.Columns,
		&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "tinyblob", Type: &schema.BinaryType{T: "tinyblob"}}},
		&schema.Column{Name: "b", Type: &schema.ColumnType{Raw: "mediumblob", Type: &schema.BinaryType{T: "mediumblob"}}},
		&schema.Column{Name: "c", Type: &schema.ColumnType{Raw: "blob", Type: &schema.BinaryType{T: "blob"}}},
		&schema.Column{Name: "d", Type: &schema.ColumnType{Raw: "longblob", Type: &schema.BinaryType{T: "longblob"}}},
		&schema.Column{Name: "e", Type: &schema.ColumnType{Raw: "binary", Type: &schema.BinaryType{T: "binary"}}},
		&schema.Column{Name: "f", Type: &schema.ColumnType{Raw: "varbinary(255)", Type: &schema.BinaryType{T: "varbinary(255)"}}},
		&schema.Column{Name: "g", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}},
		&schema.Column{Name: "h", Type: &schema.ColumnType{Raw: "varchar(255)", Type: &schema.StringType{T: "varchar(255)"}}},
		&schema.Column{Name: "i", Type: &schema.ColumnType{Raw: "tinytext", Type: &schema.StringType{T: "tinytext"}}},
		&schema.Column{Name: "j", Type: &schema.ColumnType{Raw: "mediumtext", Type: &schema.StringType{T: "mediumtext"}}},
		&schema.Column{Name: "k", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
		&schema.Column{Name: "l", Type: &schema.ColumnType{Raw: "longtext", Type: &schema.StringType{T: "longtext"}}},
		&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 6}}},
		&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}}},
		&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 2}}},
		&schema.Column{Name: "p", Type: &schema.ColumnType{Type: &schema.FloatType{T: "double", Precision: 14}}},
		&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 14}}},
		&schema.Column{Name: "r", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
		&schema.Column{Name: "s", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
		&schema.Column{Name: "t", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}}},
		&schema.Column{Name: "u", Type: &schema.ColumnType{Type: &schema.EnumType{Values: []string{"a", "b", "c"}}}},
		&schema.Column{Name: "v", Type: &schema.ColumnType{Type: &schema.StringType{T: "char(36)"}}},
		&schema.Column{Name: "x", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "line"}}},
		&schema.Column{Name: "y", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"}}},
		&schema.Column{Name: "z", Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}},
	)
	changes, err := s.drv.Diff().TableDiff(s.loadRealm().Schemas[0].Tables[0], usersT)
	s.Require().NoError(err)
	s.Len(changes, 24)
	err = s.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
	s.Require().NoError(err)
	s.ensureNoChange(usersT)
}

func (s *mysqlSuite) loadRealm() *schema.Realm {
	r, err := s.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"test"},
	})
	s.Require().NoError(err)
	return r
}

func (s *mysqlSuite) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name:  "test",
				Attrs: s.defaultAttrs(),
			},
		},
		Attrs: s.defaultAttrs(),
	}
	r.Schemas[0].Realm = r
	return r
}

func (s *mysqlSuite) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&mysql.AutoIncrement{}},
			},
			{
				Name: "x",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
			},
		},
		Attrs: s.defaultAttrs(),
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (s *mysqlSuite) posts() *schema.Table {
	usersT := s.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: s.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&mysql.AutoIncrement{}},
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
				Attrs: []schema.Attr{
					&mysql.OnUpdate{
						A: "CURRENT_TIMESTAMP",
					},
				},
			},
		},
		Attrs: s.defaultAttrs(),
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

// defaultConfig returns the default charset and
// collation configuration based on the MySQL version.
func (s *mysqlSuite) defaultAttrs() []schema.Attr {
	var (
		charset   = "latin1"
		collation = "latin1_swedish_ci"
	)
	if s.version == "8" {
		charset = "utf8mb4"
		collation = "utf8mb4_0900_ai_ci"
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

func (s *mysqlSuite) ensureNoChange(tables ...*schema.Table) {
	realm := s.loadRealm()
	s.Require().Equal(len(realm.Schemas[0].Tables), len(tables))
	for i, t := range tables {
		changes, err := s.drv.Diff().TableDiff(realm.Schemas[0].Tables[i], t)
		s.Require().NoError(err)
		s.Empty(changes)
	}
}
