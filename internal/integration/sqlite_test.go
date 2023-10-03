// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

type liteTest struct {
	*testing.T
	db   *sql.DB
	drv  migrate.Driver
	rrw  migrate.RevisionReadWriter
	file string
}

func liteRun(t *testing.T, fn func(test *liteTest)) {
	t.Parallel()
	f := path.Join(t.TempDir(), strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", f))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	tt := &liteTest{T: t, db: db, drv: drv, file: f, rrw: &rrw{}}
	fn(tt)
}

func TestSQLite_Executor(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		testExecutor(t)
	})
}

func TestSQLite_AddDropTable(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		testAddDrop(t)
	})
}

func TestSQLite_Relation(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		testRelation(t)
	})
}

func TestSQLite_ColumnCheck(t *testing.T) {
	liteRun(t, func(t *liteTest) {
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

func TestSQLite_AddIndexedColumns(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		usersT := &schema.Table{
			Name:    "users",
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
		}
		t.migrate(&schema.AddTable{T: usersT})
		t.dropTables(usersT.Name)

		// Insert 2 records to the users table, and make sure they are there
		// after executing migration.
		_, err := t.db.Exec("INSERT INTO users (id) VALUES (1), (2)")
		require.NoError(t, err)

		usersT.Columns = append(usersT.Columns, &schema.Column{
			Name:    "a",
			Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
			Default: &schema.Literal{V: "10"},
		}, &schema.Column{
			Name:    "b",
			Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
			Default: &schema.Literal{V: "20"},
		}, &schema.Column{
			Name:    "c",
			Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
			Default: &schema.Literal{V: "30"},
		})
		usersT.Indexes = append(usersT.Indexes, &schema.Index{
			Unique: true,
			Name:   "id_a_b_c_unique",
			Parts:  []*schema.IndexPart{{C: usersT.Columns[0]}, {C: usersT.Columns[1]}, {C: usersT.Columns[2]}, {C: usersT.Columns[3]}},
		})
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 4, "usersT contains 3 new columns and 1 new index")
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)

		// Scan records from the table to ensure correctness of
		// the rows transferring.
		rows, err := t.db.Query("SELECT * FROM users")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var v [4]int
		require.NoError(t, rows.Scan(&v[0], &v[1], &v[2], &v[3]))
		require.Equal(t, [4]int{1, 10, 20, 30}, v)
		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&v[0], &v[1], &v[2], &v[3]))
		require.Equal(t, [4]int{2, 10, 20, 30}, v)
		require.False(t, rows.Next())
		require.NoError(t, rows.Close())

		// Dropping a column from both table and index.
		usersT = t.loadUsers()
		idx, ok := usersT.Index("id_a_b_c_unique")
		require.True(t, ok)
		require.Len(t, idx.Parts, 4)
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		idx.Parts = idx.Parts[:len(idx.Parts)-1]
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 2)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())

		// Scan records from the table to ensure correctness of
		// the rows transferring.
		rows, err = t.db.Query("SELECT * FROM users")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var u [3]int
		require.NoError(t, rows.Scan(&u[0], &u[1], &u[2]))
		require.Equal(t, [3]int{1, 10, 20}, u)
		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&u[0], &u[1], &u[2]))
		require.Equal(t, [3]int{2, 10, 20}, u)
		require.False(t, rows.Next())
		require.NoError(t, rows.Close())

	})
}

func TestSQLite_AutoIncrement(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		usersT := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{sqlite.AutoIncrement{}}},
			},
		}
		usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
		t.migrate(&schema.AddTable{T: usersT})
		t.dropTables(usersT.Name)
		_, err := t.db.Exec("INSERT INTO users DEFAULT VALUES")
		require.NoError(t, err)
		var id int
		err = t.db.QueryRow("SELECT id FROM users").Scan(&id)
		require.NoError(t, err)
		require.Equal(t, 1, id)
	})
}

func TestSQLite_AutoIncrementSequence(t *testing.T) {
	// This test shows a bug detected in Ent when working with pre-defined auto-increment start values.
	// If there is a change somewhere to create an auto-increment with a start value, Atlas must make sure to create
	// an entry in the 'sqlite_sequence' table (and also ensure the table exists before attempting to create the entry).
	liteRun(t, func(t *liteTest) {
		t1 := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{
					Name:  "id",
					Type:  &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}},
					Attrs: []schema.Attr{&sqlite.AutoIncrement{Seq: 10}},
				},
			},
			Attrs: []schema.Attr{&sqlite.AutoIncrement{}},
		}
		t1.PrimaryKey = &schema.Index{Table: t1, Parts: []*schema.IndexPart{{C: t1.Columns[0]}}}
		t1.Columns[0].Indexes = append(t1.Columns[0].Indexes, t1.PrimaryKey)

		// Planning the changes should not result in an error.
		_ = plan(t, "col_seq", &schema.AddTable{T: t1})
	})
}

