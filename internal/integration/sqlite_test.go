// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

type liteTest struct {
	*testing.T
	db  *sql.DB
	drv *sqlite.Driver
}

func liteRun(t *testing.T, fn func(test *liteTest)) {
	db, err := sql.Open("sqlite3", "file:atlas?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	tt := &liteTest{T: t, db: db, drv: drv}
	fn(tt)
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

func TestSQLite_Ent(t *testing.T) {
	liteRun(t, func(t *liteTest) {
		testEntIntegration(t, dialect.SQLite, t.db)
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
			Default: &schema.RawExpr{X: "10"},
		}, &schema.Column{
			Name:    "b",
			Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
			Default: &schema.RawExpr{X: "20"},
		}, &schema.Column{
			Name:    "c",
			Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Null: true},
			Default: &schema.RawExpr{X: "30"},
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
			&schema.Column{Name: "notnull_int", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}}, Default: &schema.RawExpr{X: "1"}},
			&schema.Column{Name: "null_real", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real"}, Null: true}},
			&schema.Column{Name: "notnull_real", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real"}}, Default: &schema.RawExpr{X: "1.0"}},
			&schema.Column{Name: "null_text", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Null: true}},
			&schema.Column{Name: "notnull_text1", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Default: &schema.RawExpr{X: "hello"}},
			&schema.Column{Name: "notnull_text2", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}}, Default: &schema.RawExpr{X: "'hello'"}},
			&schema.Column{Name: "null_blob", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}, Null: true}},
			&schema.Column{Name: "notnull_blob", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}}, Default: &schema.RawExpr{X: "'blob'"}},
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
schema "public" {
}
table "users" {
	schema = schema.public
	column "id" {
		type = "int"
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
table "posts" {
	schema = schema.public
	column "id" {
		type = "int"
	}
	column "author_id" {
		type = "int"
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
schema "public" {
}
`
	liteRun(t, func(t *liteTest) {
		testHCLIntegration(t, full, empty)
	})
}

func (t *liteTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := sqlite.UnmarshalSpec([]byte(spec), schemahcl.Unmarshal, &desired)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.Diff().SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.Migrate().Exec(context.Background(), diff)
	require.NoError(t, err)
}

func (t *liteTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"main"},
	})
	require.NoError(t, err)
	return r
}

func (t *liteTest) loadUsers() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	users, ok := realm.Schemas[0].Table("users")
	require.True(t, ok)
	return users
}

func (t *liteTest) loadPosts() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	posts, ok := realm.Schemas[0].Table("posts")
	require.True(t, ok)
	return posts
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
					&sqlite.File{Name: ":memory:"},
				},
			},
		},
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *liteTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.Diff().TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *liteTest) migrate(changes ...schema.Change) {
	err := t.drv.Migrate().Exec(context.Background(), changes)
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
