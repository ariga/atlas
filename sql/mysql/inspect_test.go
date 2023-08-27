// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"fmt"
	"testing"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// Single table queries used by the different tests.
var (
	queryFKs              = sqltest.Escape(fmt.Sprintf(fksQuery, "?"))
	queryTable            = sqltest.Escape(fmt.Sprintf(tablesQuery, "?"))
	queryColumns          = sqltest.Escape(fmt.Sprintf(columnsExprQuery, "?"))
	queryColumnsNoExpr    = sqltest.Escape(fmt.Sprintf(columnsQuery, "?"))
	queryIndexes          = sqltest.Escape(fmt.Sprintf(indexesQuery, "?"))
	queryIndexesNoComment = sqltest.Escape(fmt.Sprintf(indexesNoCommentQuery, "?"))
	queryIndexesExpr      = sqltest.Escape(fmt.Sprintf(indexesExprQuery, "?"))
	queryMyChecks         = sqltest.Escape(fmt.Sprintf(myChecksQuery, "?"))
	queryMarChecks        = sqltest.Escape(fmt.Sprintf(marChecksQuery, "?"))
)

func TestDriver_InspectTable(t *testing.T) {
	tests := []struct {
		name    string
		version string
		before  func(mock)
		expect  func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "table collation",
			before: func(m mock) {
				m.ExpectQuery(queryTable).
					WithArgs("public").
					WillReturnRows(sqltest.Rows(`
+--------------+--------------+--------------------+--------------------+----------------+---------------+-------------------+------------------+------------------+
| TABLE_SCHEMA | TABLE_NAME   | CHARACTER_SET_NAME | TABLE_COLLATION    | AUTO_INCREMENT | TABLE_COMMENT | CREATE_OPTIONS    |      ENGINE      |  DEFAULT_ENGINE  |
+--------------+--------------+--------------------+--------------------+----------------+---------------+-------------------+------------------+------------------+
| public       | users        | utf8mb4            | utf8mb4_0900_ai_ci | nil            | Comment       | COMPRESSION="ZLIB"|       InnoDB     |       1          |
+--------------+--------------+--------------------+--------------------+----------------+---------------+-------------------+------------------+------------------+
`))
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+--------------------+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| table_name         | column_name        | column_type          | column_comment       | is_nullable | column_key | column_default | extra          | character_set_name | collation_name | generation_expression     |
+--------------------+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| users              | id                 | bigint(20)           |                      | NO          | PRI        | NULL           | auto_increment | NULL               | NULL           | NULL                      |
+--------------------+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
`))
				m.ExpectQuery(queryIndexesExpr).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+--------------------+--------------+-------------+------------+--------------+--------------+----------+--------------+------------+------------------+
| TABLE_NAME         | INDEX_NAME   | COLUMN_NAME | NON_UNIQUE | SEQ_IN_INDEX | INDEX_TYPE   | DESC     | COMMENT      | SUB_PART   | EXPRESSION       |
+--------------------+--------------+-------------+------------+--------------+--------------+----------+--------------+------------+------------------+
| users              | PRIMARY      | id          |          0 |            1 | BTREE        | 0        |              |       NULL |      NULL        |
+--------------------+--------------+-------------+------------+--------------+--------------+----------+--------------+------------+------------------+
`))
				m.noFKs()
				m.ExpectQuery(sqltest.Escape("SHOW CREATE TABLE `public`.`users`")).
					WillReturnRows(sqltest.Rows(`
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
| Table | Create Table                                                                                                                                |
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
| users | CREATE TABLE users (id bigint NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=55834574848 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin |
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]schema.Attr{
					&schema.Charset{V: "utf8mb4"},
					&schema.Collation{V: "utf8mb4_0900_ai_ci"},
					&schema.Comment{Text: "Comment"},
					&CreateOptions{V: `COMPRESSION="ZLIB"`},
					&Engine{V: "InnoDB", Default: true},
					&CreateStmt{S: "CREATE TABLE users (id bigint NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=55834574848 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin"},
					&AutoIncrement{V: 55834574848},
				}, t.Attrs)
				require.Len(t.PrimaryKey.Parts, 1)
				require.True(t.PrimaryKey.Parts[0].C == t.Columns[0])
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint"}}, Attrs: []schema.Attr{&AutoIncrement{V: 55834574848}}},
				}, t.Columns)
			},
		},
		{
			name: "int types",
			before: func(m mock) {
				m.ExpectQuery(queryTable).
					WithArgs("public").
					WillReturnRows(sqltest.Rows(`
+--------------+--------------+--------------------+--------------------+----------------+---------------+--------------------+------------------+------------------+
| TABLE_SCHEMA | TABLE_NAME   | CHARACTER_SET_NAME | TABLE_COLLATION    | AUTO_INCREMENT | TABLE_COMMENT | CREATE_OPTIONS     |      ENGINE      |  DEFAULT_ENGINE  |
+--------------+--------------+--------------------+--------------------+----------------+---------------+--------------------+------------------+------------------+
| public       | users        | utf8mb4            | utf8mb4_0900_ai_ci | nil            | Comment       | COMPRESSION="ZLIB" |       InnoDB     |       1          |
+--------------+--------------+--------------------+--------------------+----------------+---------------+--------------------+------------------+------------------+
`))
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+--------------------+------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| table_name | column_name        | column_type                  | column_comment       | is_nullable | column_key | column_default | extra          | character_set_name | collation_name | generation_expression     |
+----------- +--------------------+------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| users      | id                 | bigint(20)                   |                      | NO          | PRI        | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_tiny           | tinyint(1)                   |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_tiny_unsigned  | tinyint(4) unsigned          |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_small          | smallint(6)                  |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_small_unsigned | smallint(6) unsigned         |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_int            | bigint(11)                   |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v57_int_unsigned   | bigint(11) unsigned          |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_tiny            | tinyint                      |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_tiny_unsigned   | tinyint unsigned             |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_small           | smallint                     |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_small_unsigned  | smallint unsigned            |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_big             | bigint                       |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_big_unsigned    | bigint unsigned              | comment              | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      | v8_big_zerofill    | bigint(20) unsigned zerofill | comment              | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
+------------+--------------------+------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "v57_tiny", Type: &schema.ColumnType{Raw: "tinyint(1)", Type: &schema.BoolType{T: "bool"}}},
					{Name: "v57_tiny_unsigned", Type: &schema.ColumnType{Raw: "tinyint(4) unsigned", Type: &schema.IntegerType{T: "tinyint", Unsigned: true}}},
					{Name: "v57_small", Type: &schema.ColumnType{Raw: "smallint(6)", Type: &schema.IntegerType{T: "smallint"}}},
					{Name: "v57_small_unsigned", Type: &schema.ColumnType{Raw: "smallint(6) unsigned", Type: &schema.IntegerType{T: "smallint", Unsigned: true}}},
					{Name: "v57_int", Type: &schema.ColumnType{Raw: "bigint(11)", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "v57_int_unsigned", Type: &schema.ColumnType{Raw: "bigint(11) unsigned", Type: &schema.IntegerType{T: "bigint", Unsigned: true}}},
					{Name: "v8_tiny", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
					{Name: "v8_tiny_unsigned", Type: &schema.ColumnType{Raw: "tinyint unsigned", Type: &schema.IntegerType{T: "tinyint", Unsigned: true}}},
					{Name: "v8_small", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint"}}},
					{Name: "v8_small_unsigned", Type: &schema.ColumnType{Raw: "smallint unsigned", Type: &schema.IntegerType{T: "smallint", Unsigned: true}}},
					{Name: "v8_big", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "v8_big_unsigned", Type: &schema.ColumnType{Raw: "bigint unsigned", Type: &schema.IntegerType{T: "bigint", Unsigned: true}}, Attrs: []schema.Attr{&schema.Comment{Text: "comment"}}},
					{Name: "v8_big_zerofill", Type: &schema.ColumnType{Raw: "bigint(20) unsigned zerofill", Type: &schema.IntegerType{T: "bigint", Unsigned: true, Attrs: []schema.Attr{&DisplayWidth{N: 20}, &ZeroFill{A: "zerofill"}}}}, Attrs: []schema.Attr{&schema.Comment{Text: "comment"}}},
				}, t.Columns)
			},
		},
		{
			name:    "maria/types",
			version: "10.7.1-MariaDB",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+----------------+-------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| table_name |  column_name   | column_type                   | column_comment       | is_nullable | column_key | column_default | extra          | character_set_name | collation_name | generation_expression     |
+------------+----------------+-------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
| users      |  id            | bigint(20)                    |                      | NO          | PRI        | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  tiny_int      | tinyint(1)                    |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  longtext      | longtext                      |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  jsonc         | longtext                      |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  dt0           | datetime /* mariadb-5.3 */    |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  dt1           | datetime(6) /* mariadb-5.3 */ |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
| users      |  dt2           | time(1) /* mariadb-5.3 */     |                      | NO          |            | NULL           |                | NULL               | NULL           | NULL                      |
+------------+----------------+-------------------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+---------------------------+
`))
				m.ExpectQuery(queryIndexes).
					WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "non_unique", "key_part", "expression"}))
				m.noFKs()
				m.ExpectQuery(queryMarChecks).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+--------+------------------+-------------------------------------------+------------+
