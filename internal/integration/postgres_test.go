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

	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entm2m2types "entgo.io/ent/examples/m2m2types/ent"
	enttraversal "entgo.io/ent/examples/traversal/ent"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type pgTest struct {
	*testing.T
	db      *sql.DB
	drv     *postgres.Driver
	version string
}

var pgTests struct {
	sync.Once
	drivers map[string]*pgTest
}

func pgRun(t *testing.T, fn func(*pgTest)) {
	pgTests.Do(func() {
		pgTests.drivers = make(map[string]*pgTest)
		for version, port := range map[string]int{"10": 5430, "11": 5431, "12": 5432, "13": 5433, "14": 5434} {
			db, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=postgres dbname=test password=pass sslmode=disable", port))
			require.NoError(t, err)
			drv, err := postgres.Open(db)
			require.NoError(t, err)
			pgTests.drivers[version] = &pgTest{db: db, drv: drv, version: version}
		}
	})
	for version, tt := range pgTests.drivers {
		t.Run(version, func(t *testing.T) {
			tt := &pgTest{T: t, db: tt.db, drv: tt.drv, version: version}
			fn(tt)
		})
	}
}

func TestPostgres_AddDropTable(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		usersT := t.users()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.AddTable{T: usersT},
		})
		require.NoError(t, err)
		realm := t.loadRealm()
		changes, err := t.drv.Diff().TableDiff(realm.Schemas[0].Tables[0], usersT)
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

func TestPostgres_Relation(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		usersT, postsT := t.users(), t.posts()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{
			&schema.AddTable{T: usersT},
			&schema.AddTable{T: postsT},
		})
		require.NoError(t, err)
		t.dropTables(postsT.Name, usersT.Name)
		t.ensureNoChange(postsT, usersT)
	})
}

func TestPostgres_AddIndexedColumns(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		usersT := &schema.Table{
			Name:    "users",
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}}},
		}
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
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.NotEmpty(t, changes, "usersT contains 3 new columns and 1 new index")
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)

		// Dropping a column involves in a multi-column
		// index causes the index to be dropped as well.
		usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
		changes, err = t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(t.loadUsers())
		usersT = t.loadUsers()
		_, ok := usersT.Index("a_b_c_unique")
		require.False(t, ok)
	})
}

func TestPostgres_AddColumns(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		ctx := context.Background()
		usersT := t.users()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
		require.NoError(t, err)
		t.dropTables(usersT.Name)
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "bytea"}}},
			&schema.Column{Name: "b", Type: &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 10}}, Default: &schema.RawExpr{X: "10.1"}},
			&schema.Column{Name: "c", Type: &schema.ColumnType{Type: &schema.StringType{T: "character"}}, Default: &schema.RawExpr{X: "'y'"}},
			&schema.Column{Name: "d", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}}, Default: &schema.RawExpr{X: "0.99"}},
			&schema.Column{Name: "e", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}}, Default: &schema.RawExpr{X: "'{}'"}},
			&schema.Column{Name: "f", Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}}, Default: &schema.RawExpr{X: "'1'"}},
			&schema.Column{Name: "g", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 10}}, Default: &schema.RawExpr{X: "'1'"}},
			&schema.Column{Name: "h", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 30}}, Default: &schema.RawExpr{X: "'1'"}},
			&schema.Column{Name: "i", Type: &schema.ColumnType{Type: &schema.FloatType{T: "float", Precision: 53}}, Default: &schema.RawExpr{X: "1"}},
			&schema.Column{Name: "j", Type: &schema.ColumnType{Type: &postgres.SerialType{T: "serial"}}},
			&schema.Column{Name: "k", Type: &schema.ColumnType{Type: &postgres.CurrencyType{T: "money"}}, Default: &schema.RawExpr{X: "100"}},
			&schema.Column{Name: "l", Type: &schema.ColumnType{Type: &postgres.CurrencyType{T: "money"}, Null: true}, Default: &schema.RawExpr{X: "'52093.89'::money"}},
			&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Null: true}, Default: &schema.RawExpr{X: "false"}},
			&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "point"}, Null: true}, Default: &schema.RawExpr{X: "'(1,2)'"}},
			&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "line"}, Null: true}, Default: &schema.RawExpr{X: "'{1,2,3}'"}},
		)
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 15)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)
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
			err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
			require.NoError(t, err)
			t.dropTables(usersT.Name)
			change(usersT.Columns[0])
			changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
			require.NoError(t, err)
			require.Len(t, changes, 1)
			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
			require.NoError(t, err)
			t.ensureNoChange(usersT)
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
		ctx := context.Background()
		usersT := t.users()
		err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
		require.NoError(t, err)
		t.dropTables(usersT.Name)

		// Add column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "int[]", Type: &postgres.ArrayType{T: "int[]"}}, Default: &schema.RawExpr{X: "'{1}'"}},
		)
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)

		// Check default.
		usersT.Columns[2].Default = &schema.RawExpr{X: "ARRAY[1]"}
		t.ensureNoChange(usersT)

		// Change default.
		usersT.Columns[2].Default = &schema.RawExpr{X: "ARRAY[1,2]"}
		changes, err = t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err)
		t.ensureNoChange(usersT)
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
		err := t.drv.Migrate().Exec(ctx, []schema.Change{&schema.AddTable{T: usersT}})
		require.NoError(t, err, "create a new table with an enum column")
		t.dropTables(usersT.Name)
		t.ensureNoChange(usersT)

		// Add another enum column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "day", Type: &schema.ColumnType{Type: &schema.EnumType{T: "day", Values: []string{"sunday", "monday"}}}},
		)
		changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "add a new enum column to existing table")
		t.ensureNoChange(usersT)

		// Add a new value to an existing enum.
		e := usersT.Columns[1].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "tuesday")
		changes, err = t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append a value to existing enum")
		t.ensureNoChange(usersT)

		// Add multiple new values to an existing enum.
		e = usersT.Columns[1].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "wednesday", "thursday", "friday", "saturday")
		changes, err = t.drv.Diff().TableDiff(t.loadUsers(), usersT)
		require.NoError(t, err)
		require.Len(t, changes, 1)
		err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append multiple values to existing enum")
		t.ensureNoChange(usersT)
	})
}

