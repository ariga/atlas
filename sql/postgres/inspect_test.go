// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// Single table queries used by the different tests.
var (
	queryFKs         = sqltest.Escape(fmt.Sprintf(fksQuery, "$2"))
	queryTables      = sqltest.Escape(fmt.Sprintf(tablesQuery, "$1"))
	queryChecks      = sqltest.Escape(fmt.Sprintf(checksQuery, "$2"))
	queryColumns     = sqltest.Escape(fmt.Sprintf(columnsQuery, "$2"))
	queryCrdbColumns = sqltest.Escape(fmt.Sprintf(crdbColumnsQuery, "$2"))
	queryIndexes     = sqltest.Escape(fmt.Sprintf(indexesQuery, "$2"))
	queryCrdbIndexes = sqltest.Escape(fmt.Sprintf(crdbIndexesQuery, "$2"))
)

func TestDriver_InspectTable(t *testing.T) {
	tests := []struct {
		name   string
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "column types",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 table_name  |  column_name |          data_type          |  formatted          | is_nullable |         column_default                 | character_maximum_length | numeric_precision | datetime_precision | numeric_scale |    interval_type    | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  | identity_generation | generation_expression | comment | typtype | typelem | elemtyp |  oid
-------------+--------------+-----------------------------+---------------------|-------------+----------------------------------------+--------------------------+-------------------+--------------------+---------------+---------------------+--------------------+----------------+-------------+----------------+--------------------+------------------+---------------------+-----------------------+---------+---------+---------+---------+-------
 users       |  id          | bigint                      | int8                | NO          |                                        |                          |                64 |                    |             0 |                     |                    |                | YES         |      100       |          1         |          1       |    BY DEFAULT       |                       |         | b       |         |         |    20
 users       |  rank        | integer                     | int4                | YES         |                                        |                          |                32 |                    |             0 |                     |                    |                | NO          |                |                    |                  |                     |                       | rank    | b       |         |         |    23
 users       |  c1          | smallint                    | int2                | NO          |           1000                         |                          |                16 |                    |             0 |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    21
 users       |  c2          | bit                         | bit                 | NO          |                                        |                        1 |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1560
 users       |  c3          | bit varying                 | varbit              | NO          |                                        |                       10 |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1562
 users       |  c4          | boolean                     | bool                | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    16
 users       |  c5          | bytea                       | bytea               | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    17
 users       |  c6          | character                   | bpchar              | NO          |                                        |                      100 |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1042
 users       |  c7          | character varying           | varchar             | NO          | 'logged_in'::character varying         |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1043
 users       |  c8          | cidr                        | cidr                | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   650
 users       |  c9          | circle                      | circle              | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   718
 users       |  c10         | date                        | date                | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1082
 users       |  c11         | time with time zone         | timetz              | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1266
 users       |  c12         | double precision            | float8              | NO          |                                        |                          |                53 |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   701
 users       |  c13         | real                        | float4              | NO          |           random()                     |                          |                24 |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   700
 users       |  c14         | json                        | json                | NO          |           '{}'::json                   |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   114
 users       |  c15         | jsonb                       | jsonb               | NO          |           '{}'::jsonb                  |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  3802
 users       |  c16         | money                       | money               | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   790
 users       |  c17         | numeric                     | numeric             | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1700
 users       |  c18         | numeric                     | numeric             | NO          |                                        |                          |                 4 |                    |             4 |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1700
 users       |  c19         | integer                     | int4                | NO          | nextval('t1_c19_seq'::regclass)        |                          |                32 |                    |             0 |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    23
 users       |  c20         | uuid                        | uuid                | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  2950
 users       |  c21         | xml                         | xml                 | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |   142
 users       |  c22         | ARRAY                       | integer[]           | YES         |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1007
 users       |  c23         | USER-DEFINED                | ltree               | YES         |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         | 16535
 users       |  c24         | USER-DEFINED                | state               | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | e       |         |         | 16774
 users       |  c25         | timestamp without time zone | timestamp           | NO          |            now()                       |                          |                   |                  4 |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1114
 users       |  c26         | timestamp with time zone    | timestamptz         | NO          |                                        |                          |                   |                  6 |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1184
 users       |  c27         | time without time zone      | time                | NO          |                                        |                          |                   |                  6 |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1266
 users       |  c28         | int                         | int8                | NO          |                                        |                          |                   |                  6 |               |                     |                    |                | NO          |                |                    |                  |                     |        (c1 + c2)      |         | b       |         |         |  1267
 users       |  c29         | interval                    | interval            | NO          |                                        |                          |                   |                  6 |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1268
 users       |  c30         | interval                    | interval            | NO          |                                        |                          |                   |                  6 |               |        MONTH        |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1269
 users       |  c31         | interval                    | interval            | NO          |                                        |                          |                   |                  6 |               | MINUTE TO SECOND(6) |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  1233
 users       |  c32         | bigint                      | int4                | NO          | nextval('public.t1_c32_seq'::regclass) |                          |                32 |                    |             0 |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    23
 users       |  c33         | USER-DEFINED                | test."status""."    | NO          |  'unknown'::test."status""."           |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | e       |         |         | 16775
 users       |  c34         | ARRAY                       | state[]             | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |  16774  |  e      | 16779
 users       |  c35         | character                   | domain_char         | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | d       |  16774  |  e      | 16779
 users       |  c36         | tsvector                    | tsvector            | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |  16774  |         | 16779
 users       |  c37         | tsquery                     | tsquery             | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |  16774  |         | 16779
 users       |  c38         | datemultirange              | datemultirange      | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | m       |         |         | 4535
 users       |  c39         | numrange                    | numrange            | NO          |                                        |                          |                   |                    |               |                     |                    |                | NO          |                |                    |                  |                     |                       |         | m       |         |         | 4536
 users       |  c40         | smallint                    | int4                | NO          | nextval('"Users_c40_seq"'::regclass)   |                          |                32 |                    |             0 |                     |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    23
`))
				m.ExpectQuery(sqltest.Escape(`SELECT enumtypid, enumlabel FROM pg_enum WHERE enumtypid IN ($1, $2)`)).
					WithArgs(16774, 16775).
					WillReturnRows(sqltest.Rows(`
 enumtypid | enumlabel
-----------+-----------
     16774 | on
     16774 | off
     16775 | unknown
`))
				m.noIndexes()
				m.noFKs()
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				p := func(i int) *int { return &i }
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&Identity{Generation: "BY DEFAULT", Sequence: &Sequence{Start: 100, Increment: 1, Last: 1}}}},
					{Name: "rank", Type: &schema.ColumnType{Raw: "integer", Null: true, Type: &schema.IntegerType{T: "integer"}}, Attrs: []schema.Attr{&schema.Comment{Text: "rank"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint"}}, Default: &schema.Literal{V: "1000"}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "bit", Type: &BitType{T: "bit", Len: 1}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "bit varying", Type: &BitType{T: "bit varying", Len: 10}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "boolean", Type: &schema.BoolType{T: "boolean"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "bytea", Type: &schema.BinaryType{T: "bytea"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "character", Type: &schema.StringType{T: "character", Size: 100}}},
					{Name: "c7", Type: &schema.ColumnType{Raw: "character varying", Type: &schema.StringType{T: "character varying"}}, Default: &schema.Literal{V: "'logged_in'"}},
					{Name: "c8", Type: &schema.ColumnType{Raw: "cidr", Type: &NetworkType{T: "cidr"}}},
					{Name: "c9", Type: &schema.ColumnType{Raw: "circle", Type: &schema.SpatialType{T: "circle"}}},
					{Name: "c10", Type: &schema.ColumnType{Raw: "date", Type: &schema.TimeType{T: "date"}}},
					{Name: "c11", Type: &schema.ColumnType{Raw: "time with time zone", Type: &schema.TimeType{T: "time with time zone", Precision: p(0)}}},
					{Name: "c12", Type: &schema.ColumnType{Raw: "double precision", Type: &schema.FloatType{T: "double precision", Precision: 53}}},
					{Name: "c13", Type: &schema.ColumnType{Raw: "real", Type: &schema.FloatType{T: "real", Precision: 24}}, Default: &schema.RawExpr{X: "random()"}},
					{Name: "c14", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}, Default: &schema.Literal{V: "'{}'"}},
					{Name: "c15", Type: &schema.ColumnType{Raw: "jsonb", Type: &schema.JSONType{T: "jsonb"}}, Default: &schema.Literal{V: "'{}'"}},
					{Name: "c16", Type: &schema.ColumnType{Raw: "money", Type: &CurrencyType{T: "money"}}},
					{Name: "c17", Type: &schema.ColumnType{Raw: "numeric", Type: &schema.DecimalType{T: "numeric"}}},
					{Name: "c18", Type: &schema.ColumnType{Raw: "numeric", Type: &schema.DecimalType{T: "numeric", Precision: 4, Scale: 4}}},
					{Name: "c19", Type: &schema.ColumnType{Raw: "serial", Type: &SerialType{T: "serial", SequenceName: "t1_c19_seq"}}},
					{Name: "c20", Type: &schema.ColumnType{Raw: "uuid", Type: &schema.UUIDType{T: "uuid"}}},
					{Name: "c21", Type: &schema.ColumnType{Raw: "xml", Type: &XMLType{T: "xml"}}},
					{Name: "c22", Type: &schema.ColumnType{Raw: "ARRAY", Null: true, Type: &ArrayType{Type: &schema.IntegerType{T: "integer"}, T: "integer[]"}}},
					{Name: "c23", Type: &schema.ColumnType{Raw: "USER-DEFINED", Null: true, Type: &UserDefinedType{T: "ltree"}}},
					{Name: "c24", Type: &schema.ColumnType{Raw: "state", Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}, Schema: t.Schema}}},
					{Name: "c25", Type: &schema.ColumnType{Raw: "timestamp without time zone", Type: &schema.TimeType{T: "timestamp without time zone", Precision: p(4)}}, Default: &schema.RawExpr{X: "now()"}},
					{Name: "c26", Type: &schema.ColumnType{Raw: "timestamp with time zone", Type: &schema.TimeType{T: "timestamp with time zone", Precision: p(6)}}},
					{Name: "c27", Type: &schema.ColumnType{Raw: "time without time zone", Type: &schema.TimeType{T: "time without time zone", Precision: p(6)}}},
					{Name: "c28", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&schema.GeneratedExpr{Expr: "(c1 + c2)"}}},
					{Name: "c29", Type: &schema.ColumnType{Raw: "interval", Type: &IntervalType{T: "interval", Precision: p(6)}}},
					{Name: "c30", Type: &schema.ColumnType{Raw: "interval", Type: &IntervalType{T: "interval", F: "MONTH", Precision: p(6)}}},
					{Name: "c31", Type: &schema.ColumnType{Raw: "interval", Type: &IntervalType{T: "interval", F: "MINUTE TO SECOND", Precision: p(6)}}},
					{Name: "c32", Type: &schema.ColumnType{Raw: "bigserial", Type: &SerialType{T: "bigserial", SequenceName: "t1_c32_seq"}}},
					{Name: "c33", Type: &schema.ColumnType{Raw: `status".`, Type: &schema.EnumType{T: `status".`, Values: []string{"unknown"}, Schema: schema.New("test")}}, Default: &schema.Literal{V: "'unknown'"}},
					{Name: "c34", Type: &schema.ColumnType{Raw: "ARRAY", Type: &ArrayType{T: "state[]", Type: &schema.EnumType{T: "state", Values: []string{"on", "off"}, Schema: t.Schema}}}},
					{Name: "c35", Type: &schema.ColumnType{Raw: "character", Type: &UserDefinedType{T: "domain_char"}}},
					{Name: "c36", Type: &schema.ColumnType{Raw: "tsvector", Type: &TextSearchType{T: "tsvector"}}},
					{Name: "c37", Type: &schema.ColumnType{Raw: "tsquery", Type: &TextSearchType{T: "tsquery"}}},
					{Name: "c38", Type: &schema.ColumnType{Raw: "datemultirange", Type: &RangeType{T: "datemultirange"}}},
					{Name: "c39", Type: &schema.ColumnType{Raw: "numrange", Type: &RangeType{T: "numrange"}}},
					{Name: "c40", Type: &schema.ColumnType{Raw: "smallserial", Type: &SerialType{T: "smallserial", SequenceName: "Users_c40_seq"}}},
				}, t.Columns)
			},
		},
		{
			name: "table indexes",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
table_name | column_name |      data_type      | formatted |  is_nullable |         column_default          | character_maximum_length | numeric_precision | datetime_precision | numeric_scale | interval_type | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  | identity_generation | generation_expression | comment | typtype | typelem | elemtyp |  oid
-----------+-------------+---------------------+-----------+--------------+---------------------------------+--------------------------+-------------------+--------------------+---------------+---------------+--------------------+----------------+-------------+----------------+--------------------+------------------+---------------------+-----------------------+---------+---------+---------+---------+-------
users      | id          | bigint              | int8      |  NO          |                                 |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    20
users      | c1          | smallint            | int2      |  NO          |                                 |                          |                16 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    21
users      | parent_id   | bigint              | int8      |  YES         |                                 |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    22
users      | ts          | tsvector            | tsvector  |  NO          |                                 |                          |                   |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    23
`))
				m.ExpectQuery(queryIndexes).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
   table_name   |    index_name   | index_type  | column_name | included | primary | unique |   constraints   | predicate             |   expression              | desc | nulls_first | nulls_last | comment   |                 options               |   opclass_name    | opclass_default | opclass_params 
