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

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

type myTest struct {
	*testing.T
	db      *sql.DB
	drv     *mysql.Driver
	version string
}

var myTests struct {
	sync.Once
	drivers map[string]*myTest
}

func myRun(t *testing.T, fn func(*myTest)) {
	myTests.Do(func() {
		myTests.drivers = make(map[string]*myTest)
		for version, port := range map[string]int{"56": 3306, "57": 3307, "8": 3308} {
			db, err := sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/test?parseTime=True", port))
			require.NoError(t, err)
			drv, err := mysql.Open(db)
			require.NoError(t, err)
			myTests.drivers[version] = &myTest{db: db, drv: drv, version: version}
		}
	})
	for version, tt := range myTests.drivers {
		t.Run(version, func(t *testing.T) {
			tt := &myTest{T: t, db: tt.db, drv: tt.drv, version: version}
			fn(tt)
		})
	}
}

func TestMySQL_AddDropTable(t *testing.T) {
	myRun(t, func(t *myTest) {
		ctx := context.Background()
		usersT := t.users()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.AddTable{T: usersT},
		})
		require.NoError(t, err)
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Empty(t, changes)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.DropTable{T: usersT},
		})
		require.NoError(t, err)
		// Ensure the realm is empty.
		require.EqualValues(t, t.realm(), t.loadRealm())
	})
}

func TestMySQL_Relation(t *testing.T) {
	myRun(t, func(t *myTest) {
		ctx := context.Background()
		usersT, postsT := t.users(), t.posts()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.AddTable{T: usersT},
			&schema.AddTable{T: postsT},
		})
		require.NoError(t, err)
		t.dropTables("posts", "users")
		t.ensureNoChange(postsT, usersT)
	})
}

func TestMySQL_AddIndexedColumns(t *testing.T) {
	myRun(t, func(t *myTest) {
		ctx := context.Background()
		usersT := t.users()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.AddTable{T: usersT},
		})
		require.NoError(t, err)
		t.dropTables(usersT.Name)
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
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.NotEmpty(t, changes, "usersT contains 2 new columns and 1 new index")
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)
	})
}

func TestMySQL_AddColumns(t *testing.T) {
	myRun(t, func(t *myTest) {
		ctx := context.Background()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: t.users()}})
		require.NoError(t, err)
		usersT := t.users()
		t.dropTables(usersT.Name)
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
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 24)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)
	})
}

func TestMySQL_ColumnInt(t *testing.T) {
	ctx := context.Background()
	t.Run("ChangeType", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			for _, typ := range []string{"tinyint", "smallint", "mediumint", "bigint"} {
				usersT.Columns[0].Type.Type = &schema.IntegerType{T: typ}
				changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
				require.NoError(t, err)
				require.Len(t, changes, 1)
				err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
				require.NoError(t, err)
				t.ensureNoChange(usersT)
			}
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Default: &schema.RawExpr{X: "1"}}},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
			for _, x := range []string{"2", "'3'", "10.1"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
				require.NoError(t, err)
				require.Len(t, changes, 1)
				err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
				t.ensureNoChange(usersT)
			}
		})
	})
}

func TestMySQL_ColumnString(t *testing.T) {
	ctx := context.Background()
	t.Run("ChangeType", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(20)"}}}},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			for _, typ := range []string{"varchar(255)", "char(120)", "tinytext", "mediumtext", "longtext"} {
				usersT.Columns[0].Type.Type = &schema.StringType{T: typ}
				changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
				require.NoError(t, err)
				require.Len(t, changes, 1)
				err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
				require.NoError(t, err)
				t.ensureNoChange(usersT)
			}
		})
	})

	t.Run("AddWithDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}, Default: &schema.RawExpr{X: "hello"}},
					{Name: "b", Type: &schema.ColumnType{Type: &schema.StringType{T: "char(255)"}}, Default: &schema.RawExpr{X: "'world'"}},
				},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}, Default: &schema.RawExpr{X: "hello"}}},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
			for _, x := range []string{"2", "'3'", "'world'"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
				require.NoError(t, err)
				require.Len(t, changes, 1)
				err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
				t.ensureNoChange(usersT)
			}
		})
	})
}

func TestMySQL_ColumnBool(t *testing.T) {
	ctx := context.Background()
	t.Run("Add", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}},
					{Name: "b", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}}},
					{Name: "c", Type: &schema.ColumnType{Type: &schema.BoolType{T: "tinyint"}}},
					{Name: "d", Type: &schema.ColumnType{Type: &schema.BoolType{T: "tinyint(1)"}}},
				},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
		})
	})

	t.Run("AddWithDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "1"}},
					{Name: "b", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "0"}},
					{Name: "c", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "'1'"}},
					{Name: "d", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "'0'"}},
					{Name: "e", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "true"}},
					{Name: "f", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "false"}},
					{Name: "g", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "TRUE"}},
					{Name: "h", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "FALSE"}},
				},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}}, Default: &schema.RawExpr{X: "1"}},
				},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
			// Change default from "true" to "false" to "true".
			for _, x := range []string{"false", "true"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
				require.NoError(t, err)
				require.Len(t, changes, 1)
				err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
				t.ensureNoChange(usersT)
			}
		})
	})

	t.Run("ChangeNull", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name: "users",
				Columns: []*schema.Column{
					{Name: "a", Type: &schema.ColumnType{Type: &schema.BoolType{T: "bool"}, Null: true}},
				},
			}
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			t.ensureNoChange(usersT)
			usersT.Columns[0].Type.Null = false
			changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
			require.NoError(t, err)
			require.Len(t, changes, 1)
			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
			t.ensureNoChange(usersT)
		})
	})
}

