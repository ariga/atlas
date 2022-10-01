package spanner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"math/big"
	"testing"
	"time"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	mk.databaseOpts(databaseDialectGoogleStandardSQL)
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
+------------+-------------+------------------+----------------+-----------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
| table_name | column_name | ordinal_position | column_default | data_type | is_nullable | spanner_type | is_generated | generation_expression                       | is_stored | spanner_state |
+------------+-------------+------------------+----------------+-----------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
| Users      | Id          | 1                | NULL           | NULL      | NO          | STRING(20)   | NEVER        | NULL                                        | NULL      | COMMITTED     |
| Users      | FirstName   | 2                | NULL           | NULL      | YES         | STRING(50)   | NEVER        | NULL                                        | NULL      | COMMITTED     |
| Users      | LastName    | 3                | NULL           | NULL      | YES         | STRING(50)   | NEVER        | NULL                                        | NULL      | COMMITTED     |
| Users      | Age         | 4                | NULL           | NULL      | NO          | INT64        | NEVER        | NULL                                        | NULL      | COMMITTED     |
| Users      | FullName    | 5                | NULL           | NULL      | YES         | STRING(100)  | ALWAYS       | ARRAY_TO_STRING([FirstName, LastName], " ") | YES       | COMMITTED     |
+------------+-------------+------------------+----------------+-----------+-------------+--------------+--------------+---------------------------------------------+-----------+---------------+
`))
				m.noIndexes()
				m.noFKs()
				m.noChecks()
			},
			expect: func(require *require.Assertions, t *schema.Table, err error) {
				//p := func(i int) *int { return &i }
				require.NoError(err)
				require.Equal("Users", t.Name)
				require.EqualValues([]*schema.Column{
					// TODO: Include size params
					{Name: "Id", Type: &schema.ColumnType{Type: &StringType{T: "STRING", Size: 20, SizeIsMax: false}}},
					{Name: "FirstName", Type: &schema.ColumnType{Type: &StringType{T: "STRING", Size: 50}, Null: true}},
					{Name: "LastName", Type: &schema.ColumnType{Type: &StringType{T: "STRING", Size: 50}, Null: true}},
					{Name: "Age", Type: &schema.ColumnType{Type: &schema.IntegerType{T: "INT64"}}},
					{Name: "FullName", Type: &schema.ColumnType{Type: &StringType{T: "STRING", Size: 100}, Null: true}, Attrs: []schema.Attr{
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
			mk.databaseOpts(databaseDialectGoogleStandardSQL)
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
	mk.databaseOpts(databaseDialectGoogleStandardSQL)
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
	mk.databaseOpts(databaseDialectGoogleStandardSQL)
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

const databaseDialectGoogleStandardSQL = "GOOGLE_STANDARD_SQL"

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
		return spanner.ToSpannerError(status.Errorf(codes.InvalidArgument, "unsupported value type: %v", t))
	case nil:
	case sql.NullInt64:
	case sql.NullTime:
	case sql.NullString:
	case sql.NullFloat64:
	case sql.NullBool:
	case sql.NullInt32:
	case string:
	case spanner.NullString:
	case []string:
	case []spanner.NullString:
	case *string:
	case []*string:
	case []byte:
	case [][]byte:
	case int:
	case []int:
	case int64:
	case []int64:
	case spanner.NullInt64:
	case []spanner.NullInt64:
	case *int64:
	case []*int64:
	case bool:
	case []bool:
	case spanner.NullBool:
	case []spanner.NullBool:
	case *bool:
	case []*bool:
	case float64:
	case []float64:
	case spanner.NullFloat64:
	case []spanner.NullFloat64:
	case *float64:
	case []*float64:
	case big.Rat:
	case []big.Rat:
	case spanner.NullNumeric:
	case []spanner.NullNumeric:
	case *big.Rat:
	case []*big.Rat:
	case time.Time:
	case []time.Time:
	case spanner.NullTime:
	case []spanner.NullTime:
	case *time.Time:
	case []*time.Time:
	case civil.Date:
	case []civil.Date:
	case spanner.NullDate:
	case []spanner.NullDate:
	case *civil.Date:
	case []*civil.Date:
	case spanner.NullJSON:
	case []spanner.NullJSON:
	case spanner.GenericColumnValue:
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