----------------+-----------------+-------------+-------------+----------+---------+--------+-----------------+-----------------------+---------------------------+------+-------------+------------+-----------+---------------------------------------+-------------------+-----------------+----------------
users           | idx             | hash        |             | f        | f       | f      |                 |                       | "left"((c11)::text, 100)  | t    | t           | f          | boring    |                                       |     int4_ops      |        t        |
users           | idx1            | btree       |             | f        | f       | f      |                 | (id <> NULL::integer) | "left"((c11)::text, 100)  | t    | t           | f          |           |                                       |     int4_ops      |        t        |
users           | t1_c1_key       | btree       | c1          | f        | f       | t      | {"name": "u"}   |                       | c1                        | t    | t           | f          |           |                                       |     int4_ops      |        t        |
users           | t1_pkey         | btree       | id          | f        | t       | t      | {"t_pkey": "p"} |                       | id                        | t    | f           | f          |           |                                       |     int4_ops      |        t        |
users           | idx4            | btree       | c1          | f        | f       | t      |                 |                       | c1                        | f    | f           | f          |           |                                       |     int4_ops      |        t        |
users           | idx4            | btree       | id          | f        | f       | t      |                 |                       | id                        | f    | f           | t          |           |                                       |     int4_ops      |        t        |
users           | idx5            | btree       | c1          | f        | f       | t      |                 |                       | c1                        | f    | f           | f          |           |                                       |     int4_ops      |        t        |
users           | idx5            | btree       |             | f        | f       | t      |                 |                       | coalesce(parent_id, 0)    | f    | f           | f          |           |                                       |     int4_ops      |        t        |
users           | idx6            | brin        | c1          | f        | f       | t      |                 |                       |                           | f    | f           | f          |           | {autosummarize=true,pages_per_range=2}|     int4_ops      |        t        |
users           | idx2            | btree       |             | f        | f       | f      |                 |                       | ((c * 2))                 | f    | f           | t          |           |                                       |     int4_ops      |        t        |
users           | idx2            | btree       | c1          | f        | f       | f      |                 |                       | c                         | f    | f           | t          |           |                                       |     int4_ops      |        t        |
users           | idx2            | btree       | id          | f        | f       | f      |                 |                       | d                         | f    | f           | t          |           |                                       |     int4_ops      |        t        |
users           | idx2            | btree       | c1          | t        | f       | f      |                 |                       | c                         |      |             |            |           |                                       |     int4_ops      |        t        |
users           | idx2            | btree       | parent_id   | t        | f       | f      |                 |                       | d                         |      |             |            |           |                                       |     int4_ops      |        t        |
users           | tsx             | gist        | ts          | f        | f       | f      |                 |                       | ts                        |      |             |            |           |                                       |     tsvector_ops  |        f        | {siglen=1}
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
					{Name: "parent_id", Type: &schema.ColumnType{Raw: "bigint", Null: true, Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "ts", Type: &schema.ColumnType{Raw: "tsvector", Type: &TextSearchType{T: "tsvector"}}},
				}
				indexes := []*schema.Index{
					{Name: "idx", Table: t, Attrs: []schema.Attr{&IndexType{T: "hash"}, &schema.Comment{Text: "boring"}}, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}, Desc: true, Attrs: []schema.Attr{&IndexColumnProperty{NullsFirst: true}}}}},
					{Name: "idx1", Table: t, Attrs: []schema.Attr{&IndexType{T: "btree"}, &IndexPredicate{P: `(id <> NULL::integer)`}}, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `"left"((c11)::text, 100)`}, Desc: true, Attrs: []schema.Attr{&IndexColumnProperty{NullsFirst: true}}}}},
					{Name: "t1_c1_key", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "btree"}, &Constraint{N: "name", T: "u"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1], Desc: true, Attrs: []schema.Attr{&IndexColumnProperty{NullsFirst: true}}}}},
					{Name: "idx4", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "btree"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}, {SeqNo: 2, C: columns[0], Attrs: []schema.Attr{&IndexColumnProperty{NullsLast: true}}}}},
					{Name: "idx5", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "btree"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}, {SeqNo: 2, X: &schema.RawExpr{X: `coalesce(parent_id, 0)`}}}},
					{Name: "idx6", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "brin"}, &IndexStorageParams{AutoSummarize: true, PagesPerRange: 2}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}}},
					{Name: "idx2", Unique: false, Table: t, Attrs: []schema.Attr{&IndexType{T: "btree"}, &IndexInclude{Columns: columns[1:3]}}, Parts: []*schema.IndexPart{{SeqNo: 1, X: &schema.RawExpr{X: `((c * 2))`}, Attrs: []schema.Attr{&IndexColumnProperty{NullsLast: true}}}, {SeqNo: 2, C: columns[1], Attrs: []schema.Attr{&IndexColumnProperty{NullsLast: true}}}, {SeqNo: 3, C: columns[0], Attrs: []schema.Attr{&IndexColumnProperty{NullsLast: true}}}}},
					{Name: "tsx", Unique: false, Table: t, Attrs: []schema.Attr{&IndexType{T: "gist"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[3], Attrs: []schema.Attr{&IndexOpClass{Name: "tsvector_ops", Params: []struct{ N, V string }{{N: "siglen", V: "1"}}}}}}},
				}
				pk := &schema.Index{
					Name:   "t1_pkey",
					Unique: true,
					Table:  t,
					Attrs:  []schema.Attr{&IndexType{T: "btree"}, &Constraint{N: "t_pkey", T: "p"}},
					Parts:  []*schema.IndexPart{{SeqNo: 1, C: columns[0], Desc: true}},
				}
				columns[0].Indexes = append(columns[0].Indexes, pk, indexes[3], indexes[6])
				columns[1].Indexes = indexes[2:7]
				columns[3].Indexes = indexes[7:]
				require.EqualValues(columns, t.Columns)
				require.EqualValues(indexes, t.Indexes)
				require.EqualValues(pk, t.PrimaryKey)
			},
		},
		{
			name: "fks",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
table_name | column_name |      data_type      | formatted | is_nullable |         column_default          | character_maximum_length | numeric_precision | datetime_precision | numeric_scale | interval_type | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  | identity_generation | generation_expression | comment | typtype | typelem | elemtyp |  oid
-----------+-------------+---------------------+-----------+-------------+---------------------------------+--------------------------+-------------------+--------------------+---------------+---------------+--------------------+----------------+-------------+----------------+--------------------+------------------+---------------------+-----------------------+---------+---------+---------+---------+-------
users      | id          | integer             | int       | NO          |                                 |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    20
users      | oid         | integer             | int       | NO          |                                 |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    21
users      | uid         | integer             | int       | NO          |                                 |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |    21
`))
				m.noIndexes()
				m.ExpectQuery(queryFKs).
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
					{Symbol: "multi_column", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: &schema.Table{Name: "t1", Schema: t.Schema}, RefColumns: []*schema.Column{{Name: "gid"}, {Name: "xid"}}},
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
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
table_name |column_name | data_type | formatted | is_nullable | column_default | character_maximum_length | numeric_precision | datetime_precision | numeric_scale | interval_type | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  | identity_generation | generation_expression | comment | typtype | typelem | elemtyp | oid
-----------+------------+-----------+-----------+-------------+----------------+--------------------------+-------------------+--------------------+---------------+---------------+--------------------+----------------+-------------+----------------+--------------------+------------------+---------------------+-----------------------+---------+---------+---------+---------+-----
users      | c1         | integer   | int4      | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
users      | c2         | integer   | int4      | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
users      | c3         | integer   | int4      | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
`))
				m.noIndexes()
				m.noFKs()
				m.ExpectQuery(queryChecks).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
