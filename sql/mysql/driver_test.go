package mysql_test

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDriver_Table(t *testing.T) {
	tests := []struct {
		name   string
		opts   *schema.InspectOptions
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "table does not exist",
			before: func(m mock) {
				m.version("5.7.23")
				m.tableExists("users", false)
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.Nil(t)
				require.Error(err)
				require.True(schema.IsNotExistError(err), "expect not exists error")
			},
		},
		{
			name: "table does not exist in schema",
			opts: &schema.InspectOptions{
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
			name: "int types",
			before: func(m mock) {
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+--------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| column_name        | column_type          | is_nullable | column_key | column_default | extra          | character_set_name | collation_name |
+--------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
| id                 | bigint(20)           | NO          | PRI        | NULL           | auto_increment | NULL               | NULL           |
| v57_tiny           | tinyint(1)           | NO          |            | NULL           |                | NULL               | NULL           |
| v57_tiny_unsigned  | tinyint(4) unsigned  | NO          |            | NULL           |                | NULL               | NULL           |
| v57_small          | smallint(6)          | NO          |            | NULL           |                | NULL               | NULL           |
| v57_small_unsigned | smallint(6) unsigned | NO          |            | NULL           |                | NULL               | NULL           |
| v57_int            | bigint(11)           | NO          |            | NULL           |                | NULL               | NULL           |
| v57_int_unsigned   | bigint(11) unsigned  | NO          |            | NULL           |                | NULL               | NULL           |
| v8_tiny            | tinyint              | NO          |            | NULL           |                | NULL               | NULL           |
| v8_tiny_unsigned   | tinyint unsigned     | NO          |            | NULL           |                | NULL               | NULL           |
| v8_small           | smallint             | NO          |            | NULL           |                | NULL               | NULL           |
| v8_small_unsigned  | smallint unsigned    | NO          |            | NULL           |                | NULL               | NULL           |
| v8_big             | bigint               | NO          |            | NULL           |                | NULL               | NULL           |
| v8_big_unsigned    | bigint unsigned      | NO          |            | NULL           |                | NULL               | NULL           |
+--------------------+----------------------+-------------+------------+----------------+----------------+--------------------+----------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.Len(t.PrimaryKey, 1)
				require.True(t.PrimaryKey[0] == t.Columns[0])
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint(20)", Type: &schema.IntegerType{T: "bigint", Size: 20}}, Attrs: []schema.Attr{&mysql.AutoIncrement{A: "auto_increment"}}},
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
					{Name: "v8_big_unsigned", Type: &schema.ColumnType{Raw: "bigint unsigned", Type: &schema.IntegerType{T: "bigint", Unsigned: true}}},
				}, t.Columns)
			},
		},
		{
			name: "decimal types",
			before: func(m mock) {
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| d1          | decimal(10,2) | NO          |            | 10.20          |       | NULL               | NULL           |
| d2          | decimal(10,0) | NO          |            | 10             |       | NULL               | NULL           |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "d1", Type: &schema.ColumnType{Raw: "decimal(10,2)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}, Default: &schema.RawExpr{X: "10.20"}}},
					{Name: "d2", Type: &schema.ColumnType{Raw: "decimal(10,0)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 0}, Default: &schema.RawExpr{X: "10"}}},
				}, t.Columns)
			},
		},
		{
			name: "float types",
			before: func(m mock) {
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(sqlmock.NewRows([]string{"column_name", "column_type", "is_nullable", "column_key", "column_default", "extra", "character_set_name", "collation_name"}).
						AddRow("float", "float", "NO", "NULL", "", "", "", "").
						AddRow("double", "double", "NO", "NULL", "", "", "", "")).
					WillReturnRows(rows(`
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type  | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+--------------+-------------+------------+----------------+-------+--------------------+----------------+
| float       | float        | NO          |            |                |       | NULL               | NULL           |
| double      | double       | NO          |            |                |       | NULL               | NULL           |
+-------------+--------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
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
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | binary(20)    | NO          |            | NULL           |       | NULL               | NULL           |
| c2          | varbinary(30) | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | tinyblob      | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | mediumblob    | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | blob          | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | longblob      | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
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
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type   | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | char(20)      | NO          |            | char           |       | NULL               | NULL           |
| c2          | varchar(30)   | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | tinytext      | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | mediumtext    | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | text          | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | longtext      | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "char(20)", Type: &schema.StringType{T: "char", Size: 20}, Default: &schema.RawExpr{X: "char"}}},
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
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+---------------+-------------+------------+----------------+-------+--------------------+-------------------+
| column_name | column_type   | is_nullable | column_key | column_default | extra | character_set_name | collation_name    |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+-------------------+
| c1          | enum('a','b') | NO          |            | NULL           |       | latin1             | latin1_swedish_ci |
| c2          | enum('c','d') | NO          |            | d              |       | latin1             | latin1_swedish_ci |
+-------------+---------------+-------------+------------+----------------+-------+--------------------+-------------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "enum('a','b')", Type: &schema.EnumType{Values: []string{"a", "b"}}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "enum('c','d')", Type: &schema.EnumType{Values: []string{"c", "d"}}, Default: &schema.RawExpr{X: "d"}}},
				}, t.Columns)
			},
		},
		{
			name: "time type",
			before: func(m mock) {
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+-------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
| column_name | column_type | is_nullable | column_key | column_default    | extra                       | character_set_name | collation_name |
+-------------+-------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
| c1          | date        | NO          |            | NULL              |                             | NULL               | NULL           |
| c2          | datetime    | NO          |            | NULL              |                             | NULL               | NULL           |
| c3          | time        | NO          |            | NULL              |                             | NULL               | NULL           |
| c4          | timestamp   | NO          |            | CURRENT_TIMESTAMP | on update CURRENT_TIMESTAMP | NULL               | NULL           |
| c5          | year(4)     | NO          |            | NULL              |                             | NULL               | NULL           |
| c6          | year        | NO          |            | NULL              |                             | NULL               | NULL           |
+-------------+-------------+-------------+------------+-------------------+-----------------------------+--------------------+----------------+
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "c1", Type: &schema.ColumnType{Raw: "date", Type: &schema.TimeType{T: "date"}}},
					{Name: "c2", Type: &schema.ColumnType{Raw: "datetime", Type: &schema.TimeType{T: "datetime"}}},
					{Name: "c3", Type: &schema.ColumnType{Raw: "time", Type: &schema.TimeType{T: "time"}}},
					{Name: "c4", Type: &schema.ColumnType{Raw: "timestamp", Type: &schema.TimeType{T: "timestamp"}, Default: &schema.RawExpr{X: "CURRENT_TIMESTAMP"}}, Attrs: []schema.Attr{&mysql.OnUpdate{A: "on update current_timestamp"}}},
					{Name: "c5", Type: &schema.ColumnType{Raw: "year(4)", Type: &schema.TimeType{T: "year"}}},
					{Name: "c6", Type: &schema.ColumnType{Raw: "year", Type: &schema.TimeType{T: "year"}}},
				}, t.Columns)
			},
		},
		{
			name: "json type",
			before: func(m mock) {
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+-------------+-------------+------------+----------------+-------+--------------------+----------------+
| COLUMN_NAME | COLUMN_TYPE | IS_NULLABLE | COLUMN_KEY | COLUMN_DEFAULT | EXTRA | CHARACTER_SET_NAME | COLLATION_NAME |
+-------------+-------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | json        | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+-------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
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
				m.version("8.0.0")
				m.tableExists("users", true)
				m.ExpectQuery(escape("SELECT `column_name`, `column_type`, `is_nullable`, `column_key`, `column_default`, `extra`, `character_set_name`, `collation_name` FROM `INFORMATION_SCHEMA`.`COLUMNS` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
					WithArgs("users").
					WillReturnRows(rows(`
+-------------+--------------------+-------------+------------+----------------+-------+--------------------+----------------+
| column_name | column_type        | is_nullable | column_key | column_default | extra | character_set_name | collation_name |
+-------------+--------------------+-------------+------------+----------------+-------+--------------------+----------------+
| c1          | point              | NO          |            | NULL           |       | NULL               | NULL           |
| c2          | multipoint         | NO          |            | NULL           |       | NULL               | NULL           |
| c3          | linestring         | NO          |            | NULL           |       | NULL               | NULL           |
| c4          | multilinestring    | NO          |            | NULL           |       | NULL               | NULL           |
| c5          | polygon            | NO          |            | NULL           |       | NULL               | NULL           |
| c6          | multipolygon       | NO          |            | NULL           |       | NULL               | NULL           |
| c7          | geometry           | NO          |            | NULL           |       | NULL               | NULL           |
| c8          | geometrycollection | NO          |            | NULL           |       | NULL               | NULL           |
| c9          | geomcollection     | NO          |            | NULL           |       | NULL               | NULL           |
+-------------+--------------------+-------------+------------+----------------+-------+--------------------+----------------+
`))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			tt.before(mock{m})
			drv, err := mysql.Open(db)
			require.NoError(t, err)
			table, err := drv.Table(context.Background(), "users", tt.opts)
			tt.expect(require.New(t), table, err)
		})
	}
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	m.ExpectQuery(escape("SHOW VARIABLES LIKE 'version'")).
		WillReturnRows(sqlmock.NewRows([]string{"Variable_name", "Value"}).AddRow("version", version))
}

func (m mock) tableExists(table string, exists bool) {
	count := 0
	if exists {
		count = 1
	}
	m.ExpectQuery(escape("SELECT COUNT(*) FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = (SELECT DATABASE()) AND `TABLE_NAME` = ?")).
		WithArgs(table).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

func (m mock) tableExistsInSchema(schema, table string, exists bool) {
	count := 0
	if exists {
		count = 1
	}
	m.ExpectQuery(escape("SELECT COUNT(*) FROM `INFORMATION_SCHEMA`.`TABLES` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?")).
		WithArgs(schema, table).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
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
