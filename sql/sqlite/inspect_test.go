// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDriver_InspectTable(t *testing.T) {
	tests := []struct {
		name   string
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "table columns",
			before: func(m mock) {
				m.tableExists("users", true, "CREATE TABLE users(id INTEGER PRIMARY KEY AUTOINCREMENT, w INT GENERATED ALWAYS AS (a*10), x TEXT AS (typeof(c)) STORED, y TEXT AS (substr(b,a,a+2)))")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 name |   type       | nullable | dflt_value  | primary  | hidden
------+--------------+----------+ ------------+----------+----------
 c1   | int           |  1      |     a       |  0       |  0
 c2   | integer       |  0      |     97      |  0       |  0
 c3   | varchar(100)  |  1      |    'A'      |  0       |  0
 c4   | boolean       |  0      |             |  0       |  0
 c5   | json          |  0      |             |  0       |  0
 c6   | datetime      |  0      |             |  0       |  0
 c7   | blob          |  0      |    x'a'     |  0       |  0
 c8   | text          |  0      |             |  0       |  0
 c9   | numeric(10,2) |  0      |             |  0       |  0
 c10  | real          |  0      |             |  0       |  0
 c11  |               |  0      |             |  0       |  0
 w    | int           |  0      |             |  0       |  2
 x    | text          |  0      |             |  0       |  3
 y    | text          |  0      |             |  0       |  2
 id   | integer       |  0      |     0x1     |  1       |  0
`))
				m.noIndexes("users")
				m.noFKs("users")
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				columns := []*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Null: true, Type: &schema.IntegerType{T: "int"}, Raw: "int"}, Default: &schema.RawExpr{X: "a"}},
					{Name: "c2", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer"}, Default: &schema.Literal{V: "97"}},
					{Name: "c3", Type: &schema.ColumnType{Null: true, Type: &schema.StringType{T: "varchar", Size: 100}, Raw: "varchar(100)"}, Default: &schema.Literal{V: "'A'"}},
					{Name: "c4", Type: &schema.ColumnType{Type: &schema.BoolType{T: "boolean"}, Raw: "boolean"}},
					{Name: "c5", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}, Raw: "json"}},
					{Name: "c6", Type: &schema.ColumnType{Type: &schema.TimeType{T: "datetime"}, Raw: "datetime"}},
					{Name: "c7", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}, Raw: "blob"}, Default: &schema.Literal{V: "x'a'"}},
					{Name: "c8", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Raw: "text"}},
					{Name: "c9", Type: &schema.ColumnType{Type: &schema.DecimalType{T: "numeric", Precision: 10, Scale: 2}, Raw: "numeric(10,2)"}},
					{Name: "c10", Type: &schema.ColumnType{Type: &schema.FloatType{T: "real"}, Raw: "real"}},
					{Name: "c11", Type: &schema.ColumnType{Type: &schema.BinaryType{T: "blob"}, Raw: ""}},
					{Name: "w", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "int"}, Raw: "int"}, Attrs: []schema.Attr{&schema.GeneratedExpr{Type: "VIRTUAL", Expr: "(a*10)"}}},
					{Name: "x", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Raw: "text"}, Attrs: []schema.Attr{&schema.GeneratedExpr{Type: "STORED", Expr: "(typeof(c))"}}},
					{Name: "y", Type: &schema.ColumnType{Type: &schema.StringType{T: "text"}, Raw: "text"}, Attrs: []schema.Attr{&schema.GeneratedExpr{Type: "VIRTUAL", Expr: "(substr(b,a,a+2))"}}},
					{Name: "id", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer"}, Attrs: []schema.Attr{&AutoIncrement{}}, Default: &schema.Literal{V: "0x1"}},
				}
				require.Equal(t.Columns, columns)
				require.EqualValues(&schema.Index{
					Name:   "PRIMARY",
					Unique: true,
					Table:  t,
					Parts:  []*schema.IndexPart{{SeqNo: 1, C: columns[len(columns)-1]}},
					Attrs:  []schema.Attr{&AutoIncrement{}},
				}, t.PrimaryKey)
			},
		},
		{
			name: "table indexes",
			before: func(m mock) {
				m.tableExists("users", true, "CREATE TABLE users(id INTEGER PRIMARY KEY)")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 name |   type       | nullable | dflt_value  | primary  | hidden
------+--------------+----------+ ------------+----------+----------
 c1   | int           |  1      |             |  0       |  0
 c2   | integer       |  0      |             |  0       |  0
 c3   | json          |  0      |             |  0       |  0
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 name  |   unique     | origin | partial  |                      sql 
-------+--------------+--------+----------+-------------------------------------------------------
 c1u   |  1           |  c     |  0       | CREATE UNIQUE INDEX c1u on users(c1, c2)
 c1_c2 |  0           |  c     |  1       | CREATE INDEX c1_c2 on users(c1, c2*2) WHERE c1 <> NULL
 c1_x  |  0           |  c     |  0       | CREATE INDEX c1_x ON users (f(c1))
 c3_x  |  0           |  c     |  0       | CREATE INDEX c3_x ON users (json_extract(c3, '$.x') desc, json_extract(c3, '$.y') desc)
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexColumnsQuery, "c1u"))).
					WillReturnRows(sqltest.Rows(`
 name  |   desc |
-------+--------+
 c1   |  1      |
 c2   |  0      |
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexColumnsQuery, "c1_c2"))).
					WillReturnRows(sqltest.Rows(`
 name  |   desc |     
-------+--------+     
 c1    |  0     |     
 nil   |  0     |     
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexColumnsQuery, "c1_x"))).
					WillReturnRows(sqltest.Rows(`
 name  |   desc |
-------+--------+
 nil   |  0     |
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexColumnsQuery, "c3_x"))).
					WillReturnRows(sqltest.Rows(`
 name  |   desc |
-------+--------+
 nil   |  1     |
 nil   |  1     |
`))
				m.noFKs("users")
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				columns := []*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Null: true, Type: &schema.IntegerType{T: "int"}, Raw: "int"}},
					{Name: "c2", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer"}},
					{Name: "c3", Type: &schema.ColumnType{Type: &schema.JSONType{T: "json"}, Raw: "json"}},
				}
				indexes := []*schema.Index{
					{
						Name:   "c1u",
						Unique: true,
						Table:  t,
						Parts: []*schema.IndexPart{
							{SeqNo: 1, C: columns[0], Desc: true},
							{SeqNo: 2, C: columns[1]},
						},
						Attrs: []schema.Attr{
							&CreateStmt{S: "CREATE UNIQUE INDEX c1u on users(c1, c2)"},
							&IndexOrigin{O: "c"},
						},
					},
					{
						Name:  "c1_c2",
						Table: t,
						Parts: []*schema.IndexPart{
							{SeqNo: 1, C: columns[0]},
							{SeqNo: 2, X: &schema.RawExpr{X: "c2*2"}},
						},
						Attrs: []schema.Attr{
							&CreateStmt{S: "CREATE INDEX c1_c2 on users(c1, c2*2) WHERE c1 <> NULL"},
							&IndexOrigin{O: "c"},
							&IndexPredicate{P: "c1 <> NULL"},
						},
					},
					{
						Name:  "c1_x",
						Table: t,
						Parts: []*schema.IndexPart{
							{SeqNo: 1, X: &schema.RawExpr{X: "f(c1)"}},
						},
						Attrs: []schema.Attr{
							&CreateStmt{S: "CREATE INDEX c1_x ON users (f(c1))"},
							&IndexOrigin{O: "c"},
						},
					},
					{
						Name:  "c3_x",
						Table: t,
						Parts: []*schema.IndexPart{
							{SeqNo: 1, X: &schema.RawExpr{X: "json_extract(c3, '$.x')"}, Desc: true},
							{SeqNo: 2, X: &schema.RawExpr{X: "json_extract(c3, '$.y')"}, Desc: true},
						},
						Attrs: []schema.Attr{
							&CreateStmt{S: "CREATE INDEX c3_x ON users (json_extract(c3, '$.x') desc, json_extract(c3, '$.y') desc)"},
							&IndexOrigin{O: "c"},
						},
					},
				}
				require.Equal(t.Columns, columns)
				require.Equal(t.Indexes, indexes)
			},
		},
		{
			name: "table constraints",
			before: func(m mock) {
				m.tableExists("users", true, `
CREATE TABLE users(
	id INTEGER PRIMARY KEY,
	c1 int CHECK (c1 > 10),
	c2 integer NOT NULL CONSTRAINT c2_fk REFERENCES users (c1) ON DELETE SET NULL constraint "ck1" CHECK ((c1 + c2) % 2 = 0),
	c3 integer NOT NULL REFERENCES users (c1) ON DELETE SET NULL,
	CONSTRAINT "c1_c2_fk" FOREIGN KEY (c1, c2) REFERENCES t2 (id, c1),
	CONSTRAINT "id_nonzero" CHECK (id <> 0)
)
`)
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 name |   type       | nullable | dflt_value  | primary  | hidden
------+--------------+----------+ ------------+----------+----------
 c1   | int           |  1      |             |  0       |  0
 c2   | integer       |  0      |             |  0       |  0
 c3   | integer       |  0      |             |  0       |  0
`))
				m.noIndexes("users")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 id |   from    | to | table  | on_update   | on_delete   
----+-----------+-------------+-------------+-----------
 0  | c1        | id | t2     |  NO ACTION  | CASCADE
 0  | c2        | c1 | t2     |  NO ACTION  | CASCADE
 1  | c2        | c1 | users  |  NO ACTION  | CASCADE
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				fks := []*schema.ForeignKey{
					{Symbol: "c1_c2_fk", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: &schema.Table{Name: "t2", Schema: &schema.Schema{Name: "main"}}, RefColumns: []*schema.Column{{Name: "id"}, {Name: "c1"}}},
					{Symbol: "c2_fk", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: t},
				}
				columns := []*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Null: true, Type: &schema.IntegerType{T: "int"}, Raw: "int"}, ForeignKeys: fks[:1]},
					{Name: "c2", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer"}, ForeignKeys: fks},
					{Name: "c3", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "integer"}, Raw: "integer"}},
				}
				fks[0].Columns = columns[:2]
				fks[1].Columns = columns[1:2]
				fks[1].RefColumns = columns[:1]
				checks := []schema.Attr{
					&schema.Check{Expr: "(c1 > 10)"},
					&schema.Check{Name: "ck1", Expr: "((c1 + c2) % 2 = 0)"},
					&schema.Check{Name: "id_nonzero", Expr: "(id <> 0)"},
				}
				require.Equal(t.Columns, columns)
				require.Equal(t.ForeignKeys, fks)
				require.Equal(t.Attrs[1:], checks)
			},
		},
		{
			name: "table options",
			before: func(m mock) {
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(databasesQueryArgs, "?"))).
					WithArgs("main").
					WillReturnRows(sqltest.Rows(`
 name |   file
------+-----------
 main |
`))
				rows := sqlmock.NewRows([]string{"name", "sql", "wr", "strict"})
				rows.AddRow("users", "CREATE TABLE users(id INTEGER PRIMARY KEY) without rowid, strict", 1, 1)
				m.ExpectQuery(sqltest.Escape(tablesQuery + " AND sqlite_master.name IN (?)")).
					WithArgs("users").
					WillReturnRows(rows)
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, "users"))).
					WillReturnRows(sqltest.Rows(`
 name |   type       | nullable | dflt_value  | primary  | hidden
------+--------------+----------+ ------------+----------+----------
 id   | int          |  0       |             |  1       |  0
`))
				m.noIndexes("users")
				m.noFKs("users")
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Len(t.Attrs, 3)
				require.Equal(t.Attrs[1], &WithoutRowID{})
				require.Equal(t.Attrs[2], &Strict{})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			mk := mock{m}
			mk.systemVars("3.36.0")
			drv, err := Open(db)
			require.NoError(t, err)
			tt.before(mk)
			s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{
				Tables: []string{"users"},
				Mode:   ^schema.InspectViews,
			})
			require.NoError(t, err)
			tt.expect(require.New(t), s.Tables[0], err)
		})
	}
}

func TestRegex_TableFK(t *testing.T) {
	tests := []struct {
		input   string
		matches []string
	}{
		{
			input:   `CREATE TABLE pets (id int NOT NULL, owner_id int, CONSTRAINT "owner_fk" FOREIGN KEY(owner_id) REFERENCES users(id))`,
			matches: []string{"owner_fk", "owner_id", "users", "id"},
		},
		{
			input:   `CREATE TABLE pets (id int NOT NULL, owner_id int, CONSTRAINT "owner_fk" FOREIGN KEY (owner_id) REFERENCES users(id))`,
			matches: []string{"owner_fk", "owner_id", "users", "id"},
		},
		{
			input: `
CREATE TABLE pets (
id int NOT NULL,
owner_id int,
CONSTRAINT owner_fk
	FOREIGN KEY ("owner_id") REFERENCES "users" (id)
)`,
			matches: []string{"owner_fk", `"owner_id"`, "users", "id"},
		},
		{
			input: `
CREATE TABLE pets (
id int NOT NULL,
c int,
d int,
CONSTRAINT "c_d_fk" FOREIGN KEY (c, d) REFERENCES "users" (a, b)
)`,
			matches: []string{"c_d_fk", "c, d", "users", "a, b"},
		},
		{
			input:   `CREATE TABLE pets (id int NOT NULL,c int,d int,CONSTRAINT "c_d_fk" FOREIGN KEY (c, "d") REFERENCES "users" (a, "b"))`,
			matches: []string{"c_d_fk", `c, "d"`, "users", `a, "b"`},
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL,c int,d int,CONSTRAINT FOREIGN KEY (c, "d") REFERENCES "users" (a, "b"))`,
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL,c int,d int,CONSTRAINT name FOREIGN KEY c REFERENCES "users" (a, "b"))`,
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL,c int,d int,CONSTRAINT name FOREIGN KEY c REFERENCES (a, "b"))`,
		},
	}
	for _, tt := range tests {
		m := reFKT.FindStringSubmatch(tt.input)
		require.Equal(t, len(m) != 0, len(tt.matches) != 0)
		if len(m) > 0 {
			require.Equal(t, tt.matches, m[1:])
		}
	}
}

