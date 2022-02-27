// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"database/sql"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func tidbRun(t *testing.T, fn func(*myTest)) {
	port := 4309
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/test?parseTime=True", port))
	require.NoError(t, err)
	drv, err := mysql.Open(db)
	require.NoError(t, err)
	t.Run("TiDB5", func(t *testing.T) {
		tt := &myTest{T: t, db: db, drv: drv, version: "TiDB5", port: port}
		fn(tt)
	})
}

func TestTiDB_AddDropTable(t *testing.T) {
	tidbRun(t, func(t *myTest) {
		testAddDrop(t)
	})
}

func TestTiDB_Relation(t *testing.T) {
	tidbRun(t, func(t *myTest) {
		testRelation(t)
	})
}

func TestTiDB_AddIndexedColumns(t *testing.T) {
	tidbRun(t, func(t *myTest) {
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
		// However, on TiDB an explicit DROP/ADD INDEX is required.
		idx, ok := usersT.Index("a_b_c_unique")
		require.True(t, ok)
		idx.Parts = idx.Parts[:len(idx.Parts)-1]
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		changes = t.diff(t.loadUsers(), usersT)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())

		// Dropping a column from both table and index.
		usersT = t.loadUsers()
		idx, ok = usersT.Index("a_b_c_unique")
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
		// In MySQL, dropping a column should remove its index.
		// However, on TiDB an explicit DROP INDEX is required.
		usersT.Indexes = nil
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 2)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())
		_, ok = t.loadUsers().Index("a_b_c_unique")
		require.False(t, ok)
	})
}

func TestTiDB_AddColumns(t *testing.T) {
	tidbRun(t, func(t *myTest) {
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
			&schema.Column{Name: "z", Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 27)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestTiDB_ColumnInt(t *testing.T) {
	t.Run("ChangeType", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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

func TestTiDB_ColumnString(t *testing.T) {
	t.Run("ChangeType", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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

func TestTiDB_ColumnBool(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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

func TestTiDB_ForeignKey(t *testing.T) {
	t.Run("ChangeAction", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
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
