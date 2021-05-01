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
				require.True(schema.IsNotExistError(err))
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
				require.True(schema.IsNotExistError(err))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			tt.before(mock{m})
			drv, err := mysql.NewDriver(db)
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
