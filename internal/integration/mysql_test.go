// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

type myTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
	once    sync.Once
}

var myTests = map[string]*myTest{
	"mysql56":  {port: 3306},
	"mysql57":  {port: 3307},
	"mysql8":   {port: 3308},
	"maria107": {port: 4306},
	"maria102": {port: 4307},
	"maria103": {port: 4308},
}

func myRun(t *testing.T, fn func(*myTest)) {
	for version, tt := range myTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/test?parseTime=True", tt.port))
					require.NoError(t, err)
					// Close connection after all tests have been run.
					dbs = append(dbs, tt.db)
					tt.drv, err = mysql.Open(tt.db)
					require.NoError(t, err)
				})
				tt := &myTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
}

func TestMySQL_Executor(t *testing.T) {
	myRun(t, func(t *myTest) {
		testExecutor(t)
	})
}

func TestMySQL_AddDropTable(t *testing.T) {
	myRun(t, func(t *myTest) {
		testAddDrop(t)
	})
}

func TestMySQL_Relation(t *testing.T) {
	myRun(t, func(t *myTest) {
		testRelation(t)
	})
}

func TestMySQL_AddIndexedColumns(t *testing.T) {
	myRun(t, func(t *myTest) {
		usersT := &schema.Table{
			Name:    "users",
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
		}
		t.migrate(&schema.AddTable{T: usersT})
		t.dropTables(usersT.Name)
		usersT.Columns = append(usersT.Columns, &schema.Column{
			Name:    "a",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.RawExpr{X: "10"},
		}, &schema.Column{
			Name:    "b",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.RawExpr{X: "10"},
		}, &schema.Column{
			Name:    "c",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.RawExpr{X: "10"},
		})
		parts := usersT.Columns[len(usersT.Columns)-3:]
		usersT.Indexes = append(usersT.Indexes, &schema.Index{
			Unique: true,
			Name:   "a_b_c_unique",
			Parts:  []*schema.IndexPart{{C: parts[0]}, {C: parts[1]}, {C: parts[2]}},
		})
		changes := t.diff(t.loadUsers(), usersT)
		require.NotEmpty(t, changes, "usersT contains 2 new columns and 1 new index")
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)

		// In MySQL, dropping a column should remove it from the key.
		// However, on MariaDB an explicit DROP/ADD INDEX is required.
		if t.mariadb() {
			idx, ok := usersT.Index("a_b_c_unique")
			require.True(t, ok)
			idx.Parts = idx.Parts[:len(idx.Parts)-1]
		}
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		changes = t.diff(t.loadUsers(), usersT)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())

		// Dropping a column from both table and index.
		usersT = t.loadUsers()
		idx, ok := usersT.Index("a_b_c_unique")
		require.True(t, ok)
		require.Len(t, idx.Parts, 2)
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		idx.Parts = idx.Parts[:len(idx.Parts)-1]
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 2)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())

		// Dropping a column should remove
		// single-column keys as well.
		usersT = t.loadUsers()
		idx, ok = usersT.Index("a_b_c_unique")
		require.True(t, ok)
		require.Len(t, idx.Parts, 1)
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())
		idx, ok = t.loadUsers().Index("a_b_c_unique")
		require.False(t, ok)
	})
}

