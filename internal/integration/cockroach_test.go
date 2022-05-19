// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"ariga.io/atlas/sql/cockroach"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	"entgo.io/ent/dialect"
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
}

var crdbTests = struct {
	drivers map[string]*crdbTest
	ports   map[string]int
}{
	drivers: make(map[string]*crdbTest),
	ports: map[string]int{
		"cockroach": 26257,
	},
}

func crdbInit(d string) []io.Closer {
	var cs []io.Closer
	if d != "" {
		p, ok := crdbTests.ports[d]
		if ok {
			crdbTests.ports = map[string]int{d: p}
		} else {
			crdbTests.ports = make(map[string]int)
		}
	}
	for version, port := range crdbTests.ports {
		db, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=root dbname=defaultdb password=pass sslmode=disable", port))
		if err != nil {
			log.Fatalln(err)
		}
		cs = append(cs, db)
		drv, err := cockroach.Open(db)
		if err != nil {
			log.Fatalln(err)
		}
		crdbTests.drivers[version] = &crdbTest{db: db, drv: drv, version: version, port: port, rrw: &rrw{}}
	}
	return cs
}

func crdbRun(t *testing.T, fn func(*crdbTest)) {
	for version, tt := range crdbTests.drivers {
		t.Run(version, func(t *testing.T) {
			tt := &crdbTest{T: t, db: tt.db, drv: tt.drv, version: version, port: tt.port, rrw: tt.rrw}
			fn(tt)
		})
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
			Columns: []*schema.Column{{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&cockroach.Identity{}}}},
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
			&schema.Column{Name: "j", Type: &schema.ColumnType{Type: &cockroach.SerialType{T: "serial"}}},
			&schema.Column{Name: "m", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Null: true}, Default: &schema.Literal{V: "false"}},
			&schema.Column{Name: "n", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"}, Null: true}, Default: &schema.Literal{V: "'POINT(1 2)'"}},
			&schema.Column{Name: "o", Type: &schema.ColumnType{Type: &schema.SpatialType{T: "geometry"}, Null: true}, Default: &schema.Literal{V: "'LINESTRING(0 0, 1440 900)'"}},
			&schema.Column{Name: "q", Type: &schema.ColumnType{Type: &cockroach.ArrayType{T: "text[]"}, Null: true}, Default: &schema.Literal{V: "'{}'"}},
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

func TestCockroach_ColumnArray(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		usersT := t.users()
		t.dropTables(usersT.Name)
		t.migrate(&schema.AddTable{T: usersT})

		// Add column.
		usersT.Columns = append(
			usersT.Columns,
			&schema.Column{Name: "a", Type: &schema.ColumnType{Raw: "int[]", Type: &cockroach.ArrayType{T: "int[]"}}, Default: &schema.Literal{V: "'{1}'"}},
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

func TestCockroach_ForeignKey(t *testing.T) {
	t.Run("ChangeAction", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
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
		crdbRun(t, func(t *crdbTest) {
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
		crdbRun(t, func(t *crdbTest) {
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

func TestCockroach_Ent(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testEntIntegration(t, dialect.Postgres, t.db)
	})
}

func TestCockroach_AdvisoryLock(t *testing.T) {
	crdbRun(t, func(t *crdbTest) {
		testAdvisoryLock(t.T, t.drv.(schema.Locker))
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
		hcl, err := mysql.MarshalHCL(realm)
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

func (t *crdbTest) applyRealmHcl(spec string) {
	realm := t.loadRealm()
	var desired schema.Realm
	err := cockroach.UnmarshalHCL([]byte(spec), &desired)
	require.NoError(t, err)
	diff, err := t.drv.RealmDiff(realm, &desired)
	require.NoError(t, err)
	err = t.drv.ApplyChanges(context.Background(), diff)
	require.NoError(t, err)
}

func TestCockroach_CLI(t *testing.T) {
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
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaInspect(t, h, t.dsn(), cockroach.UnmarshalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApply(t, h, t.dsn())
		})
	})
	t.Run("SchemaApplyDryRun", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApplyDry(t, h, t.dsn())
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
			testCLISchemaApply(t, h, t.dsn(), "--var", "tenant=public")
		})
	})
	t.Run("SchemaDiffRun", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaDiff(t, t.dsn())
		})
	})
	t.Run("SchemaApplyAutoApprove", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testCLISchemaApplyAutoApprove(t, h, t.dsn())
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
					type = integer
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
					type = integer
				}
				primary_key {
					columns = [table.users.column.id]
				}
			}`
	t.Run("SchemaInspect", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			testCLIMultiSchemaInspect(t, h, t.dsn(), []string{"public", "test2"}, cockroach.UnmarshalHCL)
		})
	})
	t.Run("SchemaApply", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			t.dropSchemas("test2")
			t.dropTables("users")
			testCLIMultiSchemaApply(t, h, t.dsn(), []string{"public", "test2"}, cockroach.UnmarshalHCL)
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
		spec, err := cockroach.MarshalHCL(realm.Schemas[0])
		require.NoError(t, err)
		var s schema.Schema
		err = cockroach.UnmarshalHCL(spec, &s)
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
    "tGeom"                geometry                       default                                       null,
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
					Type:    &schema.ColumnType{Type: &cockroach.BitType{T: "bit", Len: 10}, Raw: "bit", Null: true},
					Default: &schema.RawExpr{X: "B'100'"},
				},
				{
					Name:    "tBitVar",
					Type:    &schema.ColumnType{Type: &cockroach.BitType{T: "bit varying", Len: 10}, Raw: "bit varying", Null: true},
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
					Type:    &schema.ColumnType{Type: &cockroach.NetworkType{T: "inet"}, Raw: "inet", Null: true},
					Default: &schema.RawExpr{X: "'127.0.0.1':::INET"},
				},
				{
					Name: "tGeom",
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
				// all serial types are the same in cockroach: https://www.cockroachlabs.com/docs/v21.2/serial.html#modes-of-operation
				// https://github.com/cockroachdb/cockroach/issues/75927#issuecomment-1029163946
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
					Type: &schema.ColumnType{Type: &cockroach.ArrayType{T: "text[]"}, Raw: "ARRAY", Null: true},
					Default: &schema.RawExpr{
						X: "ARRAY[]:::STRING[]",
					},
				},
				// json is alias for jsonb see: https://www.cockroachlabs.com/docs/v21.2/jsonb.html
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
					Type: &schema.ColumnType{Type: &cockroach.UUIDType{T: "uuid"}, Raw: "uuid", Null: true},
					Default: &schema.RawExpr{
						X: "'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11':::UUID",
					},
				},
				{
					Name: "tInterval",
					Type: &schema.ColumnType{Type: &schema.UnsupportedType{T: "interval"}, Raw: "interval", Null: true},
					Default: &schema.RawExpr{
						X: "'04:00:00':::INTERVAL",
					},
				},
			},
		}
		// cockroach automatically adds another column and makes it the row_id, we ignore that for this test:
		ts.PrimaryKey = nil
		ts.Indexes = nil
		ts.Columns = ts.Columns[:len(expected.Columns)]
		require.EqualValues(t, &expected, ts)
	})

	t.Run("ImplicitIndexes", func(t *testing.T) {
		crdbRun(t, func(t *crdbTest) {
			testImplicitIndexes(t, t.db)
		})
	})
}

func (t *crdbTest) dsn() string {
	return fmt.Sprintf("postgres://postgres:pass@localhost:%d/test?sslmode=disable", t.port)
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
	err := cockroach.UnmarshalHCL([]byte(spec), &desired)
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
				Attrs: []schema.Attr{&cockroach.Identity{}},
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
				Attrs: []schema.Attr{&cockroach.Identity{}},
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

func (t *crdbTest) revisions() *schema.Table {
	versionsT := &schema.Table{
		Name: "atlas_schema_revisions",
		Columns: []*schema.Column{
			{Name: "version", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "description", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "execution_state", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "executed_at", Type: &schema.ColumnType{Type: &schema.TimeType{T: "timestamp with time zone"}}},
			{Name: "execution_time", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "bigint"}}},
			{Name: "hash", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "operator_version", Type: &schema.ColumnType{Type: &schema.StringType{T: "character varying"}}},
			{Name: "meta", Type: &schema.ColumnType{Type: &schema.JSONType{T: "jsonb"}, Raw: "jsonb"}},
		},
	}
	versionsT.PrimaryKey = &schema.Index{Parts: []*schema.IndexPart{{C: versionsT.Columns[0]}}}
	return versionsT
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
			&cockroach.CType{V: "C.UTF-8"},
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
