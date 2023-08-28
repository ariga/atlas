// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type pgTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
	once    sync.Once
}

var pgTests = map[string]*pgTest{
	"postgres-ext-postgis": {port: 5429},
	"postgres10":           {port: 5430},
	"postgres11":           {port: 5431},
	"postgres12":           {port: 5432},
	"postgres13":           {port: 5433},
	"postgres14":           {port: 5434},
	"postgres15":           {port: 5435},
}

func pgRun(t *testing.T, fn func(*pgTest)) {
	for version, tt := range pgTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=postgres dbname=test password=pass sslmode=disable", tt.port))
					if err != nil {
						log.Fatalln(err)
					}
					dbs = append(dbs, tt.db) // close connection after all tests have been run
					// the postgis/postgis image enables the postgis_topology and postgis_tiger_geocoder extensions,
					// this creates a few unwanted schemas, so we drop it.
					// https://github.com/postgis/docker-postgis/issues/187
					if tt.version == "postgres-ext-postgis" {
						schemasToDrop := []string{
							"tiger",      // created by postgis_tiger_geocoder
							"tiger_data", // created by postgis_tiger_geocoder
							"topology",   // created by postgis_topology
						}
						for _, s := range schemasToDrop {
							if _, err := tt.db.Exec("DROP SCHEMA IF EXISTS " + s + " CASCADE;"); err != nil {
								log.Fatalf("error dropping schema %q: %v", s, err)
							}
						}
					}
					tt.drv, err = postgres.Open(tt.db)
					if err != nil {
						log.Fatalln(err)
					}
				})
				tt := &pgTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
}

func TestPostgres_Executor(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testExecutor(t)
	})
}

func TestPostgres_AddDropTable(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testAddDrop(t)
	})
}

func TestPostgres_Relation(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testRelation(t)
	})
}

func TestPostgres_NoSchema(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		t.Cleanup(func() {
			_, err := t.db.Exec("CREATE SCHEMA IF NOT EXISTS public")
			require.NoError(t, err)
		})
		_, err := t.db.Exec("DROP SCHEMA IF EXISTS public CASCADE")
		require.NoError(t, err)
		r, err := t.drv.InspectRealm(context.Background(), nil)
		require.NoError(t, err)
		require.Nil(t, r.Schemas)
	})
}

func TestPostgres_AddIndexedColumns(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		usersT := &schema.Table{
			Name:    "users",
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
		}
		t.migrate(&schema.AddTable{T: usersT})
		t.dropTables(usersT.Name)
		usersT.Columns = append(usersT.Columns, &schema.Column{
			Name:    "a",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.Literal{V: "10"},
		}, &schema.Column{
			Name:    "b",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.Literal{V: "10"},
		}, &schema.Column{
			Name:    "c",
			Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
			Default: &schema.Literal{V: "10"},
		})
		parts := usersT.Columns[len(usersT.Columns)-3:]
		usersT.Indexes = append(usersT.Indexes, &schema.Index{
			Unique: true,
			Name:   "a_b_c_unique",
			Parts:  []*schema.IndexPart{{C: parts[0]}, {C: parts[1]}, {C: parts[2]}},
		})
		changes := t.diff(t.loadUsers(), usersT)
		require.NotEmpty(t, changes, "usersT contains 3 new columns and 1 new index")
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)

		// Dropping a column involves in a multi-column
		// index causes the index to be dropped as well.
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		changes = t.diff(t.loadUsers(), usersT)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, t.loadUsers())
		usersT = t.loadUsers()
		_, ok := usersT.Index("a_b_c_unique")
		require.False(t, ok)
	})
}

func TestPostgres_ColumnCheck(t *testing.T) {
	pgRun(t, func(t *pgTest) {
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

func TestPostgres_AddColumns(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})
		_, err := t.db.Exec("CREATE EXTENSION IF NOT EXISTS hstore")
		require.NoError(t, err)
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "bytea"}}},
			&schema.Column{Name: "b", Type: &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 10}}, Default: &schema.Literal{V: "10.1"}},
			&schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.StringType{T: "character"}}, Default: &schema.Literal{V: "'y'"}},
			&schema.Column{Name: "d", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}}, Default: &schema.Literal{V: "0.99"}},
			&schema.Column{Name: "e", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}}, Default: &schema.Literal{V: "'{}'"}},
			&schema.Column{Name: "f", Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}}, Default: &schema.Literal{V: "'1'"}},
			&schema.Column{Name: "g", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 10}}, Default: &schema.Literal{V: "'1'"}},
			&schema.Column{Name: "h", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 30}}, Default: &schema.Literal{V: "'1'"}},
			&schema.Column{Name: "i", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 53}}, Default: &schema.Literal{V: "1"}},
			&schema.Column{Name: "j", Type: &schema.ColumnType{Type: &postgres.SerialType{T: "serial"}}},
			&schema.Column{Name: "k", Type: &schema.ColumnType{Type: &postgres.CurrencyType{T: "money"}}, Default: &schema.Literal{V: "'100'"}},
			&schema.Column{Name: "l", Type: &schema.ColumnType{Type: &postgres.CurrencyType{T: "money"}, Null: true}, Default: &schema.RawExpr{X: "'52093.89'::money"}},
			&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Null: true}, Default: &schema.Literal{V: "false"}},
			&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"}, Null: true}, Default: &schema.Literal{V: "'(1,2)'"}},
			&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "line"}, Null: true}, Default: &schema.Literal{V: "'{1,2,3}'"}},
			&schema.Column{Name: "p", Type: &schema.ColumnType{Type: &postgres.UserDefinedType{T: "hstore"}, Null: true}, Default: &schema.RawExpr{X: "'a => 1'"}},
			&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &postgres.ArrayType{Type: &schema.StringType{T: "text"}, T: "text[]"}, Null: true}, Default: &schema.Literal{V: "'{}'"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 17)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestPostgres_ColumnInt(t *testing.T) {
	ctx := context.Background()
	run := func(t *testing.T, change func(*schema.Column)) {
		pgRun(t, func(t *pgTest) {
			usersT := &schema.Table{
				Name:    "users",
				Columns: []*schema.Column{{Name: "a", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}}},
			}
			err := t.drv.ApplyChanges(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			change(usersT.Columns[0])
			changes := t.diff(t.loadUsers(), usersT)
			require.Len(t, changes, 1)
			t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
			ensureNoChange(t, usersT)
		})
	}

	t.Run("ChangeNull", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Type.Null = true
		})
	})

	t.Run("ChangeType", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Type.Type.(*schema.IntegerType).T = "integer"
		})
	})

	t.Run("ChangeDefault", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Default = &schema.RawExpr{X: "0"}
		})
	})
}

