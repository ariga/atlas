// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

var tidbTests = map[string]*myTest{
	"tidb5": {port: 4309},
	"tidb6": {port: 4310},
}

func tidbRun(t *testing.T, fn func(*myTest)) {
	for version, tt := range tidbTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/test?parseTime=True", tt.port))
					if err != nil {
						log.Fatalln(err)
					}
					dbs = append(dbs, tt.db) // close connection after all tests have been run
					tt.drv, err = mysql.Open(tt.db)
					if err != nil {
						log.Fatalln(err)
					}
				})
				tt := &myTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
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

func TestTiDB_HCL_Realm(t *testing.T) {
	tidbRun(t, func(t *myTest) {
		t.dropSchemas("second")
		realm := t.loadRealm()
		hcl, err := mysql.MarshalHCL(realm)
		require.NoError(t, err)
		wa := string(hcl) + `
schema "second" {
}
`
		t.applyRealmHcl(wa)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{})
		require.NoError(t, err)
		_, ok := realm.Schema("test")
		require.True(t, ok)
		_, ok = realm.Schema("second")
		require.True(t, ok)
	})
}

func TestTiDB_DefaultsHCL(t *testing.T) {
	n := "atlas_defaults"
	tidbRun(t, func(t *myTest) {
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

func TestTiDB_CLI_MultiSchema(t *testing.T) {
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
		tidbRun(t, func(t *myTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLIMultiSchemaInspect(t, fmt.Sprintf(h, charset.V, collate.V, charset.V, collate.V), t.url(""), []string{"test", "test2"}, mysql.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLIMultiSchemaApply(t, fmt.Sprintf(h, charset.V, collate.V, charset.V, collate.V), t.url(""), []string{"test", "test2"}, mysql.EvalHCL)
		})
	})
}

func TestTiDB_CLI(t *testing.T) {
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
		tidbRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaInspect(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"), mysql.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
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
		tidbRun(t, func(t *myTest) {
			testCLISchemaApply(t, h, t.url("test"), "--var", "tenant=test")
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
			attrs := t.defaultAttrs()
			charset, collate := attrs[0].(*schema.Charset), attrs[1].(*schema.Collation)
			testCLISchemaApplyDry(t, fmt.Sprintf(h, charset.V, collate.V), t.url("test"))
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
			testCLISchemaDiff(t, t.url("test"))
		})
	})
}

func TestTiDB_HCL(t *testing.T) {
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
	tidbRun(t, func(t *myTest) {
		testHCLIntegration(t, full, empty)
	})
}

func TestTiDB_Sanity(t *testing.T) {
	n := "atlas_types_sanity"
	t.Run("Common", func(t *testing.T) {
		ddl := `
create table atlas_types_sanity
(
    tBit                        bit(10)              default b'1000000001'                                       null,
		tInt                        int(10)              default 4                                               not null,
		tTinyInt                    tinyint(10)          default 8                                                   null,
		tSmallInt                   smallint(10)         default 2                                                   null,
		tMediumInt                  mediumint(10)        default 11                                                  null,
		tBigInt                     bigint(10)           default 4                                                   null,
		tDecimal                    decimal              default 4                                                   null,
		tNumeric                    numeric              default 4                                               not null,
		tFloat                      float         			 default 4                                                   null,
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
		tSet                        set('a','b')         default                                                     null
) CHARSET = latin1;
`
		tidbRun(t, func(t *myTest) {
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
					&schema.Collation{
						V: "latin1_bin",
					},
				},
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					{
						Name:    "tBit",
						Type:    &schema.ColumnType{Type: &mysql.BitType{T: "bit", Size: 10}, Raw: "bit(10) unsigned", Null: true},
						Default: &schema.Literal{V: "b'1000000001'"},
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
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "float"},
							Raw: "float", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tDouble",
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "double"},
							Raw: "double", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tReal",
						Type: &schema.ColumnType{Type: &schema.FloatType{T: "double"},
							Raw: "double", Null: true},
						Default: &schema.Literal{V: "4"},
					},
					{
						Name: "tTimestamp",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"},
							Raw: "timestamp", Null: true},
						Default: &schema.RawExpr{
							X: "CURRENT_TIMESTAMP",
						},
					},
					{
						Name: "tTimestampFraction",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp", Precision: intp(6)},
							Raw: "timestamp(6)", Null: true},
						Default: &schema.RawExpr{
							X: "CURRENT_TIMESTAMP(6)",
						},
					},
					{
						Name: "tTimestampOnUpdate",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp"},
							Raw: "timestamp", Null: true},
						Default: &schema.RawExpr{
							X: "CURRENT_TIMESTAMP",
						},
						Attrs: []schema.Attr{
							&mysql.OnUpdate{
								A: "CURRENT_TIMESTAMP",
							},
						},
					},
					{
						Name: "tTimestampFractionOnUpdate",
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp", Precision: intp(6)},
							Raw: "timestamp(6)", Null: true},
						Default: &schema.RawExpr{
							X: "CURRENT_TIMESTAMP(6)",
						},
						Attrs: []schema.Attr{
							&mysql.OnUpdate{
								A: "CURRENT_TIMESTAMP(6)",
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
						Type: &schema.ColumnType{Type: &schema.TimeType{T: "year", Precision: intp(t.intByVersion(map[string]int{"mysql8": 0}, 4))},
							Raw: t.valueByVersion(map[string]string{"mysql8": "year"}, "year(4) unsigned"), Null: true},
					},
					{
						Name: "tVarchar",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 10},
							Raw: "varchar(10)", Null: true},
						Default: &schema.Literal{V: t.quoted("Titan")},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tChar",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "char", Size: 25},
							Raw: "char(25)", Null: false},
						Default: &schema.Literal{V: t.quoted("Olimpia")},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
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
						Type: &schema.ColumnType{Type: &schema.StringType{T: "tinytext"},
							Raw: "tinytext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tTinyText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "tinytext"},
							Raw: "tinytext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tMediumText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "mediumtext", Size: 0},
							Raw: "mediumtext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tLongText",
						Type: &schema.ColumnType{Type: &schema.StringType{T: "longtext", Size: 0},
							Raw: "longtext", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tEnum",
						Type: &schema.ColumnType{Type: &schema.EnumType{T: "enum", Values: []string{"a", "b"}},
							Raw: "enum('a','b')", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
					},
					{
						Name: "tSet",
						Type: &schema.ColumnType{Type: &mysql.SetType{Values: []string{"a", "b"}},
							Raw: "set('a','b')", Null: true},
						Attrs: []schema.Attr{
							&schema.Charset{V: "latin1"},
							&schema.Collation{V: "latin1_bin"},
						},
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
	) CHARSET = latin1;
	`
		tidbRun(t, func(t *myTest) {
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
					return []schema.Attr{
						&schema.Charset{V: "latin1"},
						&schema.Collation{V: "latin1_bin"},
					}
				}(),
				Schema: realm.Schemas[0],
				Columns: []*schema.Column{
					{Name: "tJSON", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}, Raw: "json", Null: true}},
				},
			}
			rmCreateStmt(ts)
			require.EqualValues(t, &expected, ts)
		})
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		tidbRun(t, func(t *myTest) {
			testImplicitIndexes(t, t.db)
		})
	})

	t.Run("AltersOrder", func(t *testing.T) {
		ddl := `
		create table tidb_alter_order(
			tBigInt bigint(10) default 4 null,
			INDEX   i  (tBigInt)
		);
	`
		tidbRun(t, func(t *myTest) {
			t.dropTables("tidb_alter_order")
			_, err := t.db.Exec(ddl)
			require.NoError(t, err)
			tbl := t.loadTable("tidb_alter_order")
			require.NotNil(t, tbl)
			to := schema.Table{
				Name: "tidb_alter_order",
				Attrs: func() []schema.Attr {
					return []schema.Attr{
						&schema.Collation{V: "utf8mb4_bin"},
						&schema.Charset{V: "utf8mb4"},
					}
				}(),
				Columns: []*schema.Column{
					{
						Name: "tBigInt2",
						Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false},
							Raw: t.valueByVersion(map[string]string{"mysql8": "bigint"}, "bigint(10)"), Null: true},
						Default: &schema.Literal{V: "4"},
					},
				},
			}
			to.AddIndexes(
				&schema.Index{Name: "i2", Parts: []*schema.IndexPart{
					{
						C:    to.Columns[0],
						Desc: true,
					},
				}})
			changes, err := t.drv.SchemaDiff(schema.New("test").AddTables(tbl), schema.New("test").AddTables(&to))
			require.NoError(t, err)
			err = t.drv.ApplyChanges(context.Background(), changes)
			require.NoError(t, err)
			t.migrate()
			rmCreateStmt(tbl)
		})
	})
}