func TestMySQL_ForeignKey(t *testing.T) {
	ctx := context.Background()
	t.Run("ChangeAction", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT, postsT := t.users(), t.posts()
			t.dropTables(postsT.Name, usersT.Name)
			err := t.drv.Migrate().Exec(ctx, []schema.Change{
				&schema.AddTable{T: usersT},
				&schema.AddTable{T: postsT},
			})
			require.NoError(t, err)
			t.ensureNoChange(postsT, usersT)

			postsT = t.loadPosts()
			fk, ok := postsT.ForeignKey("author_id")
			require.True(t, ok)
			fk.OnUpdate = schema.SetNull
			fk.OnDelete = schema.Cascade
			changes, err := t.drv.Diff().TableDiff(t.loadPosts(), postsT)
			require.NoError(t, err)
			require.Len(t, changes, 1)
			modifyF, ok := changes[0].(*schema.ModifyForeignKey)
			require.True(t, ok)
			require.True(t, modifyF.Change == schema.ChangeUpdateAction|schema.ChangeDeleteAction)

			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: postsT, Changes: changes}})
			require.NoError(t, err)
			t.ensureNoChange(postsT, usersT)
		})
	})

	t.Run("UnsetNull", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT, postsT := t.users(), t.posts()
			t.dropTables(postsT.Name, usersT.Name)
			fk, ok := postsT.ForeignKey("author_id")
			require.True(t, ok)
			fk.OnDelete = schema.SetNull
			fk.OnUpdate = schema.SetNull
			err := t.drv.Migrate().Exec(ctx, []schema.Change{
				&schema.AddTable{T: usersT},
				&schema.AddTable{T: postsT},
			})
			require.NoError(t, err)
			t.ensureNoChange(postsT, usersT)

			postsT = t.loadPosts()
			c, ok := postsT.Column("author_id")
			require.True(t, ok)
			c.Type.Null = false
			fk, ok = postsT.ForeignKey("author_id")
			require.True(t, ok)
			fk.OnUpdate = schema.NoAction
			fk.OnDelete = schema.NoAction
			changes, err := t.drv.Diff().TableDiff(t.loadPosts(), postsT)
			require.NoError(t, err)
			require.Len(t, changes, 2)
			modifyC, ok := changes[0].(*schema.ModifyColumn)
			require.True(t, ok)
			require.True(t, modifyC.Change == schema.ChangeNull)
			modifyF, ok := changes[1].(*schema.ModifyForeignKey)
			require.True(t, ok)
			require.True(t, modifyF.Change == schema.ChangeUpdateAction|schema.ChangeDeleteAction)

			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: postsT, Changes: changes}})
			require.NoError(t, err)
			t.ensureNoChange(postsT, usersT)
		})
	})
}

func TestMySQL_HCL(t *testing.T) {
	myRun(t, func(t *myTest) {
		t.applyHcl(`
schema "test" {
}
table "users" {
	schema = "test"
	column "email" {
		type = "string"
	}
}
`)
		users := t.loadUsers()
		t.dropTables(users.Name)
		column, ok := users.Column("email")
		require.True(t, ok, "expected name column")
		require.Equal(t, "users", users.Name)
		require.Equal(t, "email", column.Name)
		require.Equal(t, column.Type.Raw, "varchar(255)")
		t.applyHcl(`
schema "test" {
}
`)
		require.Empty(t, t.realm().Schemas[0].Tables)
	})
}

func (t *myTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := mysql.UnmarshalSpec([]byte(spec), schemahcl.Unmarshal, &desired)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.Diff().SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.Migrate().Exec(context.Background(), diff)
	require.NoError(t, err)
}

func (t *myTest) ensureNoChange(tables ...*schema.Table) {
	realm := t.loadRealm()
	require.Equal(t, len(realm.Schemas[0].Tables), len(tables))
	for i := range tables {
		changes, err := t.drv.Diff().TableDiff(realm.Schemas[0].Tables[i], tables[i])
		require.NoError(t, err)
		require.Empty(t, changes)
	}
}

func (t *myTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *myTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name:  "test",
				Attrs: t.defaultAttrs(),
			},
		},
		Attrs: t.defaultAttrs(),
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *myTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
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
		Attrs: t.defaultAttrs(),
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (t *myTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
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

func (t *myTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"test"},
	})
	require.NoError(t, err)
	return r
}

func (t *myTest) loadUsers() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	users, ok := realm.Schemas[0].Table("users")
	require.True(t, ok)
	return users
}

func (t *myTest) loadPosts() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	users, ok := realm.Schemas[0].Table("posts")
	require.True(t, ok)
	return users
}

// defaultConfig returns the default charset and
// collation configuration based on the MySQL version.
func (t *myTest) defaultAttrs() []schema.Attr {
	var (
		charset   = "latin1"
		collation = "latin1_swedish_ci"
	)
	if t.version == "8" {
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