func TestSQLite_AddColumns(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		usersT := &schema.Table{
			Name: "users",
			Columns: []*schema.Column{
				{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{sqlite.AutoIncrement{}}},
			},
		}
		usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
		t.migrate(&schema.AddTable{T: usersT})
		t.dropTables(usersT.Name)
		_, err := t.db.Exec("INSERT INTO users (id) VALUES (1), (2)")
		require.NoError(t, err)
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "null_int", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true}},
			&schema.Column{Name: "notnull_int", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Default: &schema.Literal{V: "1"}},
			&schema.Column{Name: "null_real", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real"}, Null: true}},
			&schema.Column{Name: "notnull_real", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real"}}, Default: &schema.Literal{V: "1.0"}},
			&schema.Column{Name: "null_text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
			&schema.Column{Name: "notnull_text1", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Default: &schema.Literal{V: "hello"}},
			&schema.Column{Name: "notnull_text2", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Default: &schema.Literal{V: "'hello'"}},
			&schema.Column{Name: "null_blob", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}, Null: true}},
			&schema.Column{Name: "notnull_blob", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}}, Default: &schema.Literal{V: "'blob'"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 9)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)

		// Scan records from the table to ensure correctness of
		// the rows transferring.
		rows, err := t.db.Query("SELECT id, notnull_int FROM users")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var v [2]int
		require.NoError(t, rows.Scan(&v[0], &v[1]))
		require.Equal(t, [2]int{1, 1}, v)
		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&v[0], &v[1]))
		require.Equal(t, [2]int{2, 1}, v)
		require.False(t, rows.Next())
		require.NoError(t, rows.Close())
	})
}

func TestSQLite_ColumnInt(t *testing.T) {
	t.Run("ChangeTypeNull", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			usersT.Columns[0].Type.Null = true
			usersT.Columns[0].Type.Type = &schema.FloatType{T: "real"}
			changes := t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 1)
			require.Equal(t, schema.ChangeNull|schema.ChangeType, changes[0].(*schema.ModifyColumn).Change)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Default: &schema.Literal{V: "1"}}},
			}
			t.migrate(&schema.AddTable{T: usersT})
			t.dropTables(usersT.Name)
			ensureNoChange(t, usersT)
			for _, x := range []string{"2", "'3'", "10.1"} {
				usersT.Columns[0].Default.(*schema.Literal).V = x
				changes := t.diff(t.loadUsers(), usersT)
				require.Len(t, changes, 1)
				t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
				ensureNoChange(t, usersT)
				_, err := t.db.Exec("INSERT INTO users DEFAULT VALUES")
				require.NoError(t, err)
			}

			rows, err := t.db.Query("SELECT a FROM users")
			require.NoError(t, err)
			for _, e := range []driver.Value{2, 3, 10.1} {
				var v driver.Value
				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&v))
				require.EqualValues(t, e, v)
			}
			require.False(t, rows.Next())
			require.NoError(t, rows.Close())
		})
	})
}