| table  | CONSTRAINT_NAME  | CHECK_CLAUSE                              |  ENFORCED  |
+--------+------------------+-------------------------------------------+------------+
| users  | jsonc            | json_valid(` + "`jsonc`" + `)             |  YES       |
| users  | users_chk_1      | longtext <> '\'\'""'                      |  YES       |
+--------+------------------+-------------------------------------------+------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "tiny_int", Type: &schema.ColumnType{Raw: "tinyint(1)", Type: &schema.BoolType{T: "bool"}}},
					{Name: "longtext", Type: &schema.ColumnType{Raw: "longtext", Type: &schema.StringType{T: "longtext"}}},
					{Name: "jsonc", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
					{Name: "dt0", Type: &schema.ColumnType{Raw: "datetime /* mariadb-5.3 */", Type: &schema.TimeType{T: "datetime"}}},
					{Name: "dt1", Type: &schema.ColumnType{Raw: "datetime(6) /* mariadb-5.3 */", Type: &schema.TimeType{T: "datetime", Precision: sqlx.P(6)}}},
					{Name: "dt2", Type: &schema.ColumnType{Raw: "time(1) /* mariadb-5.3 */", Type: &schema.TimeType{T: "time", Precision: sqlx.P(1)}}},
				}, t.Columns)
				require.EqualValues([]schema.Attr{
					&schema.Check{Name: "users_chk_1", Expr: `longtext <> '\'\'""'`},
				}, t.Attrs)
			},
		},
		{
			name: "decimal types",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+--------------+------------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| table_name |  column_name |      column_type       | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name | generation_expression     |
+------------+--------------+------------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      |  d1          | decimal(10,2)          |                | NO          |            | 10.20          |       | NULL               | NULL           | NULL                      |
| users      |  d2          | decimal(10,0)          |                | NO          |            | 10             |       | NULL               | NULL           | NULL                      |
| users      |  d3          | decimal(10,2) unsigned |                | NO          |            | 10.20          |       | NULL               | NULL           | NULL                      |
| users      |  d4          | decimal(10,0) unsigned |                | NO          |            | 10             |       | NULL               | NULL           | NULL                      |
+------------+-------------+-------------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "d1", Type: &schema.ColumnType{Raw: "decimal(10,2)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}}, Default: &schema.Literal{V: "10.20"}},
					{Name: "d2", Type: &schema.ColumnType{Raw: "decimal(10,0)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 0}}, Default: &schema.Literal{V: "10"}},
					{Name: "d3", Type: &schema.ColumnType{Raw: "decimal(10,2) unsigned", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2, Unsigned: true}}, Default: &schema.Literal{V: "10.20"}},
					{Name: "d4", Type: &schema.ColumnType{Raw: "decimal(10,0) unsigned", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 0, Unsigned: true}}, Default: &schema.Literal{V: "10"}},
				}, t.Columns)
			},
		},
		{
			name: "float types",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+----------------------------+--------------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| table_name |    column_name             | column_type              | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name | generation_expression     |
+------------+----------------------------+--------------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      |  float                     | float                    |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
| users      |  double                    | double                   |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
| users      |  float_unsigned            | float unsigned           |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
| users      |  double_unsigned           | double unsigned          |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
| users      |  float_unsigned_p          | float(10) unsigned       |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
| users      |  doubled_zerofill_unsigned | double unsigned zerofill |                | NO          |            |                |       | NULL               | NULL           | NULL                      |
+------------+-------------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "float", Type: &schema.ColumnType{Raw: "float", Type: &schema.FloatType{T: "float"}}},
					{Name: "double", Type: &schema.ColumnType{Raw: "double", Type: &schema.FloatType{T: "double"}}},
					{Name: "float_unsigned", Type: &schema.ColumnType{Raw: "float unsigned", Type: &schema.FloatType{T: "float", Unsigned: true}}},
					{Name: "double_unsigned", Type: &schema.ColumnType{Raw: "double unsigned", Type: &schema.FloatType{T: "double", Unsigned: true}}},
					{Name: "float_unsigned_p", Type: &schema.ColumnType{Raw: "float(10) unsigned", Type: &schema.FloatType{T: "float", Precision: 10, Unsigned: true}}},
					{Name: "doubled_zerofill_unsigned", Type: &schema.ColumnType{Raw: "double unsigned zerofill", Type: &schema.FloatType{T: "double", Unsigned: true}}},
				}, t.Columns)
			},
		},
		{
			name: "binary types",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+--------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| table_name |  column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name | generation_expression     |
+------------+--------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      |  c1          | binary(20)    |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      |  c2          | varbinary(30) |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      |  c3          | tinyblob      |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      |  c4          | mediumblob    |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      |  c5          | blob          |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      |  c6          | longblob      |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
+------------+--------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				p := func(i int) *int { return &i }
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "binary(20)", Type: &schema.BinaryType{T: "binary", Size: p(20)}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "varbinary(30)", Type: &schema.BinaryType{T: "varbinary", Size: p(30)}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "tinyblob", Type: &schema.BinaryType{T: "tinyblob"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "mediumblob", Type: &schema.BinaryType{T: "mediumblob"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "blob", Type: &schema.BinaryType{T: "blob"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "longblob", Type: &schema.BinaryType{T: "longblob"}}},
				}, t.Columns)
			},
		},
		{
			name: "bit type",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| TABLE_NAME |COLUMN_NAME | COLUMN_TYPE | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA | CHARACTER_SET_NAME | COLLATION_NAME | GENERATION_EXPRESSION     |
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      |c1          | bit        |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                       |
| users      |c2          | bit(1)     |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                       |
| users      |c3          | bit(2)     |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                       |
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "bit", Type: &BitType{T: "bit"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "bit(1)", Type: &BitType{T: "bit", Size: 1}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "bit(2)", Type: &BitType{T: "bit", Size: 2}}},
				}, t.Columns)
			},
		},
		{
			name: "string types",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+---------------+----------------+-------------+------------+--------------------------------------------+-------------------+--------------------+----------------+---------------------------+
| table_name | column_name | column_type   | column_comment | is_nullable | column_key | column_default                             | extra             | character_set_name | collation_name | generation_expression     |
+------------+-------------+---------------+----------------+-------------+------------+--------------------------------------------+-------------------+--------------------+----------------+---------------------------+
| users      | c1          | char(20)      |                | NO          |            | char                                       |                   | NULL               | NULL           | NULL                      |
| users      | c2          | varchar(30)   |                | NO          |            | NULL                                       |                   | NULL               | NULL           | NULL                      |
| users      | c3          | tinytext      |                | NO          |            | NULL                                       |                   | NULL               | NULL           | NULL                      |
| users      | c4          | mediumtext    |                | NO          |            | NULL                                       |                   | NULL               | NULL           | NULL                      |
| users      | c5          | text          |                | NO          |            | NULL                                       |                   | NULL               | NULL           | NULL                      |
| users      | c6          | longtext      |                | NO          |            | NULL                                       |                   | NULL               | NULL           | NULL                      |
| users      | c7          | varchar(20)   |                | NO          |            | concat(_latin1\'Hello \',` + "`name`" + `) | DEFAULT_GENERATED | NULL               | NULL           | NULL                      |
+------------+-------------+---------------+----------------+-------------+------------+--------------------------------------------+-------------------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "char(20)", Type: &schema.StringType{T: "char", Size: 20}}, Default: &schema.Literal{V: `"char"`}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "varchar(30)", Type: &schema.StringType{T: "varchar", Size: 30}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "tinytext", Type: &schema.StringType{T: "tinytext"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "mediumtext", Type: &schema.StringType{T: "mediumtext"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "longtext", Type: &schema.StringType{T: "longtext"}}},
					{Name: "c7", Type: &schema.ColumnType{Raw: "varchar(20)", Type: &schema.StringType{T: "varchar", Size: 20}}, Default: &schema.RawExpr{X: "(concat(_latin1'Hello ',`name`))"}},
				}, t.Columns)
			},
		},
		{
			name: "enum type",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+---------------------------+
| table_name | column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name    | generation_expression     |
+------------+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+---------------------------+
| users      | c1          | enum('a','b') |                | NO          |            | NULL           |       | latin1             | latin1_swedish_ci | NULL                      |
| users      | c2          | enum('c','d') |                | NO          |            | d              |       | latin1             | latin1_swedish_ci | NULL                      |
+------------+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "enum('a','b')", Type: &schema.EnumType{T: "enum", Values: []string{"a", "b"}}}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}, &schema.Collation{V: "latin1_swedish_ci"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "enum('c','d')", Type: &schema.EnumType{T: "enum", Values: []string{"c", "d"}}}, Default: &schema.Literal{V: `"d"`}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}, &schema.Collation{V: "latin1_swedish_ci"}}},
				}, t.Columns)
			},
		},
		{
			name: "time type",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+--------------+-------------------+-------------+------------+----------------------+--------------------------------+--------------------+----------------+---------------------------+
| table_name | column_name | column_type  | column_comment    | is_nullable | column_key | column_default       | extra                          | character_set_name | collation_name | generation_expression     |
+------------+-------------+--------------+-------------------+-------------+------------+----------------------+--------------------------------+--------------------+----------------+---------------------------+
| users      | c1          | date         |                   | NO          |            | NULL                 |                                | NULL               | NULL           | NULL                      |
| users      | c2          | datetime     |                   | NO          |            | NULL                 |                                | NULL               | NULL           | NULL                      |
| users      | c3          | time         |                   | NO          |            | NULL                 |                                | NULL               | NULL           | NULL                      |
| users      | c4          | timestamp    |                   | NO          |            | CURRENT_TIMESTAMP    | on update CURRENT_TIMESTAMP    | NULL               | NULL           | NULL                      |
| users      | c5          | year(4)      |                   | NO          |            | NULL                 |                                | NULL               | NULL           | NULL                      |
| users      | c6          | year         |                   | NO          |            | NULL                 |                                | NULL               | NULL           | NULL                      |
| users      | c7          | timestamp(6) |                   | NO          |            | CURRENT_TIMESTAMP(6) | on update CURRENT_TIMESTAMP(6) | NULL               | NULL           | NULL                      |
+------------+--------------+-------------------+-------------+------------+----------------------+--------------------------------+--------------------+-------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				p := func(i int) *int { return &i }
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "date", Type: &schema.TimeType{T: "date"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "datetime", Type: &schema.TimeType{T: "datetime"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "time", Type: &schema.TimeType{T: "time"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}, Attrs: []schema.Attr{&OnUpdate{A: "CURRENT_TIMESTAMP"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "year(4)", Type: &schema.TimeType{T: "year", Precision: p(4)}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "year", Type: &schema.TimeType{T: "year"}}},
					{Name: "c7", Type: &schema.ColumnType{Raw: "timestamp(6)", Type: &schema.TimeType{T: "timestamp", Precision: p(6)}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP(6)"}, Attrs: []schema.Attr{&OnUpdate{A: "CURRENT_TIMESTAMP(6)"}}},
				}, t.Columns)
			},
		},
		{
			name: "json type",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| TABLE_NAME |COLUMN_NAME | COLUMN_TYPE | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA | CHARACTER_SET_NAME | COLLATION_NAME | GENERATION_EXPRESSION     |
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      |c1          | json        |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
+------------+------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "json", Type: &schema.JSONType{T: "json"}}},
				}, t.Columns)
			},
		},
		{
			name: "spatial type",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| table_name | column_name | column_type        | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name | GENERATION_EXPRESSION     |
+------------+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
| users      | c1          | point              |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c2          | multipoint         |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c3          | linestring         |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c4          | multilinestring    |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c5          | polygon            |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c6          | multipolygon       |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c7          | geometry           |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c8          | geometrycollection |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
| users      | c9          | geomcollection     |                | NO          |            | NULL           |       | NULL               | NULL           | NULL                      |
+------------+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "point", Type: &schema.SpatialType{T: "point"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "multipoint", Type: &schema.SpatialType{T: "multipoint"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "linestring", Type: &schema.SpatialType{T: "linestring"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "multilinestring", Type: &schema.SpatialType{T: "multilinestring"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "polygon", Type: &schema.SpatialType{T: "polygon"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "multipolygon", Type: &schema.SpatialType{T: "multipolygon"}}},
					{Name: "c7", Type: &schema.ColumnType{Raw: "geometry", Type: &schema.SpatialType{T: "geometry"}}},
					{Name: "c8", Type: &schema.ColumnType{Raw: "geometrycollection", Type: &schema.SpatialType{T: "geometrycollection"}}},
					{Name: "c9", Type: &schema.ColumnType{Raw: "geomcollection", Type: &schema.SpatialType{T: "geomcollection"}}},
				}, t.Columns)
			},
		},
		{
			name: "generated columns",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+-------------+----------------+-------------+------------+----------------+-------------------+--------------------+----------------+--------------------------------------+
| TABLE_NAME | COLUMN_NAME | COLUMN_TYPE | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA             | CHARACTER_SET_NAME | COLLATION_NAME | GENERATION_EXPRESSION                |
+------------+-------------+-------------+----------------+-------------+------------+----------------+-------------------+--------------------+----------------+--------------------------------------+
| users      | c1          | int         |                | NO          |            | NULL           |                   | NULL               | NULL           |                                      |
| users      | c2          | int         |                | NO          |            | NULL           | VIRTUAL GENERATED | NULL               | NULL           | ` + "(`c1` * `c1`)" + `              |
| users      | c3          | int         |                | NO          |            | NULL           | STORED GENERATED  | NULL               | NULL           | ` + "(`c1` + `c2`)" + `              |
| users      | c4          | varchar(20) |                | NO          |            | NULL           | STORED GENERATED  | NULL               | NULL           | concat(_latin1\'\\\'\',_latin1\'"\') |
+------------+-------------+-------------+----------------+-------------+------------+----------------+-------------------+--------------------+----------------+--------------------------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&schema.GeneratedExpr{Expr: "(`c1` * `c1`)", Type: "VIRTUAL"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&schema.GeneratedExpr{Expr: "(`c1` + `c2`)", Type: "STORED"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "varchar(20)", Type: &schema.StringType{T: "varchar", Size: 20}}, Attrs: []schema.Attr{&schema.GeneratedExpr{Expr: "concat(_latin1'\\'',_latin1'\"')", Type: "STORED"}}},
				}, t.Columns)
			},
		},
		{
			name: "indexes",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| TABLE_NAME | COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     | GENERATION_EXPRESSION     |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| users      | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
| users      | nickname    | varchar(255) |                | NO          | UNI        | NULL           |                | utf8mb4            | utf8mb4_0900_ai_ci | NULL                      |
| users      | oid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               | NULL                      |
| users      | uid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               | NULL                      |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
`))
				m.ExpectQuery(queryIndexesExpr).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
| TABLE_NAME   | INDEX_NAME   | COLUMN_NAME | NON_UNIQUE | SEQ_IN_INDEX | INDEX_TYPE   | DESC    | COMMENT      | SUB_PART   | EXPRESSION       |
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
| users        | nickname     | nickname    |          0 |            1 | BTREE        | nil     |              |        255 |      NULL        |
| users        | lower_nick   | NULL        |          1 |            1 | HASH         | 0       |              |       NULL | lower(nickname)  |
| users        | non_unique   | oid         |          1 |            1 | BTREE        | 0       |              |       NULL |      NULL        |
| users        | non_unique   | uid         |          1 |            2 | BTREE        | 0       |              |       NULL |      NULL        |
| users        | PRIMARY      | id          |          0 |            1 | BTREE        | 0       |              |       NULL |      NULL        |
| users        | unique_index | uid         |          0 |            1 | BTREE        | 1       |              |       NULL |      NULL        |
| users        | unique_index | oid         |          0 |            2 | BTREE        | 1       |              |       NULL |      NULL        |
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
`))
				m.noFKs()
				m.ExpectQuery(sqltest.Escape("SHOW CREATE TABLE `public`.`users`")).
					WillReturnRows(sqltest.Rows(`
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
| Table | Create Table                                                                                                                                |
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
| users | CREATE TABLE users (id bigint NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=55834574848 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin |
+-------+---------------------------------------------------------------------------------------------------------------------------------------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				indexes := []*schema.Index{
					{Name: "nickname", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "BTREE"}}}, // Implicitly created by the UNIQUE clause.
					{Name: "lower_nick", Table: t, Attrs: []schema.Attr{&IndexType{T: "HASH"}}},
					{Name: "non_unique", Table: t, Attrs: []schema.Attr{&IndexType{T: "BTREE"}}},
					{Name: "unique_index", Unique: true, Table: t, Attrs: []schema.Attr{&IndexType{T: "BTREE"}}},
				}
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					{Name: "nickname", Type: &schema.ColumnType{Raw: "varchar(255)", Type: &schema.StringType{T: "varchar", Size: 255}}, Indexes: indexes[0:1], Attrs: []schema.Attr{&schema.Charset{V: "utf8mb4"}, &schema.Collation{V: "utf8mb4_0900_ai_ci"}}},
					{Name: "oid", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Indexes: indexes[2:]},
					{Name: "uid", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Indexes: indexes[2:]},
				}
				// nickname
				indexes[0].Parts = []*schema.IndexPart{
					{SeqNo: 1, C: columns[1], Attrs: []schema.Attr{&SubPart{Len: 255}}},
				}
				// lower(nickname)
				indexes[1].Parts = []*schema.IndexPart{
					{SeqNo: 1, X: &schema.RawExpr{X: "lower(nickname)"}},
				}
				// oid, uid
				indexes[2].Parts = []*schema.IndexPart{
					{SeqNo: 1, C: columns[2]},
					{SeqNo: 2, C: columns[3]},
				}
				// uid, oid
				indexes[3].Parts = []*schema.IndexPart{
					{SeqNo: 1, C: columns[3], Desc: true},
					{SeqNo: 2, C: columns[2], Desc: true},
				}
				require.EqualValues(columns, t.Columns)
				require.EqualValues(indexes, t.Indexes)
			},
		},
		{
			name:    "indexes/not_support_comment",
			version: "5.1.60",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumnsNoExpr).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| TABLE_NAME | COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     | GENERATION_EXPRESSION     |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| users      | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
`))
				m.ExpectQuery(queryIndexesNoComment).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