func TestMySQL_AddColumns(t *testing.T) {
	myRun(t, func(t *myTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "tinyblob", Type: &schema.BinaryType{T: "tinyblob"}}},
			&schema.Column{Name: "b", Type: &schema.ColumnType{Raw: "mediumblob", Type: &schema.BinaryType{T: "mediumblob"}}},
			&schema.Column{Name: "c", Type: &schema.ColumnType{Raw: "blob", Type: &schema.BinaryType{T: "blob"}}},
			&schema.Column{Name: "d", Type: &schema.ColumnType{Raw: "longblob", Type: &schema.BinaryType{T: "longblob"}}},
			&schema.Column{Name: "e", Type: &schema.ColumnType{Raw: "binary", Type: &schema.BinaryType{T: "binary"}}},
			&schema.Column{Name: "f", Type: &schema.ColumnType{Raw: "varbinary(255)", Type: &schema.BinaryType{T: "varbinary(255)"}}, Default: &schema.Literal{V: "foo"}},
			&schema.Column{Name: "g", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 255}}},
			&schema.Column{Name: "h", Type: &schema.ColumnType{Raw: "varchar(255)", Type: &schema.StringType{T: "varchar(255)"}}},
			&schema.Column{Name: "i", Type: &schema.ColumnType{Raw: "tinytext", Type: &schema.StringType{T: "tinytext"}}},
			&schema.Column{Name: "j", Type: &schema.ColumnType{Raw: "mediumtext", Type: &schema.StringType{T: "mediumtext"}}},
			&schema.Column{Name: "k", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
			&schema.Column{Name: "l", Type: &schema.ColumnType{Raw: "longtext", Type: &schema.StringType{T: "longtext"}}},
			&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 6}}},
			&schema.Column{Name: "m1", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal"}}},
			&schema.Column{Name: "m2", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 2}}},
			&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}}},
			&schema.Column{Name: "n1", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric"}}},
			&schema.Column{Name: "n2", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 2}}},
			&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 2}}},
			&schema.Column{Name: "p", Type: &schema.ColumnType{Type: &schema.FloatType{T: "double", Precision: 14}}},
			&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 14}}},
			&schema.Column{Name: "r", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
			&schema.Column{Name: "s", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			&schema.Column{Name: "t", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}}},
			&schema.Column{Name: "u", Type: &schema.ColumnType{Type: &schema.EnumType{T: "enum", Values: []string{"a", "b", "c"}}}},
			&schema.Column{Name: "v", Type: &schema.ColumnType{Type: &schema.StringType{T: "char(36)"}}},
			&schema.Column{Name: "x", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "line"}}},
			&schema.Column{Name: "y", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"}}},
			&schema.Column{Name: "z", Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 28)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestMySQL_ColumnInt(t *testing.T) {
	t.Run("ChangeType", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			for _, typ := range []string{"tinyint", "smallint", "mediumint", "bigint"} {
				usersT.Columns[0].Type.Type = &schema.IntegerType{T: typ}
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
			}
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Default: &schema.RawExpr{X: "1"}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
			for _, x := range []string{"2", "'3'", "10.1"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
			}
		})
	})
}

func TestMySQL_ColumnString(t *testing.T) {
	t.Run("ChangeType", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(20)"}}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			for _, typ := range []string{"varchar(255)", "char(120)", "tinytext", "mediumtext", "longtext"} {
				usersT.Columns[0].Type.Type = &schema.StringType{T: typ}
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
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
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar(255)"}}, Default: &schema.RawExpr{X: "hello"}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
			for _, x := range []string{"2", "'3'", "'world'"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
			}
		})
	})
}

func TestMySQL_ColumnBool(t *testing.T) {
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
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
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
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
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
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
			// Change default from "true" to "false" to "true".
			for _, x := range []string{"false", "true"} {
				usersT.Columns[0].Default.(*schema.RawExpr).X = x
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
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
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
			usersT.Columns[0].Type.Null = false
			changes := t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 1)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)
		})
	})
}

func TestMySQL_ColumnCheck(t *testing.T) {
	myRun(t, func(t *myTest) {
		// Checks are not supported in all versions.
		if t.version == "mysql56" || t.version == "mysql57" {
			t.Skip()
		}
		usersT := &schema.Table{
			Name:  "users",
			Attrs: []schema.Attr{schema.NewCheck().SetName("users_c_check").SetExpr("c > 5")},
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
				{Name: "c", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}},
			},
		}
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})
		ensureNoChange(t, usersT)
	})
}

