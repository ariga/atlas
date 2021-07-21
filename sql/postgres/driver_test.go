package postgres

import (
	"context"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDriver_InspectTable(t *testing.T) {
	tests := []struct {
		name   string
		opts   *schema.InspectTableOptions
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "table does not exist",
			before: func(m mock) {
				m.version("100000")
				m.tableExists("public", "users", false)
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.Nil(t)
				require.Error(err)
				require.True(schema.IsNotExistError(err), "expect not exists error")
			},
		},
		{
			name: "table does not exist in schema",
			opts: &schema.InspectTableOptions{
				Schema: "postgres",
			},
			before: func(m mock) {
				m.version("100000")
				m.tableExistsInSchema("postgres", "users", false)
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.Nil(t)
				require.Error(err)
				require.True(schema.IsNotExistError(err), "expect not exists error")
			},
		},
		{
			name: "column types",
			before: func(m mock) {
				m.version("130000")
				m.tableExists("public", "users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 column_name |      data_type      | is_nullable |         column_default          | character_maximum_length | numeric_precision | numeric_scale | character_set_name | collation_name | udt_name | is_identity | comment | typtype |  oid  
-------------+---------------------+-------------+---------------------------------+--------------------------+-------------------+---------------+--------------------+----------------+----------+-------------+---------+---------+-------
 id          | bigint              | NO          |                                 |                          |                64 |             0 |                    |                | int8     | NO          |         | b       |    20
 rank        | integer             | YES         |                                 |                          |                32 |             0 |                    |                | int4     | NO          | rank    | b       |    23
 c1          | smallint            | NO          |                                 |                          |                16 |             0 |                    |                | int2     | NO          |         | b       |    21
 c2          | bit                 | NO          |                                 |                        1 |                   |               |                    |                | bit      | NO          |         | b       |  1560
 c3          | bit varying         | NO          |                                 |                       10 |                   |               |                    |                | varbit   | NO          |         | b       |  1562
 c4          | boolean             | NO          |                                 |                          |                   |               |                    |                | bool     | NO          |         | b       |    16
 c5          | bytea               | NO          |                                 |                          |                   |               |                    |                | bytea    | NO          |         | b       |    17
 c6          | character           | NO          |                                 |                      100 |                   |               |                    |                | bpchar   | NO          |         | b       |  1042
 c7          | character varying   | NO          |                                 |                          |                   |               |                    |                | varchar  | NO          |         | b       |  1043
 c8          | cidr                | NO          |                                 |                          |                   |               |                    |                | cidr     | NO          |         | b       |   650
 c9          | circle              | NO          |                                 |                          |                   |               |                    |                | circle   | NO          |         | b       |   718
 c10         | date                | NO          |                                 |                          |                   |               |                    |                | date     | NO          |         | b       |  1082
 c11         | time with time zone | NO          |                                 |                          |                   |               |                    |                | timetz   | NO          |         | b       |  1266
 c12         | double precision    | NO          |                                 |                          |                53 |               |                    |                | float8   | NO          |         | b       |   701
 c13         | real                | NO          |                                 |                          |                24 |               |                    |                | float4   | NO          |         | b       |   700
 c14         | json                | NO          |                                 |                          |                   |               |                    |                | json     | NO          |         | b       |   114
 c15         | jsonb               | NO          |                                 |                          |                   |               |                    |                | jsonb    | NO          |         | b       |  3802
 c16         | money               | NO          |                                 |                          |                   |               |                    |                | money    | NO          |         | b       |   790
 c17         | numeric             | NO          |                                 |                          |                   |               |                    |                | numeric  | NO          |         | b       |  1700
 c18         | numeric             | NO          |                                 |                          |                 4 |             4 |                    |                | numeric  | NO          |         | b       |  1700
 c19         | integer             | NO          | nextval('t1_c19_seq'::regclass) |                          |                32 |             0 |                    |                | int4     | NO          |         | b       |    23
 c20         | uuid                | NO          |                                 |                          |                   |               |                    |                | uuid     | NO          |         | b       |  2950
 c21         | xml                 | NO          |                                 |                          |                   |               |                    |                | xml      | NO          |         | b       |   142
 c22         | ARRAY               | YES         |                                 |                          |                   |               |                    |                | _int4    | NO          |         | b       |  1007
 c23         | USER-DEFINED        | YES         |                                 |                          |                   |               |                    |                | ltree    | NO          |         | b       | 16535
 c24         | USER-DEFINED        | NO          |                                 |                          |                   |               |                    |                | state    | NO          |         | e       | 16774
`))
				m.ExpectQuery(sqltest.Escape(`SELECT enumtypid, enumlabel FROM pg_enum WHERE enumtypid IN (?)`)).
					WithArgs(16774).
					WillReturnRows(sqltest.Rows(`
 enumtypid | enumlabel
-----------+-----------
     16774 | on
     16774 | off
`))
				m.noIndexes()
				m.noFKs()
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "rank", Type: &schema.ColumnType{Raw: "integer", Null: true, Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&schema.Comment{Text: "rank"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "bit", Type: &BitType{T: "bit", Len: 1}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "bit varying", Type: &BitType{T: "bit varying", Len: 10}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "boolean", Type: &schema.BoolType{T: "boolean"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "bytea", Type: &schema.BinaryType{T: "bytea"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "character", Type: &schema.StringType{T: "character", Size: 100}}},
					{Name: "c7", Type: &schema.ColumnType{Raw: "character varying", Type: &schema.StringType{T: "character varying"}}},
					{Name: "c8", Type: &schema.ColumnType{Raw: "cidr", Type: &NetworkType{T: "cidr"}}},
					{Name: "c9", Type: &schema.ColumnType{Raw: "circle", Type: &schema.SpatialType{T: "circle"}}},
					{Name: "c10", Type: &schema.ColumnType{Raw: "date", Type: &schema.TimeType{T: "date"}}},
					{Name: "c11", Type: &schema.ColumnType{Raw: "time with time zone", Type: &schema.TimeType{T: "time with time zone"}}},
					{Name: "c12", Type: &schema.ColumnType{Raw: "double precision", Type: &schema.FloatType{T: "double precision", Precision: 53}}},
					{Name: "c13", Type: &schema.ColumnType{Raw: "real", Type: &schema.FloatType{T: "real", Precision: 24}}},
					{Name: "c14", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
					{Name: "c15", Type: &schema.ColumnType{Raw: "jsonb", Type: &schema.JSONType{T: "jsonb"}}},
					{Name: "c16", Type: &schema.ColumnType{Raw: "money", Type: &CurrencyType{T: "money"}}},
					{Name: "c17", Type: &schema.ColumnType{Raw: "numeric", Type: &schema.DecimalType{T: "numeric"}}},
					{Name: "c18", Type: &schema.ColumnType{Raw: "numeric", Type: &schema.DecimalType{T: "numeric", Precision: 4, Scale: 4}}},
					{Name: "c19", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}, Default: &schema.RawExpr{X: "nextval('t1_c19_seq'::regclass)"}},
					{Name: "c20", Type: &schema.ColumnType{Raw: "uuid", Type: &UUIDType{T: "uuid"}}},
					{Name: "c21", Type: &schema.ColumnType{Raw: "xml", Type: &XMLType{T: "xml"}}},
					{Name: "c22", Type: &schema.ColumnType{Raw: "ARRAY", Null: true, Type: &ArrayType{T: "int4"}}},
					{Name: "c23", Type: &schema.ColumnType{Raw: "USER-DEFINED", Null: true, Type: &UserDefinedType{T: "ltree"}}},
					{Name: "c24", Type: &schema.ColumnType{Raw: "USER-DEFINED", Type: &EnumType{T: "state", ID: 16774, Values: []string{"on", "off"}}}},
				}, t.Columns)
			},
		},
		{
			name: "table indexes",
			before: func(m mock) {
				m.version("130000")
				m.tableExists("public", "users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 column_name |      data_type      | is_nullable |         column_default          | character_maximum_length | numeric_precision | numeric_scale | character_set_name | collation_name | udt_name | is_identity | comment | typtype |  oid
-------------+---------------------+-------------+---------------------------------+--------------------------+-------------------+---------------+--------------------+----------------+----------+-------------+---------+---------+-------
 id          | bigint              | NO          |                                 |                          |                64 |             0 |                    |                | int8     | NO          |         | b       |    20
 c1          | smallint            | NO          |                                 |                          |                16 |             0 |                    |                | int2     | NO          |         | b       |    21
`))
				m.ExpectQuery(sqltest.Escape(indexesQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
    index_name    | column_name | primary | unique | constraint_type | predicate             |   expression              | asc | desc | nulls_first | nulls_last | comment
------------------+-------------+---------+--------+-----------------+-----------------------+---------------------------+-----+------+-------------+------------+-----------
 idx              | left        | f       | f      |                 |                       | "left"((c11)::text, 100)  | f   | t    | t           | f          | boring
 idx1             | left        | f       | f      |                 | (id <> NULL::integer) | "left"((c11)::text, 100)  | f   | t    | t           | f          |
 t1_c1_key        | c1          | f       | t      | u               |                       |                           | f   | t    | t           | f          |
 t1_pkey          | id          | t       | t      | p               |                       |                           | f   | t    | f           | f          |
 idx4             | c1          | f       | t      |                 |                       |                           | t   | f    | f           | f          |
 idx4             | id          | f       | t      |                 |                       |                           | t   | f    | f           | t          |

`))
				m.noFKs()
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint"}}},
				}
				require.EqualValues(columns, t.Columns)
				indexes := []*schema.Index{
					{Name: "idx", Table: t, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}, Attrs: []schema.Attr{&IndexColumnProperty{Desc: true, NullsFirst: true}}}}, Attrs: []schema.Attr{&schema.Comment{Text: "boring"}}},
					{Name: "idx1", Table: t, Attrs: []schema.Attr{&IndexPredicate{P: `(id <> NULL::integer)`}}, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}, Attrs: []schema.Attr{&IndexColumnProperty{Desc: true, NullsFirst: true}}}}},
					{Name: "t1_c1_key", Unique: true, Table: t, Attrs: []schema.Attr{&ConType{T: "u"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1], Attrs: []schema.Attr{&IndexColumnProperty{Desc: true, NullsFirst: true}}}}},
					{Name: "idx4", Unique: true, Table: t, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1], Attrs: []schema.Attr{&IndexColumnProperty{Asc: true}}}, {SeqNo: 2, C: columns[0], Attrs: []schema.Attr{&IndexColumnProperty{Asc: true, NullsLast: true}}}}},
				}
				require.EqualValues(indexes, t.Indexes)
				require.EqualValues(&schema.Index{
					Name:   "t1_pkey",
					Unique: true,
					Table:  t,
					Attrs:  []schema.Attr{&ConType{T: "p"}},
					Parts:  []*schema.IndexPart{{SeqNo: 1, C: columns[0], Attrs: []schema.Attr{&IndexColumnProperty{Desc: true}}}},
				}, t.PrimaryKey)
			},
		},
		{
			name: "fks",
			before: func(m mock) {
				m.version("130000")
				m.tableExists("public", "users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("public", "users").
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 column_name |      data_type      | is_nullable |         column_default          | character_maximum_length | numeric_precision | numeric_scale | character_set_name | collation_name | udt_name | is_identity | comment | typtype |  oid
-------------+---------------------+-------------+---------------------------------+--------------------------+-------------------+---------------+--------------------+----------------+----------+-------------+---------+---------+-------
 id          | integer             | NO          |                                 |                          |                32 |             0 |                    |                | int      | NO          |         | b       |    20
 oid         | integer             | NO          |                                 |                          |                32 |             0 |                    |                | int      | NO          |         | b       |    21
 uid         | integer             | NO          |                                 |                          |                32 |             0 |                    |                | int      | NO          |         | b       |    21
`))
				m.noIndexes()
				m.ExpectQuery(sqltest.Escape(fksQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 constraint_name | table_name | column_name | table_schema | referenced_table_name | referenced_column_name | referenced_schema_name | update_rule | delete_rule
-----------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------
 multi_column    | users      | id          | public       | t1                    | gid                    | public                 | NO ACTION   | CASCADE
 multi_column    | users      | id          | public       | t1                    | xid                    | public                 | NO ACTION   | CASCADE
 multi_column    | users      | oid         | public       | t1                    | gid                    | public                 | NO ACTION   | CASCADE
 multi_column    | users      | oid         | public       | t1                    | xid                    | public                 | NO ACTION   | CASCADE
 self_reference  | users      | uid         | public       | users                 | id                     | public                 | NO ACTION   | CASCADE

`))
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Equal("public", t.Schema.Name)
				fks := []*schema.ForeignKey{
					{Symbol: "multi_column", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}}, RefColumns: []*schema.Column{{Name: "gid"}, {Name: "xid"}}},
					{Symbol: "self_reference", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: t},
				}
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}, ForeignKeys: fks[0:1]},
					{Name: "oid", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}, ForeignKeys: fks[0:1]},
					{Name: "uid", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}, ForeignKeys: fks[1:2]},
				}
				fks[0].Columns = columns[:2]
				fks[1].Columns = columns[2:]
				fks[1].RefColumns = columns[:1]
				require.EqualValues(columns, t.Columns)
				require.EqualValues(fks, t.ForeignKeys)
			},
		},
		{
			name: "check",
			before: func(m mock) {
				m.version("130000")
				m.tableExists("public", "users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("public", "users").
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 column_name | data_type | is_nullable | column_default | character_maximum_length | numeric_precision | numeric_scale | character_set_name | collation_name | udt_name | is_identity | comment | typtype | oid
-------------+-----------+-------------+----------------+--------------------------+-------------------+---------------+--------------------+----------------+----------+-------------+---------+---------+-----
 c1          | integer   | NO          |                |                          |                32 |             0 |                    |                | int4     | NO          |         | b       |  23
 c2          | integer   | NO          |                |                          |                32 |             0 |                    |                | int4     | NO          |         | b       |  23
 c3          | integer   | NO          |                |                          |                32 |             0 |                    |                | int4     | NO          |         | b       |  23
`))
				m.noIndexes()
				m.noFKs()
				m.ExpectQuery(sqltest.Escape(checksQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 constraint_name    |       expression        | column_name | column_indexes 
--------------------+-------------------------+-------------+----------------
 boring             | (c1 > 1)                | c1          | {1}
 users_c2_check     | (c2 > 0)                | c2          | {2}
 users_c2_check1    | (c2 > 0)                | c2          | {2}
 users_check        | ((c2 + c1) > 2)         | c2          | {2,1}
 users_check        | ((c2 + c1) > 2)         | c1          | {2,1}
 users_check1       | (((c2 + c1) + c3) > 10) | c2          | {2,1,3}
 users_check1       | (((c2 + c1) + c3) > 10) | c1          | {2,1,3}
 users_check1       | (((c2 + c1) + c3) > 10) | c3          | {2,1,3}
`))
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Equal("public", t.Schema.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}},
				}, t.Columns)
				require.EqualValues([]schema.Attr{
					&Check{Name: "boring", Clause: "(c1 > 1)", Columns: []string{"c1"}},
					&Check{Name: "users_c2_check", Clause: "(c2 > 0)", Columns: []string{"c2"}},
					&Check{Name: "users_c2_check1", Clause: "(c2 > 0)", Columns: []string{"c2"}},
					&Check{Name: "users_check", Clause: "((c2 + c1) > 2)", Columns: []string{"c2", "c1"}},
					&Check{Name: "users_check1", Clause: "(((c2 + c1) + c3) > 10)", Columns: []string{"c2", "c1", "c3"}},
				}, t.Attrs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			tt.before(mock{m})
			drv, err := Open(db)
			require.NoError(t, err)
			table, err := drv.InspectTable(context.Background(), "users", tt.opts)
			tt.expect(require.New(t), table, err)
		})
	}
}

func TestDriver_Realm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(schemasQuery + " WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')")).
		WillReturnRows(sqltest.Rows(`
    schema_name
--------------------
 test
 public
`))
	mk.tables("test")
	mk.tables("public")
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
				},
				{
					Name: "public",
				},
			},
			// Server default configuration.
			Attrs: []schema.Attr{
				&schema.Collation{
					V: "en_US.utf8",
				},
				&CType{
					V: "en_US.utf8",
				},
			},
		}
		r.Schemas[0].Realm = r
		r.Schemas[1].Realm = r
		return r
	}(), realm)

	mk.ExpectQuery(sqltest.Escape(schemasQuery+" WHERE schema_name IN ($1, $2)")).
		WithArgs("test", "public").
		WillReturnRows(sqltest.Rows(`
    schema_name
--------------------
 test
 public
`))
	mk.tables("test")
	mk.tables("public")
	realm, err = drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Schemas: []string{"test", "public"}})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
				},
				{
					Name: "public",
				},
			},
			// Server default configuration.
			Attrs: []schema.Attr{
				&schema.Collation{
					V: "en_US.utf8",
				},
				&CType{
					V: "en_US.utf8",
				},
			},
		}
		r.Schemas[0].Realm = r
		r.Schemas[1].Realm = r
		return r
	}(), realm)
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	m.ExpectQuery(sqltest.Escape(paramsQuery)).
		WillReturnRows(sqltest.Rows(`
  setting   
------------
 en_US.utf8
 en_US.utf8
 ` + version + `
`))
}

func (m mock) tableExists(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_comment"})
	if exists {
		rows.AddRow(schema, nil)
	}
	m.ExpectQuery(sqltest.Escape(tableQuery)).
		WithArgs(table).
		WillReturnRows(rows)
}

func (m mock) tableExistsInSchema(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_comment"})
	if exists {
		rows.AddRow(schema, nil)
	}
	m.ExpectQuery(sqltest.Escape(tableSchemaQuery)).
		WithArgs(table, schema).
		WillReturnRows(rows)
}

func (m mock) noIndexes() {
	m.ExpectQuery(sqltest.Escape(indexesQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "primary", "unique", "constraint_type", "predicate", "expression"}))
}

func (m mock) noFKs() {
	m.ExpectQuery(sqltest.Escape(fksQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name", "referenced_table_schema", "update_rule", "delete_rule"}))
}

func (m mock) noChecks() {
	m.ExpectQuery(sqltest.Escape(checksQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "expression", "column_name", "column_indexes"}))
}

func (m mock) tables(schema string, names ...string) {
	rows := sqlmock.NewRows([]string{"table_name"})
	for i := range names {
		rows.AddRow(names[i])
	}
	m.ExpectQuery(sqltest.Escape(tablesQuery)).
		WithArgs(schema).
		WillReturnRows(rows)
}