func TestSQLite_ForeignKey(t *testing.T) {
	t.Run("ChangeAction", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			usersT, postsT := t.users(), t.posts()
			t.dropTables(postsT.Name, usersT.Name)
			t.migrate(&schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
			ensureNoChange(t, postsT, usersT)

			postsT = t.loadPosts()
			// The "author_id" constraint. SQLite does not support
			// getting the foreign-key constraint names at the moment.
			fk := postsT.ForeignKeys[0]
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
		liteRun(t, func(t *liteTest) {
			usersT, postsT := t.users(), t.posts()
			t.dropTables(postsT.Name, usersT.Name)
			fk := postsT.ForeignKeys[0]
			fk.OnDelete = schema.SetNull
			fk.OnUpdate = schema.SetNull
			t.migrate(&schema.AddTable{T: usersT}, &schema.AddTable{T: postsT})
			ensureNoChange(t, postsT, usersT)

			postsT = t.loadPosts()
			c, ok := postsT.Column("author_id")
			require.True(t, ok)
			c.Type.Null = false
			fk = postsT.ForeignKeys[0]
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
		liteRun(t, func(t *liteTest) {
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

func TestSQLite_HCL(t *testing.T) {
	full := `
schema "main" {
}
table "users" {
	schema = schema.main
	column "id" {
		type = int
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
table "posts" {
	schema = schema.main
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
schema "main" {
}
`
	liteRun(t, func(t *liteTest) {
		testHCLIntegration(t, full, empty)
	})
}

func TestSQLite_DefaultsHCL(t *testing.T) {
	n := "atlas_defaults"
	liteRun(t, func(t *liteTest) {
		ddl := `
create table atlas_defaults
(
	string varchar(255) default "hello_world",
	quoted varchar(100) default 'never say "never"',
	d date default current_timestamp,
	n integer default 0x100 
)
`
		t.dropTables(n)
		_, err := t.db.Exec(ddl)
		require.NoError(t, err)
		realm := t.loadRealm()
		spec, err := sqlite.MarshalHCL(realm.Schemas[0])
		require.NoError(t, err)
		var s schema.Schema
		err = sqlite.EvalHCLBytes(spec, &s, nil)
		require.NoError(t, err)
		t.dropTables(n)
		t.applyHcl(string(spec))
		ensureNoChange(t, realm.Schemas[0].Tables[0])
	})
}

func TestSQLite_CLI(t *testing.T) {
	h := `
			schema "main" {
			}
			table "users" {
				schema = schema.main
				column "id" {
					type = int
				}
			}`
	t.Run("InspectFromEnv", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			env := fmt.Sprintf(`
env "hello" {
	url = "%s"
	src = "./schema.hcl"
}
`, t.url(""))
			wd, _ := os.Getwd()
			envfile := filepath.Join(wd, "atlas.hcl")
			err := os.WriteFile(envfile, []byte(env), 0600)
			t.Cleanup(func() {
				os.Remove(envfile)
			})
			require.NoError(t, err)

			testCLISchemaInspectEnv(t, h, "hello", sqlite.EvalHCL)
		})
	})
	t.Run("SchemaInspect", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testCLISchemaInspect(t, h, t.url(""), sqlite.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testCLISchemaApply(t, h, t.url(""))
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
		liteRun(t, func(t *liteTest) {
			testCLISchemaApply(t, h, t.url(""), "--var", "tenant=main")
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testCLISchemaApplyDry(t, h, t.url(""))
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testCLISchemaDiff(t, t.url(""))
		})
	})
	t.Run("SchemaApplyAutoApprove", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testCLISchemaApplyAutoApprove(t, h, t.url(""))
		})
	})
}

func TestSQLite_Sanity(t *testing.T) {
	n := "atlas_types_sanity"
	ddl := `
create table atlas_types_sanity
(
    "tInteger"            integer(10)                     default 100                                   null,
    "tInt"                int(10)                         default 100                                   null,
    "tTinyIny"            tinyint(10)                     default 100                                   null,
    "tSmallInt"           smallint(10)                    default 100                                   null,
    "tMediumInt"          mediumint(10)                   default 100                                   null,
    "tIntegerBigInt"      bigint(10)                      default 100                                   null,
    "tUnsignedBigInt"     unsigned big int(10)            default 100                                   null,
    "tInt2"               int2(10)                        default 100                                   null,
    "tInt8"               int8(10)                        default 100                                   null,
    "tReal"               real(10)                        default 100                                   null,
    "tDouble"             double(10)                      default 100                                   null,
    "tDoublePrecision"    double precision(10)            default 100                                   null,
    "tFloat"              float(10)                       default 100                                   null,
    "tText"               text(10)                        default 'I am Text'                       not null,
    "tCharacter"          character(10)                   default 'I am Text'                       not null,
    "tVarchar"            varchar(10)                     default 'I am Text'                       not null,
    "tVaryingCharacter"   varying character(10)           default 'I am Text'                       not null,
    "tNchar"              nchar(10)                       default 'I am Text'                       not null,
    "tNativeCharacter"    native character(10)            default 'I am Text'                       not null,
    "tNVarChar"           nvarchar(10)                    default 'I am Text'                       not null,
    "tClob"               clob(10)                        default 'I am Text'                       not null,
    "tBlob"               blob(10)                        default 'A'                               not null,
    "tNumeric"            numeric(10)                     default 100                               not null,
    "tDecimal"            decimal(10,5)                   default 100                               not null,
    "tBoolean"            boolean                         default false                             not null,
    "tDate"               date                            default 'now()'                           not null ,
    "tDatetime"           datetime                        default 'now()'                           not null 
);
`
	liteRun(t, func(t *liteTest) {
		t.dropTables(n)
		_, err := t.db.Exec(ddl)
		require.NoError(t, err)
		realm := t.loadRealm()
		require.Len(t, realm.Schemas, 1)
		ts, ok := realm.Schemas[0].Table(n)
		require.True(t, ok)
		expected := schema.Table{
			Name:   n,
			Schema: realm.Schemas[0],
			Attrs:  ts.Attrs,
			Columns: []*schema.Column{
				{
					Name: "tInteger",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer", Unsigned: false}, Raw: "integer(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tInt",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int", Unsigned: false}, Raw: "int(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tTinyIny",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "tinyint", Unsigned: false}, Raw: "tinyint(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tSmallInt",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint", Unsigned: false}, Raw: "smallint(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tMediumInt",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "mediumint", Unsigned: false}, Raw: "mediumint(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tIntegerBigInt",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tUnsignedBigInt",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "unsigned big int", Unsigned: false}, Raw: "unsigned big int(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tInt2",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int2", Unsigned: false}, Raw: "int2(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tInt8",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int8", Unsigned: false}, Raw: "int8(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tReal",
					Type: &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 0}, Raw: "real(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tDouble",
					Type: &schema.ColumnType{Type: &schema.FloatType{T: "double", Precision: 0}, Raw: "double(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tDoublePrecision",
					Type: &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 0}, Raw: "double precision(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tFloat",
					Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 0}, Raw: "float(10)", Null: true},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tText",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "text", Size: 10}, Raw: "text(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tCharacter",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tVarchar",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "varchar", Size: 10}, Raw: "varchar(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tVaryingCharacter",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "varying character", Size: 10}, Raw: "varying character(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tNchar",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "nchar", Size: 10}, Raw: "nchar(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tNativeCharacter",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "native character", Size: 10}, Raw: "native character(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tNVarChar",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "nvarchar", Size: 10}, Raw: "nvarchar(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tClob",
					Type: &schema.ColumnType{Type: &schema.StringType{T: "clob", Size: 10}, Raw: "clob(10)", Null: false},
					Default: &schema.Literal{
						V: "'I am Text'",
					},
				},
				{
					Name: "tBlob",
					Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}, Raw: "blob(10)", Null: false},
					Default: &schema.Literal{
						V: "'A'",
					},
				},
				{
					Name: "tNumeric",
					Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10}, Raw: "numeric(10)", Null: false},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tDecimal",
					Type: &schema.ColumnType{Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 5}, Raw: "decimal(10,5)", Null: false},
					Default: &schema.Literal{
						V: "100",
					},
				},
				{
					Name: "tBoolean",
					Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Raw: "boolean", Null: false},
					Default: &schema.Literal{
						V: "false",
					},
				},
				{
					Name: "tDate",
					Type: &schema.ColumnType{Type: &schema.TimeType{T: "date"}, Raw: "date", Null: false},
					Default: &schema.Literal{
						V: "'now()'",
					},
				},
				{
					Name: "tDatetime",
					Type: &schema.ColumnType{Type: &schema.TimeType{T: "datetime"}, Raw: "datetime", Null: false},
					Default: &schema.Literal{
						V: "'now()'",
					},
				},
			},
		}
		require.EqualValues(t, &expected, ts)
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		liteRun(t, func(t *liteTest) {
			testImplicitIndexes(t, t.db)
		})
	})
}

func (t *liteTest) driver() migrate.Driver {
	return t.drv
}

func (t *liteTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *liteTest) dropSchemas(...string) {}

func (t *liteTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := sqlite.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *liteTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"main"},
		Mode:    schema.InspectSchemas | schema.InspectTables,
	})
	require.NoError(t, err)
	return r
}

func (t *liteTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *liteTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *liteTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *liteTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name: "id",
				Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
			},
			{
				Name: "x",
				Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}},
			},
		},
	}
	usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
	return usersT
}

func (t *liteTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
			{
				Name:    "author_id",
				Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
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

func (t *liteTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "main",
				Attrs: []schema.Attr{
					&sqlite.File{Name: t.file},
				},
			},
		},
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *liteTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *liteTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *liteTest) dropTables(names ...string) {
	t.Cleanup(func() {
		for i := range names {
			_, err := t.db.Exec("DROP TABLE IF EXISTS " + names[i])
			require.NoError(t.T, err, "drop tables %q", names[i])
		}
	})
}

func (t *liteTest) url(_ string) string {
	return fmt.Sprintf("sqlite://file:%s?cache=shared&_fk=1", t.file)
}

func (t *liteTest) applyRealmHcl(spec string) {
	t.applyHcl(spec)
}