func TestRegex_ColumnFK(t *testing.T) {
	tests := []struct {
		input   string
		matches []string
	}{
		{
			input:   `CREATE TABLE pets (id int NOT NULL, owner_id int CONSTRAINT "owner_fk" REFERENCES users(id))`,
			matches: []string{"owner_id", "owner_fk", "users", "id"},
		},
		{
			input:   `CREATE TABLE pets (id int NOT NULL, owner_id int CONSTRAINT "owner_fk" REFERENCES users(id))`,
			matches: []string{"owner_id", "owner_fk", "users", "id"},
		},
		{
			input: `
CREATE TABLE pets (
	id int NOT NULL,
	c int REFERENCES users(id),
	d int CONSTRAINT "dfk" REFERENCES users(id)
)`,
			matches: []string{"d", "dfk", "users", "id"},
		},
		{
			input: `
CREATE TABLE t1 (
	c int REFERENCES users(id),
	d text CONSTRAINT "dfk" CHECK (d <> '') REFERENCES t2(d)
)`,
		},
	}
	for _, tt := range tests {
		m := reFKC.FindStringSubmatch(tt.input)
		require.Equal(t, len(m) != 0, len(tt.matches) != 0)
		if len(m) > 0 {
			require.Equal(t, tt.matches, m[1:])
		}
	}
}