func TestPostgres_ColumnArray(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})

		// Add column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "int[]", Type: &postgres.ArrayType{Type: &schema.IntegerType{T: "int"}, T: "int[]"}}, Default: &schema.Literal{V: "'{1}'"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)

		// Check default.
		usersT.Columns[2].Default = &schema.RawExpr{X: "ARRAY[1]"}
		ensureNoChange(t, usersT)

		// Change default.
		usersT.Columns[2].Default = &schema.RawExpr{X: "ARRAY[1,2]"}
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestPostgres_Enums(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		usersT := &schema.Table{
			Name:   "users",
			Schema: t.realm().Schemas[0],
			Columns: []*schema.Column{
				{Name: "state", Type: &schema.ColumnType{Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}}}},
			},
		}
		t.Cleanup(func() {
			_, err := t.drv.ExecContext(ctx, "DROP TYPE IF EXISTS state, day")
			require.NoError(t, err)
		})

		// Create table with an enum column.
		err := t.drv.ApplyChanges(ctx, []schema.Change{
			&schema.AddObject{O: &schema.EnumType{T: "state", Values: []string{"on", "off"}, Schema: usersT.Schema}},
			&schema.AddTable{T: usersT},
		})
		require.NoError(t, err, "create a new table with an enum column")
		t.dropTables(usersT.Name)
		ensureNoChange(t, usersT)

		// Add another enum column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "day", Type: &schema.ColumnType{Type: &schema.EnumType{T: "day", Values: []string{"sunday", "monday"}}}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		err = t.drv.ApplyChanges(ctx, []schema.Change{
			&schema.AddObject{O: &schema.EnumType{T: "day", Values: []string{"sunday", "monday"}, Schema: usersT.Schema}},
			&schema.ModifyTable{T: usersT, Changes: changes},
		})
		require.NoError(t, err, "add a new enum column to existing table")
		ensureNoChange(t, usersT)
	})
}

func TestPostgres_ForeignKey(t *testing.T) {
	t.Run("ChangeAction", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
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
		pgRun(t, func(t *pgTest) {
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
		pgRun(t, func(t *pgTest) {
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

func TestPostgres_EntGlobalUniqueID(t *testing.T) {
	// Migration to global unique identifiers.
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		t.dropTables("global_id")
		_, err := t.driver().ExecContext(ctx, "CREATE TABLE global_id (id int NOT NULL GENERATED BY DEFAULT AS IDENTITY, PRIMARY KEY(id))")
		require.NoError(t, err)
		_, err = t.driver().ExecContext(ctx, "ALTER TABLE global_id ALTER COLUMN id RESTART WITH 1024")
		require.NoError(t, err)
		_, err = t.driver().ExecContext(ctx, "INSERT INTO global_id VALUES (default), (default)")
		require.NoError(t, err)
		var id int
		require.NoError(t, t.db.QueryRow("SELECT id FROM global_id").Scan(&id))
		require.Equal(t, 1024, id)
		_, err = t.driver().ExecContext(ctx, "DELETE FROM global_id WHERE id = 1024")
		require.NoError(t, err)

		globalT := t.loadTable("global_id")
		c, ok := globalT.Column("id")
		require.True(t, ok)
		require.EqualValues(t, 1, c.Attrs[0].(*postgres.Identity).Sequence.Start)
		t.migrate(&schema.ModifyTable{
			T: globalT,
			Changes: []schema.Change{
				&schema.ModifyColumn{
					From: globalT.Columns[0],
					To: schema.NewIntColumn("id", "int").
						AddAttrs(&postgres.Identity{
							Generation: "BY DEFAULT",
							Sequence: &postgres.Sequence{
								Start: 1024,
							},
						}),
					Change: schema.ChangeAttr,
				},
			},
		})
		_, err = t.driver().ExecContext(ctx, "INSERT INTO global_id VALUES (default), (default)")
		require.NoError(t, err)
		globalT = t.loadTable("global_id")
		c, ok = globalT.Column("id")
		require.True(t, ok)
		require.EqualValues(t, 1024, c.Attrs[0].(*postgres.Identity).Sequence.Start)
	})
}

func TestPostgres_AdvisoryLock(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testAdvisoryLock(t.T, t.drv.(schema.Locker))
	})
}

func TestPostgres_HCL(t *testing.T) {
	full := `
schema "public" {
}
table "users" {
	schema = schema.public
	column "id" {
		type = int
	}
	primary_key {
		columns = [table.users.column.id]
	}
}
table "posts" {
	schema = schema.public
	column "id" {
		type = int
	}
	column "tags" {
		type = sql("text[]")
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
schema "public" {
}
`
	pgRun(t, func(t *pgTest) {
		testHCLIntegration(t, full, empty)
	})
}

func TestPostgres_HCL_Realm(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		t.dropSchemas("second")
		realm := t.loadRealm()
		hcl, err := postgres.MarshalHCL(realm)
		require.NoError(t, err)
		wa := string(hcl) + `
schema "second" {
  comment = "second schema"
}
`
		t.applyRealmHcl(wa)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{})
		require.NoError(t, err)
		_, ok := realm.Schema("public")
		require.True(t, ok)
		s2, ok := realm.Schema("second")
		require.True(t, ok)
		require.Len(t, s2.Attrs, 1)
		require.Equal(t, "second schema", s2.Attrs[0].(*schema.Comment).Text)
	})
}

