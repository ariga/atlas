package mysql

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDriver_Table(t *testing.T) {
	tests := []struct {
		name   string
		opts   *schema.InspectTableOptions
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "table does not exist",
			before: func(m mock) {
				m.version("5.7.23")
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
				Schema: "public",
			},
			before: func(m mock) {
				m.version("5.7.23")
				m.tableExistsInSchema("public", "users", false)
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.Nil(t)
				require.Error(err)
				require.True(schema.IsNotExistError(err), "expect not exists error")
			},
		},
		{
			name: "table collation",
			before: func(m mock) {
				m.version("8.0.13")
				m.ExpectQuery(escape(tableQuery)).
					WithArgs("users").
					WillReturnRows(rows(`
+--------------+--------------------+--------------------+----------------+---------------+
| TABLE_SCHEMA | CHARACTER_SET_NAME | TABLE_COLLATION    | AUTO_INCREMENT | TABLE_COMMENT |
+--------------+--------------------+--------------------+----------------+---------------+
| test         | utf8mb4            | utf8mb4_0900_ai_ci | nil            | Comment       |
+--------------+--------------------+--------------------+----------------+---------------+
`))
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("test", "users").
					WillReturnRows(rows(`
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| column_name        | column_type          | column_comment       | is_nullable | column_key | column_default | extra          | character_set_name | collation_name |
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| id                 | bigint(20)           |                      | NO          | PRI        | NULL           | auto_increment | NULL               | NULL           |
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]schema.Attr{
					&schema.Charset{V: "utf8mb4"},
					&schema.Collation{V: "utf8mb4_0900_ai_ci"},
					&schema.Comment{Text: "Comment"},
				}, t.Attrs)
				require.Len(t.PrimaryKey, 1)
				require.True(t.PrimaryKey[0] == t.Columns[0])
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint", Size: 20}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
				}, t.Columns)
			},
		},
		{
			name: "int types",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| column_name        | column_type          | column_comment       | is_nullable | column_key | column_default | extra          | character_set_name | collation_name |
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| id                 | bigint(20)           |                      | NO          | PRI        | NULL           | auto_increment | NULL               | NULL           |
| v57_tiny           | tinyint(1)           |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v57_tiny_unsigned  | tinyint(4) unsigned  |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v57_small          | smallint(6)          |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v57_small_unsigned | smallint(6) unsigned |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v57_int            | bigint(11)           |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v57_int_unsigned   | bigint(11) unsigned  |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_tiny            | tinyint              |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_tiny_unsigned   | tinyint unsigned     |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_small           | smallint             |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_small_unsigned  | smallint unsigned    |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_big             | bigint               |                      | NO          |            | NULL           |                | NULL               | NULL           |
| v8_big_unsigned    | bigint unsigned      | comment              | NO          |            | NULL           |                | NULL               | NULL           |
+--------------------+----------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Len(t.PrimaryKey, 1)
				require.True(t.PrimaryKey[0] == t.Columns[0])
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint", Size: 20}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
					{Name: "v57_tiny", Type: &schema.ColumnType{Raw: "tinyint(1)", Type: &schema.BoolType{T: "tinyint"}}},
					{Name: "v57_tiny_unsigned", Type: &schema.ColumnType{Raw: "tinyint(4) unsigned", Type: &schema.IntegerType{T: "tinyint", Size: 4, Unsigned: true}}},
					{Name: "v57_small", Type: &schema.ColumnType{Raw: "smallint(6)", Type: &schema.IntegerType{T: "smallint", Size: 6}}},
					{Name: "v57_small_unsigned", Type: &schema.ColumnType{Raw: "smallint(6) unsigned", Type: &schema.IntegerType{T: "smallint", Size: 6, Unsigned: true}}},
					{Name: "v57_int", Type: &schema.ColumnType{Raw: "bigint(11)", Type: &schema.IntegerType{T: "bigint", Size: 11}}},
					{Name: "v57_int_unsigned", Type: &schema.ColumnType{Raw: "bigint(11) unsigned", Type: &schema.IntegerType{T: "bigint", Size: 11, Unsigned: true}}},
					{Name: "v8_tiny", Type: &schema.ColumnType{Raw: "tinyint", Type: &schema.IntegerType{T: "tinyint"}}},
					{Name: "v8_tiny_unsigned", Type: &schema.ColumnType{Raw: "tinyint unsigned", Type: &schema.IntegerType{T: "tinyint", Unsigned: true}}},
					{Name: "v8_small", Type: &schema.ColumnType{Raw: "smallint", Type: &schema.IntegerType{T: "smallint"}}},
					{Name: "v8_small_unsigned", Type: &schema.ColumnType{Raw: "smallint unsigned", Type: &schema.IntegerType{T: "smallint", Unsigned: true}}},
					{Name: "v8_big", Type: &schema.ColumnType{Raw: "bigint", Type: &schema.IntegerType{T: "bigint"}}},
					{Name: "v8_big_unsigned", Type: &schema.ColumnType{Raw: "bigint unsigned", Type: &schema.IntegerType{T: "bigint", Unsigned: true}}, Attrs: []schema.Attr{&schema.Comment{Text: "comment"}}},
				}, t.Columns)
			},
		},
		{
			name: "decimal types",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| d1          | decimal(10,2) |                | NO          |            | 10.20          |       | NULL               | NULL           |
| d2          | decimal(10,0) |                | NO          |            | 10             |       | NULL               | NULL           |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "d1", Type: &schema.ColumnType{Raw: "decimal(10,2)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}}, Default: &schema.RawExpr{X: "10.20"}},
					{Name: "d2", Type: &schema.ColumnType{Raw: "decimal(10,0)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 0}}, Default: &schema.RawExpr{X: "10"}},
				}, t.Columns)
			},
		},
		{
			name: "float types",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type  | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+--------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| float       | float        |                | NO          |            |                |       | NULL               | NULL           |
| double      | double       |                | NO          |            |                |       | NULL               | NULL           |
+-------------+--------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
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
				}, t.Columns)
			},
		},
		{
			name: "binary types",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | binary(20)    |                | NO          |            | NULL           |       | NULL               | NULL           |
| c2          | varbinary(30) |                | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | tinyblob      |                | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | mediumblob    |                | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | blob          |                | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | longblob      |                | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "binary(20)", Type: &schema.BinaryType{T: "binary", Size: 20}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "varbinary(30)", Type: &schema.BinaryType{T: "varbinary", Size: 30}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "tinyblob", Type: &schema.BinaryType{T: "tinyblob"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "mediumblob", Type: &schema.BinaryType{T: "mediumblob"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "blob", Type: &schema.BinaryType{T: "blob"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "longblob", Type: &schema.BinaryType{T: "longblob"}}},
				}, t.Columns)
			},
		},
		{
			name: "string types",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | char(20)      |                | NO          |            | char           |       | NULL               | NULL           |
| c2          | varchar(30)   |                | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | tinytext      |                | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | mediumtext    |                | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | text          |                | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | longtext      |                | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "char(20)", Type: &schema.StringType{T: "char", Size: 20}}, Default: &schema.RawExpr{X: "char"}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "varchar(30)", Type: &schema.StringType{T: "varchar", Size: 30}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "tinytext", Type: &schema.StringType{T: "tinytext"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "mediumtext", Type: &schema.StringType{T: "mediumtext"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "text", Type: &schema.StringType{T: "text"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "longtext", Type: &schema.StringType{T: "longtext"}}},
				}, t.Columns)
			},
		},
		{
			name: "enum type",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+
| column_name | column_type   | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name    |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+
| c1          | enum('a','b') |                | NO          |            | NULL           |       | latin1             | latin1_swedish_ci |
| c2          | enum('c','d') |                | NO          |            | d              |       | latin1             | latin1_swedish_ci |
+-------------+---------------+----------------+-------------+------------+----------------+-------+--------------------+-------------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "enum('a','b')", Type: &schema.EnumType{Values: []string{"a", "b"}}}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}, &schema.Collation{V: "latin1_swedish_ci"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "enum('c','d')", Type: &schema.EnumType{Values: []string{"c", "d"}}}, Default: &schema.RawExpr{X: "d"}, Attrs: []schema.Attr{&schema.Charset{V: "latin1"}, &schema.Collation{V: "latin1_swedish_ci"}}},
				}, t.Columns)
			},
		},
		{
			name: "time type",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+-------------+-------------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
| column_name | column_type | column_comment    | is_nullable | column_key | column_default    | extra                       | character_set_name | collation_name |
+-------------+-------------+-------------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
| c1          | date        |                   | NO          |            | NULL              |                             | NULL               | NULL           |
| c2          | datetime    |                   | NO          |            | NULL              |                             | NULL               | NULL           |
| c3          | time        |                   | NO          |            | NULL              |                             | NULL               | NULL           |
| c4          | timestamp   |                   | NO          |            | CURRENT_TIMESTAMP | on update CURRENT_TIMESTAMP | NULL               | NULL           |
| c5          | year(4)     |                   | NO          |            | NULL              |                             | NULL               | NULL           |
| c6          | year        |                   | NO          |            | NULL              |                             | NULL               | NULL           |
+-------------+-------------+-------------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
`))
				m.noIndexes()
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "date", Type: &schema.TimeType{T: "date"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "datetime", Type: &schema.TimeType{T: "datetime"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "time", Type: &schema.TimeType{T: "time"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}, Attrs: []schema.Attr{&OnUpdate{A: "on update current_timestamp"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "year(4)", Type: &schema.TimeType{T: "year"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "year", Type: &schema.TimeType{T: "year"}}},
				}, t.Columns)
			},
		},
		{
			name: "json type",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| COLUMN_NAME | COLUMN_TYPE | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA | CHARACTER_SET_NAME | COLLATION_NAME |
+-------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | json        |                | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+-------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
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
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type        | column_comment | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | point              |                | NO          |            | NULL           |       | NULL               | NULL           |
| c2          | multipoint         |                | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | linestring         |                | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | multilinestring    |                | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | polygon            |                | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | multipolygon       |                | NO          |            | NULL           |       | NULL               | NULL           |
| c7          | geometry           |                | NO          |            | NULL           |       | NULL               | NULL           |
| c8          | geometrycollection |                | NO          |            | NULL           |       | NULL               | NULL           |
| c9          | geomcollection     |                | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+--------------------+----------------+-------------+------------+----------------+-------+--------------------+----------------+
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
			name: "indexes",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| id          | int          |                | NO          | PRI        | NULL           | auto_increment | NULL               | NULL               |
| nickname    | varchar(255) |                | NO          | UNI        | NULL           |                | utf8mb4            | utf8mb4_0900_ai_ci |
| oid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               |
| uid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
`))
				m.ExpectQuery(escape(indexesExprQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+--------------+-------------+------------+--------------+------------+------------------+
| INDEX_NAME   | COLUMN_NAME | NON_UNIQUE | SEQ_IN_INDEX | SUB_PART   | EXPRESSION       |
+--------------+-------------+------------+--------------+------------+------------------+
| nickname     | nickname    |          0 |            1 |        255 |      NULL        |
| lower_nick   | NULL        |          1 |            1 |       NULL | lower(nickname)  |
| non_unique   | oid         |          1 |            1 |       NULL |      NULL        |
| non_unique   | uid         |          1 |            2 |       NULL |      NULL        |
| PRIMARY      | id          |          0 |            1 |       NULL |      NULL        |
| unique_index | uid         |          0 |            1 |       NULL |      NULL        |
| unique_index | oid         |          0 |            2 |       NULL |      NULL        |
+--------------+-------------+------------+--------------+------------+------------------+
`))
				m.noFKs()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				indexes := []*schema.Index{
					{Name: "nickname", Unique: true, Table: t}, // Implicitly created by the UNIQUE clause.
					{Name: "lower_nick", Table: t},
					{Name: "non_unique", Table: t},
					{Name: "unique_index", Unique: true, Table: t},
				}
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
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
					{SeqNo: 1, C: columns[3]},
					{SeqNo: 2, C: columns[2]},
				}
				require.EqualValues(columns, t.Columns)
				require.EqualValues(indexes, t.Indexes)
			},
		},
		{
			name: "fks",
			before: func(m mock) {
				m.version("8.0.13")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| id          | int          |                | NO          | PRI        | NULL           | auto_increment | NULL               | NULL               |
| oid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               |
| uid         | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
`))
				m.noIndexes()
				m.ExpectQuery(escape(fksQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
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
					{Symbol: "multi_column", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: &schema.Table{Name: "t1", Schema: &schema.Schema{Name: "public"}}, RefColumns: []*schema.Column{{Name: "gid"}, {Name: "xid"}}},
					{Symbol: "self_reference", Table: t, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: t},
				}
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}, ForeignKeys: fks[0:1]},
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
			name: "checks",
			before: func(m mock) {
				m.version("8.0.16")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| id          | int          |                | NO          | PRI        | NULL           | auto_increment | NULL               | NULL               |
| c1          | int          |                | NO          | MUL        | NULL           |                | NULL               | NULL               |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
`))
				m.noIndexes()
				m.noFKs()
				m.ExpectQuery(escape(checksQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------------+----------+-----------------------------------------------------------------+
| CONSTRAINT_NAME   | ENFORCED | CHECK_CLAUSE                                                    |
+-------------------+----------+-----------------------------------------------------------------+
| users_chk_1       | YES      | (` + "`c6`" + ` <>_latin1\'foo\\\'s\')                          |
+-------------------+----------+-----------------------------------------------------------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Equal("public", t.Schema.Name)
				columns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
					{Name: "c1", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}},
				}
				require.EqualValues(columns, t.Columns)
				require.EqualValues([]schema.Attr{&Check{Name: "users_chk_1", Clause: "(`c6` <>_latin1\\'foo\\'s\\')", Enforced: true}}, t.Attrs)
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
			table, err := drv.Table(context.Background(), "users", tt.opts)
			tt.expect(require.New(t), table, err)
		})
	}
}