func TestMySQL_ForeignKey(t *testing.T) {
	t.Run("ChangeAction", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT, postsT := t.users(), t.posts()
			t.dropTables(postsT.Name, usersT.Name)
			t.migrate(&schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
			ensureNoChange(t, postsT, usersT)

			postsT = t.loadPosts()
			fk, ok := postsT.ForeignKey("author_id")
			require.True(t, ok)
			fk.OnUpdate = schema.SetNull
			fk.OnDelete = schema.Cascade
			changes := t.diff(t.loadPosts(), postsT)
			require.Len(t, changes, 1)
			modifyF, ok := changes[0].(*schema.ModifyForeignKey)
			require.True(t, ok)
			require.True(t, modifyF.Change == schema.ChangeUpdateAction|schema.ChangeDeleteAction)

			t.migrate(&schema.ModifyTable{T: postsT, Changes: changes})
			ensureNoChange(t, postsT, usersT)
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
			t.migrate(&schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
			ensureNoChange(t, postsT, usersT)

			postsT = t.loadPosts()
			c, ok := postsT.Column("author_id")
			require.True(t, ok)
			c.Type.Null = false
			fk, ok = postsT.ForeignKey("author_id")
			require.True(t, ok)
			fk.OnUpdate = schema.NoAction
			fk.OnDelete = schema.NoAction
			changes := t.diff(t.loadPosts(), postsT)
			require.Len(t, changes, 2)
			modifyC, ok := changes[0].(*schema.ModifyColumn)
			require.True(t, ok)
			require.True(t, modifyC.Change == schema.ChangeNull)
			modifyF, ok := changes[1].(*schema.ModifyForeignKey)
			require.True(t, ok)
			require.True(t, modifyF.Change == schema.ChangeUpdateAction|schema.ChangeDeleteAction)

			t.migrate(&schema.ModifyTable{T: postsT, Changes: changes})
			ensureNoChange(t, postsT, usersT)
		})
	})

	t.Run("AddDrop", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			usersT := t.users()
			t.dropTables(usersT.Name)
			t.migrate(&schema.AddTable{T: usersT})
			ensureNoChange(t, usersT)

			// Add foreign key.
			usersT.Columns = append(usersT.Columns, &schema.Column{
				Name: "spouse_id",
				Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			})
			usersT.ForeignKeys = append(usersT.ForeignKeys, &schema.ForeignKey{
				Symbol:     "spouse_id",
				Table:      usersT,
				Columns:    usersT.Columns[len(usersT.Columns)-1:],
				RefTable:   usersT,
				RefColumns: usersT.Columns[:1],
				OnDelete:   schema.NoAction,
			})

			changes := t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 2)
			addC, ok := changes[0].(*schema.AddColumn)
			require.True(t, ok)
			require.Equal(t, "spouse_id", addC.C.Name)
			addF, ok := changes[1].(*schema.AddForeignKey)
			require.True(t, ok)
			require.Equal(t, "spouse_id", addF.F.Symbol)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)

			// Drop foreign keys.
			usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
			usersT.ForeignKeys = usersT.ForeignKeys[:len(usersT.ForeignKeys)-1]
			changes = t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 2)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)
		})
	})
}

func TestMySQL_AdvisoryLock(t *testing.T) {
	myRun(t, func(t *myTest) {
		testAdvisoryLock(t.T, t.drv.(schema.Locker))
	})
}

func TestMySQL_HCL(t *testing.T) {
	full := `
schema "test" {
}
table "users" {
	schema = schema.test
	column "id" {
		type = int
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
table "posts" {
	schema = schema.test
	column "id" {
		type = int
	}
	column "author_id" {
		type = int
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
schema "test" {
}
`
	myRun(t, func(t *myTest) {
		testHCLIntegration(t, full, empty)
	})
}

func TestMySQL_Snapshot(t *testing.T) {
	myRun(t, func(t *myTest) {
		db, err := sql.Open("mysql", fmt.Sprintf("root:pass@tcp(localhost:%d)/", t.port))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, db.Close())
		})
		drv, err := mysql.Open(db)
		require.NoError(t, err)

		_, err = drv.(migrate.Snapshoter).Snapshot(context.Background())
		require.ErrorAs(t, err, new(*migrate.NotCleanError))

		r, err := t.driver().InspectRealm(context.Background(), &schema.InspectRealmOption{
			Mode: ^schema.InspectViews,
		})
		require.NoError(t, err)
		restore, err := t.driver().(migrate.Snapshoter).Snapshot(context.Background())
		require.NoError(t, err) // connected to test schema
		t.migrate(&schema.AddTable{T: schema.NewTable("my_table").AddColumns(
			schema.NewIntColumn("col_1", "integer").SetNull(true),
			schema.NewIntColumn("col_2", "bigint"),
		)})
		t.Cleanup(func() {
			t.dropTables("my_table")
		})
		require.NoError(t, restore(context.Background()))
		r1, err := t.driver().InspectRealm(context.Background(), &schema.InspectRealmOption{
			Mode: ^schema.InspectViews,
		})
		require.NoError(t, err)
		diff, err := t.driver().RealmDiff(r1, r)
		require.NoError(t, err)
		require.Zero(t, diff)
	})
}

