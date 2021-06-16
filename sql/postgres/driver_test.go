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
			name: "int types",
			before: func(m mock) {
				m.version("130000")
				m.tableExists("public", "users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("public", "users").
					WillReturnRows(sqltest.Rows(`
 column_name | data_type | is_nullable | column_default | coalesce | coalesce | coalesce | character_set_name | collation_name | udt_name | is_identity | comment 
-------------+-----------+-------------+----------------+----------+----------+----------+--------------------+----------------+----------+-------------+---------
 id          | bigint    | NO          |                |        0 |       64 |        0 |                    |                | int8     | YES         | 
 rank        | integer   | YES         |                |        0 |       32 |        0 |                    |                | int4     | NO          | rank
`))
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "id", Type: &schema.ColumnType{Raw: "bigint"}},
					{Name: "rank", Type: &schema.ColumnType{Raw: "integer", Null: true}, Attrs: []schema.Attr{&schema.Comment{Text: "rank"}}},
				}, t.Columns)
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
