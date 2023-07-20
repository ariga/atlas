// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqltest

import (
	"database/sql/driver"
	"regexp"
	"strings"
	"unicode"

	"github.com/DATA-DOG/go-sqlmock"
)

// Rows converts MySQL/PostgreSQL table output to sql.Rows.
// All row values are parsed as text except the "nil" and NULL keywords.
// For example:
//
//	+-------------+-------------+-------------+----------------+
//	| column_name | column_type | is_nullable | column_default |
//	+-------------+-------------+-------------+----------------+
//	| c1          | float       | YES         | nil            |
//	| c2          | int         | YES         |                |
//	| c3          | double      | YES         | NULL           |
//	+-------------+-------------+-------------+----------------+
func Rows(table string) *sqlmock.Rows {
	var (
		nc    int
		rows  *sqlmock.Rows
		lines = strings.Split(table, "\n")
	)
	for i := 0; i < len(lines); i++ {
		line := strings.TrimFunc(lines[i], unicode.IsSpace)
		// Skip new lines, header and footer.
		if line == "" || strings.IndexAny(line, "+-") == 0 {
			continue
		}
		columns := strings.FieldsFunc(line, func(r rune) bool {
			return r == '|'
		})
		for i, c := range columns {
			columns[i] = strings.TrimSpace(c)
		}
		if rows == nil {
			nc = len(columns)
			rows = sqlmock.NewRows(columns)
		} else {
			values := make([]driver.Value, nc)
			for i, c := range columns {
				switch c {
				case "", "nil", "NULL":
				default:
					values[i] = c
				}
			}
			rows.AddRow(values...)
		}
	}
	return rows
}

// Escape escapes all regular expression metacharacters in the given query.
func Escape(query string) string {
	rows := strings.Split(query, "\n")
	for i := range rows {
		rows[i] = strings.TrimPrefix(rows[i], " ")
	}
	query = strings.Join(rows, " ")
	return strings.TrimSpace(regexp.QuoteMeta(query)) + "$"
}
