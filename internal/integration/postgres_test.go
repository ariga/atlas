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
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	"entgo.io/ent/dialect"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type pgTest struct {
	*testing.T
	db      *sql.DB
	drv     *postgres.Driver
	version string
	port    int
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
			pgTests.drivers[version] = &pgTest{db: db, drv: drv, version: version, port: port}
		}
	})
	for version, tt := range pgTests.drivers {
		t.Run(version, func(t *testing.T) {
			tt := &pgTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port}
			fn(tt)
		})
	}
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
			&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &postgres.ArrayType{T: "text[]"}, Null: true}, Default: &schema.Literal{V: "'{}'"}},
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
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "int[]", Type: &postgres.ArrayType{T: "int[]"}}, Default: &schema.Literal{V: "'{1}'"}},
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
		err := t.drv.ApplyChanges(ctx, []schema.Change{&schema.AddTable{T: usersT}})
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
		err = t.drv.ApplyChanges(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "add a new enum column to existing table")
		ensureNoChange(t, usersT)

		// Add a new value to an existing enum.
		e := usersT.Columns[1].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "tuesday")
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		err = t.drv.ApplyChanges(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append a value to existing enum")
		ensureNoChange(t, usersT)

		// Add multiple new values to an existing enum.
		e = usersT.Columns[1].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "wednesday", "thursday", "friday", "saturday")
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		err = t.drv.ApplyChanges(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append multiple values to existing enum")
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

func TestPostgres_Ent(t *testing.T) {
	pgRun(t, func(t *pgTest) {
		testEntIntegration(t, dialect.Postgres, t.db)
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
			testCLISchemaInspect(t, h, t.dsn(), postgres.UnmarshalSpec, schemahcl.New(schemahcl.WithTypes(postgres.TypeRegistry.Specs())))
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		pgRun(t, func(t *pgTest) {
			testCLISchemaApply(t, h, t.dsn())
		})
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
					Default: &schema.RawExpr{X: t.valueByVersion(map[string]string{"10": "B'100'::\"bit\""}, "'100'::\"bit\"")},
				},
				{
					Name:    "tBitVar",
					Type:    &schema.ColumnType{Type: &postgres.BitType{T: "bit varying", Len: 10}, Raw: "bit varying", Null: true},
					Default: &schema.RawExpr{X: t.valueByVersion(map[string]string{"10": "B'100'::\"bit\""}, "'100'::\"bit\"")},
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
					Default: &schema.RawExpr{X: "'atlas'::bpchar"},
				},
				{
					Name:    "tChar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character", Null: true},
					Default: &schema.RawExpr{X: "'atlas'::bpchar"},
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
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone"}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimeWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time with time zone"}, Raw: "time with time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimeWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone"}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "CURRENT_TIME"},
				},
				{
					Name:    "tTimestamp",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone"}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone"}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone"}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now()"},
				},
				{
					Name:    "tTimestampWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone"}, Raw: "timestamp without time zone", Null: true},
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
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint", Unsigned: false}, Raw: "smallint", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tSmallSerial_seq\"'::regclass)",
					},
				},
				{
					Name: "tSerial",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer", Unsigned: false}, Raw: "integer", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tSerial_seq\"'::regclass)",
					},
				},
				{
					Name: "tBigSerial",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tBigSerial_seq\"'::regclass)",
					},
				},
				{
					Name: "tSerial2",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "smallint", Unsigned: false}, Raw: "smallint", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tSerial2_seq\"'::regclass)",
					},
				},
				{
					Name: "tSerial4",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer", Unsigned: false}, Raw: "integer", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tSerial4_seq\"'::regclass)",
					},
				},
				{
					Name: "tSerial8",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "nextval('\"atlas_types_sanity_tSerial8_seq\"'::regclass)",
					},
				},
				{
					Name: "tArray",
					Type: &schema.ColumnType{Type: &postgres.ArrayType{T: "text[]"}, Raw: "ARRAY", Null: true},
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
					Type: &schema.ColumnType{Type: &postgres.UUIDType{T: "uuid"}, Raw: "uuid", Null: true},
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
					Type: &schema.ColumnType{Type: &schema.UnsupportedType{T: "interval"}, Raw: "interval", Null: true},
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
}

func (t *pgTest) dsn() string {
	return fmt.Sprintf("postgres://postgres:pass@localhost:%d/test?sslmode=disable", t.port)
}

func (t *pgTest) applyHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Schema
	err := postgres.UnmarshalSpec([]byte(spec), schemahcl.New(schemahcl.WithTypes(postgres.TypeRegistry.Specs())), &desired)
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
