// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type crdbTest struct {
	*testing.T
	db      *sql.DB
	drv     migrate.Driver
	rrw     migrate.RevisionReadWriter
	version string
	port    int
	once    sync.Once
}

var crdbTests = map[string]*crdbTest{
	"cockroach": {port: 26257},
}

func crdbRun(t *testing.T, fn func(*crdbTest)) {
	for version, tt := range crdbTests {
		if flagVersion == "" || flagVersion == version {
			t.Run(version, func(t *testing.T) {
				tt.once.Do(func() {
					var err error
					tt.version = version
					tt.rrw = &rrw{}
					tt.db, err = sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=root dbname=defaultdb password=pass sslmode=disable", tt.port))
					if err != nil {
						log.Fatalln(err)
					}
					dbs = append(dbs, tt.db) // close connection after all tests have been run
					tt.drv, err = postgres.Open(tt.db)
					if err != nil {
						log.Fatalln(err)
					}
				})
				tt := &crdbTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
				fn(tt)
			})
		}
	}
}

func TestCockroach_Executor(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testExecutor(t)
	})
}

func TestCockroach_AddDropTable(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testAddDrop(t)
	})
}

func TestCockroach_Relation(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testRelation(t)
	})
}

func TestCockroach_AddIndexedColumns(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		s := &schema.Schema{
			Name: "public",
		}
		usersT := &schema.Table{
			Name:    "users",
			Schema:  s,
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&postgres.Identity{}}}},
		}
		usersT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: usersT.Columns[0]}}}
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

func TestCockroach_AddColumns(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})
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
			&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Null: true}, Default: &schema.Literal{V: "false"}},
			&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"}, Null: true}, Default: &schema.Literal{V: "'POINT(1 2)'"}},
			&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"}, Null: true}, Default: &schema.Literal{V: "'LINESTRING(0 0, 1440 900)'"}},
			&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &postgres.ArrayType{Type: &schema.StringType{T: "text"}, T: "text[]"}, Null: true}, Default: &schema.Literal{V: "'{}'"}},
		)
		changes := t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 14)
		t.migrate(&schema.ModifyTable{T: usersT, Changes: changes})
		ensureNoChange(t, usersT)
	})
}

func TestCockroach_ColumnInt(t *testing.T) {
	ctx := context.Background()
	run := func(t *testing.T, change func(*schema.Column)) {
		crdbRun(t, func(t *crdbTest) {
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

	t.Run("ChangeDefault", func(t *testing.T) {
		run(t, func(c *schema.Column) {
			c.Default = &schema.RawExpr{X: "0"}
		})
	})
}

func TestCockroach_ColumnArray(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})

		// Add column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "bigint[]", Type: &postgres.ArrayType{Type: &schema.IntegerType{T: "bigint"}, T: "bigint[]"}}, Default: &schema.Literal{V: "'{1}'"}},
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

func TestCockroach_Enums(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
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
		e := usersT.Columns[2].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "tuesday")
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		err = t.drv.ApplyChanges(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append a value to existing enum")
		ensureNoChange(t, usersT)

		// Add multiple new values to an existing enum.
		e = usersT.Columns[2].Type.Type.(*schema.EnumType)
		e.Values = append(e.Values, "wednesday", "thursday", "friday", "saturday")
		changes = t.diff(t.loadUsers(), usersT)
		require.Len(t, changes, 1)
		err = t.drv.ApplyChanges(ctx, []schema.Change{&schema.ModifyTable{T: usersT, Changes: changes}})
		require.NoError(t, err, "append multiple values to existing enum")
		ensureNoChange(t, usersT)
	})
}

func TestCockroach_HCL(t *testing.T) {
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
	crdbRun(t, func(t *crdbTest) {
		testHCLIntegration(t, full, empty)
	})
}

