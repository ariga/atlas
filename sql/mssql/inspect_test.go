// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"database/sql/driver"
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
 t1         | c24         | real           | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 24        | 0     
 t1         | c25         | nchar          | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 2          | 0         | 0     
 t1         | c26         | binary         | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 1          | 0         | 0     
 t1         | c27         | varbinary      | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | NULL                         | 1          | 0         | 0     
 t1         | c28         | varchar        | NULL                   | 1           | 0               | NULL        | NULL          | NULL               | SQL_Latin1_General_CP1_CI_AS | 1          | 0         | 0     
`))
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesQuery, nArgs(1, 1)))).
					WithArgs("dbo", "t1").
					WillReturnRows(sqltest.Rows(`
 table_name | index_name | index_type   | column_name | comment   | filter_expr | primary | is_unique | included | is_desc | seq_in_index
------------+------------+--------------+-------------+-----------+-------------+---------+-----------+----------+---------+--------------
 t1         | PK_t1      | CLUSTERED    | id          | NULL      | NULL        | 1       | 1         | 0        | 0       | 1
 t1         | i1         | NONCLUSTERED | c1          | Index One | NULL        | 0       | 1         | 0        | 0       | 1
 t1         | i1         | NONCLUSTERED | c2          | Index One | NULL        | 0       | 1         | 0        | 0       | 2
 t1         | i2         | NONCLUSTERED | c21         | NULL      | NULL        | 0       | 0         | 1        | 0       | 0
 t1         | i2         | NONCLUSTERED | c22         | NULL      | NULL        | 0       | 0         | 1        | 0       | 0
 t1         | i2         | NONCLUSTERED | c4          | NULL      | NULL        | 0       | 0         | 0        | 0       | 1
 t1         | i2         | NONCLUSTERED | c5          | NULL      | NULL        | 0       | 0         | 0        | 1       | 2
 t1         | i3         | NONCLUSTERED | id          | NULL      | ([c5]=(1))  | 0       | 0         | 0        | 0       | 1
 t1         | i4         | NONCLUSTERED | c25         | NULL      | ([c5]=(1))  | 0       | 0         | 1        | 0       | 0
 t1         | i4         | NONCLUSTERED | c5          | NULL      | ([c5]=(1))  | 0       | 0         | 0        | 0       | 1
 t1         | i4         | NONCLUSTERED | c28         | NULL      | ([c5]=(1))  | 0       | 0         | 0        | 0       | 2