func TestPostgres_HCL_ForeignKeyCrossSchema(t *testing.T) {
	const expected = `table "credit_cards" {
  schema = schema.financial
  column "id" {
    null = false
    type = serial
  }
  column "user_id" {
    null = false
    type = integer
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
}
table "financial" "t" {
  schema = schema.financial
  column "t_id" {
    null = false
    type = uuid
  }
  primary_key {
    columns = [column.t_id]
  }
  foreign_key "fk_t_id" {
    columns     = [column.t_id]
    ref_columns = [table.users.t.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
table "financial" "users" {
  schema = schema.financial
  column "id" {
    null = false
    type = serial
  }
}
table "users" "t" {
  schema = schema.users
  column "id" {
    null = false
    type = uuid
  }
  primary_key {
    columns = [column.id]
  }
}
table "users" "users" {
  schema = schema.users
  column "id" {
    null = false
    type = bigserial
  }
  column "email" {
    null = false
    type = character_varying
  }
  primary_key {
    columns = [column.id]
  }
}
schema "financial" {
}
schema "users" {
}
`
	pgRun(t, func(t *pgTest) {
		t.dropSchemas("financial", "users")
		realm := t.loadRealm()
		hcl, err := postgres.MarshalHCL(realm)
		require.NoError(t, err)
		t.applyRealmHcl(string(hcl) + "\n" + expected)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Schemas: []string{"users", "financial"}})
		require.NoError(t, err)
		actual, err := postgres.MarshalHCL(realm)
		require.NoError(t, err)
		require.Equal(t, expected, string(actual))
	})
}

func (t *pgTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := postgres.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func TestPostgres_Snapshot(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		client, err := sqlclient.Open(context.Background(), fmt.Sprintf("postgres://postgres:pass@localhost:%d/test?sslmode=disable&search_path=another", t.port))
		require.NoError(t, err)

		_, err = client.ExecContext(context.Background(), "CREATE SCHEMA another")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err = client.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS another")
			require.NoError(t, client.Close())
		})
		drv := client.Driver

		_, err = t.driver().(migrate.Snapshoter).Snapshot(context.Background())
		require.ErrorAs(t, err, new(*migrate.NotCleanError))

		r, err := drv.InspectRealm(context.Background(), nil)
		require.NoError(t, err)
		restore, err := drv.(migrate.Snapshoter).Snapshot(context.Background())
		require.NoError(t, err) // connected to test schema
		require.NoError(t, drv.ApplyChanges(context.Background(), []schema.Change{
			&schema.AddTable{T: schema.NewTable("my_table").
				AddColumns(
					schema.NewIntColumn("col_1", "integer").SetNull(true),
					schema.NewIntColumn("col_2", "bigint"),
				),
			},
		}))
		t.Cleanup(func() {
			t.dropTables("my_table")
		})
		require.NoError(t, restore(context.Background()))
		r1, err := drv.InspectRealm(context.Background(), nil)
		require.NoError(t, err)
		diff, err := drv.RealmDiff(r1, r)
		require.NoError(t, err)
		require.Zero(t, diff)
	})
}

func TestPostgres_CLI_MigrateApplyBC(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testCLIMigrateApplyBC(t, "postgres")
	})
}

func TestPostgres_CLI(t *testing.T) {
	h := `
			schema "public" {
			}
			table "users" {
				schema = schema.public
				column "id" {
					type = integer
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaInspect(t, h, t.url(""), postgres.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaApply(t, h, t.url(""))
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaApplyDry(t, h, t.url(""))
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
		pgRun(t, func(t *pgTest) {
			testCLISchemaApply(t, h, t.url(""), "--var", "tenant=public")
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaDiff(t, t.url(""))
		})
	})
	t.Run("SchemaApplyAutoApprove", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaApplyAutoApprove(t, h, t.url(""))
		})
	})
	t.Run("SchemaApplyFromMigrationDir", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaApplyFromMigrationDir(t)
		})
	})
}

func TestPostgres_CLI_MultiSchema(t *testing.T) {
	h := `
			schema "public" {
			}
			table "users" {
				schema = schema.public
				column "id" {
					type = integer
				}
				primary_key {
					columns = [column.id]
				}
			}
			schema "test2" {
			}
			table "pets" {
				schema = schema.test2
				column "id" {
					type = integer
				}
				column "owner_id" {
					type = integer
				}
				primary_key {
					columns = [column.id]
				}
				foreign_key "owner_id" {
					columns     = [column.owner_id]
					ref_columns = [table.users.column.id]
					on_delete   = NO_ACTION
					on_update   = NO_ACTION
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			t.dropTables("users")
			t.dropSchemas("test2")
			testCLIMultiSchemaInspect(t, h, t.url(""), []string{"public", "test2"}, postgres.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			t.dropTables("users")
			t.dropSchemas("test2")
			testCLIMultiSchemaApply(t, h, t.url(""), []string{"public", "test2"}, postgres.EvalHCL)
		})
	})
}