func TestCockroach_HCL_Realm(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		t.dropSchemas("second")
		realm := t.loadRealm()
		hcl, err := postgres.MarshalHCL(realm)
		require.NoError(t, err)
		wa := string(hcl) + `
schema "second" {
}
`
		t.applyRealmHcl(wa)
		realm, err = t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{})
		require.NoError(t, err)
		_, ok := realm.Schema("public")
		require.True(t, ok)
		_, ok = realm.Schema("second")
		require.True(t, ok)
	})
}

func TestCockroach_CLI(t *testing.T) {
	h := `
			schema "public" {
			}
			table "users" {
				schema = schema.public
				column "id" {
					type = bigint
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaInspect(t, h, t.url(""), postgres.EvalHCL, "-s", "public")
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApply(t, h, t.url(""), "-s", "public")
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
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
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApply(t, h, t.url(""), "--var", "tenant=public", "-s", "public")
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaDiff(t, t.url(""))
		})
	})
	t.Run("SchemaApplyAutoApprove", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApplyAutoApprove(t, h, t.url(""), "-s", "public")
		})
	})
}

func TestCockroach_CLI_MultiSchema(t *testing.T) {
	h := `
			schema "public" {	
			}
			table "users" {
				schema = schema.public
				column "id" {
					type = bigint
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}
			schema "test2" {	
			}
			table "users" {
				schema = schema.test2
				column "id" {
					type = bigint
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			testCLIMultiSchemaInspect(t, h, t.url(""), []string{"public", "test2"}, postgres.EvalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			testCLIMultiSchemaApply(t, h, t.url(""), []string{"public", "test2"}, postgres.EvalHCL)
		})
	})
}