table_name   | constraint_name    |       expression        | column_name | column_indexes | no_inherit
-------------+--------------------+-------------------------+-------------+----------------+----------------
users        | boring             | (c1 > 1)                | c1          | {1}            | t
users        | users_c2_check     | (c2 > 0)                | c2          | {2}            | f
users        | users_c2_check1    | (c2 > 0)                | c2          | {2}            | f
users        | users_check        | ((c2 + c1) > 2)         | c2          | {2,1}          | f
users        | users_check        | ((c2 + c1) > 2)         | c1          | {2,1}          | f
users        | users_check1       | (((c2 + c1) + c3) > 10) | c2          | {2,1,3}        | f
users        | users_check1       | (((c2 + c1) + c3) > 10) | c1          | {2,1,3}        | f
users        | users_check1       | (((c2 + c1) + c3) > 10) | c3          | {2,1,3}        | f
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
					&schema.Check{Name: "boring", Expr: "(c1 > 1)", Attrs: []schema.Attr{&CheckColumns{Columns: []string{"c1"}}, &NoInherit{}}},
					&schema.Check{Name: "users_c2_check", Expr: "(c2 > 0)", Attrs: []schema.Attr{&CheckColumns{Columns: []string{"c2"}}}},
					&schema.Check{Name: "users_c2_check1", Expr: "(c2 > 0)", Attrs: []schema.Attr{&CheckColumns{Columns: []string{"c2"}}}},
					&schema.Check{Name: "users_check", Expr: "((c2 + c1) > 2)", Attrs: []schema.Attr{&CheckColumns{Columns: []string{"c2", "c1"}}}},
					&schema.Check{Name: "users_check1", Expr: "(((c2 + c1) + c3) > 10)", Attrs: []schema.Attr{&CheckColumns{Columns: []string{"c2", "c1", "c3"}}}},
				}, t.Attrs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			mk := mock{m}
			mk.version("130000")
			var drv migrate.Driver
			drv, err = Open(db)
			require.NoError(t, err)
			mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= $1"))).
				WithArgs("public").
				WillReturnRows(sqltest.Rows(`
    schema_name
--------------------
 public
`))
			tt.before(mk)
			s, err := drv.InspectSchema(context.Background(), "public", nil)
			require.NoError(t, err)
			tt.expect(require.New(t), s.Tables[0], err)
		})
	}
}