func TestPostgres_NormalizeRealm(t *testing.T) {
	bin := cliPath(t)
	pgRun(t, func(t *pgTest) {
		dir := t.TempDir()
		_, err := t.db.Exec("CREATE DATABASE normalized_realm")
		require.NoError(t, err)
		defer t.db.Exec("DROP DATABASE IF EXISTS normalized_realm")
		hcl := `
schema "public" {}
enum "status" {
  schema = schema.public
  values = ["active", "inactive"]
}

table "users" {
  schema = schema.public
  column "id" { type = serial }
  column "e"  { type = enum.status }
  column "ae" { type = sql("status[]") }
}

schema "other" {}
table "posts" {
  schema = schema.other
  column "id" { type = integer }
}

table "with_default" {
  schema = schema.other
  column "name" {
    type = varchar
	default = sql("lower('Hello')")
  }
}
`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		require.NoError(t, err)
		out, err := exec.Command(
			bin, "schema", "inspect",
			"--url", fmt.Sprintf("file://%s", filepath.Join(dir, "schema.hcl")),
			"--dev-url", fmt.Sprintf("postgres://postgres:pass@localhost:%d/normalized_realm?sslmode=disable", t.port),
		).CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, `table "posts" {
  schema = schema.other
  column "id" {
    null = false
    type = integer
  }
}
table "with_default" {
  schema = schema.other
  column "name" {
    null    = false
    type    = character_varying
    default = sql("lower('Hello'::text)")
  }
}
table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "e" {
    null = false
    type = enum.status
  }
  column "ae" {
    null = false
    type = sql("status[]")
  }
}
enum "status" {
  schema = schema.public
  values = ["active", "inactive"]
}
schema "other" {
}
schema "public" {
  comment = "standard public schema"
}
`, string(out))
		err = t.drv.(migrate.CleanChecker).CheckClean(context.Background(), nil)
		require.NoError(t, err)
	})
}

func TestPostgres_MigrateDiffRealm(t *testing.T) {
	bin := cliPath(t)
	pgRun(t, func(t *pgTest) {
		dir := t.TempDir()
		_, err := t.db.Exec("CREATE DATABASE migrate_diff")
		require.NoError(t, err)
		defer t.db.Exec("DROP DATABASE IF EXISTS migrate_diff")

		hcl := `
schema "public" {}
table "users" {
	schema = schema.public
	column "id" { type = integer }
}
schema "other" {}
table "posts" {
	schema = schema.other
	column "id" { type = integer }
}
`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		diff := func(name string) string {
			out, err := exec.Command(
				bin, "migrate", "diff", name,
				"--dir", fmt.Sprintf("file://%s", filepath.Join(dir, "migrations")),
				"--to", fmt.Sprintf("file://%s", filepath.Join(dir, "schema.hcl")),
				"--dev-url", fmt.Sprintf("postgres://postgres:pass@localhost:%d/migrate_diff?sslmode=disable", t.port),
			).CombinedOutput()
			require.NoError(t, err, string(out))
			return strings.TrimSpace(string(out))
		}
		require.Empty(t, diff("initial"))

		// Expect one file and read its contents.
		files, err := os.ReadDir(filepath.Join(dir, "migrations"))
		require.NoError(t, err)
		require.Equal(t, 2, len(files))
		require.Equal(t, "atlas.sum", files[1].Name())
		b, err := os.ReadFile(filepath.Join(dir, "migrations", files[0].Name()))
		require.NoError(t, err)
		require.Equal(t,
			`-- Add new schema named "other"
CREATE SCHEMA "other";
-- Create "users" table
CREATE TABLE "public"."users" ("id" integer NOT NULL);
-- Create "posts" table
CREATE TABLE "other"."posts" ("id" integer NOT NULL);
`, string(b))
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made", diff("no_change"))

		// Append a change to the schema and expect a migration to be created.
		hcl += `
table "other" "users" {
	schema = schema.other
	column "id" { type = integer }
}`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		require.Empty(t, diff("second"))
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made", diff("no_change"))
		files, err = os.ReadDir(filepath.Join(dir, "migrations"))
		require.NoError(t, err)
		require.Equal(t, 3, len(files), dir)
		b, err = os.ReadFile(filepath.Join(dir, "migrations", files[1].Name()))
		require.NoError(t, err)
		require.Equal(t,
			`-- Create "users" table
CREATE TABLE "other"."users" ("id" integer NOT NULL);
`, string(b))
	})
}

