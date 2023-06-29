// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

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
		// The schema in the `schema` parameter (if given).
		schema  string
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
		sqlclient.OpenerFunc(opener),
		sqlclient.RegisterDriverOpener(Open),
		sqlclient.RegisterURLParser(parser{}),
	)
}

func opener(_ context.Context, u *url.URL) (*sqlclient.Client, error) {
	ur := parser{}.ParseURL(u)
	db, err := sql.Open(DriverName, ur.DSN)
	if err != nil {
		return nil, err
	}
	drv, err := Open(db)
	if err != nil {
		if cerr := db.Close(); cerr != nil {
			err = fmt.Errorf("%w: %v", err, cerr)
		}
		return nil, err
	}
	switch drv := drv.(type) {
	case *Driver:
		drv.schema = ur.Schema
	}
	return &sqlclient.Client{
		Name:   DriverName,
		DB:     db,
		URL:    ur,
		Driver: drv,
	}, nil
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

// Snapshot implements migrate.Snapshoter.
func (d *Driver) Snapshot(ctx context.Context) (migrate.RestoreFunc, error) {
	// The connection is considered bound to the realm.
	if d.schema != "" {
		s, err := d.InspectSchema(ctx, d.schema, nil)
		if err != nil {
			return nil, err
		}
		if len(s.Tables) > 0 {
			return nil, &migrate.NotCleanError{Reason: fmt.Sprintf("found table %q in connected schema", s.Tables[0].Name)}
		}
		return func(ctx context.Context) error {
			current, err := d.InspectSchema(ctx, s.Name, nil)
			if err != nil {
				return err
			}
			changes, err := d.SchemaDiff(current, s)
			if err != nil {
				return err
			}
			return d.ApplyChanges(ctx, changes)
		}, nil
	}
	// Not bound to a schema.
	realm, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return nil, err
	}
	restore := func(ctx context.Context) error {
		current, err := d.InspectRealm(ctx, nil)
		if err != nil {
			return err
		}
		changes, err := d.RealmDiff(current, realm)
		if err != nil {
			return err
		}
		return d.ApplyChanges(ctx, changes)
	}
	// MS-SQL is considered clean, if there are no schemas or the dbo schema has no tables.
	if len(realm.Schemas) == 0 {
		return restore, nil
	}
	if s, ok := realm.Schema("dbo"); len(realm.Schemas) == 1 && ok {
		if len(s.Tables) > 0 {
			return nil, &migrate.NotCleanError{Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name)}
		}
		return restore, nil
	}
	return nil, &migrate.NotCleanError{Reason: fmt.Sprintf("found schema %q", realm.Schemas[0].Name)}
}

// CheckClean implements migrate.CleanChecker.
func (d *Driver) CheckClean(ctx context.Context, revT *migrate.TableIdent) error {
	if revT == nil { // accept nil values
		revT = &migrate.TableIdent{}
	}
	if d.schema != "" {
		switch s, err := d.InspectSchema(ctx, d.schema, nil); {
		case err != nil:
			return err
		case len(s.Tables) == 0, (revT.Schema == "" || s.Name == revT.Schema) && len(s.Tables) == 1 && s.Tables[0].Name == revT.Name:
			return nil
		default:
			return &migrate.NotCleanError{Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name)}
		}
	}
	r, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return err
	}
	for _, s := range r.Schemas {
		switch {
		case len(s.Tables) == 0 && s.Name == "dbo":
		case len(s.Tables) == 0 || s.Name != revT.Schema:
			return &migrate.NotCleanError{Reason: fmt.Sprintf("found schema %q", s.Name)}
		case len(s.Tables) > 1:
			return &migrate.NotCleanError{Reason: fmt.Sprintf("found %d tables in schema %q", len(s.Tables), s.Name)}
		case len(s.Tables) == 1 && s.Tables[0].Name != revT.Name:
			return &migrate.NotCleanError{Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name)}
		}
	}
	return nil
}

type parser struct{}

// ParseURL implements the sqlclient.URLParser interface.
func (parser) ParseURL(u *url.URL) *sqlclient.URL {
	// "schema" is used to specify the schema name.
	// It is not part of the default SQL driver.
	return &sqlclient.URL{
		URL:    u,
		DSN:    u.String(),
		Schema: u.Query().Get("schema"),
	}
}

// ChangeSchema implements the sqlclient.SchemaChanger interface.
func (parser) ChangeSchema(u *url.URL, s string) *url.URL {
	nu := *u
	q := nu.Query()
	q.Set("schema", s)
	nu.RawQuery = q.Encode()
	return &nu
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
	IndexTypeNonClustered = "NONCLUSTERED" // Default
	IndexTypeClustered    = "CLUSTERED"
	IndexTypeFullText     = "FULLTEXT"
	IndexTypeHash         = "HASH"
	IndexTypeSpatial      = "SPATIAL"
	IndexTypeUnique       = "UNIQUE"
	IndexTypeXML          = "XML"
)

const (
	computedPersisted = "PERSISTED"
)