func TestRegex_Checks(t *testing.T) {
	tests := []struct {
		input  string
		checks []*schema.Check
	}{
		{
			input: `CREATE TABLE pets (id int NOT NULL, owner_id int CONSTRAINT "ck1" CHECK (owner_id <> 0))`,
			checks: []*schema.Check{
				{Name: "ck1", Expr: "(owner_id <> 0)"},
			},
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL, owner_id int CHECK (owner_id <> 0) CONSTRAINT "ck1")`,
			checks: []*schema.Check{
				{Expr: "(owner_id <> 0)"},
			},
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL CHECK ("id" <> 0), owner_id int CONSTRAINT "ck1" CHECK ((owner_id) <> 0))`,
			checks: []*schema.Check{
				{Expr: `("id" <> 0)`},
				{Name: "ck1", Expr: "((owner_id) <> 0)"},
			},
		},
		{
			input: `CREATE TABLE pets (id int NOT NULL CHECK ("(" <> ')'), owner_id int CONSTRAINT "ck1" CHECK ((owner_id) <> 0))`,
			checks: []*schema.Check{
				{Expr: `("(" <> ')')`},
				{Name: "ck1", Expr: "((owner_id) <> 0)"},
			},
		},
		{
			input: "CREATE TABLE pets (\n\tid int NOT NULL CHECK (id <> 0) CHECK ((id % 2) = 0)\n,\n\towner_id int CHECK ((owner_id) <> 0)\n)",
			checks: []*schema.Check{
				{Expr: "(id <> 0)"},
				{Expr: "((id % 2) = 0)"},
				{Expr: "((owner_id) <> 0)"},
			},
		},
		{
			input: `CREATE TABLE t1(
				x INTEGER CHECK( x<5 ),
				y REAL CHECK( y>x ))`,
			checks: []*schema.Check{
				{Expr: "( x<5 )"},
				{Expr: "( y>x )"},
			},
		},
		{
			input: `CREATE TABLE t(
				x INTEGER CONSTRAINT one CHECK( typeof(coalesce(x,0))=="integer" ),
				y NUMERIC CONSTRAINT two CHECK( typeof(coalesce(y,0.1))=='real' ),
				z TEXT CONSTRAINT three CHECK( typeof(coalesce(z,''))=='text' )
			)`,
			checks: []*schema.Check{
				{Name: "one", Expr: `( typeof(coalesce(x,0))=="integer" )`},
				{Name: "two", Expr: `( typeof(coalesce(y,0.1))=='real' )`},
				{Name: "three", Expr: `( typeof(coalesce(z,''))=='text' )`},
			},
		},
		{
			input: `CREATE TABLE t(
				x char check ('foo''(' <> 1)
			)`,
			checks: []*schema.Check{
				{Expr: `('foo''(' <> 1)`},
			},
		},
		// Invalid inputs.
		{
			input: "CREATE TABLE t(x char check)",
		},
		{
			input: "CREATE TABLE t(x char constraint x check)",
		},
	}
	for _, tt := range tests {
		const name = "users"
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		mk := mock{m}
		mk.systemVars("3.36.0")
		mk.tableExists(name, true, tt.input)
		mk.noColumns(name)
		mk.noIndexes(name)
		mk.noFKs(name)
		drv, err := Open(db)
		require.NoError(t, err)
		s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{
			Tables: []string{name},
			Mode:   ^schema.InspectViews,
		})
		require.NoError(t, err)
		table := s.Tables[0]
		require.Equal(t, len(table.Attrs[1:]), len(tt.checks))
		for i := range tt.checks {
			require.Equal(t, tt.checks[i], table.Attrs[i+1])
		}
	}
}

