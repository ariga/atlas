// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
)

type (
	// Driver represents a MySQL driver for introspecting database schemas,
	// generating diff between schema elements and apply migrations changes.
	Driver struct {
		conn
		schema.Differ
		schema.Inspector
		migrate.PlanApplier
	}

	// database connection and its information.
	conn struct {
		schema.ExecQuerier
		// System variables that are set on `Open`.
		version string
		collate string
		charset string
	}
)

// Open opens a new MySQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	c := conn{ExecQuerier: db}
	if err := db.QueryRow(variablesQuery).Scan(&c.version, &c.collate, &c.charset); err != nil {
		return nil, fmt.Errorf("mysql: scanning system variables: %w", err)
	}
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

// supportsCheck reports if the connected database supports
// the CHECK clause, and return the querying for getting them.
func (d *conn) supportsCheck() (string, bool) {
	v, q := "8.0.16", myChecksQuery
	if d.mariadb() {
		v, q = "10.2.1", marChecksQuery
	}
	return q, d.gteV(v)
}

// supportsIndexExpr reports if the connected database supports
// index expressions (functional key part).
func (d *conn) supportsIndexExpr() bool {
	return !d.mariadb() && d.gteV("8.0.13")
}

// supportsDisplayWidth reports if the connected database supports
// getting the display width information from the information schema.
func (d *conn) supportsDisplayWidth() bool {
	// MySQL v8.0.19 dropped the display width
	// information from the information schema
	return d.mariadb() || d.ltV("8.0.19")
}

// supportsExprDefault reports if the connected database supports
// expressions in the DEFAULT clause on column definition.
func (d *conn) supportsExprDefault() bool {
	v := "8.0.13"
	if d.mariadb() {
		v = "10.2.1"
	}
	return d.gteV(v)
}

// supportsEnforceCheck reports if the connected database supports
// the ENFORCED option in CHECK constraint syntax.
func (d *conn) supportsEnforceCheck() bool {
	return !d.mariadb() && d.gteV("8.0.16")
}

// mariadb reports if the Driver is connected to a MariaDB database.
func (d *conn) mariadb() bool {
	return strings.Index(d.version, "MariaDB") > 0
}

// compareV returns an integer comparing two versions according to
// semantic version precedence.
func (d *conn) compareV(w string) int {
	v := d.version
	if d.mariadb() {
		v = v[:strings.Index(v, "MariaDB")-1]
	}
	return semver.Compare("v"+v, "v"+w)
}

// gteV reports if the connection version is >= w.
func (d *conn) gteV(w string) bool { return d.compareV(w) >= 0 }

// ltV reports if the connection version is < w.
func (d *conn) ltV(w string) bool { return d.compareV(w) == -1 }

// MySQL standard unescape field function from its codebase:
// https://github.com/mysql/mysql-server/blob/8.0/sql/dd/impl/utils.cc
func unescape(s string) string {
	var b strings.Builder
	for i, c := range s {
		if c != '\\' || i+1 < len(s) && s[i+1] != '\\' && s[i+1] != '=' && s[i+1] != ';' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// MySQL standard column types as defined in its codebase. Name and order
// is organized differently than MySQL.
//
// https://github.com/mysql/mysql-server/blob/8.0/include/field_types.h
// https://github.com/mysql/mysql-server/blob/8.0/sql/dd/types/column.h
// https://github.com/mysql/mysql-server/blob/8.0/sql/sql_show.cc
// https://github.com/mysql/mysql-server/blob/8.0/sql/gis/geometries.cc
const (
	tBit       = "bit"       // MYSQL_TYPE_BIT
	tInt       = "int"       // MYSQL_TYPE_LONG
	tTinyInt   = "tinyint"   // MYSQL_TYPE_TINY
	tSmallInt  = "smallint"  // MYSQL_TYPE_SHORT
	tMediumInt = "mediumint" // MYSQL_TYPE_INT24
	tBigInt    = "bigint"    // MYSQL_TYPE_LONGLONG

	tDecimal = "decimal" // MYSQL_TYPE_DECIMAL
	tNumeric = "numeric" // MYSQL_TYPE_DECIMAL (numeric_type rule in sql_yacc.yy)
	tFloat   = "float"   // MYSQL_TYPE_FLOAT
	tDouble  = "double"  // MYSQL_TYPE_DOUBLE
	tReal    = "real"    // MYSQL_TYPE_FLOAT or MYSQL_TYPE_DOUBLE (real_type in sql_yacc.yy)

	tTimestamp = "timestamp" // MYSQL_TYPE_TIMESTAMP
	tDate      = "date"      // MYSQL_TYPE_DATE
	tTime      = "time"      // MYSQL_TYPE_TIME
	tDateTime  = "datetime"  // MYSQL_TYPE_DATETIME
	tYear      = "year"      // MYSQL_TYPE_YEAR

	tVarchar    = "varchar"    // MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_VARCHAR
	tChar       = "char"       // MYSQL_TYPE_STRING
	tVarBinary  = "varbinary"  // MYSQL_TYPE_VAR_STRING + NULL CHARACTER_SET.
	tBinary     = "binary"     // MYSQL_TYPE_STRING + NULL CHARACTER_SET.
	tBlob       = "blob"       // MYSQL_TYPE_BLOB
	tTinyBlob   = "tinyblob"   // MYSQL_TYPE_TINYBLOB
	tMediumBlob = "mediumblob" // MYSQL_TYPE_MEDIUM_BLOB
	tLongBlob   = "longblob"   // MYSQL_TYPE_LONG_BLOB
	tText       = "text"       // MYSQL_TYPE_BLOB + CHARACTER_SET utf8mb4
	tTinyText   = "tinytext"   // MYSQL_TYPE_TINYBLOB + CHARACTER_SET utf8mb4
	tMediumText = "mediumtext" // MYSQL_TYPE_MEDIUM_BLOB + CHARACTER_SET utf8mb4
	tLongText   = "longtext"   // MYSQL_TYPE_LONG_BLOB with + CHARACTER_SET utf8mb4

	tEnum = "enum" // MYSQL_TYPE_ENUM
	tSet  = "set"  // MYSQL_TYPE_SET
	tJSON = "json" // MYSQL_TYPE_JSON

	tGeometry           = "geometry"           // MYSQL_TYPE_GEOMETRY
	tPoint              = "point"              // Geometry_type::kPoint
	tMultiPoint         = "multipoint"         // Geometry_type::kMultipoint
	tLineString         = "linestring"         // Geometry_type::kLinestring
	tMultiLineString    = "multilinestring"    // Geometry_type::kMultilinestring
	tPolygon            = "polygon"            // Geometry_type::kPolygon
	tMultiPolygon       = "multipolygon"       // Geometry_type::kMultipolygon
	tGeoCollection      = "geomcollection"     // Geometry_type::kGeometrycollection
	tGeometryCollection = "geometrycollection" // Geometry_type::kGeometrycollection
)

// Additional common constants in MySQL.
const (
	currentTS     = "current_timestamp"
	defaultGen    = "default_generated"
	autoIncrement = "auto_increment"
)
