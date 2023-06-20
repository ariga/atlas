// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// Single table queries used by the different tests.
var (
	queryTable = sqltest.Escape(fmt.Sprintf(tablesQuery, "@1"))
)

func TestDriver_InspectSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		opts   *schema.InspectOptions
		before func(mock)
		expect func(*require.Assertions, *schema.Schema, error)
	}{
		{
			name:   "attached schema",
			schema: "",
			before: func(m mock) {
				m.version("16.0.4035.4")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= SCHEMA_NAME()"))).
					WillReturnRows(sqltest.Rows(`
 SCHEMA_NAME
-------------
 dbo
				`))
				m.tables("dbo")
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				require.EqualValues(func() *schema.Schema {
					realm := &schema.Realm{
						Schemas: []*schema.Schema{
							{
								Name:  "dbo",
								Attrs: nil,
							},
						},
						Attrs: []schema.Attr{
							&schema.Collation{
								V: "SQL_Latin1_General_CP1_CI_AS",
							},
						},
					}
					realm.Schemas[0].Realm = realm
					return realm.Schemas[0]
				}(), s)
			},
		},
		{
			name:   "attached schema with tables",
			schema: "",
			before: func(m mock) {
				m.version("16.0.4035.4")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= SCHEMA_NAME()"))).
					WillReturnRows(sqltest.Rows(`
 schema_name 
-------------
 dbo         
`))
				m.tables("dbo", "t1")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, nArgs(1, 1)))).
					WithArgs("dbo", "t1").
					WillReturnRows(sqltest.Rows(`
 table_name | column_name | type_name      | comment                | is_nullable | is_user_defined | is_identity | identity_seek | identity_increment | collation_name               | max_length | precision | scale 
------------|-------------|----------------+------------------------|-------------|-----------------|-------------|---------------|--------------------|------------------------------|------------|-----------|-------
 t1         | id          | int            | NULL                   | 0           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 0     
 t1         | c1          | bigint         | NULL                   | 0           | 0               | 1           | 701           | 1000               | NULL                         | 8          | 19        | 0     
 t1         | c2          | smallint       | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 2          | 5         | 0     
 t1         | c3          | tinyint        | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 1          | 3         | 0     
 t1         | c4          | binary         | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 50         | 0         | 0     
 t1         | c5          | bit            | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 1          | 1         | 0     
 t1         | c6          | date           | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 3          | 10        | 0     
 t1         | c7          | datetime       | This is datetime       | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 8          | 23        | 3     
 t1         | c8          | datetime2      | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 7          | 24        | 4     
 t1         | c9          | datetime2      | Datetime default scale | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 8          | 27        | 7     
 t1         | c10         | datetimeoffset | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 10         | 34        | 7     
 t1         | c11         | decimal        | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 9          | 12        | 9     
 t1         | c12         | timestamp      | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 8          | 0         | 0     
 t1         | c13         | real           | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 24        | 0     
 t1         | c14         | real           | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 24        | 0     
 t1         | c15         | money          | Tien tien tien         | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 8          | 19        | 4     
 t1         | c16         | smallmoney     | small tien             | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 4     
 t1         | c17         | nchar          | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 1402       | 0         | 0     
 t1         | c18         | nvarchar       | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 100        | 0         | 0     
 t1         | c19         | nvarchar       | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | -1         | 0         | 0     
 t1         | c20         | varbinary      | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 50         | 0         | 0     
 t1         | c21         | varbinary      | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | -1         | 0         | 0     
 t1         | c22         | varchar        | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 50         | 0         | 0     
 t1         | c23         | char           | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 50         | 0         | 0     
 `))
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				require.EqualValues(func() *schema.Schema {
					realm := &schema.Realm{
						Schemas: []*schema.Schema{
							{
								Name: "dbo",
								Tables: []*schema.Table{
									{
										Name:  "t1",
										Attrs: nil,
										Columns: []*schema.Column{
											{Name: "id", Type: &schema.ColumnType{
												Raw:  "int",
												Type: &schema.IntegerType{T: "int"},
											}},
											{Name: "c1", Type: &schema.ColumnType{
												Raw:  "bigint",
												Type: &schema.IntegerType{T: "bigint"},
											}, Attrs: []schema.Attr{
												&Identity{Seek: 701, Increment: 1000},
											}},
											{Name: "c2", Type: &schema.ColumnType{
												Null: true, Raw: "smallint",
												Type: &schema.IntegerType{T: "smallint"},
											}},
											{Name: "c3", Type: &schema.ColumnType{
												Null: true, Raw: "tinyint",
												Type: &schema.IntegerType{T: "tinyint"},
											}},
											{Name: "c4", Type: &schema.ColumnType{
												Null: true, Raw: "binary",
												Type: &schema.BinaryType{T: "binary", Size: sqlx.P[int](50)},
											}},
											{Name: "c5", Type: &schema.ColumnType{
												Null: true, Raw: "bit",
												Type: &BitType{T: "bit"},
											}},
											{Name: "c6", Type: &schema.ColumnType{
												Null: true, Raw: "date",
												Type: &schema.TimeType{T: "date"},
											}},
											{Name: "c7", Type: &schema.ColumnType{
												Null: true, Raw: "datetime",
												Type: &schema.TimeType{T: "datetime"},
											}, Attrs: []schema.Attr{
												&schema.Comment{Text: "This is datetime"},
											}},
											{Name: "c8", Type: &schema.ColumnType{
												Null: true, Raw: "datetime2",
												Type: &schema.TimeType{T: "datetime2", Precision: sqlx.P[int](24), Scale: sqlx.P[int](4)},
											}},
											{Name: "c9", Type: &schema.ColumnType{
												Null: true, Raw: "datetime2",
												Type: &schema.TimeType{T: "datetime2", Precision: sqlx.P[int](27), Scale: sqlx.P[int](7)},
											}, Attrs: []schema.Attr{
												&schema.Comment{Text: "Datetime default scale"},
											}},
											{Name: "c10", Type: &schema.ColumnType{
												Null: true, Raw: "datetimeoffset",
												Type: &schema.TimeType{T: "datetimeoffset", Precision: sqlx.P[int](34), Scale: sqlx.P[int](7)},
											}},
											{Name: "c11", Type: &schema.ColumnType{
												Null: true, Raw: "decimal",
												Type: &schema.DecimalType{T: "decimal", Precision: 12, Scale: 9},
											}},
											{Name: "c12", Type: &schema.ColumnType{
												Null: true, Raw: "timestamp",
												Type: &RowVersionType{T: "rowversion"},
											}},
											{Name: "c13", Type: &schema.ColumnType{
												Null: true, Raw: "real",
												Type: &schema.FloatType{T: "real", Precision: 24},
											}},
											{Name: "c14", Type: &schema.ColumnType{
												Null: true, Raw: "real",
												Type: &schema.FloatType{T: "real", Precision: 24},
											}},
											{Name: "c15", Type: &schema.ColumnType{
												Null: true, Raw: "money",
												Type: &MoneyType{T: "money"},
											}, Attrs: []schema.Attr{
												&schema.Comment{Text: "Tien tien tien"},
											}},
											{Name: "c16", Type: &schema.ColumnType{
												Null: true, Raw: "smallmoney",
												Type: &MoneyType{T: "smallmoney"},
											}, Attrs: []schema.Attr{
												&schema.Comment{Text: "small tien"},
											}},
											{Name: "c17", Type: &schema.ColumnType{
												Null: true, Raw: "nchar",
												Type: &schema.StringType{T: "nchar", Size: 1402},
											}, Attrs: []schema.Attr{
												&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
											}},
											{Name: "c18", Type: &schema.ColumnType{
												Null: true, Raw: "nvarchar",
												Type: &schema.StringType{T: "nvarchar", Size: 100},
											}, Attrs: []schema.Attr{
												&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
											}},
											{Name: "c19", Type: &schema.ColumnType{
												Null: true, Raw: "nvarchar",
												Type: &schema.StringType{T: "nvarchar", Size: -1},
											}, Attrs: []schema.Attr{
												&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
											}},
											{Name: "c20", Type: &schema.ColumnType{
												Null: true, Raw: "varbinary",
												Type: &schema.BinaryType{T: "varbinary", Size: sqlx.P[int](50)},
											}},
											{Name: "c21", Type: &schema.ColumnType{
												Null: true, Raw: "varbinary",
												Type: &schema.BinaryType{T: "varbinary", Size: sqlx.P[int](-1)},
											}},
											{Name: "c22", Type: &schema.ColumnType{
												Null: true, Raw: "varchar",
												Type: &schema.StringType{T: "varchar", Size: 50},
											}, Attrs: []schema.Attr{
												&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
											}},
											{Name: "c23", Type: &schema.ColumnType{
												Null: true, Raw: "char",
												Type: &schema.StringType{T: "char", Size: 50},
											}, Attrs: []schema.Attr{
												&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
											}},
										},
									},
								},
							},
						},
						Attrs: []schema.Attr{
							&schema.Collation{
								V: "SQL_Latin1_General_CP1_CI_AS",
							},
						},
					}
					realm.Schemas[0].Tables[0].Schema = realm.Schemas[0]
					realm.Schemas[0].Realm = realm
					return realm.Schemas[0]
				}(), s)
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
			tables, err := drv.InspectSchema(context.Background(), tt.schema, tt.opts)
			tt.expect(require.New(t), tables, err)
		})
	}
}

type mock struct {
	sqlmock.Sqlmock
}

func (m mock) version(version string) {
	rows := sqlmock.NewRows([]string{
		"ProductVersion", "Collation", "SqlCharSetName",
	})
	rows.AddRow(version, "SQL_Latin1_General_CP1_CI_AS", "iso_1")
	m.ExpectQuery(sqltest.Escape(propertiesQuery)).
		WillReturnRows(rows)
}

func (m mock) tables(schema string, tables ...string) {
	rows := sqlmock.NewRows([]string{"schema", "table", "comment", "is_memory_optimized"})
	for _, t := range tables {
		rows.AddRow(schema, t, nil, 0)
	}
	m.ExpectQuery(queryTable).
		WithArgs(schema).
		WillReturnRows(rows)
}