func TestPostgres_Migrate_Mixin(t *testing.T) {
	bin := cliPath(t)
	pgRun(t, func(t *pgTest) {
		dir := t.TempDir()
		_, err := t.db.Exec("CREATE DATABASE migrate_mixin")
		require.NoError(t, err)
		defer t.db.Exec("DROP DATABASE IF EXISTS migrate_mixin")

		hcl := `
schema "public" {}
table "users" {
	schema = schema.public
	embed = mixin.base
}
schema "other" {}
table "posts" {
	schema = schema.other
	embed = mixin.base
}
mixin "base" {
	column "id" { type = integer }
}
`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		diff := func(name string) string {
			out, err := exec.Command(
				bin, "migrate", "diff", name,
				"--dir", fmt.Sprintf("file://%s", filepath.Join(dir, "migrations")),
				"--to", fmt.Sprintf("file://%s", filepath.Join(dir, "schema.hcl")),
				"--dev-url", fmt.Sprintf("postgres://postgres:pass@localhost:%d/migrate_mixin?sslmode=disable", t.port),
			).CombinedOutput()
			require.NoError(t, err, string(out))
			return strings.TrimSpace(string(out))
		}
		require.Empty(t, diff("initial"))

		// Expect one file and read its contents.
		files, err := os.ReadDir(filepath.Join(dir, "migrations"))
		require.NoError(t, err)
		require.Equal(t, 2, len(files))
		require.Equal(t, "atlas.sum", files[1].Name())
		b, err := os.ReadFile(filepath.Join(dir, "migrations", files[0].Name()))
		require.NoError(t, err)
		require.Equal(t,
			`-- Add new schema named "other"
CREATE SCHEMA "other";
-- Create "users" table
CREATE TABLE "public"."users" ("id" integer NOT NULL);
-- Create "posts" table
CREATE TABLE "other"."posts" ("id" integer NOT NULL);
`, string(b))
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made", diff("no_change"))

		// Append a change to the schema and expect a migration to be created.
		hcl += `
table "other" "users" {
	schema = schema.other
	column "id" { type = integer }
}`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		require.Empty(t, diff("second"))
		require.Equal(t, "The migration directory is synced with the desired state, no changes to be made", diff("no_change"))
		files, err = os.ReadDir(filepath.Join(dir, "migrations"))
		require.NoError(t, err)
		require.Equal(t, 3, len(files), dir)
		b, err = os.ReadFile(filepath.Join(dir, "migrations", files[1].Name()))
		require.NoError(t, err)
		require.Equal(t,
			`-- Create "users" table
CREATE TABLE "other"."users" ("id" integer NOT NULL);
`, string(b))
	})
}

func TestPostgres_SchemaDiff(t *testing.T) {
	bin := cliPath(t)
	pgRun(t, func(t *pgTest) {
		dir := t.TempDir()
		_, err := t.db.Exec("CREATE DATABASE test1")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := t.db.Exec("DROP DATABASE IF EXISTS test1")
			require.NoError(t, err)
		})
		_, err = t.db.Exec("CREATE DATABASE test2")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err = t.db.Exec("DROP DATABASE IF EXISTS test2")
			require.NoError(t, err)
		})

		diff := func(db1, db2 string) string {
			out, err := exec.Command(
				bin, "schema", "diff",
				"--from", fmt.Sprintf("postgres://postgres:pass@localhost:%d/%s", t.port, db1),
				"--to", fmt.Sprintf("postgres://postgres:pass@localhost:%d/%s", t.port, db2),
			).CombinedOutput()
			require.NoError(t, err, string(out))
			return strings.TrimSpace(string(out))
		}
		// Diff a database with itself.
		require.Equal(t, "Schemas are synced, no changes to be made.", diff("test1?sslmode=disable", "test2?sslmode=disable"))

		// Create schemas on test2 database.
		hcl := `
schema "public" {
  comment = "standard public schema"
}
table "users" {
	schema = schema.public
	column "id" { type = integer }
}
schema "other" {}
table "posts" {
	schema = schema.other
	column "id" { type = integer }
}
`
		err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(hcl), 0600)
		require.NoError(t, err)
		out, err := exec.Command(
			bin, "schema", "apply",
			"-u", fmt.Sprintf("postgres://postgres:pass@localhost:%d/test2?sslmode=disable", t.port),
			"-f", fmt.Sprintf(filepath.Join(dir, "schema.hcl")),
			"--auto-approve",
		).CombinedOutput()
		require.NoError(t, err, string(out))

		// Diff a database with different one.
		require.Equal(t, `-- Add new schema named "other"
CREATE SCHEMA "other";
-- Create "users" table
CREATE TABLE "public"."users" ("id" integer NOT NULL);
-- Create "posts" table
CREATE TABLE "other"."posts" ("id" integer NOT NULL);`, diff("test1?sslmode=disable", "test2?sslmode=disable"))

		// Diffing schema should both tables and comments (from 'public' to 'other').
		require.Equal(t, `-- Create "users" table
CREATE TABLE "users" ("id" integer NOT NULL);
-- Drop "posts" table
DROP TABLE "posts";`, diff("test2?sslmode=disable&search_path=other", "test2?sslmode=disable&search_path=public"))
		// diff between schema and database
		out, err = exec.Command(
			bin, "schema", "diff",
			"--from", fmt.Sprintf("postgres://postgres:pass@localhost:%d/test2?sslmode=disable", t.port),
			"--to", fmt.Sprintf("postgres://postgres:pass@localhost:%d/test2?sslmode=disable&search_path=public", t.port),
		).CombinedOutput()
		require.Error(t, err, string(out))
		require.Equal(t, "Error: cannot diff a schema with a database connection: \"\" <> \"public\"\n", string(out))
	})
}

