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
	rows := sqlmock.NewRows([]string{"schema", "table", "comment"})
	for _, t := range tables {
		rows.AddRow(schema, t, nil)
	}
	m.ExpectQuery(queryTable).
		WithArgs(schema).
		WillReturnRows(rows)
}