func TestMySQL_CLI_MigrateApplyBC(t *testing.T) {
	myRun(t, func(t *myTest) {
		testCLIMigrateApplyBC(t, "mysql")
	})
}

func TestMySQL_CLI_MigrateApplyLock(t *testing.T) {
	myRun(t, func(t *myTest) {
		t.dropSchemas("mysqlock")
		t.migrate(&schema.AddSchema{S: schema.New("mysqlock")})
		var (
			b  atomic.Bool
			wg sync.WaitGroup
		)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				out, err := exec.Command(
					execPath(t),
					"migrate", "apply",
					"--dir", "file://testdata/migrations/mysqlock",
					"--url", t.url("mysqlock"),
				).CombinedOutput()
				require.NoError(t, err, string(out))
				switch {
				// Nop.
				case err == nil && strings.HasPrefix(string(out), "No migration files to execute"):
				// Successful run.
				case err == nil && strings.HasPrefix(string(out), "Migrating to version 3"):
					if b.Swap(true) {
						t.Errorf("migration ran twice: %s", out)
					}
				}
			}(i)
		}
		wg.Wait()
		require.True(t, b.Load(), "Migration should run successfully exactly once")
	})
}

func TestMySQL_CLI(t *testing.T) {
	h := `
			schema "test" {
				charset   = "%s"
				collation = "%s"
			}
			table "users" {
				schema = schema.test
				column "id" {
					type = int
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaInspect(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"), mysql.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaApply(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"))
		})
	})
	t.Run("SchemaApplyWithVars", func(t *testing.T) {
		h := `
variable "tenant" {
	type = string
}
schema "tenant" {
	name = var.tenant
}
table "users" {
	schema = schema.tenant
	column "id" {
		type = int
	}
}
`
		myRun(t, func(t *myTest) {
			testCLISchemaApply(t, h, t.url("test"), "--var", "tenant=test")
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaApplyDry(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"))
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			testCLISchemaDiff(t, t.url("test"))
		})
	})
	t.Run("SchemaApplyAutoApprove", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaApplyAutoApprove(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"))
		})
	})
	t.Run("SchemaApplyFromMigrationDir", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			testCLISchemaApplyFromMigrationDir(t)
		})
	})
}

func TestMySQL_CLI_MultiSchema(t *testing.T) {
	h := `
			schema "test" {
				charset   = "%s"
				collation = "%s"
			}
			table "users" {
				schema = schema.test
				column "id" {
					type = int
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}
			schema "test2" {
				charset   = "%s"
				collation = "%s"
			}
			table "users" {
				schema = schema.test2
				column "id" {
					type = int
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLIMultiSchemaInspect(t, fmt.Sprintf(h, charset.V, collate.V, charset.V, collate.V), t.url(""), []string{"test", "test2"}, mysql.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLIMultiSchemaApply(t, fmt.Sprintf(h, charset.V, collate.V, charset.V, collate.V), t.url(""), []string{"test", "test2"}, mysql.EvalHCL)
		})
	})
}

func TestMySQL_HCL_Realm(t *testing.T) {
	myRun(t, func(t *myTest) {
		t.dropSchemas("second")
		realm := t.loadRealm()
		hcl, err := mysql.MarshalHCL(realm)
		require.NoError(t, err)
		wa := string(hcl) + `
schema "second" {
}
`
		t.applyRealmHcl(wa)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
			Mode: ^schema.InspectViews,
		})
		require.NoError(t, err)
		_, ok := realm.Schema("test")
		require.True(t, ok)
		_, ok = realm.Schema("second")
		require.True(t, ok)
	})
}

