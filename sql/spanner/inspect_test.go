// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"
	"testing"
	"time"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type mockValueConverter struct{}

// ConvertValue implements the sqlmock.ValueConverter interface and satisfies the acceptable Spanner types.
func (mockValueConverter) ConvertValue(v interface{}) (driver.Value, error) {
	return driver.String.ConvertValue(v)
}

func TestDriver_InspectSchema(t *testing.T) {
	db, m, err := sqlmock.New(sqlmock.ValueConverterOption(mockValueConverter{}))
	require.NoError(t, err)
	mk := mock{m}
	mk.databaseOpts(dialectGoogleStandardSQL)
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(schemasQueryArgs)).
		WithArgs([]string{""}).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(""))

	m.ExpectQuery(sqltest.Escape(tablesQuery)).
		WithArgs([]string{""}).
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "parent_table_name", "on_delete_action", "spanner_state"}))
	s, err := drv.InspectSchema(context.Background(), "", &schema.InspectOptions{})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Schema {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "default",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r.Schemas[0]
	}(), s)
}

func TestDriver_InspectTable(t *testing.T) {
	tests := []struct {
		name   string
		before func(mock)
		expect func(*require.Assertions, *schema.Table, error)
	}{
		{
			name: "column types",
			before: func(m mock) {
				m.tableExists("", "Users", true)
				m.ExpectQuery(sqltest.Escape(columnsQuery)).
					WithArgs("", []string{"Users"}).
					WillReturnRows(sqltest.Rows(`
+------------+-------------+------------------+----------------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
| table_name | column_name | ordinal_position | column_default | is_nullable | spanner_type | is_generated | generation_expression                       | is_stored | spanner_state |
+------------+-------------+------------------+----------------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
| Users      | Id          | 1                | NULL           | false       | STRING(20)   | false        | NULL                                        | false     | COMMITTED     |
| Users      | FirstName   | 2                | NULL           | true        | STRING(50)   | false        | NULL                                        | false     | COMMITTED     |
| Users      | LastName    | 3                | NULL           | true        | STRING(50)   | false        | NULL                                        | false     | COMMITTED     |
| Users      | Age         | 4                | NULL           | false       | INT64        | false        | NULL                                        | false     | COMMITTED     |
| Users      | FullName    | 5                | NULL           | true        | STRING(MAX)  | true         | ARRAY_TO_STRING([FirstName, LastName], " ") | true      | COMMITTED     |
+------------+-------------+------------------+----------------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
`))
				m.noIndexes()
				m.noFKs()
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				require.NoError(err)
				require.Equal("Users", t.Name)
				require.EqualValues([]*schema.Column{
					{Name: "Id", Type: &schema.ColumnType{Raw: "STRING(20)", Type: &schema.StringType{T: "STRING", Size: 20}}},
					{Name: "FirstName", Type: &schema.ColumnType{Raw: "STRING(50)", Type: &schema.StringType{T: "STRING", Size: 50}, Null: true}},
					{Name: "LastName", Type: &schema.ColumnType{Raw: "STRING(50)", Type: &schema.StringType{T: "STRING", Size: 50}, Null: true}},
					{Name: "Age", Type: &schema.ColumnType{Raw: "INT64", Type: &schema.IntegerType{T: "INT64"}}},
					{Name: "FullName", Type: &schema.ColumnType{Raw: "STRING(MAX)", Type: &schema.StringType{T: "STRING", Attrs: []schema.Attr{&MaxSize{}}}, Null: true}, Attrs: []schema.Attr{
						&schema.GeneratedExpr{
							Expr: `ARRAY_TO_STRING([FirstName, LastName], " ")`,
							Type: "STORED",
						}},
					},
				}, t.Columns)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, m, err := sqlmock.New(sqlmock.ValueConverterOption(mockValueConverter{}))
			require.NoError(t, err)
			mk := mock{m}
			mk.databaseOpts(dialectGoogleStandardSQL)
			var drv migrate.Driver
			drv, err = Open(db)
			require.NoError(t, err)
			mk.ExpectQuery(sqltest.Escape(schemasQueryArgs)).
				WithArgs([]string{""}).
				WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(""))
			tt.before(mk)
			s, err := drv.InspectSchema(context.Background(), "", nil)
			require.NoError(t, err)
			tt.expect(require.New(t), s.Tables[0], err)
		})
	}
}