`))
				m.noFKs("dbo", "t1")
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				require.EqualValues(func() *schema.Schema {
					table := &schema.Table{
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
							{Name: "c24", Type: &schema.ColumnType{
								Null: true, Raw: "real",
								Type: &schema.FloatType{T: "real", Precision: 24},
							}},
							{Name: "c25", Type: &schema.ColumnType{
								Null: true, Raw: "nchar",
								Type: &schema.StringType{T: "nchar", Size: 2},
							}, Attrs: []schema.Attr{
								&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
							}},
							{Name: "c26", Type: &schema.ColumnType{
								Null: true, Raw: "binary",
								Type: &schema.BinaryType{T: "binary", Size: sqlx.P(1)},
							}},
							{Name: "c27", Type: &schema.ColumnType{
								Null: true, Raw: "varbinary",
								Type: &schema.BinaryType{T: "varbinary", Size: sqlx.P(1)},
							}},
							{Name: "c28", Type: &schema.ColumnType{
								Null: true, Raw: "varchar",
								Type: &schema.StringType{T: "varchar", Size: 1},
							}, Attrs: []schema.Attr{
								&schema.Collation{V: "SQL_Latin1_General_CP1_CI_AS"},
							}},
						},
					}
					indexes := []*schema.Index{
						{Table: table, Name: "PK_t1", Unique: true, Attrs: []schema.Attr{
							&IndexType{T: "CLUSTERED"},
						}, Parts: []*schema.IndexPart{
							{SeqNo: 1, C: table.Columns[0]},
						}},
						{Table: table, Name: "i1", Unique: true, Attrs: []schema.Attr{
							&IndexType{T: "NONCLUSTERED"},
							&schema.Comment{Text: "Index One"},
						}, Parts: []*schema.IndexPart{
							{SeqNo: 1, C: table.Columns[1]},
							{SeqNo: 2, C: table.Columns[2]},
						}},
						{Table: table, Name: "i2", Attrs: []schema.Attr{
							&IndexType{T: "NONCLUSTERED"},
							&IndexInclude{Columns: []*schema.Column{
								table.Columns[21], table.Columns[22],
							}},
						}, Parts: []*schema.IndexPart{
							{SeqNo: 1, C: table.Columns[4]},
							{SeqNo: 2, C: table.Columns[5], Desc: true},
						}},
						{Table: table, Name: "i3", Attrs: []schema.Attr{
							&IndexType{T: "NONCLUSTERED"},
							&IndexPredicate{P: "([c5]=(1))"},
						}, Parts: []*schema.IndexPart{
							{SeqNo: 1, C: table.Columns[0]},
						}},
						{Table: table, Name: "i4", Attrs: []schema.Attr{
							&IndexType{T: "NONCLUSTERED"},
							&IndexPredicate{P: "([c5]=(1))"},
							&IndexInclude{Columns: []*schema.Column{
								table.Columns[25],
							}},
						}, Parts: []*schema.IndexPart{
							{SeqNo: 1, C: table.Columns[5]},
							{SeqNo: 2, C: table.Columns[28]},
						}},
					}
					table.Columns[0].Indexes = []*schema.Index{indexes[0], indexes[3]}
					table.Columns[1].Indexes = []*schema.Index{indexes[1]}
					table.Columns[2].Indexes = []*schema.Index{indexes[1]}
					table.Columns[4].Indexes = []*schema.Index{indexes[2]}
					table.Columns[5].Indexes = []*schema.Index{indexes[2], indexes[4]}
					table.Columns[28].Indexes = []*schema.Index{indexes[4]}

					table.Indexes = []*schema.Index{indexes[1], indexes[2], indexes[3], indexes[4]}
					table.PrimaryKey = indexes[0]
					realm := &schema.Realm{
						Schemas: []*schema.Schema{
							{
								Name:   "dbo",
								Tables: []*schema.Table{table},
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
		{
			name: "table and fks",
			before: func(m mock) {
				m.version("16.0.4035.4")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(schemasQueryArgs, "= SCHEMA_NAME()"))).
					WillReturnRows(sqltest.Rows(`
 SCHEMA_NAME
-------------
 dbo
				`))
				m.tables("dbo", "t1", "t2")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(columnsQuery, nArgs(1, 2)))).
					WithArgs("dbo", "t1", "t2").
					WillReturnRows(sqltest.Rows(`
 table_name | column_name | type_name      | comment                | is_nullable | is_user_defined | is_identity | identity_seek | identity_increment | collation_name               | max_length | precision | scale 
------------|-------------|----------------+------------------------|-------------|-----------------|-------------|---------------|--------------------|------------------------------|------------|-----------|-------
 t1         | id          | int            | NULL                   | 0           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 0     
 t1         | c1          | bigint         | NULL                   | 0           | 0               | 1           | 701           | 1000               | NULL                         | 8          | 19        | 0     
 t2         | id          | int            | NULL                   | 0           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 0     
 t2         | fk_t1       | int            | NULL                   | 0           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 0     
 t2         | fk2         | int            | NULL                   | 0           | 0               | NULL        | NULL          | NULL               | NULL                         | 4          | 10        | 0     
`))
				m.noIndexes("dbo", "t1", "t2")
				m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, nArgs(1, 2)))).
					WithArgs("dbo", "t1", "t2").
					WillReturnRows(sqltest.Rows(`
 constraint_name | table_name | column_name | table_schema | referenced_table_name | referenced_column_name | referenced_table_schema | update_rule | delete_rule 