| TABLE_NAME   | INDEX_NAME   | COLUMN_NAME | NON_UNIQUE | SEQ_IN_INDEX | INDEX_TYPE   | DESC    | COMMENT      | SUB_PART   | EXPRESSION       |
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
| users        | PRIMARY      | id          |          0 |            1 | BTREE        | 0       | NULL         |       NULL |      NULL        |
+--------------+--------------+-------------+------------+--------------+--------------+---------+--------------+------------+------------------+
`))
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				// nothing to expect, ExpectQuery is enough for this test
				require.NoError(err)
			},
		},
		{
			name: "fks",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| TABLE_NAME | COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     | GENERATION_EXPRESSION     |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| users      | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
| users      | oid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               | NULL                      |
| users      | uid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               | NULL                      |
+------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
`))
				m.noIndexes()
				m.ExpectQuery(queryFKs).
					WithArgs("public", "public", "users").
					WillReturnRows(sqltest.Rows(`
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| CONSTRAINT_NAME  | TABLE_NAME | COLUMN_NAME | TABLE_SCHEMA | REFERENCED_TABLE_NAME | REFERENCED_COLUMN_NAME | REFERENCED_SCHEMA_NAME | UPDATE_RULE | DELETE_RULE |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| multi_column     | users      | id          | public       | t1                    | gid                    | public                 | NO ACTION   | CASCADE     |
| multi_column     | users      | oid         | public       | t1                    | xid                    | public                 | NO ACTION   | CASCADE     |
| self_reference   | users      | uid         | public       | users                 | id                     | public                 | NO ACTION   | CASCADE     |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+ ------------+-------------+
`))
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
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, ForeignKeys: fks[0:1]},
					{Name: "oid", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, ForeignKeys: fks[0:1]},
					{Name: "uid", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, ForeignKeys: fks[1:2]},
				}
				fks[0].Columns = columns[:2]
				fks[1].Columns = columns[2:]
				fks[1].RefColumns = columns[:1]
				require.EqualValues(columns, t.Columns)
				require.EqualValues(fks, t.ForeignKeys)
			},
		},
		{
			name:    "checks",
			version: "8.0.16",
			before: func(m mock) {
				m.tableExists("public", "users", true)
				m.ExpectQuery(queryColumns).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| TABLE_NAME  | COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     | GENERATION_EXPRESSION     |
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| users       | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
| users       | c1          | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               | NULL                      |
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
`))
				m.noIndexes()
				m.noFKs()
				m.ExpectQuery(queryMyChecks).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
+-------------------+-------------------+-------------------------------------------+------------+
| TABLE_NAME        | CONSTRAINT_NAME   | CHECK_CLAUSE                              |  ENFORCED  |
+-------------------+-------------------+-------------------------------------------+------------+
| users             | users_chk_1       | (` + "`c6`" + ` <> _latin1\'foo\\\'s\')   |  YES       |
| users             | users_chk_2       | (c1 <> _latin1\'dev/atlas\')              |  YES       |
| users             | users_chk_3       | (c1 <> _latin1\'a\\\'b""\')               |  YES       |
| users             | users_chk_4       | (c1 <> in (_latin1\'usa\',_latin1\'uk\')) |  YES       |
| users             | users_chk_5       | (c1 <> _latin1\'\\\\\\\\\\\'\\\'\')       |  YES       |
+-------------------+-------------------+-------------------------------------------+------------+
`))
				m.ExpectQuery(sqltest.Escape("SHOW CREATE TABLE `public`.`users`")).
					WillReturnRows(sqltest.Rows(`
+-------+------------------------+
| Table | Create Table           |
+-------+------------------------+
| users | CREATE TABLE users()   |
+-------+------------------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Equal("public", t.Schema.Name)
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
				}
				require.EqualValues(columns, t.Columns)
				require.EqualValues([]schema.Attr{
					&schema.Check{Name: "users_chk_1", Expr: "(`c6` <> _latin1'foo\\'s')"},
					&schema.Check{Name: "users_chk_2", Expr: "(c1 <> _latin1'dev/atlas')"},
					&schema.Check{Name: "users_chk_3", Expr: `(c1 <> _latin1'a\'b""')`},
					&schema.Check{Name: "users_chk_4", Expr: `(c1 <> in (_latin1'usa',_latin1'uk'))`},
					&schema.Check{Name: "users_chk_5", Expr: `(c1 <> _latin1'\\\\\'\'')`},
				}, t.Attrs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			mk := mock{m}
			if tt.version == "" {
				tt.version = "8.0.13"
			}
			mk.version(tt.version)
			mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= ?"))).
				WithArgs("public").
				WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| public      | utf8mb4                    | utf8mb4_unicode_ci     |
+-------------+----------------------------+------------------------+
				`))
			tt.before(mk)
			drv, err := Open(db)
			require.NoError(t, err)
			s, err := drv.InspectSchema(context.Background(), "public", &schema.InspectOptions{
				Mode: ^schema.InspectViews,
			})
			require.NoError(t, err)
			require.NotNil(t, s)
			tt.expect(require.New(t), s.Tables[0], err)
		})
	}
}