func TestCockroach_DefaultsHCL(t *testing.T) {
	n := "atlas_defaults"
	crdbRun(t, func(t *crdbTest) {
		ddl := `
create table atlas_defaults
(
	string varchar(255) default 'hello_world',
	quoted varchar(100) default 'never say "never"',
	tBit bit(10) default B'10101',
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

func TestCockroach_Sanity(t *testing.T) {
	n := "atlas_types_sanity"
	ddl := `
create table atlas_types_sanity
(
    "tBit"                 bit(10)                     default B'100'                                   null,
    "tBitVar"              bit varying(10)             default B'100'                                   null,
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
    "tInet"                inet                        default '127.0.0.1'                              null,
    "tGeometry"            geometry                    default                                          null,
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
    "tArray"               text[10]                     default '{}'                                    null,
    "tJSON"                json                         default '{"key":"value"}'                       null,
    "tJSONB"               jsonb                        default '{"key":"value"}'                       null,
    "tUUID"                uuid                         default  'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11' null,
    "tInterval"            interval                     default '4 hours'                               null
);
`
	crdbRun(t, func(t *crdbTest) {
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
					Default: &schema.RawExpr{X: "B'100'"},
				},
				{
					Name:    "tBitVar",
					Type:    &schema.ColumnType{Type: &postgres.BitType{T: "bit varying", Len: 10}, Raw: "bit varying", Null: true},
					Default: &schema.RawExpr{X: "B'100'"},
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
					Default: &schema.RawExpr{X: "'\\x01':::BYTES"},
				},
				{
					Name:    "tCharacter",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character", Null: true},
					Default: &schema.RawExpr{X: "'atlas':::STRING"},
				},
				{
					Name:    "tChar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character", Size: 10}, Raw: "character", Null: true},
					Default: &schema.RawExpr{X: "'atlas':::STRING"},
				},
				{
					Name:    "tCharVar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character varying", Size: 10}, Raw: "character varying", Null: true},
					Default: &schema.RawExpr{X: "'atlas':::STRING"},
				},
				{
					Name:    "tVarChar",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "character varying", Size: 10}, Raw: "character varying", Null: true},
					Default: &schema.RawExpr{X: "'atlas':::STRING"},
				},
				{
					Name:    "tText",
					Type:    &schema.ColumnType{Type: &schema.StringType{T: "text"}, Raw: "text", Null: true},
					Default: &schema.RawExpr{X: "'atlas':::STRING"},
				},
				{
					Name:    "tSmallInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}, Raw: "smallint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInteger",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tBigInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInt",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInt2",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "smallint"}, Raw: "smallint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInt4",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInt8",
					Type:    &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}, Raw: "bigint", Null: true},
					Default: &schema.RawExpr{X: "10:::INT8"},
				},
				{
					Name:    "tInet",
					Type:    &schema.ColumnType{Type: &postgres.NetworkType{T: "inet"}, Raw: "inet", Null: true},
					Default: &schema.RawExpr{X: "'127.0.0.1':::INET"},
				},
				{
					Name: "tGeometry",
					Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"}, Raw: "geometry", Null: true},
				},
				{
					Name:    "tDate",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "date"}, Raw: "date", Null: true},
					Default: &schema.RawExpr{X: "current_date()"},
				},
				{
					Name:    "tTime",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone", Precision: intp(6)}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "current_time():::TIME"},
				},
				{
					Name:    "tTimeWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time with time zone", Precision: intp(6)}, Raw: "time with time zone", Null: true},
					Default: &schema.RawExpr{X: "current_time():::TIMETZ"},
				},
				{
					Name:    "tTimeWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "time without time zone", Precision: intp(6)}, Raw: "time without time zone", Null: true},
					Default: &schema.RawExpr{X: "current_time():::TIME"},
				},
				{
					Name:    "tTimestamp",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(6)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now():::TIMESTAMP"},
				},
				{
					Name:    "tTimestampTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone", Precision: intp(6)}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now():::TIMESTAMPTZ"},
				},
				{
					Name:    "tTimestampWTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone", Precision: intp(6)}, Raw: "timestamp with time zone", Null: true},
					Default: &schema.RawExpr{X: "now():::TIMESTAMPTZ"},
				},
				{
					Name:    "tTimestampWOTZ",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(6)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now():::TIMESTAMP"},
				},
				{
					Name:    "tTimestampPrec",
					Type:    &schema.ColumnType{Type: &schema.TimeType{T: "timestamp without time zone", Precision: intp(4)}, Raw: "timestamp without time zone", Null: true},
					Default: &schema.RawExpr{X: "now():::TIMESTAMP"},
				},
				{
					Name:    "tDouble",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 53}, Raw: "double precision", Null: true},
					Default: &schema.RawExpr{X: "0.0:::FLOAT8"},
				},
				{
					Name:    "tReal",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 24}, Raw: "real", Null: true},
					Default: &schema.RawExpr{X: "0.0:::FLOAT8"},
				},
				{
					Name:    "tFloat8",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "double precision", Precision: 53}, Raw: "double precision", Null: true},
					Default: &schema.RawExpr{X: "0.0:::FLOAT8"},
				},
				{
					Name:    "tFloat4",
					Type:    &schema.ColumnType{Type: &schema.FloatType{T: "real", Precision: 24}, Raw: "real", Null: true},
					Default: &schema.RawExpr{X: "0.0:::FLOAT8"},
				},
				{
					Name:    "tNumeric",
					Type:    &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 0}, Raw: "numeric", Null: true},
					Default: &schema.RawExpr{X: "0:::DECIMAL"},
				},
				{
					Name:    "tDecimal",
					Type:    &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 0}, Raw: "numeric", Null: true},
					Default: &schema.RawExpr{X: "0:::DECIMAL"},
				},
				{
					Name: "tSmallSerial",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tSerial",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tBigSerial",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tSerial2",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tSerial4",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tSerial8",
					Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint", Unsigned: false}, Raw: "bigint", Null: false},
					Default: &schema.RawExpr{
						X: "unique_rowid()",
					},
				},
				{
					Name: "tArray",
					Type: &schema.ColumnType{Type: &postgres.ArrayType{Type: &schema.StringType{T: "text"}, T: "text[]"}, Raw: "ARRAY", Null: true},
					Default: &schema.RawExpr{
						X: "ARRAY[]:::STRING[]",
					},
				},
				{
					Name: "tJSON",
					Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}, Raw: "jsonb", Null: true},
					Default: &schema.RawExpr{
						X: "'{\"key\": \"value\"}':::JSONB",
					},
				},
				{
					Name: "tJSONB",
					Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}, Raw: "jsonb", Null: true},
					Default: &schema.RawExpr{
						X: "'{\"key\": \"value\"}':::JSONB",
					},
				},
				{
					Name: "tUUID",
					Type: &schema.ColumnType{Type: &postgres.UUIDType{T: "uuid"}, Raw: "uuid", Null: true},
					Default: &schema.RawExpr{
						X: "'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11':::UUID",
					},
				},
				{
					Name: "tInterval",
					Type: &schema.ColumnType{Type: &postgres.IntervalType{T: "interval", Precision: intp(6)}, Raw: "interval", Null: true},
					Default: &schema.RawExpr{
						X: "'04:00:00':::INTERVAL",
					},
				},
			},
		}
		for i, c := range expected.Columns {
			require.EqualValues(t, ts.Columns[i], c, c.Name)
		}
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testImplicitIndexes(t, t.db)
		})
	})
}

func (t *crdbTest) url(_ string) string {
	return fmt.Sprintf("postgres://root:pass@localhost:%d/defaultdb?sslmode=disable", t.port)
}

func (t *crdbTest) driver() migrate.Driver {
	return t.drv
}

func (t *crdbTest) revisionsStorage() migrate.RevisionReadWriter {
	return t.rrw
}

func (t *crdbTest) applyHcl(spec string) {
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

func (t *crdbTest) valueByVersion(values map[string]string, defaults string) string {
	if v, ok := values[t.version]; ok {
		return v
	}
	return defaults
}

func (t *crdbTest) loadRealm() *schema.Realm {
	r, err := t.drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Schemas: []string{"public"},
	})
	require.NoError(t, err)
	return r
}

func (t *crdbTest) loadUsers() *schema.Table {
	return t.loadTable("users")
}

func (t *crdbTest) loadPosts() *schema.Table {
	return t.loadTable("posts")
}

func (t *crdbTest) loadTable(name string) *schema.Table {
	realm := t.loadRealm()
	require.Len(t, realm.Schemas, 1)
	table, ok := realm.Schemas[0].Table(name)
	require.True(t, ok)
	return table
}

func (t *crdbTest) users() *schema.Table {
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

func (t *crdbTest) posts() *schema.Table {
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

func (t *crdbTest) realm() *schema.Realm {
	r := &schema.Realm{
		Schemas: []*schema.Schema{
			{
				Name: "public",
			},
		},
		Attrs: []schema.Attr{
			&schema.Collation{V: "C.UTF-8"},
			&postgres.CType{V: "C.UTF-8"},
		},
	}
	r.Schemas[0].Realm = r
	return r
}

func (t *crdbTest) diff(t1, t2 *schema.Table) []schema.Change {
	changes, err := t.drv.TableDiff(t1, t2)
	require.NoError(t, err)
	return changes
}

func (t *crdbTest) migrate(changes ...schema.Change) {
	err := t.drv.ApplyChanges(context.Background(), changes)
	require.NoError(t, err)
}

func (t *crdbTest) dropTables(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP TABLE IF EXISTS " + strings.Join(names, ", "))
		require.NoError(t.T, err, "drop tables %q", names)
	})
}

func (t *crdbTest) dropSchemas(names ...string) {
	t.Cleanup(func() {
		_, err := t.db.Exec("DROP SCHEMA IF EXISTS " + strings.Join(names, ", ") + " CASCADE")
		require.NoError(t.T, err, "drop schema %q", names)
	})
}
func (t *crdbTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := postgres.EvalHCLBytes([]byte(spec), &desired, nil)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}
