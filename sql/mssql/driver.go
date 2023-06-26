// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// Driver represents a MSSQL driver for introspecting database schemas,
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
		version string
		charset string
		collate string
	}
)

// DriverName holds the name used for registration.
const DriverName = "sqlserver"

func init() {
	sqlclient.Register(
		DriverName,
		sqlclient.DriverOpener(Open),
	)
}

// Open opens a new Spanner driver.
func Open(db schema.ExecQuerier) (migrate.Driver, error) {
	c := conn{ExecQuerier: db}
	rows, err := db.QueryContext(context.Background(), propertiesQuery)
	if err != nil {
		return nil, fmt.Errorf("mssql: query server property: %w", err)
	}
	if err := sqlx.ScanOne(rows, &c.version, &c.collate, &c.charset); err != nil {
		return nil, fmt.Errorf("mssql: scan server property: %w", err)
	}
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

// MS-SQL standard column types as defined in its codebase.
//
// https://learn.microsoft.com/en-us/sql/t-sql/data-types/data-types-transact-sql?view=sql-server-ver16
const (
	// Exact numerics
	TypeBigInt     = "bigint"
	TypeBit        = "bit"
	TypeDecimal    = "decimal"
	TypeInt        = "int"
	TypeMoney      = "money"
	TypeNumeric    = "numeric"
	TypeSmallInt   = "smallint"
	TypeSmallMoney = "smallmoney"
	TypeTinyInt    = "tinyint"

	// Approximate numerics
	TypeFloat = "float"
	TypeReal  = "real"

	// Date and time
	TypeDate           = "date"
	TypeDateTime       = "datetime"
	TypeDateTime2      = "datetime2"
	TypeDateTimeOffset = "datetimeoffset"
	TypeSmallDateTime  = "smalldatetime"
	TypeTime           = "time"

	// Character strings
	TypeChar    = "char"
	TypeText    = "text"
	TypeVarchar = "varchar"

	// Unicode character strings
	TypeNChar    = "nchar"
	TypeNText    = "ntext"
	TypeNVarchar = "nvarchar"

	// Binary strings
	TypeBinary    = "binary"
	TypeImage     = "image"
	TypeVarBinary = "varbinary"

	// Other data types
	TypeGeography        = "geography"
	TypeGeometry         = "geometry"
	TypeHierarchyID      = "hierarchyid"
	TypeRowVersion       = "rowversion"
	TypeSQLVariant       = "sql_variant"
	TypeUniqueIdentifier = "uniqueidentifier"
	TypeXML              = "xml"
)

// ISO synonym type
const (
	typeISODec                      = "dec"                        // decimal
	typeISODoublePrecision          = "double precision"           // float(53)
	typeISOCharacter                = "character"                  // char
	typeISOCharVarying              = "charvarying"                // varchar
	typeISOCharacterVarying         = "charactervarying"           // varchar
	typeISONationalChar             = "national char"              // nchar
	typeISONationalCharacter        = "national character"         // nchar
	typeISONationalCharVarying      = "national char varying"      // nvarchar
	typeISONationalCharacterVarying = "national character varying" // nvarchar
	typeISONationalText             = "national text"              // ntext
)

// ANSI synonym type
const (
	typeANSIBinaryVarying = "binary varying"
)

// List of supported index types.
const (
	IndexTypeClustered  = "CLUSTERED"
	IndexTypeFullText   = "FULLTEXT"
	IndexTypeHash       = "HASH"
	IndexTypeNonCluster = "NONCLUSTERED"
	IndexTypeSpatial    = "SPATIAL"
	IndexTypeUnique     = "UNIQUE"
	IndexTypeXML        = "XML"
)

const (
	computedPersisted = "PERSISTED"
)