func TestDriver_InspectSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		before func(mock)
		expect func(*require.Assertions, *schema.Schema, error)
	}{
		{
			name: "attached schema",
			before: func(m mock) {
				m.version("5.7.23")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= SCHEMA()"))).
					WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| public      | utf8mb4                    | utf8mb4_unicode_ci     |
+-------------+----------------------------+------------------------+
				`))
				m.tables("public")
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				require.EqualValues(func() *schema.Schema {
					realm := &schema.Realm{
						Schemas: []*schema.Schema{
							{
								Name: "public",
								Attrs: []schema.Attr{
									&schema.Charset{V: "utf8mb4"},
									&schema.Collation{V: "utf8mb4_unicode_ci"},
								},
							},
						},
						Attrs: []schema.Attr{
							&schema.Charset{
								V: "utf8",
							},
							&schema.Collation{
								V: "utf8_general_ci",
							},
						},
					}
					realm.Schemas[0].Realm = realm
					return realm.Schemas[0]
				}(), s)
			},
		},
		{
			name:   "multi table",
			schema: "public",
			before: func(m mock) {
				m.version("8.0.13")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= ?"))).
					WithArgs("public").
					WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| public      | utf8mb4                    | utf8mb4_unicode_ci     |
+-------------+----------------------------+------------------------+
`))
				m.tables("public", "users", "pets")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsExprQuery, "?, ?"))).
					WithArgs("public", "users", "pets").
					WillReturnRows(sqltest.Rows(`
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| TABLE_NAME  | COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     | GENERATION_EXPRESSION     |
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
| users       | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
| users       | spouse_id   | int          |                | YES         | NULL       | NULL           |                | NULL               | NULL               | NULL                      |
| pets        | id          | int          |                | NO          | PRI        | NULL           |                | NULL               | NULL               | NULL                      |
| pets        | owner_id    | int          |                | YES         | NULL       | NULL           |                | NULL               | NULL               | NULL                      |
+-------------+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+---------------------------+
				`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesExprQuery, "?, ?"))).
					WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "non_unique", "key_part", "expression"}))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, "?, ?"))).
					WithArgs("public", "public", "users", "pets").
					WillReturnRows(sqltest.Rows(`
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| CONSTRAINT_NAME  | TABLE_NAME | COLUMN_NAME | TABLE_SCHEMA | REFERENCED_TABLE_NAME | REFERENCED_COLUMN_NAME | REFERENCED_SCHEMA_NAME | UPDATE_RULE | DELETE_RULE |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| spouse_id        | users      | spouse_id   | public       | users                 | id                     | public                 | NO ACTION   | CASCADE     |
| owner_id         | pets       | owner_id    | public       | users                 | id                     | public                 | NO ACTION   | CASCADE     |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
				`))
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				ts := s.Tables
				require.Len(ts, 2)
				users, pets := ts[0], ts[1]

				require.Equal("users", users.Name)
				userFKs := []*schema.ForeignKey{
					{Symbol: "spouse_id", Table: users, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: users},
				}
				userColumns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					{Name: "spouse_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}, Null: true}, ForeignKeys: userFKs},
				}
				userFKs[0].Columns = userColumns[1:]
				userFKs[0].RefColumns = userColumns[:1]
				require.EqualValues(userColumns, users.Columns)
				require.EqualValues(userFKs, users.ForeignKeys)

				require.Equal("pets", pets.Name)
				petsFKs := []*schema.ForeignKey{
					{Symbol: "owner_id", Table: pets, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: users, RefColumns: userColumns[:1]},
				}
				petsColumns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
					{Name: "owner_id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}, Null: true}, ForeignKeys: petsFKs},
				}
				petsFKs[0].Columns = petsColumns[1:]
				require.EqualValues(petsColumns, pets.Columns)
				require.EqualValues(petsFKs, pets.ForeignKeys)
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
			tables, err := drv.InspectSchema(context.Background(), tt.schema, &schema.InspectOptions{
				Mode: ^schema.InspectViews,
			})
			tt.expect(require.New(t), tables, err)
		})
	}
}