func TestMySQL_HCL_ForeignKeyCrossSchema(t *testing.T) {
	const expected = `table "credit_cards" {
  schema = schema.financial
  column "id" {
    null = false
    type = int
  }
  column "user_id" {
    null = false
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "user_id_fkey" {
    columns = [column.user_id]
  }
}
table "financial" "users" {
  schema = schema.financial
  column "id" {
    null = false
    type = int
  }
}
table "users" "users" {
  schema = schema.users
  column "id" {
    null = false
    type = int
  }
  column "email" {
    null = false
    type = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "financial" {
  charset = "utf8mb4"
  collate = "utf8mb4_general_ci"
}
schema "users" {
  charset = "utf8mb4"
  collate = "utf8mb4_general_ci"
}
`
	myRun(t, func(t *myTest) {
		t.dropSchemas("financial", "users")
		realm := t.loadRealm()
		hcl, err := mysql.MarshalHCL(realm)
		require.NoError(t, err)
		t.applyRealmHcl(string(hcl) + "\n" + expected)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
			Mode: ^schema.InspectViews,
			Schemas: []string{"users", "financial"},
		})
		require.NoError(t, err)
		actual, err := mysql.MarshalHCL(realm)
		require.NoError(t, err)
		require.Equal(t, expected, string(actual))
	})
}

func TestMySQL_DefaultsHCL(t *testing.T) {
	n := "atlas_defaults"
	myRun(t, func(t *myTest) {
		ddl := `
create table atlas_defaults
(
	string varchar(255) default "hello_world",
	quoted varchar(100) default 'never say "never"',
	tBit bit(10) default b'10101',
	ts timestamp default CURRENT_TIMESTAMP,
	number int default 42
)
`
		t.dropTables(n)
		_, err := t.db.Exec(ddl)
		require.NoError(t, err)
		realm := t.loadRealm()
		spec, err := mysql.MarshalHCL(realm.Schemas[0])
		require.NoError(t, err)
		var s schema.Realm
		err = mysql.EvalHCLBytes(spec, &s, nil)
		require.NoError(t, err)
		t.dropTables(n)
		t.applyHcl(string(spec))
		ensureNoChange(t, realm.Schemas[0].Tables[0])
	})
}