func TestPostgres_MigrateDiffEnt(t *testing.T) {
	bin := cliPath(t)
	pgRun(t, func(t *pgTest) {
		dir := t.TempDir()
		_, err := t.db.Exec("CREATE SCHEMA IF NOT EXISTS entdev")
		require.NoError(t, err)
		t.dropSchemas("entdev")

		entdir, migratedir := filepath.Join(dir, "entschema"), filepath.Join(dir, "migrations")
		require.NoError(t, os.Mkdir(entdir, 0755))
		err = os.WriteFile(filepath.Join(entdir, "user.go"), []byte(`
package schema

import "entgo.io/ent"

type User struct { ent.Schema }
`), 0600)
		require.NoError(t, err)
		for _, ts := range [][]string{{"init", "entschema"}, {"tidy"}} {
			cmd := exec.Command("go", append([]string{"mod"}, ts...)...)
			cmd.Dir = dir
			require.NoError(t, cmd.Run())
		}
		cmd := exec.Command(
			bin, "migrate", "diff",
			"--dev-url", fmt.Sprintf("postgres://postgres:pass@localhost:%d/test?search_path=entdev&sslmode=disable", t.port),
			"--to", "ent://"+entdir,
			"--dir", "file://"+migratedir,
		)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		local, err := migrate.NewLocalDir(migratedir)
		require.NoError(t, err)
		files, err := local.Files()
		require.NoError(t, err)
		require.Equal(t, 1, len(files))
		stmts, err := files[0].Stmts()
		require.NoError(t, err)
		require.Equal(t, []string{
			`CREATE TABLE "users" ("id" bigint NOT NULL GENERATED BY DEFAULT AS IDENTITY, PRIMARY KEY ("id"));`,
		}, stmts)
	})
}

func TestPostgres_DefaultsHCL(t *testing.T) {
	n := "atlas_defaults"
	pgRun(t, func(t *pgTest) {
		ddl := `
create table atlas_defaults
(
	string varchar(255) default 'hello_world',
	quoted varchar(100) default 'never say "never"',
	tBit bit(10) default b'10101',
	ts timestamp default CURRENT_TIMESTAMP,
	tstz timestamp with time zone default CURRENT_TIMESTAMP,
	number int default 42
)
`
		t.dropTables(n)
		_, err := t.db.Exec(ddl)
		require.NoError(t, err)
		realm := t.loadRealm()
		spec, err := postgres.MarshalHCL(realm.Schemas[0])
		require.NoError(t, err)
		var s schema.Schema
		err = postgres.EvalHCLBytes(spec, &s, nil)
		require.NoError(t, err)
		t.dropTables(n)
		t.applyHcl(string(spec))
		ensureNoChange(t, realm.Schemas[0].Tables[0])
	})
}

