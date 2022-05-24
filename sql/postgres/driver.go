// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"golang.org/x/mod/semver"
)

type (
	// Driver represents a PostgreSQL driver for introspecting database schemas,
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
		collate string
		ctype   string
		version string
		crdb    bool
	}
)

func init() {
	sqlclient.Register(
		"postgres",
		sqlclient.DriverOpener(Open),
		sqlclient.RegisterCodec(MarshalHCL, EvalHCL),
		sqlclient.RegisterURLParser(func(u *url.URL) *sqlclient.URL {
			return &sqlclient.URL{URL: u, DSN: u.String(), Schema: u.Query().Get("search_path")}
		}),
	)
	sqlclient.Register(
		"cockroach",
		sqlclient.DriverOpener(OpenCRDB),
		sqlclient.RegisterCodec(MarshalHCL, EvalHCL),
		sqlclient.RegisterFlavours("crdb"),
		sqlclient.RegisterURLParser(func(u *url.URL) *sqlclient.URL {
			return &sqlclient.URL{URL: u, DSN: u.String(), Schema: u.Query().Get("search_path")}
		}),
	)
}

// OpenCRDB opens a new CRDB driver.
func OpenCRDB(db schema.ExecQuerier) (migrate.Driver, error) {
	drv, err := open(db)
	if err != nil {
		return nil, err
	}
	c := drv.conn
	c.crdb = true
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &crdbDiff{diff{c}}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

// Open opens a new PostgreSQL driver.
func Open(db schema.ExecQuerier) (migrate.Driver, error) {
	return open(db)
}

func open(db schema.ExecQuerier) (*Driver, error) {
	c := conn{ExecQuerier: db}
	rows, err := db.QueryContext(context.Background(), paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning system variables: %w", err)
	}
	params, err := sqlx.ScanStrings(rows)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed scanning rows: %w", err)
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
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

func (d *Driver) dev() *sqlx.DevDriver {
	return &sqlx.DevDriver{Driver: d, MaxNameLen: 63}
}

// NormalizeRealm returns the normal representation of the given database.
func (d *Driver) NormalizeRealm(ctx context.Context, r *schema.Realm) (*schema.Realm, error) {
	return d.dev().NormalizeRealm(ctx, r)
}

// NormalizeSchema returns the normal representation of the given database.
func (d *Driver) NormalizeSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	return d.dev().NormalizeSchema(ctx, s)
}

// Lock implements the schema.Locker interface.
func (d *Driver) Lock(ctx context.Context, name string, timeout time.Duration) (schema.UnlockFunc, error) {
	conn, err := sqlx.SingleConn(ctx, d.ExecQuerier)
	if err != nil {
		return nil, err
	}
	h := fnv.New32()
	h.Write([]byte(name))
	id := h.Sum32()
	if err := acquire(ctx, conn, id, timeout); err != nil {
		conn.Close()
		return nil, err
	}
	return func() error {
		defer conn.Close()
		rows, err := conn.QueryContext(ctx, "SELECT pg_advisory_unlock($1)", id)
		if err != nil {
			return err
		}
		switch released, err := sqlx.ScanNullBool(rows); {
		case err != nil:
			return err
		case !released.Valid || !released.Bool:
			return fmt.Errorf("sql/postgres: failed releasing lock %d", id)
		}
		return nil
	}, nil
}

func acquire(ctx context.Context, conn schema.ExecQuerier, id uint32, timeout time.Duration) error {
	switch {
	// With timeout (context-based).
	case timeout > 0:
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		fallthrough
	// Infinite timeout.
	case timeout < 0:
		rows, err := conn.QueryContext(ctx, "SELECT pg_advisory_lock($1)", id)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			err = schema.ErrLocked
		}
		if err != nil {
			return err
		}
		return rows.Close()
	// No timeout.
	default:
		rows, err := conn.QueryContext(ctx, "SELECT pg_try_advisory_lock($1)", id)
		if err != nil {
			return err
		}
		acquired, err := sqlx.ScanNullBool(rows)
		if err != nil {
			return err
		}
		if !acquired.Bool {
			return schema.ErrLocked
		}
		return nil
	}
}

// Standard column types (and their aliases) as defined in
// PostgreSQL codebase/website.
const (
	TypeBit     = "bit"
	TypeBitVar  = "bit varying"
	TypeBoolean = "boolean"
	TypeBool    = "bool" // boolean.
	TypeBytea   = "bytea"

	TypeCharacter = "character"
	TypeChar      = "char" // character
	TypeCharVar   = "character varying"
	TypeVarChar   = "varchar" // character varying
	TypeText      = "text"

	TypeSmallInt = "smallint"
	TypeInteger  = "integer"
	TypeBigInt   = "bigint"
	TypeInt      = "int"   // integer.
	TypeInt2     = "int2"  // smallint.
	TypeInt4     = "int4"  // integer.
	TypeInt8     = "int8"  // bigint.
	TypeInt64    = "int64" // cockroach bigint.

	TypeCIDR     = "cidr"
	TypeInet     = "inet"
	TypeMACAddr  = "macaddr"
	TypeMACAddr8 = "macaddr8"

	TypeCircle   = "circle"
	TypeLine     = "line"
	TypeLseg     = "lseg"
	TypeBox      = "box"
	TypePath     = "path"
	TypePolygon  = "polygon"
	TypePoint    = "point"
	TypeGeometry = "geometry" // cockroach geometry.

	TypeDate          = "date"
	TypeTime          = "time"   // time without time zone
	TypeTimeTZ        = "timetz" // time with time zone
	TypeTimeWTZ       = "time with time zone"
	TypeTimeWOTZ      = "time without time zone"
	TypeTimestamp     = "timestamp" // timestamp without time zone
	TypeTimestampTZ   = "timestamptz"
	TypeTimestampWTZ  = "timestamp with time zone"
	TypeTimestampWOTZ = "timestamp without time zone"

	TypeDouble = "double precision"
	TypeReal   = "real"
	TypeFloat8 = "float8" // double precision
	TypeFloat4 = "float4" // real

	TypeNumeric = "numeric"
	TypeDecimal = "decimal" // numeric

	TypeSmallSerial = "smallserial" // smallint with auto_increment.
	TypeSerial      = "serial"      // integer with auto_increment.
	TypeBigSerial   = "bigserial"   // bigint with auto_increment.
	TypeSerial2     = "serial2"     // smallserial
	TypeSerial4     = "serial4"     // serial
	TypeSerial8     = "serial8"     // bigserial

	TypeArray       = "array"
	TypeXML         = "xml"
	TypeJSON        = "json"
	TypeJSONB       = "jsonb"
	TypeUUID        = "uuid"
	TypeMoney       = "money"
	TypeInterval    = "interval"
	TypeUserDefined = "user-defined"
)

// List of supported index types.
const (
	IndexTypeBTree      = "BTREE"
	IndexTypeHash       = "HASH"
	IndexTypeGIN        = "GIN"
	IndexTypeGiST       = "GIST"
	IndexTypeBRIN       = "BRIN"
	defaultPagePerRange = 128
)

// List of "GENERATED" types.
const (
	GeneratedTypeAlways    = "ALWAYS"
	GeneratedTypeByDefault = "BY_DEFAULT" // BY DEFAULT.
)

// List of PARTITION KEY types.
const (
	PartitionTypeRange = "RANGE"
	PartitionTypeList  = "LIST"
	PartitionTypeHash  = "HASH"
)
