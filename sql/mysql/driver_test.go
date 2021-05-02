package mysql_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

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
					WillReturnRows(sqlmock.NewRows([]string{"column_name", "column_type", "is_nullable", "column_key", "column_default", "extra", "character_set_name", "collation_name"}).
						AddRow("id", "bigint(20)", "NO", "PRI", nil, "auto_increment", "", "").
						AddRow("v57_tiny", "tinyint(1)", "NO", "YES", "NULL", "", "", "").
						AddRow("v57_tiny_unsigned", "tinyint(4) unsigned", "NO", "YES", "NULL", "", "", "").
						AddRow("v57_small", "smallint(6)", "NO", "YES", "NULL", "", "", "").
						AddRow("v57_small_unsigned", "smallint(6) unsigned", "NO", "YES", "NULL", "", "", "").
						AddRow("v57_int", "bigint(11)", "NO", "YES", "NULL", "", "", "").
						AddRow("v57_int_unsigned", "bigint(11) unsigned", "NO", "YES", "NULL", "", "", "").
						// Numeric types format for version >= 8.0.19.
						AddRow("v8_tiny", "tinyint", "NO", "YES", "NULL", "", "", "").
						AddRow("v8_tiny_unsigned", "tinyint unsigned", "NO", "YES", "NULL", "", "", "").
						AddRow("v8_small", "smallint", "NO", "YES", "NULL", "", "", "").
						AddRow("v8_small_unsigned", "smallint unsigned", "NO", "YES", "NULL", "", "", "").
						AddRow("v8_big", "bigint", "NO", "YES", "NULL", "", "", "").
						AddRow("v8_big_unsigned", "bigint unsigned", "NO", "YES", "NULL", "", "", ""))
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
					WillReturnRows(sqlmock.NewRows([]string{"column_name", "column_type", "is_nullable", "column_key", "column_default", "extra", "character_set_name", "collation_name"}).
						AddRow("decimal", "decimal(10,2)", "NO", "NULL", "10.20", "", "", "").
						AddRow("numeric", "decimal(10,0)", "NO", "NULL", "10", "", "", ""))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "decimal", Type: &schema.ColumnType{Raw: "decimal(10,2)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}, Default: &schema.RawExpr{X: "10.20"}}},
					{Name: "numeric", Type: &schema.ColumnType{Raw: "decimal(10,0)", Type: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 0}, Default: &schema.RawExpr{X: "10"}}},
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
						AddRow("double", "double", "NO", "NULL", "", "", "", ""))
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
