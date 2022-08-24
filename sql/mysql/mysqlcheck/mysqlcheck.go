// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysqlcheck

import (
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlcheck/datadepend"
	"ariga.io/atlas/sql/sqlcheck/destructive"
)

func addNotNull(p *datadepend.ColumnPass) (diags []sqlcheck.Diagnostic, err error) {
	// Two types of reporting, implicit rows update and
	// changes that may cause the migration to fail.
	mightFail := func(tt string) {
		diags = append(diags, sqlcheck.Diagnostic{
			Pos: p.Change.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q will fail in case table %q is not empty",
				tt, p.Column.Name, p.Table.Name,
			),
		})
	}
	implicitUpdate := func(tt, v string) {
		diags = append(diags, sqlcheck.Diagnostic{
			Pos: p.Change.Pos,
			Text: fmt.Sprintf(
				"Adding a non-nullable %q column %q on table %q without a default value implicitly sets existing rows with %s",
				tt, p.Column.Name, p.Table.Name, v,
			),
		})
	}
	drv, ok := p.Dev.Driver.(*mysql.Driver)
	if !ok {
		return nil, fmt.Errorf("unexpected migrate driver %T", p.Dev.Driver)
	}
	switch ct := p.Column.Type.Type.(type) {
	case *mysql.BitType, *schema.BoolType, *schema.IntegerType, *schema.DecimalType, *schema.FloatType, *schema.BinaryType:
		tt, err := mysql.FormatType(p.Column.Type.Type)
		if err != nil {
			return nil, err
		}
		implicitUpdate(tt, "0")
	case *schema.StringType:
		switch ct.T {
		case mysql.TypeVarchar, mysql.TypeChar:
			implicitUpdate(ct.T, `""`)
		case mysql.TypeText, mysql.TypeTinyText, mysql.TypeMediumText, mysql.TypeLongText:
			// On MySQL, Existing rows are updated with ''. Skip it
			// as we cannot propose and detect multi-steps migration
			// (ALTER + UPDATE) at this stage.
			if drv.Maria() {
				implicitUpdate(ct.T, `""`)
			}
		}
	case *schema.EnumType:
		if len(ct.Values) == 0 {
			return nil, fmt.Errorf("unexpected empty values for enum column %q.%q", p.Table.Name, p.Column.Name)
		}
		implicitUpdate("enum", strconv.Quote(ct.Values[0]))
	case *mysql.SetType:
		implicitUpdate("set", `""`)
	case *schema.JSONType:
		// On MySQL, Existing rows are updated with 'null' JSON. Same as TEXT
		// columns, we cannot propose multi-steps migration (ALTER + UPDATE)
		// as it cannot be detected at this stage.
		if drv.Maria() {
			implicitUpdate(ct.T, `""`)
		}
	case *schema.TimeType:
		switch ct.T {
		case mysql.TypeDate, mysql.TypeDateTime:
			if drv.Maria() {
				implicitUpdate(ct.T, "00:00:00")
			} else {
				// The suggested solution is to add a DEFAULT clause
				// with valid value or set the column to nullable.
				mightFail(ct.T)
			}
		case mysql.TypeYear:
			implicitUpdate(ct.T, "0000")
		case mysql.TypeTime:
			implicitUpdate(ct.T, "00:00:00")
		case mysql.TypeTimestamp:
			v := "CURRENT_TIMESTAMP"
			switch {
			case drv.Maria():
				// Maria has a special behavior for the first TIMESTAMP column.
				// See: https://mariadb.com/kb/en/timestamp/#automatic-values
				for i := 0; i < len(p.Table.Columns) && p.Table.Columns[i].Name != p.Column.Name; i++ {
					tt, err := mysql.FormatType(p.Table.Columns[i].Type.Type)
					if err != nil {
						return nil, err
					}
					if strings.HasPrefix(tt, mysql.TypeTimestamp) {
						v = "0000-00-00 00:00:00"
						break
					}
				}
			// Following MySQL 8.0.2, the explicit_defaults_for_timestamp
			// system variable is now enabled by default.
			case drv.GTE("8.0.2"):
				v = "0000-00-00 00:00:00"
			}
			implicitUpdate(ct.T, v)
		}
	case *schema.SpatialType:
		if drv.Maria() {
			implicitUpdate(ct.T, `""`)
		} else {
			// The suggested solution is to add the column as
			// null, update values and then set it to not-null.
			mightFail(ct.T)
		}
	}
	return
}

func init() {
	sqlcheck.Register(mysql.DriverName, func(r *schemahcl.Resource) (sqlcheck.Analyzer, error) {
		ds, err := destructive.New(r)
		if err != nil {
			return nil, err
		}
		dd, err := datadepend.New(r, datadepend.Handler{
			AddNotNull: addNotNull,
		})
		if err != nil {
			return nil, err
		}
		return sqlcheck.Analyzers{ds, dd}, nil
	})
}