func TestMySQL_Sanity(t *testing.T) {
	n := "atlas_types_sanity"
	t.Run("Common", func(t *testing.T) {
		ddl := `
create table atlas_types_sanity
(
    tBit                        bit(10)              default b'100'                                              null,
    tInt                        int(10)              default 4                                               not null,
    tTinyInt                    tinyint(10)          default 8                                                   null,
    tSmallInt                   smallint(10)         default 2                                                   null,
    tMediumInt                  mediumint(10)        default 11                                                  null,
    tBigInt                     bigint(10)           default 4                                                   null,
    tDecimal                    decimal              default 4                                                   null,
    tNumeric                    numeric              default 4                                               not null,
    tFloat                      float(10, 0)         default 4                                                   null,
    tDouble                     double(10, 0)        default 4                                                   null,
    tReal                       double(10, 0)        default 4                                                   null,
    tTimestamp                  timestamp            default CURRENT_TIMESTAMP                                   null,
    tTimestampFraction          timestamp(6)         default CURRENT_TIMESTAMP(6)                                null,
    tTimestampOnUpdate          timestamp            default CURRENT_TIMESTAMP    ON UPDATE CURRENT_TIMESTAMP    null,
    tTimestampFractionOnUpdate  timestamp(6)         default CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) null,
    tDate                       date                                                                             null,
    tTime                       time                                                                             null,
    tDateTime                   datetime                                                                         null,
    tYear                       year                                                                             null,
    tVarchar                    varchar(10)          default 'Titan'                                             null,
    tChar                       char(25)             default 'Olimpia'                                       not null,
    tVarBinary                  varbinary(30)        default 'Titan'                                             null,
    tBinary                     binary(5)            default 'Titan'                                             null,
    tBlob                       blob(5)              default                                                     null,
    tTinyBlob                   tinyblob                                                                         null,
    tMediumBlob                 mediumblob           default                                                     null,
    tLongBlob                   longblob             default                                                     null,
    tText                       text(13)             default                                                     null,
    tTinyText                   tinytext             default                                                     null,
    tMediumText                 mediumtext           default                                                     null,
    tLongText                   longtext             default                                                     null,
    tEnum                       enum('a','b')        default                                                     null,
    tSet                        set('a','b')         default                                                     null,
    tGeometry                   geometry             default                                                     null,
    tPoint                      point                default                                                     null,
    tMultiPoint                 multipoint           default                                                     null,
    tLineString                 linestring           default                                                     null,
    tMultiLineString            multilinestring      default                                                     null,
    tPolygon                    polygon              default                                                     null,
    tMultiPolygon               multipolygon         default                                                     null,
    tGeometryCollection         geometrycollection   default                                                     null
) CHARSET = latin1 COLLATE latin1_swedish_ci;
`
		myRun(t, func(t *myTest) {
			t.dropTables(n)
			_, err := t.db.Exec(ddl)
			require.NoError(t, err)
			realm := t.loadRealm()
			require.Len(t, realm.Schemas, 1)
			ts, ok := realm.Schemas[0].Table(n)
			require.True(t, ok)
			expected := schema.Table{
				Name: n,
				Attrs: []schema.Attr{
					&schema.Charset{V: "latin1"},
					&schema.Collation{V: "latin1_swedish_ci"},
					&mysql.Engine{V: "InnoDB", Default: true},
				},
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					{
						Name:    "tBit",
						Type:    &schema.ColumnType{Type: &mysql.BitType{T: "bit", Size: 10}, Raw: "bit(10)", Null: true},
						Default: &schema.Literal{V: "b'100'"},
					},
					{
						Name: "tInt",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "int"}, "int(10)"), Null: false},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tTinyInt",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "tinyint", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "tinyint"}, "tinyint(10)"), Null: true},
						Default: &schema.Literal{V: "8"},
					},
					{
						Name: "tSmallInt",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "smallint"}, "smallint(10)"), Null: true},
						Default: &schema.Literal{V: "2"},
					},
					{
						Name: "tMediumInt",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "mediumint", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "mediumint"}, "mediumint(10)"), Null: true},
						Default: &schema.Literal{V: "11"},
					},
					{
						Name: "tBigInt",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "bigint"}, "bigint(10)"), Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tDecimal",
						Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 10},
							Raw: "decimal(10,0)", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tNumeric",
						Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 10},
							Raw: "decimal(10,0)", Null: false},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tFloat",
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 10},
							Raw: "float(10,0)", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tDouble",
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "double", Precision: 10},
							Raw: "double(10,0)", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tReal",
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "double", Precision: 10},
							Raw: "double(10,0)", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tTimestamp",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"},
							Raw: "timestamp", Null: true},
						Default: &schema.RawExpr{
							X: func() string {
								if t.mariadb() {
									return "(current_timestamp())"
								}
								return "CURRENT_TIMESTAMP"
							}(),
						},
					},
					{
						Name: "tTimestampFraction",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp", Precision: intp(6)},
							Raw: "timestamp(6)", Null: true},
						Default: &schema.RawExpr{
							X: func() string {
								if t.mariadb() {
									return "(current_timestamp(6))"
								}
								return "CURRENT_TIMESTAMP(6)"
							}(),
						},
					},
					{
						Name: "tTimestampOnUpdate",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"},
							Raw: "timestamp", Null: true},
						Default: &schema.RawExpr{
							X: func() string {
								if t.mariadb() {
									return "(current_timestamp())"
								}
								return "CURRENT_TIMESTAMP"
							}(),
						},
						Attrs: []schema.Attr{
							&mysql.OnUpdate{
								A: func() string {
									if t.mariadb() {
										return "current_timestamp()"
									}
									return "CURRENT_TIMESTAMP"
								}(),
							},
						},
					},
					{
						Name: "tTimestampFractionOnUpdate",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp", Precision: intp(6)}, Raw: "timestamp(6)", Null: true},
						Default: &schema.RawExpr{
							X: func() string {
								if t.mariadb() {
									return "(current_timestamp(6))"
								}
								return "CURRENT_TIMESTAMP(6)"
							}(),
						},
						Attrs: []schema.Attr{
							&mysql.OnUpdate{
								A: func() string {
									if t.mariadb() {
										return "current_timestamp(6)"
									}
									return "CURRENT_TIMESTAMP(6)"
								}(),
							},
						},
					},
					{
						Name: "tDate",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "date"},
							Raw: "date", Null: true},
					},
					{
						Name: "tTime",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "time"},
							Raw: "time", Null: true},
					},
					{
						Name: "tDateTime",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "datetime"},
							Raw: "datetime", Null: true},
					},
					{
						Name: "tYear",
						Type: &schema.ColumnType{
							Type: &schema.TimeType{
								T: "year",
								Precision: func() *int {
									// From MySQL 8.0.19, display width is deprecated in YEAR types.
									if t.version == "mysql8" {
										return nil
									}
									p := 4
									return &p
								}(),
							},
							Raw: t.valueByVersion(map[string]string{"mysql8": "year"}, "year(4)"), Null: true},
					},
					{
						Name: "tVarchar",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 10},
							Raw: "varchar(10)", Null: true},
						Default: &schema.Literal{V: t.quoted("Titan")},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tChar",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "char", Size: 25},
							Raw: "char(25)", Null: false},
						Default: &schema.Literal{V: t.quoted("Olimpia")},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tVarBinary",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "varbinary", Size: intp(30)},
							Raw: "varbinary(30)", Null: true},
						Default: &schema.Literal{V: t.valueByVersion(map[string]string{"mysql8": "0x546974616E"}, t.quoted("Titan"))},
					},
					{
						Name: "tBinary",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "binary", Size: intp(5)},
							Raw: "binary(5)", Null: true},
						Default: &schema.Literal{V: t.valueByVersion(map[string]string{"mysql8": "0x546974616E"}, t.quoted("Titan"))},
					},
					{
						Name: "tBlob",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "tinyblob"},
							Raw: "tinyblob", Null: true},
					},
					{
						Name: "tTinyBlob",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "tinyblob"},
							Raw: "tinyblob", Null: true},
					},
					{
						Name: "tMediumBlob",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "mediumblob"},
							Raw: "mediumblob", Null: true},
					},
					{
						Name: "tLongBlob",
						Type: &schema.ColumnType{Type: &schema.BinaryType{T: "longblob"},
							Raw: "longblob", Null: true},
					},
					{
						Name: "tText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "tinytext", Size: 0},
							Raw: "tinytext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tTinyText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "tinytext", Size: 0},
							Raw: "tinytext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tMediumText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "mediumtext", Size: 0},
							Raw: "mediumtext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tLongText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "longtext", Size: 0},
							Raw: "longtext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tEnum",
						Type: &schema.ColumnType{Type: &schema.EnumType{T: "enum", Values: []string{"a", "b"}},
							Raw: "enum('a','b')", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tSet",
						Type: &schema.ColumnType{Type: &mysql.SetType{Values: []string{"a", "b"}},
							Raw: "set('a','b')", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
						},
					},
					{
						Name: "tGeometry",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"},
							Raw: "geometry", Null: true},
					},
					{
						Name: "tPoint",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"},
							Raw: "point", Null: true},
					},
					{
						Name: "tMultiPoint",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "multipoint"},
							Raw: "multipoint", Null: true},
					},
					{
						Name: "tLineString",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "linestring"},
							Raw: "linestring", Null: true},
					},
					{
						Name: "tMultiLineString",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "multilinestring"},
							Raw: "multilinestring", Null: true},
					},
					{
						Name: "tPolygon",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "polygon"},
							Raw: "polygon", Null: true},
					},
					{
						Name: "tMultiPolygon",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: "multipolygon"},
							Raw: "multipolygon", Null: true},
					},
					{
						Name: "tGeometryCollection",
						Type: &schema.ColumnType{Type: &schema.SpatialType{T: t.valueByVersion(
							map[string]string{"mysql8": "geomcollection"}, "geometrycollection")},
							Raw: t.valueByVersion(map[string]string{"mysql8": "geomcollection"},
								"geometrycollection"), Null: true},
					},
				},
			}
			rmCreateStmt(ts)
			require.EqualValues(t, &expected, ts)
			t.hclDriftTest(n, realm, expected)
		})
	})
	t.Run("JSON", func(t *testing.T) {
		ddl := `
create table atlas_types_sanity
(
    tJSON         json          default                   null
) CHARSET = latin1 COLLATE latin1_swedish_ci;
`
		myRun(t, func(t *myTest) {
			if t.version == "mysql56" {
				return
			}
			t.dropTables(n)
			_, err := t.db.Exec(ddl)
			require.NoError(t, err)
			realm := t.loadRealm()
			require.Len(t, realm.Schemas, 1)
			ts, ok := realm.Schemas[0].Table(n)
			require.True(t, ok)
			expected := schema.Table{
				Name: n,
				Attrs: func() []schema.Attr {
					if t.version == "maria107" {
						return []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_swedish_ci"},
							&mysql.Engine{V: "InnoDB", Default: true},
							&schema.Check{Name: "tJSON", Expr: "json_valid(`tJSON`)"},
						}
					}
					return []schema.Attr{
						&schema.Charset{V: "latin1"},
						&schema.Collation{V: "latin1_swedish_ci"},
						&mysql.Engine{V: "InnoDB", Default: true},
					}
				}(),
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					func() *schema.Column {
						c := &schema.Column{Name: "tJSON", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}, Raw: "json", Null: true}}
						switch t.version {
						case "maria107":
							c.Attrs = []schema.Attr{}
						case "maria102", "maria103":
							c.Type.Raw = "longtext"
							c.Type.Type = &schema.StringType{T: "longtext"}
							c.Attrs = []schema.Attr{
								&schema.Charset{V: "utf8mb4"},
								&schema.Collation{V: "utf8mb4_bin"},
							}
						}
						return c
					}(),
				},
			}
			rmCreateStmt(ts)
			require.EqualValues(t, &expected, ts)
		})
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		myRun(t, func(t *myTest) {
			testImplicitIndexes(t, t.db)
		})
	})
}