func TestDriver_Tables(t *testing.T) {
	tests := []struct {
		name   string
		opts   *schema.InspectTableOptions
		before func(mock)
		expect func(*require.Assertions, []*schema.Table, error)
	}{
		{
			name: "no tables",
			before: func(m mock) {
				m.version("5.7.23")
				m.tables()
			},
			expect: func(require *require.Assertions, ts []*schema.Table, err error) {
				require.NoError(err)
				require.Empty(ts)
			},
		},
		{
			name: "no tables in schema",
			before: func(m mock) {
				m.version("5.7.23")
				m.tablesInSchema("public")
			},
			opts: &schema.InspectTableOptions{
				Schema: "public",
			},
			expect: func(require *require.Assertions, ts []*schema.Table, err error) {
				require.NoError(err)
				require.Empty(ts)
			},
		},
		{
			name: "multi table",
			before: func(m mock) {
				m.version("8.0.13")
				m.tables("users", "pets")
				m.tableExists("public", "users", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| id          | int          |                | NO          | PRI        | NULL           | auto_increment | NULL               | NULL               |
| spouse_id   | int          |                | YES         | NULL       | NULL           |                | NULL               | NULL               |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
`))
				m.noIndexes()
				m.ExpectQuery(escape(fksQuery)).
					WithArgs("public", "users").
					WillReturnRows(rows(`
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| CONSTRAINT_NAME  | TABLE_NAME | COLUMN_NAME | TABLE_SCHEMA | REFERENCED_TABLE_NAME | REFERENCED_COLUMN_NAME | REFERENCED_SCHEMA_NAME | UPDATE_RULE | DELETE_RULE |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| spouse_id        | users      | spouse_id   | public       | users                 | id                     | public                 | NO ACTION   | CASCADE     |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
`))

				m.tableExists("public", "pets", true)
				m.ExpectQuery(escape(columnsQuery)).
					WithArgs("public", "pets").
					WillReturnRows(rows(`
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| COLUMN_NAME | COLUMN_TYPE  | COLUMN_COMMENT | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA          | CHARACTER_SET_NAME | COLLATION_NAME     |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
| id          | int          |                | NO          | PRI        | NULL           | auto_increment | NULL               | NULL               |
| owner_id    | int          |                | YES         | NULL       | NULL           |                | NULL               | NULL               |
+-------------+--------------+----------------+-------------+------------+----------------+----------------+--------------------+--------------------+
`))
				m.noIndexes()
				m.ExpectQuery(escape(fksQuery)).
					WithArgs("public", "pets").
					WillReturnRows(rows(`
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| CONSTRAINT_NAME  | TABLE_NAME | COLUMN_NAME | TABLE_SCHEMA | REFERENCED_TABLE_NAME | REFERENCED_COLUMN_NAME | REFERENCED_SCHEMA_NAME | UPDATE_RULE | DELETE_RULE |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
| owner_id         | pets       | owner_id    | public       | users                 | id                     | public                 | NO ACTION   | CASCADE     |
+------------------+------------+-------------+--------------+-----------------------+------------------------+------------------------+-------------+-------------+
`))
			},
			expect: func(require *require.Assertions, ts []*schema.Table, err error) {
				require.NoError(err)
				require.Len(ts, 2)
				users, pets := ts[0], ts[1]

				require.Equal("users", users.Name)
				userFKs := []*schema.ForeignKey{
					{Symbol: "spouse_id", Table: users, OnUpdate: schema.NoAction, OnDelete: schema.Cascade, RefTable: users},
				}
				userColumns := []*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
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
					{Name: "id", Type: &schema.ColumnType{Raw: "int", Type: &schema.IntegerType{T: "int"}}, Attrs: []schema.Attr{&AutoIncrement{A: "auto_increment"}}},
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
			tables, err := drv.Tables(context.Background(), tt.opts)
			tt.expect(require.New(t), tables, err)
		})
	}
}

func TestDriver_Realm(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectQuery(escape(schemasQuery + " WHERE `SCHEMA_NAME` NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')")).
		WillReturnRows(rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| test        | utf8mb4                    | utf8mb4_unicode_ci     |
+-------------+----------------------------+------------------------+
`))
	mk.tablesInSchema("test")
	drv, err := Open(db)
	require.NoError(t, err)
	realm, err := drv.Realm(context.Background(), &schema.InspectRealmOption{})
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
					Tables: []*schema.Table{},
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

	mk.ExpectQuery(escape(schemasQuery+" WHERE `SCHEMA_NAME` IN (?, ?)")).
		WithArgs("test", "public").
		WillReturnRows(rows(`
+-------------+----------------------------+------------------------+
| SCHEMA_NAME | DEFAULT_CHARACTER_SET_NAME | DEFAULT_COLLATION_NAME |
+-------------+----------------------------+------------------------+
| test        | utf8mb4                    | utf8mb4_unicode_ci     |
| public      | utf8                       | utf8_general_ci        |
+-------------+----------------------------+------------------------+
`))
	mk.tablesInSchema("test")
	mk.tablesInSchema("public")
	realm, err = drv.Realm(context.Background(), &schema.InspectRealmOption{Schemas: []string{"test", "public"}})
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
					Tables: []*schema.Table{},
				},
				{
					Name: "public",
					Attrs: []schema.Attr{
						&schema.Charset{V: "utf8"},
						&schema.Collation{V: "utf8_general_ci"},
					},
					Tables: []*schema.Table{},
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

func TestDriver_Exec(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mk := mock{m}
	mk.version("8.0.13")
	mk.ExpectExec(escape("DROP TABLE `users`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("DROP TABLE `public`.`pets`")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mk.ExpectExec(escape("CREATE TABLE `pets`(`a` int NOT NULL, `b` bigint NOT NULL, `c` bigint NULL, PRIMARY KEY(`a`, `b`))")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	drv, err := Open(db)
	require.NoError(t, err)
	err = drv.Exec(context.Background(), []schema.Change{
		&schema.DropTable{T: &schema.Table{Name: "users"}},
		&schema.DropTable{T: &schema.Table{Name: "pets", Schema: &schema.Schema{Name: "public"}}},
		&schema.AddTable{
			T: func() *schema.Table {
				t := &schema.Table{
					Name: "pets",
					Columns: []*schema.Column{
						{Name: "a", Type: &schema.ColumnType{Raw: "int"}},
						{Name: "b", Type: &schema.ColumnType{Raw: "bigint"}},
						{Name: "c", Type: &schema.ColumnType{Raw: "bigint", Null: true}},
					},
				}
				t.PrimaryKey = t.Columns[:2]
				return t
			}(),
		},
	})
	require.NoError(t, err)
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	m.ExpectQuery(escape(variablesQuery)).
		WillReturnRows(rows(`
+-----------------+--------------------+------------------------+
| @@version       | @@collation_server | @@character_set_server |
+-----------------+--------------------+------------------------+
| ` + version + ` | utf8_general_ci    | utf8                   |
+-----------------+--------------------+------------------------+
`))
}

func (m mock) noIndexes() {
	m.ExpectQuery(escape(indexesExprQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "non_unique", "key_part", "expression"}))
}

func (m mock) noFKs() {
	m.ExpectQuery(escape(fksQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"CONSTRAINT_NAME", "TABLE_NAME", "COLUMN_NAME", "REFERENCED_TABLE_NAME", "REFERENCED_COLUMN_NAME", "REFERENCED_TABLE_SCHEMA", "UPDATE_RULE", "DELETE_RULE"}))
}

func (m mock) tables(names ...string) {
	rows := sqlmock.NewRows([]string{"table_name"})
	for i := range names {
		rows.AddRow(names[i])
	}
	m.ExpectQuery(escape(tablesQuery)).
		WillReturnRows(rows)
}

func (m mock) tablesInSchema(schema string, names ...string) {
	rows := sqlmock.NewRows([]string{"table_name"})
	for i := range names {
		rows.AddRow(names[i])
	}
	m.ExpectQuery(escape(tablesSchemaQuery)).
		WithArgs(schema).
		WillReturnRows(rows)
}

func (m mock) tableExists(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_collation", "character_set", "auto_increment", "table_comment"})
	if exists {
		rows.AddRow(schema, nil, nil, nil, nil)
	}
	m.ExpectQuery(escape(tableQuery)).
		WithArgs(table).
		WillReturnRows(rows)
}

func (m mock) tableExistsInSchema(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_collation", "character_set", "auto_increment", "table_comment"})
	if exists {
		rows.AddRow(schema, nil, nil, nil, nil)
	}
	m.ExpectQuery(escape(tableSchemaQuery)).
		WithArgs(table, schema).
		WillReturnRows(rows)
}

func escape(query string) string {
	rows := strings.Split(query, "\n")
	for i := range rows {
		rows[i] = strings.TrimPrefix(rows[i], " ")
	}
	query = strings.Join(rows, " ")
	return strings.TrimSpace(regexp.QuoteMeta(query)) + "$"
}

// rows converts MySQL table output to sql.Rows.
// All row values are parsed as text except the "nil" keyword. For example:
//
// 		+-------------+-------------+-------------+----------------+
//		| column_name | column_type | is_nullable | column_default |
//		+-------------+-------------+-------------+----------------+
//		| c1          | float       | YES         | nil            |
//		| c2          | int         | YES         |                |
//		| c3          | double      | YES         | NULL           |
//		+-------------+-------------+-------------+----------------+
//
func rows(table string) *sqlmock.Rows {
	var (
		rows  *sqlmock.Rows
		lines = strings.Split(table, "\n")
	)
	for i := 0; i < len(lines); i++ {
		line := strings.TrimFunc(lines[i], unicode.IsSpace)
		// Skip new lines, header and footer.
		if !strings.HasPrefix(line, "|") {
			continue
		}
		columns := strings.FieldsFunc(line, func(r rune) bool {
			return r == '|'
		})
		for i, c := range columns {
			columns[i] = strings.TrimSpace(c)
		}
		if rows == nil {
			rows = sqlmock.NewRows(columns)
		} else {
			values := make([]driver.Value, len(columns))
			for i, c := range columns {
				if c != "nil" {
					values[i] = c
				}
			}
			rows.AddRow(values...)
		}
	}
	return rows
}