func TestPostgres_Sanity(t *testing.T) {
	n := "atlas_types_sanity"
	ddl := `
DROP TYPE IF EXISTS address;
CREATE TYPE address AS (city VARCHAR(90), street VARCHAR(90));
create table atlas_types_sanity
(
    "tBit"                 bit(10)                     default b'100'                                   null,
    "tBitVar"              bit varying(10)             default b'100'                                   null,
    "tBoolean"             boolean                     default false                                not null,
    "tBool"                bool                        default false                                not null,
    "tBytea"               bytea                       default E'\\001'                             not null,
    "tCharacter"           character(10)               default 'atlas'                                  null,
    "tChar"                char(10)                    default 'atlas'                                  null,
    "tCharVar"             character varying(10)       default 'atlas'                                  null,
    "tVarChar"             varchar(10)                 default 'atlas'                                  null,
    "tText"                text                        default 'atlas'                                  null,
    "tSmallInt"            smallint                    default '10'                                     null,
    "tInteger"             integer                     default '10'                                     null,
    "tBigInt"              bigint                      default '10'                                     null,
    "tInt"                 int                         default '10'                                     null,
    "tInt2"                int2                        default '10'                                     null,
    "tInt4"                int4                        default '10'                                     null,
    "tInt8"                int8                        default '10'                                     null,
    "tCIDR"                cidr                        default '127.0.0.1'                              null,
    "tInet"                inet                        default '127.0.0.1'                              null,
    "tMACAddr"             macaddr                     default '08:00:2b:01:02:03'                      null,
    "tMACAddr8"            macaddr8                    default '08:00:2b:01:02:03:04:05'                null,
    "tCircle"              circle                      default                                          null,
    "tLine"                line                        default                                          null,
    "tLseg"                lseg                        default                                          null,
    "tBox"                 box                         default                                          null,
    "tPath"                path                        default                                          null,
    "tPoint"               point                       default                                          null,
    "tDate"                date                        default current_date                             null,
    "tTime"                time                        default current_time                             null,
    "tTimeWTZ"             time with time zone         default current_time                             null,
    "tTimeWOTZ"            time without time zone      default current_time                             null,
    "tTimestamp"           timestamp                   default now()                                    null,
    "tTimestampTZ"         timestamptz                 default now()                                    null,
    "tTimestampWTZ"        timestamp with time zone    default now()                                    null,
    "tTimestampWOTZ"       timestamp without time zone default now()                                    null,
    "tTimestampPrec"       timestamp(4)                default now()                                    null,
    "tDouble"              double precision            default 0                                        null,
    "tReal"                real                        default 0                                        null,
    "tFloat8"              float8                      default 0                                        null,
    "tFloat4"              float4                      default 0                                        null,
    "tNumeric"             numeric                     default 0                                        null,
    "tDecimal"             decimal                     default 0                                        null,
    "tSmallSerial"         smallserial                                                                      ,
    "tSerial"              serial                                                                           ,
    "tBigSerial"           bigserial                                                                        ,
    "tSerial2"             serial2                                                                          ,
    "tSerial4"             serial4                                                                          ,
    "tSerial8"             serial8                                                                          ,
    "tArray"               text[10][10]                 default '{}'                                    null,
    "tXML"                 xml                          default '<a>foo</a>'                            null,
    "tJSON"                json                         default '{"key":"value"}'                       null,
    "tJSONB"               jsonb                        default '{"key":"value"}'                       null,
    "tUUID"                uuid                         default  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11' null,
    "tMoney"               money                        default  18                                     null,
    "tInterval"            interval                     default '4 hours'                               null,
    "tUserDefined"         address                      default '("ab","cd")'                           null
);
`
	pgRun(t, func(t *pgTest) {
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
			Columns: []*schema.Column{
				{
					Name:    "tBit",
					Type:    &schema.ColumnType{Type: &postgres.BitType{T: "bit", Len: 10}, Raw: "bit", Null: true},
					Default: &schema.RawExpr{X: t.valueByVersion(map[string]string{"postgres10": "B'100'::\"bit\""}, "'100'::\"bit\"")},
				},
				{
					Name:    "tBitVar",
					Type:    &schema.ColumnType{Type: &postgres.BitType{T: "bit varying", Len: 10}, Raw: "bit varying", Null: true},
					Default: &schema.RawExpr{X: t.valueByVersion(map[string]string{"postgres10": "B'100'::\"bit\""}, "'100'::\"bit\"")},
				},
				{
					Name:    "tBoolean",
					Type:    &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Raw: "boolean", Null: false},
					Default: &schema.Literal{V: "false"},
				},
				{
					Name:    "tBool",
					Type:    &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Raw: "boolean", Null: false},
					Default: &schema.Literal{V: "false"},
				},
				{
					Name:    "tBytea",
					Type:    &schema.ColumnType{Type: &schema.BinaryType{T: "bytea"}, Raw: "bytea", Null: false},
					Default: &schema.Literal{V: "'\\x01'"},
				},
				{
					Name:    "tCharacter",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character", Null: true},
					Default: &schema.Literal{V: "'atlas'"},
				},
				{
					Name:    "tChar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character", Null: true},
					Default: &schema.Literal{V: "'atlas'"},
				},
				{
					Name:    "tCharVar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character varying", Size: 10}, Raw: "character varying", Null: true},
					Default: &schema.Literal{V: "'atlas'"},
				},
				{
					Name:    "tVarChar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character varying", Size: 10}, Raw: "character varying", Null: true},
					Default: &schema.Literal{V: "'atlas'"},
				},
				{
					Name:    "tText",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "text"}, Raw: "text", Null: true},
					Default: &schema.Literal{V: "'atlas'"},
				},
				{
					Name:    "tSmallInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}, Raw: "smallint", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tInteger",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tBigInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tInt2",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}, Raw: "smallint", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tInt4",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tInt8",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.Literal{V: "10"},
				},
				{
					Name:    "tCIDR",
					Type:    &schema.ColumnType{Type: &postgres.NetworkType{T: "cidr"}, Raw: "cidr", Null: true},
					Default: &schema.Literal{V: "'127.0.0.1/32'"},
				},
				{
					Name:    "tInet",
					Type:    &schema.ColumnType{Type: &postgres.NetworkType{T: "inet"}, Raw: "inet", Null: true},
					Default: &schema.Literal{V: "'127.0.0.1'"},
				},
				{
					Name:    "tMACAddr",
					Type:    &schema.ColumnType{Type: &postgres.NetworkType{T: "macaddr"}, Raw: "macaddr", Null: true},
					Default: &schema.Literal{V: "'08:00:2b:01:02:03'"},
				},
				{
					Name:    "tMACAddr8",
					Type:    &schema.ColumnType{Type: &postgres.NetworkType{T: "macaddr8"}, Raw: "macaddr8", Null: true},
					Default: &schema.Literal{V: "'08:00:2b:01:02:03:04:05'"},
				},
				{
					Name: "tCircle",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "circle"}, Raw: "circle", Null: true},
				},
				{
					Name: "tLine",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "line"}, Raw: "line", Null: true},
				},
				{
					Name: "tLseg",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "lseg"}, Raw: "lseg", Null: true},
				},
				{
					Name: "tBox",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "box"}, Raw: "box", Null: true},
				},
				{
					Name: "tPath",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "path"}, Raw: "path", Null: true},
				},
				{
					Name: "tPoint",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"}, Raw: "point", Null: true},
				},
				{
					Name:    "tDate",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "date"}, Raw: "date", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_DATE"},
				},
				{
					Name:    "tTime",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone", Precision: intp(6)}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimeWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time with time zone", Precision: intp(6)}, Raw: "time with time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimeWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone", Precision: intp(6)}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimestamp",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(6)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone", Precision: intp(6)}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone", Precision: intp(6)}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(6)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampPrec",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(4)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tDouble",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 53}, Raw: "double precision", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name:    "tReal",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 24}, Raw: "real", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name:    "tFloat8",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 53}, Raw: "double precision", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name:    "tFloat4",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 24}, Raw: "real", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name:    "tNumeric",
					Type:    &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 0}, Raw: "numeric", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name:    "tDecimal",
					Type:    &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 0}, Raw: "numeric", Null: true},
					Default: &schema.Literal{V: "0"},
				},
				{
					Name: "tSmallSerial",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "smallserial", SequenceName: "atlas_types_sanity_tSmallSerial_seq"}, Raw: "smallserial", Null: false},
				},
				{
					Name: "tSerial",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "serial", SequenceName: "atlas_types_sanity_tSerial_seq"}, Raw: "serial", Null: false},
				},
				{
					Name: "tBigSerial",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "bigserial", SequenceName: "atlas_types_sanity_tBigSerial_seq"}, Raw: "bigserial", Null: false},
				},
				{
					Name: "tSerial2",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "smallserial", SequenceName: "atlas_types_sanity_tSerial2_seq"}, Raw: "smallserial", Null: false},
				},
				{
					Name: "tSerial4",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "serial", SequenceName: "atlas_types_sanity_tSerial4_seq"}, Raw: "serial", Null: false},
				},
				{
					Name: "tSerial8",
					Type: &schema.ColumnType{Type: &postgres.SerialType{T: "bigserial", SequenceName: "atlas_types_sanity_tSerial8_seq"}, Raw: "bigserial", Null: false},
				},
				{
					Name: "tArray",
					Type: &schema.ColumnType{Type: &postgres.ArrayType{Type: &schema.StringType{T: "text"}, T: "text[]"}, Raw: "ARRAY", Null: true},
					Default: &schema.Literal{
						V: "'{}'",
					},
				},
				{
					Name: "tXML",
					Type: &schema.ColumnType{Type: &postgres.XMLType{T: "xml"}, Raw: "xml", Null: true},
					Default: &schema.Literal{
						V: "'<a>foo</a>'",
					},
				},
				{
					Name: "tJSON",
					Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}, Raw: "json", Null: true},
					Default: &schema.Literal{
						V: "'{\"key\":\"value\"}'",
					},
				},
				{
					Name: "tJSONB",
					Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}, Raw: "jsonb", Null: true},
					Default: &schema.Literal{
						V: "'{\"key\": \"value\"}'",
					},
				},
				{
					Name: "tUUID",
					Type: &schema.ColumnType{Type: &schema.UUIDType{T: "uuid"}, Raw: "uuid", Null: true},
					Default: &schema.Literal{
						V: "'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'",
					},
				},
				{
					Name: "tMoney",
					Type: &schema.ColumnType{Type: &postgres.CurrencyType{T: "money"}, Raw: "money", Null: true},
					Default: &schema.Literal{
						V: "18",
					},
				},
				{
					Name: "tInterval",
					Type: &schema.ColumnType{Type: &postgres.IntervalType{T: "interval", Precision: intp(6)}, Raw: "interval", Null: true},
					Default: &schema.RawExpr{
						X: "'04:00:00'::interval",
					},
				},
				{
					Name: "tUserDefined",
					Type: &schema.ColumnType{Type: &postgres.UserDefinedType{T: "address"}, Raw: "USER-DEFINED", Null: true},
					Default: &schema.RawExpr{
						X: "'(ab,cd)'::address",
					},
				},
			},
		}
		require.EqualValues(t, &expected, ts)
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testImplicitIndexes(t, t.db)
		})
	})
}