func TestPostgres_ForeignKey(t *testing.T) {
	ctx := context.Background()
	t.Run("ChangeAction", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
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
		pgRun(t, func(t *pgTest) {
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

	t.Run("AddDrop", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			usersT := t.users()
			t.dropTables(usersT.Name)
			err := t.drv.Migrate().Exec(ctx, []schema.Change{
				&schema.AddTable{T: usersT},
			})
			require.NoError(t, err)
			t.ensureNoChange(usersT)

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

			changes, err := t.drv.Diff().TableDiff(t.loadUsers(), usersT)
			require.NoError(t, err)
			require.Len(t, changes, 2)
			addC, ok := changes[0].(*schema.AddColumn)
			require.True(t, ok)
			require.Equal(t, "spouse_id", addC.C.Name)
			addF, ok := changes[1].(*schema.AddForeignKey)
			require.True(t, ok)
			require.Equal(t, "spouse_id", addF.F.Symbol)
			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
			require.NoError(t, err)
			t.ensureNoChange(usersT)

			// Drop foreign keys.
			usersT.Columns = usersT.Columns[:len(usersT.Columns)-1]
			usersT.ForeignKeys = usersT.ForeignKeys[:len(usersT.ForeignKeys)-1]
			changes, err = t.drv.Diff().TableDiff(t.loadUsers(), usersT)
			require.NoError(t, err)
			require.Len(t, changes, 2)
			err = t.drv.Migrate().Exec(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
			require.NoError(t, err)
			t.ensureNoChange(usersT)
		})
	})
}

func (t *pgTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"public"},
	})
	require.NoError(t, err)
	return r
}

func (t *pgTest) loadUsers() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	users, ok := realm.Schemas[0].Table("users")
	require.True(t, ok)
	return users
}

func (t *pgTest) loadPosts() *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	posts, ok := realm.Schemas[0].Table("posts")
	require.True(t, ok)
	return posts
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

func (t *pgTest) realm() *schema.Realm {
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

func (t *pgTest) ensureNoChange(tables ...*schema.Table) {
	realm := t.loadRealm()
	require.Equal(t, len(realm.Schemas[0].Tables), len(tables))
	for i := range tables {
		changes, err := t.drv.Diff().TableDiff(realm.Schemas[0].Tables[i], tables[i])
		require.NoError(t, err)
		require.Empty(t, changes)
	}
}

func (t *pgTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func TestPostgres_Ent(t *testing.T) {
	ctx := context.Background()
	t.Run("ent-traversal", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			drv := entsql.OpenDB(dialect.Postgres, t.db)
			client := enttraversal.NewClient(enttraversal.Driver(drv))
			err := client.Schema.Create(ctx)
			require.NoError(t, err)
			r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
				Schemas: []string{"public"},
			})
			require.NoError(t, err)
			require.NotEmpty(t, r.Schemas)
			changes, err := t.drv.Diff().SchemaDiff(r.Schemas[0], &schema.Schema{})
			require.NoError(t, err)
			require.NotEmpty(t, changes)
			err = t.drv.Migrate().Exec(ctx, changes)
			require.NoError(t, err)
			e, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
				Schemas: []string{"public"},
			})
			require.NotEmpty(t, e.Schemas)
			require.Empty(t, e.Schemas[0].Tables)
			changes, err = t.drv.Diff().SchemaDiff(e.Schemas[0], r.Schemas[0])
			require.NoError(t, err)
			require.NotEmpty(t, changes)
			err = t.drv.Migrate().Exec(ctx, changes)
			require.NoError(t, err)
			err = client.Schema.Create(ctx)
			t.ensureNoChange(r.Schemas[0].Tables...)
		})
	})

	t.Run("ent-m2m2types", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			drv := entsql.OpenDB(dialect.Postgres, t.db)
			client := entm2m2types.NewClient(entm2m2types.Driver(drv))
			err := client.Schema.Create(ctx)
			require.NoError(t, err)
			r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
				Schemas: []string{"public"},
			})
			require.NotEmpty(t, r.Schemas)
			require.NoError(t, err)
			changes, err := t.drv.Diff().SchemaDiff(r.Schemas[0], &schema.Schema{})
			require.NoError(t, err)
			require.NotEmpty(t, changes)
			err = t.drv.Migrate().Exec(ctx, changes)
			require.NoError(t, err)
			e, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
				Schemas: []string{"public"},
			})
			require.NotEmpty(t, e.Schemas)
			require.Empty(t, e.Schemas[0].Tables)
			changes, err = t.drv.Diff().SchemaDiff(e.Schemas[0], r.Schemas[0])
			require.NoError(t, err)
			require.NotEmpty(t, changes)
			err = t.drv.Migrate().Exec(ctx, changes)
			require.NoError(t, err)
			err = client.Schema.Create(ctx)
			t.ensureNoChange(r.Schemas[0].Tables...)
		})
	})
}