func TestDriver_Realm(t *testing.T) {
	db, m, err := sqlmock.New(sqlmock.ValueConverterOption(mockValueConverter{}))
	require.NoError(t, err)
	mk := mock{m}
	mk.databaseOpts(dialectGoogleStandardSQL)
	drv, err := Open(db)
	require.NoError(t, err)
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(""))
	m.ExpectQuery(sqltest.Escape(tablesQuery)).
		WithArgs([]string{""}).
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "parent_table_name", "on_delete_action", "spanner_state"}))
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "default",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r
	}(), realm)

	mk.ExpectQuery(sqltest.Escape(schemasQueryArgs)).
		WithArgs([]string{""}).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(""))
	m.ExpectQuery(sqltest.Escape(tablesQuery)).
		WithArgs([]string{""}).
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "parent_table_name", "on_delete_action", "spanner_state"}))
	realm, err = drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Schemas: []string{""}})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "default",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r
	}(), realm)
}

func TestInspectMode_InspectRealm(t *testing.T) {
	db, m, err := sqlmock.New(sqlmock.ValueConverterOption(mockValueConverter{}))
	require.NoError(t, err)
	mk := mock{m}
	mk.databaseOpts(dialectGoogleStandardSQL)
	mk.ExpectQuery(sqltest.Escape(schemasQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow(""))
	drv, err := Open(db)
	realm, err := drv.InspectRealm(context.Background(), &schema.InspectRealmOption{Mode: schema.InspectSchemas})
	require.NoError(t, err)
	require.EqualValues(t, func() *schema.Realm {
		r := &schema.Realm{
			Schemas: []*schema.Schema{
				{
					Name: "default",
				},
			},
		}
		r.Schemas[0].Realm = r
		return r
	}(), realm)
}

const dialectGoogleStandardSQL = "GOOGLE_STANDARD_SQL"

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) CheckNamedValue(value *driver.NamedValue) (err error) {
	if value == nil {
		return nil
	}
	switch t := value.Value.(type) {
	default:
		// Default is to fail, unless it is one of the following supported types.
		return fmt.Errorf("unsupported value type: %v", t)
	case nil:
	case sql.NullInt64:
	case sql.NullTime:
	case sql.NullString:
	case sql.NullFloat64:
	case sql.NullBool:
	case sql.NullInt32:
	case string:
	case []string:
	case *string:
	case []*string:
	case []byte:
	case [][]byte:
	case int:
	case []int:
	case int64:
	case []int64:
	case *int64:
	case []*int64:
	case bool:
	case []bool:
	case *bool:
	case []*bool:
	case float64:
	case []float64:
	case *float64:
	case []*float64:
	case big.Rat:
	case []big.Rat:
	case *big.Rat:
	case []*big.Rat:
	case time.Time:
	case []time.Time:
	case *time.Time:
	case []*time.Time:
	}
	return nil
}

func (m mock) databaseOpts(dialect string) {
	m.ExpectQuery(sqltest.Escape(paramsQuery)).
		WillReturnRows(sqltest.Rows(`
  option_value
------------
 ` + dialect + `
`))
}

func (m mock) tableExists(schema, table string, exists bool) {
	rows := sqlmock.NewRows([]string{"table_schema", "table_name", "parent_table_name", "on_delete_action", "spanner_state"})
	if exists {
		rows.AddRow(schema, table, nil, nil, nil)
	}
	m.ExpectQuery(sqltest.Escape(tablesQuery)).
		WithArgs([]string{schema}).
		WillReturnRows(rows)
}

func (m mock) noIndexes() {
	m.ExpectQuery(sqltest.Escape(indexesQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "index_name", "column_name", "primary", "unique", "constraint_type", "predicate", "expression", "options"}))
}

func (m mock) noFKs() {
	m.ExpectQuery(sqltest.Escape(fksQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name", "referenced_table_schema", "update_rule", "delete_rule"}))
}

func (m mock) noChecks() {
	m.ExpectQuery(sqltest.Escape(checksQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "constraint_name", "expression", "column_name", "column_indexes"}))
}
