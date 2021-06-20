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
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint", Size: 8}}},
					{Name: "rank", Type: &schema.ColumnType{Raw: "integer", Null: true, Type: &schema.IntegerType{T: "integer", Size: 4}}, Attrs: []schema.Attr{&schema.Comment{Text: "rank"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint", Size: 2}}},
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
					{Name: "c19", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer", Size: 4}}, Default: &schema.RawExpr{X: "nextval('t1_c19_seq'::regclass)"}},
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
    index_name    | column_name | primary | unique | constraint_type |       predicate       |        expression
------------------+-------------+---------+--------+-----------------+-----------------------+--------------------------
 idx              | left        | f       | f      |                 |                       | "left"((c11)::text, 100)
 idx1             | left        | f       | f      |                 | (id <> NULL::integer) | "left"((c11)::text, 100)
 t1_c1_key        | c1          | f       | t      | u               |                       |
 t1_pkey          | id          | t       | t      | p               |                       |
 idx4             | c1          | f       | t      |                 |                       |
 idx4             | id          | f       | t      |                 |                       |
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint", Size: 8}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint", Size: 2}}},
				}
				require.EqualValues(columns, t.Columns)
				indexes := []*schema.Index{
					{Name: "idx", Table: t, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}}}},
					{Name: "idx1", Table: t, Attrs: []schema.Attr{&IndexPredicate{P: `(id <> NULL::integer)`}}, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}}}},
					{Name: "t1_c1_key", Unique: true, Table: t, Attrs: []schema.Attr{&ConType{T: "u"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}}},
					{Name: "idx4", Unique: true, Table: t, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}, {SeqNo: 2, C: columns[0]}}},
				}
				require.EqualValues(indexes, t.Indexes)
				require.EqualValues(&schema.Index{
					Name:   "t1_pkey",
					Unique: true,
					Table:  t,
					Attrs:  []schema.Attr{&ConType{T: "p"}},
					Parts:  []*schema.IndexPart{{SeqNo: 1, C: columns[0]}},
				}, t.PrimaryKey)
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