func TestRegex_GeneratedExpr(t *testing.T) {
	tests := []struct {
		input  string
		column *schema.Column
	}{
		{
			input: "CREATE TABLE t1( a NOT NULL DEFAULT 'aaa', b AS(c) NOT NULL, c NOT NULL DEFAULT 'ccc');",
			column: schema.NewColumn("b").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(c)", Type: "VIRTUAL"}),
		},
		{
			input: "CREATE TABLE t4(a NOT NULL DEFAULT 123, b AS(a*10+4) STORED UNIQUE);",
			column: schema.NewColumn("b").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(a*10+4)", Type: "VIRTUAL"}),
		},
		{
			input: "CREATE TABLE t1(aa , bb AS (17) UNIQUE);",
			column: schema.NewColumn("bb").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(17)", Type: "VIRTUAL"}),
		},
		{
			input: "CREATE TABLE t1( a integer primary key, b int generated always as (a+5), c text GENERATED   ALWAYS as (printf('%08x',a)));",
			column: schema.NewColumn("c").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(printf('%08x',a))", Type: "VIRTUAL"}),
		},
		{
			input: "CREATE TABLE t1( a integer primary key, b int generated always as (a+5), c text GENERATED   ALWAYS as (printf('%08x',a)), d Generated\nAlways\nAS\n('xy\\()zzy'));",
			column: schema.NewColumn("d").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "('xy\\()zzy')", Type: "VIRTUAL"}),
		},
		{
			input: "CREATE TABLE t0( c0 AS (('a', 9) < ('b', c1)), c1 AS (1), c2 CHECK (1 = c1) );",
			column: schema.NewColumn("c0").
				SetGeneratedExpr(&schema.GeneratedExpr{Expr: "(('a', 9) < ('b', c1))", Type: "VIRTUAL"}),
		},
	}
	for _, tt := range tests {
		const name = "users"
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		mk := mock{m}
		mk.systemVars("3.36.0")
		mk.tableExists(name, true, tt.input)
		m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, name))).
			WillReturnRows(sqltest.Rows(fmt.Sprintf(`
 name |   type       | nullable | dflt_value  | primary  | hidden
------+--------------+----------+ ------------+----------+----------
 %s   | int           |  1      |     a       |  0       |  2
`, tt.column.Name)))
		mk.noIndexes(name)
		mk.noFKs(name)
		drv, err := Open(db)
		require.NoError(t, err)
		s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{
			Tables: []string{name},
			Mode:   ^schema.InspectViews,
		})
		require.NoError(t, err)
		require.Equal(t, tt.column.Attrs, s.Tables[0].Columns[0].Attrs)
	}
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) systemVars(version string) {
	m.ExpectQuery(sqltest.Escape("SELECT sqlite_version()")).
		WillReturnRows(sqltest.Rows(`
     version    
----------------
 ` + version + `
`))
	m.ExpectQuery(sqltest.Escape("SELECT name FROM pragma_collation_list()")).
		WillReturnRows(sqltest.Rows(`
  pragma_collation_list   
------------------------
      decimal
      uint
      RTRIM
      NOCASE
      BINARY
`))
}

func (m mock) tableExists(table string, exists bool, stmt ...string) {
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(databasesQueryArgs, "?"))).
		WithArgs("main").
		WillReturnRows(sqltest.Rows(`
 name |   file    
------+-----------
 main |   
`))
	rows := sqlmock.NewRows([]string{"name", "sql", "wr", "strict"})
	if exists {
		rows.AddRow(table, stmt[0], nil, nil)
	}
	m.ExpectQuery(sqltest.Escape(tablesQuery + " AND sqlite_master.name IN (?)")).
		WithArgs(table).
		WillReturnRows(rows)
}

func (m mock) noColumns(table string) {
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, table))).
		WillReturnRows(sqlmock.NewRows([]string{"name", "type", "nullable", "dflt_value", "primary"}))
}

func (m mock) noIndexes(table string) {
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesQuery, table))).
		WillReturnRows(sqlmock.NewRows([]string{"name", "unique", "origin", "partial", "sql"}))
}

func (m mock) noFKs(table string) {
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, table))).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from", "to", "table", "on_update", "on_delete"}))
}