func TestDriver_InspectPartitionedTable(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= CURRENT_SCHEMA()"))).
		WillReturnRows(sqltest.Rows(`
   schema_name
--------------------
public
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "$1"))).
		WithArgs("public").
		WillReturnRows(sqltest.Rows(`
 table_schema | table_name  | comment | partition_attrs | partition_strategy |                  partition_exprs                   
--------------+-------------+---------+-----------------+--------------------+----------------------------------------------------
 public       | logs1       |         |                 |                    | 
 public       | logs2       |         | 1               | r                  | 
 public       | logs3       |         | 2 0 0           | l                  | (a + b), (a + (b * 2))

`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, "$2, $3, $4"))).
		WithArgs("public", "logs1", "logs2", "logs3").
		WillReturnRows(sqltest.Rows(`
table_name |column_name | data_type | formatted | is_nullable | column_default | character_maximum_length | numeric_precision | datetime_precision | numeric_scale | interval_type | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  | identity_generation | generation_expression | comment | typtype | typelem | elemtyp | oid
-----------+------------+-----------+-----------+-------------+----------------+--------------------------+-------------------+--------------------+---------------+---------------+--------------------+----------------+-------------+----------------+--------------------+------------------+---------------------+-----------------------+---------+---------+---------+---------+-----
logs1      | c1         | integer   | integer   | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
logs2      | c2         | integer   | integer   | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
logs2      | c3         | integer   | integer   | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
logs3      | c4         | integer   | integer   | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
logs3      | c5         | integer   | integer   | NO          |                |                          |                32 |                    |             0 |               |                    |                | NO          |                |                    |                  |                     |                       |         | b       |         |         |  23
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesQuery, "$2, $3, $4"))).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "primary", "unique", "constraint_type", "predicate", "expression"}))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, "$2, $3, $4"))).
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name", "referenced_table_schema", "update_rule", "delete_rule"}))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(checksQuery, "$2, $3, $4"))).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "constraint_name", "expression", "column_name", "column_indexes"}))
	s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{})
	require.NoError(t, err)

	t1, ok := s.Table("logs1")
	require.True(t, ok)
	require.Empty(t, t1.Attrs)

	t2, ok := s.Table("logs2")
	require.True(t, ok)
	require.Len(t, t2.Attrs, 1)
	key := t2.Attrs[0].(*Partition)
	require.Equal(t, PartitionTypeRange, key.T)
	require.Equal(t, []*PartitionPart{
		{C: &schema.Column{Name: "c2", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}}},
	}, key.Parts)

	t3, ok := s.Table("logs3")
	require.True(t, ok)
	require.Len(t, t3.Attrs, 1)
	key = t3.Attrs[0].(*Partition)
	require.Equal(t, PartitionTypeList, key.T)
	require.Equal(t, []*PartitionPart{
		{C: &schema.Column{Name: "c5", Type: &schema.ColumnType{Raw: "integer", Type: &schema.IntegerType{T: "integer"}}}},
		{X: &schema.RawExpr{X: "(a + b)"}},
		{X: &schema.RawExpr{X: "(a + (b * 2))"}},
	}, key.Parts)
}

