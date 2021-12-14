// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"fmt"

	"ariga.io/atlas/sql/internal/sqlx"

	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
)

type (
	// Driver represents a PostgreSQL driver for introspecting database schemas,
	// generating diff between schema elements and apply migrations changes.
	Driver struct {
		conn
		schema.Differ
		schema.Execer
		schema.Inspector
	}

	// database connection and its information.
	conn struct {
		schema.ExecQuerier
		// System variables that are set on `Open`.
		collate string
		ctype   string
		version string
	}
)

// Open opens a new PostgreSQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	c := conn{ExecQuerier: db}
	rows, err := db.Query(paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning system variables: %w", err)
	}
	defer rows.Close()
	params := make([]string, 0, 3)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("postgres: failed scanning row value: %w", err)
		}
		params = append(params, v)
	}
	if len(params) != 3 {
		return nil, fmt.Errorf("postgres: unexpected number of rows: %d", len(params))
	}
	c.collate, c.ctype, c.version = params[0], params[1], params[2]
	if len(c.version) != 6 {
		return nil, fmt.Errorf("postgres: malformed version: %s", c.version)
	}
	c.version = fmt.Sprintf("%s.%s.%s", c.version[:2], c.version[2:4], c.version[4:])
	if semver.Compare("v"+c.version, "v10.0.0") != -1 {
		return nil, fmt.Errorf("postgres: unsupported postgres version: %s", c.version)
	}
	return &Driver{
		conn:      c,
		Differ:    &sqlx.Diff{DiffDriver: &diff{c}},
		Execer:    &migrate{c},
		Inspector: &inspect{c},
	}, nil
}

// Standard column types (and their aliases) as defined in
// PostgreSQL codebase/website.
const (
	tBit     = "bit"
	tBitVar  = "bit varying"
	tBoolean = "boolean"
	tBool    = "bool" // boolean.
	tBytea   = "bytea"

	tCharacter = "character"
	tChar      = "char" // character
	tCharVar   = "character varying"
	tVarChar   = "varchar" // character varying
	tText      = "text"

	tSmallInt = "smallint"
	tInteger  = "integer"
	tBigInt   = "bigint"
	tInt      = "int"  // integer.
	tInt2     = "int2" // smallint.
	tInt4     = "int4" // integer.
	tInt8     = "int8" // bigint.

	tCIDR     = "cidr"
	tInet     = "inet"
	tMACAddr  = "macaddr"
	tMACAddr8 = "macaddr8"

	tCircle  = "circle"
	tLine    = "line"
	tLseg    = "lseg"
	tBox     = "box"
	tPath    = "path"
	tPolygon = "polygon"
	tPoint   = "point"

	tDate          = "date"
	tTime          = "time" // time without time zone
	tTimeWTZ       = "time with time zone"
	tTimeWOTZ      = "time without time zone"
	tTimestamp     = "timestamp" // timestamp without time zone
	tTimestampTZ   = "timestamptz"
	tTimestampWTZ  = "timestamp with time zone"
	tTimestampWOTZ = "timestamp without time zone"

	tDouble = "double precision"
	tReal   = "real"
	tFloat8 = "float8" // double precision
	tFloat4 = "float4" // real

	tNumeric = "numeric"
	tDecimal = "decimal" // numeric

	tSmallSerial = "smallserial" // smallint with auto_increment.
	tSerial      = "serial"      // integer with auto_increment.
	tBigSerial   = "bigserial"   // bigint with auto_increment.
	tSerial2     = "serial2"     // smallserial
	tSerial4     = "serial4"     // serial
	tSerial8     = "serial8"     // bigserial

	tArray       = "array"
	tXML         = "xml"
	tJSON        = "json"
	tJSONB       = "jsonb"
	tUUID        = "uuid"
	tMoney       = "money"
	tInterval    = "interval"
	tUserDefined = "user-defined"
)