func TestDriver_Realm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| test        | utf8mb4                    | utf8mb4_unicode_ci     |
+-------------+----------------------------+------------------------+
`))
	mk.tables("test")
	drv, err := Open(db)
	require.NoError(t, err)
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Mode: ^schema.InspectViews,
	})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
					Attrs: []schema.Attr{
						&schema.Charset{V: "utf8mb4"},
						&schema.Collation{V: "utf8mb4_unicode_ci"},
					},
				},
			},
			// Server default configuration.
			Attrs: []schema.Attr{
				&schema.Charset{
					V: "utf8",
				},
				&schema.Collation{
					V: "utf8_general_ci",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r
	}(), realm)

	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "IN (?, ?)"))).
		WithArgs("test", "public").
		WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| test        | utf8mb4                    | utf8mb4_unicode_ci     |
| public      | utf8                       | utf8_general_ci        |
+-------------+----------------------------+------------------------+
`))
	mk.ExpectQuery(sqltest.Escape(fmt.Sprintf(tablesQuery, "?, ?"))).
		WithArgs("test", "public").
		WillReturnRows(sqlmock.NewRows([]string{"schema", "table", "charset", "collate", "inc", "comment", "options"}))
	realm, err = drv.InspectRealm(context.Background(), &schema.InspectRealmOption{
		Mode:    ^schema.InspectViews,
		Schemas: []string{"test", "public"},
	})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
					Attrs: []schema.Attr{
						&schema.Charset{V: "utf8mb4"},
						&schema.Collation{V: "utf8mb4_unicode_ci"},
					},
				},
				{
					Name: "public",
					Attrs: []schema.Attr{
						&schema.Charset{V: "utf8"},
						&schema.Collation{V: "utf8_general_ci"},
					},
				},
			},
			// Server default configuration.
			Attrs: []schema.Attr{
				&schema.Charset{
					V: "utf8",
				},
				&schema.Collation{
					V: "utf8_general_ci",
				},
			},
		}
		r.Schemas[0].Realm = r
		r.Schemas[1].Realm = r
		return r
	}(), realm)
}