-----------------|------------|-------------|--------------|-----------------------|------------------------|-------------------------|-------------|-------------
 fk_t2_t1        | t2         | fk_t1       | dbo          | t1                    | id                     | dbo                     | NO_ACTION   | NO_ACTION   
 gtm_fk2_t1      | t2         | fk2         | dbo          | t1                    | id                     | gtm                     | NO_ACTION   | NO_ACTION   
`))
			},
			expect: func(require *require.Assertions, s *schema.Schema, err error) {
				require.NoError(err)
				require.EqualValues(func() *schema.Schema {
					dboTables := []*schema.Table{{
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
						},
					}, {
						Name:  "t2",
						Attrs: nil,
						Columns: []*schema.Column{
							{Name: "id", Type: &schema.ColumnType{
								Raw:  "int",
								Type: &schema.IntegerType{T: "int"},
							}},
							{Name: "fk_t1", Type: &schema.ColumnType{
								Raw:  "int",
								Type: &schema.IntegerType{T: "int"},
							}},
							{Name: "fk2", Type: &schema.ColumnType{
								Raw:  "int",
								Type: &schema.IntegerType{T: "int"},
							}},
						},
					}}
					dboTable1FK1 := &schema.ForeignKey{
						Symbol: "fk_t2_t1",
						Table:  dboTables[1],
						Columns: []*schema.Column{
							dboTables[1].Columns[1],
						},
						RefTable: dboTables[0],
						RefColumns: []*schema.Column{
							dboTables[0].Columns[0],
						},
						OnUpdate: "NO_ACTION",
						OnDelete: "NO_ACTION",
					}
					dboTable1FK2 := &schema.ForeignKey{
						Symbol: "gtm_fk2_t1",
						Table:  dboTables[1],
						Columns: []*schema.Column{
							dboTables[1].Columns[2],
						},
						RefTable: &schema.Table{
							Name:   "t1",
							Schema: &schema.Schema{Name: "gtm"},
						},
						RefColumns: []*schema.Column{{
							Name: "id",
						}},
						OnUpdate: "NO_ACTION",
						OnDelete: "NO_ACTION",
					}
					dboTables[1].Columns[1].ForeignKeys = []*schema.ForeignKey{dboTable1FK1}
					dboTables[1].Columns[2].ForeignKeys = []*schema.ForeignKey{dboTable1FK2}
					dboTables[1].ForeignKeys = []*schema.ForeignKey{dboTable1FK1, dboTable1FK2}
					dbo := &schema.Schema{Name: "dbo", Tables: nil}
					dbo.Realm = &schema.Realm{
						Schemas: []*schema.Schema{dbo},
						Attrs: []schema.Attr{
							&schema.Collation{
								V: "SQL_Latin1_General_CP1_CI_AS",
							},
						},
					}
					dbo.AddTables(dboTables...)
					return dbo
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

func (m mock) noIndexes(schema string, tables ...string) {
	args := []driver.Value{schema}
	for _, t := range tables {
		args = append(args, t)
	}
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(indexesQuery, nArgs(1, len(tables))))).
		WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{
			"table_name", "index_name", "index_type", "column_name", "comment", "filter_expr", "primary", "is_unique", "included", "is_desc", "seq_in_index",
		}))
}

func (m mock) noFKs(schema string, tables ...string) {
	args := []driver.Value{schema}
	for _, t := range tables {
		args = append(args, t)
	}
	m.ExpectQuery(sqltest.Escape(fmt.Sprintf(fksQuery, nArgs(1, len(tables))))).
		WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{
			"constraint_name", "table_name", "column_name", "table_schema", "referenced_table_name", "referenced_column_name", "referenced_table_schema", "update_rule", "delete_rule",
		}))
}
