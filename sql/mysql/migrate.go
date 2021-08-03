// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A Migrate provides migration capabilities for schema elements.
type migrate struct{ *Driver }

// Migrate returns a MySQL schema executor.
func (d *Driver) Migrate() schema.Execer {
	return &sqlx.Migrate{
		MigrateDriver: &migrate{Driver: d},
	}
}

// QuoteChar returns the character that is used for
// quoting a MySQL identifier (e.g. x => `x`).
func (m *migrate) QuoteChar() byte {
	return '`'
}

// WriteTableAttr writes the given table attribute to the SQL statement
// builder when a table is created or altered.
func (m *migrate) WriteTableAttr(b *sqlx.Builder, a schema.Attr) {
	switch a := a.(type) {
	case *AutoIncrement:
		b.P("AUTO_INCREMENT")
		if a.V != 0 {
			b.P(strconv.FormatInt(a.V, 10))
		}
	case *schema.Charset:
		b.P("CHARACTER SET", a.V)
	default:
		m.attr(b, a)
	}
}

// WriteColumnAttr writes the given column attribute to the SQL statement
// builder when a column is created or modified using the ALTER statement.
func (m *migrate) WriteColumnAttr(b *sqlx.Builder, a schema.Attr) {
	switch a := a.(type) {
	case *OnUpdate:
		b.P("ON UPDATE", a.A)
	case *AutoIncrement:
		b.P("AUTO_INCREMENT")
		if a.V != 0 {
			b.P(strconv.FormatInt(a.V, 10))
		}
	default:
		m.attr(b, a)
	}
}

// WriteIndexAttr writes the given index (or primary-key) attribute to the SQL
// statement builder when an index is created or modified using the ALTER statement.
func (m *migrate) WriteIndexAttr(b *sqlx.Builder, a schema.Attr) {
	m.attr(b, a)
}

// WriteIndexPartAttr writes the given index-part attribute of a primary-key or
// a simple index to the SQL statement builder when an index is created or modified
// using the ALTER statement.
func (m *migrate) WriteIndexPartAttr(b *sqlx.Builder, a schema.Attr) {
	if c, ok := a.(*schema.Collation); ok && c.V == "D" {
		b.P("DESC")
	}
}

func (m *migrate) attr(b *sqlx.Builder, a schema.Attr) {
	switch a := a.(type) {
	case *schema.Collation:
		b.P("COLLATE", a.V)
	case *schema.Comment:
		b.P("COMMENT", "'"+strings.ReplaceAll(a.Text, "'", "\\'")+"'")
	}
}