func TestInspectMode_InspectRealm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqltest.Rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| test        | latin1                     | lain1_ci               |
+-------------+----------------------------+------------------------+
`))
	drv, err := Open(db)
	require.NoError(t, err)
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Mode: schema.InspectSchemas})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "test",
					Attrs: []schema.Attr{
						&schema.Charset{V: "latin1"},
						&schema.Collation{V: "lain1_ci"},
					},
				},
			},
			// Server default configuration.
			Attrs: []schema.Attr{
				&schema.Charset{
					V: "utf8",
				},
				&schema.Collation{
					V: "utf8_general_ci",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r
	}(), realm)
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	m.ExpectQuery(sqltest.Escape(variablesQuery)).
		WillReturnRows(sqltest.Rows(`
+-----------------+--------------------+------------------------+--------------------------+ 
| @@version       | @@collation_server | @@character_set_server | @@lower_case_table_names | 
+-----------------+--------------------+------------------------+--------------------------+ 
| ` + version + ` | utf8_general_ci    | utf8                   | 0                        | 
+-----------------+--------------------+------------------------+--------------------------+ 
`))
}

func (m mock) lcmode(version, mode string) {
	m.ExpectQuery(sqltest.Escape(variablesQuery)).
		WillReturnRows(sqltest.Rows(`