func TestDriver_InspectCRDBSchema(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.ExpectQuery(sqltest.Escape(paramsQuery)).
		WillReturnRows(sqltest.Rows(`
					setting
				------------
				130000
				en_US.utf8
				en_US.utf8
				cockroach
				`))
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= $1"))).
		WithArgs("public").
		WillReturnRows(sqltest.Rows(`
schema_name
--------------------
public
`))
	mk.tableExists("public", "users", true)
	mk.ExpectQuery(queryCrdbColumns).
		WithArgs("public", "users").
		WillReturnRows(sqltest.Rows(`
table_name  | column_name | data_type | formatted | is_nullable |              column_default               | character_maximum_length | numeric_precision | datetime_precision | numeric_scale | interval_type | character_set_name | collation_name | is_identity | identity_start | identity_increment |   identity_last  |  identity_generation  | generation_expression | comment | typtype | typelem | elemtyp | oid 
------------+-------------+-----------+-----------+-------------+-------------------------------------------+--------------------------+-------------------+--------------------+---------------+---------------+--------------------+----------------|-------------+----------------+--------------------+------------------+-----------------------+-----------------------+---------+---------+---------+---------+-----
users       | a           | bigint    | bigint    | NO          |                                           |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                       |                       |         | b       |         |         | 20 
users       | b           | bigint    | bigint    | NO          |                                           |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                       |                       |         | b       |         |         | 20 
users       | c           | bigint    | bigint    | NO          |                                           |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                       |                       |         | b       |         |         | 20 
users       | d           | bigint    | bigint    | NO          |                                           |                          |                64 |                    |             0 |               |                    |                | NO          |                |                    |                  |                       |                       |         | b       |         |         | 20 
`))
	mk.ExpectQuery(queryCrdbIndexes).
		WithArgs("public", "users").
		WillReturnRows(sqltest.Rows(`
table_name  | index_name | column_name | primary | unique | constraint_type |                                   create_stmt                                   | predicate | expression | comment 
------------+------------+-------------+---------+--------+-----------------+---------------------------------------------------------------------------------+-----------+------------+---------
users       | idx1       | a           | false   | false  |                 | CREATE INDEX idx1 ON defaultdb.public.serial USING btree (a ASC)                |           | a          |  
users       | idx2       | b           | false   | true   | u               | CREATE UNIQUE INDEX idx2 ON defaultdb.public.serial USING btree (b ASC)         |           | b          |  
users       | idx3       | c           | false   | false  |                 | CREATE INDEX idx3 ON defaultdb.public.serial USING btree (c DESC)               |           | c          | boring 
users       | idx4       | d           | false   | false  |                 | CREATE INDEX idx5 ON defaultdb.public.serial USING btree (d ASC) WHERE (d < 10) | d < 10    | d          |  
users       | idx5       | a           | false   | false  |                 | CREATE INDEX idx5 ON defaultdb.public.serial USING btree (a ASC, b ASC, c ASC)  |           | a          |  
users       | idx5       | b           | false   | false  |                 | CREATE INDEX idx5 ON defaultdb.public.serial USING btree (a ASC, b ASC, c ASC)  |           | b          |  
users       | idx5       | c           | false   | false  |                 | CREATE INDEX idx5 ON defaultdb.public.serial USING btree (a ASC, b ASC, c ASC)  |           | c          |  
`))
	mk.noFKs()
	mk.noChecks()
	s, err := drv.InspectSchema(context.Background(), "public", nil)
	require.NoError(t, err)
	tbl := s.Tables[0]
	require.Equal(t, "users", tbl.Name)
	columns := []*schema.Column{
		{Name: "a", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
		{Name: "b", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
		{Name: "c", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
		{Name: "d", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
	}
	indexes := []*schema.Index{
		{Name: "idx1", Table: tbl, Attrs: []schema.Attr{&IndexType{T: "btree"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[0]}}},
		{Name: "idx2", Unique: true, Table: tbl, Attrs: []schema.Attr{&IndexType{T: "btree"}, &Constraint{T: "u"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[1]}}},
		{Name: "idx3", Table: tbl, Attrs: []schema.Attr{&IndexType{T: "btree"}, &schema.Comment{Text: "boring"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[2], Desc: true}}},
		{Name: "idx4", Table: tbl, Attrs: []schema.Attr{&IndexType{T: "btree"}, &IndexPredicate{P: `d < 10`}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[3]}}},
		{Name: "idx5", Table: tbl, Attrs: []schema.Attr{&IndexType{T: "btree"}}, Parts: []*schema.IndexPart{{SeqNo: 1, C: columns[0]}, {SeqNo: 2, C: columns[1]}, {SeqNo: 3, C: columns[2]}}},
	}
	columns[0].Indexes = []*schema.Index{indexes[0], indexes[4]}
	columns[1].Indexes = []*schema.Index{indexes[1], indexes[4]}
	columns[2].Indexes = []*schema.Index{indexes[2], indexes[4]}
	columns[3].Indexes = []*schema.Index{indexes[3]}
	require.EqualValues(t, columns, tbl.Columns)
	require.EqualValues(t, indexes, tbl.Indexes)
}

func TestDriver_InspectSchema(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= CURRENT_SCHEMA()"))).
		WillReturnRows(sqltest.Rows(`
   schema_name
--------------------
test
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "$1"))).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "comment", "partition_attrs", "partition_strategy", "partition_exprs"}))
	s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Schema {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
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
		return r.Schemas[0]
	}(), s)
}

func TestDriver_Realm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqltest.Rows(`
   schema_name
--------------------
test
public
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "$1, $2"))).
		WithArgs("test", "public").
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "comment", "partition_attrs", "partition_strategy", "partition_exprs"}))
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

	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "IN ($1, $2)"))).
		WithArgs("test", "public").
		WillReturnRows(sqltest.Rows(`
   schema_name
--------------------
  test
  public
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "$1, $2"))).
		WithArgs("test", "public").
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "comment", "partition_attrs", "partition_strategy", "partition_exprs"}))
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

	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= $1"))).
		WithArgs("test").
		WillReturnRows(sqltest.Rows(`
 schema_name
--------------------
 test
`))
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "$1"))).
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "comment", "partition_attrs", "partition_strategy", "partition_exprs"}))
	realm, err = drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Schemas: []string{"test"}})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
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
		return r
	}(), realm)
}

func TestInspectMode_InspectRealm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("130000")
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqltest.Rows(`
   schema_name
--------------------
test
public
`))
	drv, err := Open(db)
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Mode: schema.InspectSchemas})
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

func TestIndexOpClass_UnmarshalText(t *testing.T) {
	var op IndexOpClass
	require.NoError(t, op.UnmarshalText([]byte("int4_ops")))
	require.EqualValues(t, IndexOpClass{Name: "int4_ops"}, op)

	op = IndexOpClass{}
	require.NoError(t, op.UnmarshalText([]byte("")))
	require.EqualValues(t, IndexOpClass{}, op)

	require.NoError(t, op.UnmarshalText([]byte("int4_ops(siglen=1)")))
	require.EqualValues(t, IndexOpClass{Name: "int4_ops", Params: []struct{ N, V string }{{"siglen", "1"}}}, op)
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	m.ExpectQuery(sqltest.Escape(paramsQuery)).
		WillReturnRows(sqltest.Rows(`
  setting
------------
 ` + version + `
 en_US.utf8
 en_US.utf8
`))
}

func (m mock) tableExists(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_name", "table_comment", "partition_attrs", "partition_strategy", "partition_exprs"})
	if exists {
		rows.AddRow(schema, table, nil, nil, nil, nil)
	}
	m.ExpectQuery(queryTables).
		WithArgs(schema).
		WillReturnRows(rows)
}

func (m mock) noIndexes() {
	m.ExpectQuery(queryIndexes).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "primary", "unique", "constraint_type", "predicate", "expression", "options"}))
}

func (m mock) noFKs() {
	m.ExpectQuery(queryFKs).
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name", "referenced_table_schema", "update_rule", "delete_rule"}))
}

func (m mock) noChecks() {
	m.ExpectQuery(queryChecks).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "constraint_name", "expression", "column_name", "column_indexes"}))
}