func (t *myTest) url(dbname string) string {
	d := "mysql"
	pass := ":pass"
	if t.tidb() {
		pass = ""
	}
	if t.mariadb() {
		d = "mariadb"
	}
	return fmt.Sprintf("%s://root%s@localhost:%d/%s?parseTime=true", d, pass, t.port, dbname)
}

func (t *myTest) driver() migrate.Driver {
	return t.drv
}

func (t *myTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *myTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := mysql.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	require.NoError(t, err)
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *myTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := mysql.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *myTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *myTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *myTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *myTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		for _, n := range names {
			_, err := t.db.Exec("DROP DATABASE IF EXISTS " + n)
			require.NoError(t.T, err, "drop db %q", names)
		}
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

func (t *myTest) valueByVersion(values map[string]string, defaults string) string {
	if v, ok := values[t.version]; ok {
		return v
	}
	return defaults
}

func (t *myTest) intByVersion(values map[string]int, defaults int) int {
	if v, ok := values[t.version]; ok {
		return v
	}
	return defaults
}

func (t *myTest) quoted(s string) string {
	c := "\""
	if t.mariadb() {
		c = "'"
	}
	return c + s + c
}

func (t *myTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"test"},
		Mode:    ^schema.InspectViews,
	})
	require.NoError(t, err)
	return r
}

func (t *myTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *myTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *myTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *myTest) mariadb() bool { return strings.HasPrefix(t.version, "maria") }
func (t *myTest) tidb() bool    { return strings.HasPrefix(t.version, "tidb") }

// defaultConfig returns the default charset and
// collation configuration based on the MySQL version.
func (t *myTest) defaultAttrs() []schema.Attr {
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

func (t *myTest) hclDriftTest(n string, realm *schema.Realm, expected schema.Table) {
	spec, err := mysql.MarshalHCL(realm.Schemas[0])
	require.NoError(t, err)
	t.dropTables(n)
	t.applyHcl(string(spec))
	realm = t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	ts, ok := realm.Schemas[0].Table(n)
	require.True(t, ok)
	rmCreateStmt(ts)
	require.EqualValues(t, &expected, ts)
}

func rmCreateStmt(t *schema.Table) {
	for i := range t.Attrs {
		if _, ok := t.Attrs[i].(*mysql.CreateStmt); ok {
			t.Attrs = append(t.Attrs[:i], t.Attrs[i+1:]...)
			return
		}
	}
}

func intp(i int) *int { return &i }