+-----------------+--------------------+------------------------+--------------------------+ 
| @@version       | @@collation_server | @@character_set_server | @@lower_case_table_names | 
+-----------------+--------------------+------------------------+--------------------------+ 
| ` + version + ` | utf8_general_ci    | utf8                   | ` + mode + `             |
+-----------------+--------------------+------------------------+--------------------------+ 
`))
}

func (m mock) noIndexes() {
	m.ExpectQuery(queryIndexesExpr).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "non_unique", "key_part", "expression"}))
}

func (m mock) noFKs() {
	m.ExpectQuery(queryFKs).
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "CONSTRAINT_NAME", "TABLE_NAME", "COLUMN_NAME", "REFERENCED_TABLE_NAME", "REFERENCED_COLUMN_NAME", "REFERENCED_TABLE_SCHEMA", "UPDATE_RULE", "DELETE_RULE"}))
}

func (m mock) tableExists(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_name", "table_collation", "character_set", "auto_increment", "table_comment", "create_options", "engine", "default_engine"})
	if exists {
		rows.AddRow(schema, table, nil, nil, nil, nil, nil, nil, nil)
	}
	m.ExpectQuery(queryTable).
		WithArgs(schema).
		WillReturnRows(rows)
}

func (m mock) tables(schema string, tables ...string) {
	rows := sqlmock.NewRows([]string{"schema", "table", "charset", "collate", "inc", "comment", "options", "engine", "default_engine"})
	for _, t := range tables {
		rows.AddRow(schema, t, nil, nil, nil, nil, nil, nil, nil)
	}
	m.ExpectQuery(queryTable).
		WithArgs(schema).
		WillReturnRows(rows)
}