func (t *pgTest) url(schema string) string {
	var (
		format = "postgres://postgres:pass@localhost:%d/test?sslmode=disable"
		args   = []any{t.port}
	)
	if schema != "" {
		format += "&search_path=%s"
		args = append(args, schema)
	}
	return fmt.Sprintf(format, args...)
}

func (t *pgTest) driver() migrate.Driver {
	return t.drv
}

func (t *pgTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *pgTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := postgres.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	existing := realm.Schemas[0]
	diff, err := t.drv.SchemaDiff(existing, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func (t *pgTest) valueByVersion(values map[string]string, defaults string) string {
	if v, ok := values[t.version]; ok {
		return v
	}
	return defaults
}

func (t *pgTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"public"},
		Mode:    ^schema.InspectViews,
	})
	require.NoError(t, err)
	return r
}

func (t *pgTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *pgTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *pgTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *pgTest) users() *schema.Table {
	usersT := &schema.Table{
		Name:   "users",
		Schema: t.realm().Schemas[0],
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

func (t *pgTest) posts() *schema.Table {
	usersT := t.users()
	postsT := &schema.Table{
		Name:   "posts",
		Schema: t.realm().Schemas[0],
		Columns: []*schema.Column{
			{
				Name:  "id",
				Type:  &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}},
				Attrs: []schema.Attr{&postgres.Identity{}},
			},
			{
				Name:    "author_id",
				Type:    &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}, Null: true},
				Default: &schema.Literal{V: "10"},
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

func (t *pgTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
				Attrs: []schema.Attr{
					&schema.Comment{Text: "standard public schema"},
				},
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

func (t *pgTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *pgTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *pgTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *pgTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP SCHEMA IF EXISTS " + strings.Join(names, ", ") + " CASCADE")
		require.NoError(t.T, err, "drop schema %q", names)
	})
}
