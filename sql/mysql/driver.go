// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/mysql/internal/mysqlversion"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// Driver represents a MySQL driver for introspecting database schemas,
	// generating diff between schema elements and apply migrations changes.
	Driver struct {
		*conn
		schema.Differ
		schema.Inspector
		migrate.PlanApplier
	}

	// database connection and its information.
	conn struct {
		schema.ExecQuerier
		// The schema was set in the path (schema connection).
		schema string
		// System variables that are set on `Open`.
		mysqlversion.V
		collate string
		charset string
		lcnames int
	}
)

var _ interface {
	migrate.Snapshoter
	migrate.StmtScanner
	migrate.CleanChecker
	schema.TypeParseFormatter
} = (*Driver)(nil)

// DriverName holds the name used for registration.
const DriverName = "mysql"

func init() {
	sqlclient.Register(
		DriverName,
		sqlclient.OpenerFunc(opener),
		sqlclient.RegisterDriverOpener(Open),
		sqlclient.RegisterCodec(MarshalHCL, EvalHCL),
		sqlclient.RegisterFlavours("mysql+unix", "maria", "maria+unix", "mariadb", "mariadb+unix"),
		sqlclient.RegisterURLParser(parser{}),
	)
}

// Open opens a new MySQL driver.
func Open(db schema.ExecQuerier) (migrate.Driver, error) {
	c := &conn{ExecQuerier: db}
	rows, err := db.QueryContext(context.Background(), variablesQuery)
	if err != nil {
		return nil, fmt.Errorf("mysql: query system variables: %w", err)
	}
	if err := sqlx.ScanOne(rows, &c.V, &c.collate, &c.charset, &c.lcnames); err != nil {
		return nil, fmt.Errorf("mysql: scan system variables: %w", err)
	}
	if c.TiDB() {
		return &Driver{
			conn:        c,
			Differ:      &sqlx.Diff{DiffDriver: &tdiff{diff{conn: c}}},
			Inspector:   &tinspect{inspect{c}},
			PlanApplier: &tplanApply{planApply{c}},
		}, nil
	}
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{conn: c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
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
	drv.(*Driver).schema = ur.Schema
	return &sqlclient.Client{
		Name:   DriverName,
		DB:     db,
		URL:    ur,
		Driver: drv,
	}, nil
}

// NormalizeRealm returns the normal representation of the given database.
func (d *Driver) NormalizeRealm(ctx context.Context, r *schema.Realm) (*schema.Realm, error) {
	return (&sqlx.DevDriver{Driver: d}).NormalizeRealm(ctx, r)
}

// NormalizeSchema returns the normal representation of the given database.
func (d *Driver) NormalizeSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	return (&sqlx.DevDriver{Driver: d}).NormalizeSchema(ctx, s)
}

// Lock implements the schema.Locker interface.
func (d *Driver) Lock(ctx context.Context, name string, timeout time.Duration) (schema.UnlockFunc, error) {
	conn, err := sqlx.SingleConn(ctx, d.ExecQuerier)
	if err != nil {
		return nil, err
	}
	if err := acquire(ctx, conn, name, timeout); err != nil {
		conn.Close()
		return nil, err
	}
	return func() error {
		defer conn.Close()
		rows, err := conn.QueryContext(ctx, "SELECT RELEASE_LOCK(?)", name)
		if err != nil {
			return err
		}
		switch released, err := sqlx.ScanNullBool(rows); {
		case err != nil:
			return err
		case !released.Valid || !released.Bool:
			return fmt.Errorf("sql/mysql: failed releasing a named lock %q", name)
		}
		return nil
	}, nil
}

// Snapshot implements migrate.Snapshoter.
func (d *Driver) Snapshot(ctx context.Context) (migrate.RestoreFunc, error) {
	// If the connection is bound to a schema, we can restore the state if the schema has no tables.
	s, err := d.InspectSchema(ctx, "", nil)
	if err != nil && !schema.IsNotExistError(err) {
		return nil, err
	}
	// If a schema was found, it has to have no tables attached to be considered clean.
	if s != nil {
		if len(s.Tables) > 0 {
			return nil, &migrate.NotCleanError{
				State:  schema.NewRealm(s),
				Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name),
			}
		}
		return d.SchemaRestoreFunc(s), nil
	}
	// Otherwise, the database can not have any schema.
	realm, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return nil, err
	}
	if len(realm.Schemas) > 0 {
		return nil, &migrate.NotCleanError{State: realm, Reason: fmt.Sprintf("found schema %q", realm.Schemas[0].Name)}
	}
	return d.RealmRestoreFunc(realm), nil
}

// SchemaRestoreFunc returns a function that restores the given schema to its desired state.
func (d *Driver) SchemaRestoreFunc(desired *schema.Schema) migrate.RestoreFunc {
	return func(ctx context.Context) error {
		current, err := d.InspectSchema(ctx, desired.Name, nil)
		if err != nil {
			return err
		}
		changes, err := d.SchemaDiff(current, desired)
		if err != nil {
			return err
		}
		return d.ApplyChanges(ctx, changes)
	}
}

// RealmRestoreFunc returns a function that restores the given realm to its desired state.
func (d *Driver) RealmRestoreFunc(desired *schema.Realm) migrate.RestoreFunc {
	return func(ctx context.Context) error {
		current, err := d.InspectRealm(ctx, nil)
		if err != nil {
			return err
		}
		changes, err := d.RealmDiff(current, desired)
		if err != nil {
			return err
		}
		return d.ApplyChanges(ctx, changes)
	}
}

// CheckClean implements migrate.CleanChecker.
func (d *Driver) CheckClean(ctx context.Context, revT *migrate.TableIdent) error {
	if revT == nil { // accept nil values
		revT = &migrate.TableIdent{}
	}
	s, err := d.InspectSchema(ctx, "", nil)
	if err != nil && !schema.IsNotExistError(err) {
		return err
	}
	if s != nil {
		if len(s.Tables) == 0 || (revT.Schema == "" || s.Name == revT.Schema) && len(s.Tables) == 1 && s.Tables[0].Name == revT.Name {
			return nil
		}
		return &migrate.NotCleanError{
			State:  schema.NewRealm(s),
			Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name),
		}
	}
	r, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return err
	}
	switch n := len(r.Schemas); {
	case n > 1:
		return &migrate.NotCleanError{State: r, Reason: fmt.Sprintf("found multiple schemas: %d", len(r.Schemas))}
	case n == 1 && r.Schemas[0].Name != revT.Schema:
		return &migrate.NotCleanError{State: r, Reason: fmt.Sprintf("found schema %q", r.Schemas[0].Name)}
	case n == 1 && len(r.Schemas[0].Tables) > 1:
		return &migrate.NotCleanError{State: r, Reason: fmt.Sprintf("found multiple tables: %d", len(r.Schemas[0].Tables))}
	case n == 1 && len(r.Schemas[0].Tables) == 1 && r.Schemas[0].Tables[0].Name != revT.Name:
		return &migrate.NotCleanError{State: r, Reason: fmt.Sprintf("found table %q", r.Schemas[0].Tables[0].Name)}
	}
	return nil
}

// Version returns the version of the connected database.
func (d *Driver) Version() string {
	return string(d.conn.V)
}

// FormatType converts schema type to its column form in the database.
func (*Driver) FormatType(t schema.Type) (string, error) {
	return FormatType(t)
}

// ParseType returns the schema.Type value represented by the given string.
func (*Driver) ParseType(s string) (schema.Type, error) {
	return ParseType(s)
}

// StmtBuilder is a helper method used to build statements with MySQL formatting.
func (*Driver) StmtBuilder(opts migrate.PlanOptions) *sqlx.Builder {
	return &sqlx.Builder{
		QuoteOpening: '`',
		QuoteClosing: '`',
		Schema:       opts.SchemaQualifier,
		Indent:       opts.Indent,
	}
}

// ScanStmts implements migrate.StmtScanner.
func (*Driver) ScanStmts(input string) ([]*migrate.Stmt, error) {
	return (&migrate.Scanner{
		ScannerOptions: migrate.ScannerOptions{
			MatchBegin: true,
			// The following are not support by MySQL/MariaDB.
			MatchBeginAtomic: false,
			MatchDollarQuote: false,
		},
	}).Scan(input)
}

func acquire(ctx context.Context, conn schema.ExecQuerier, name string, timeout time.Duration) error {
	rows, err := conn.QueryContext(ctx, "SELECT GET_LOCK(?, ?)", name, int(timeout.Seconds()))
	if err != nil {
		return err
	}
	switch acquired, err := sqlx.ScanNullBool(rows); {
	case err != nil:
		return err
	case !acquired.Valid:
		// NULL is returned in case of an unexpected internal error.
		return fmt.Errorf("sql/mysql: unexpected internal error on Lock(%q, %s)", name, timeout)
	case !acquired.Bool:
		return schema.ErrLocked
	}
	return nil
}

// unescape strings with backslashes returned
// for SQL expressions from information schema.
func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c != '\\' || i == len(s)-1:
			b.WriteByte(c)
		case s[i+1] == '\'', s[i+1] == '\\':
			b.WriteByte(s[i+1])
			i++
		}
	}
	return b.String()
}

type parser struct{}

// ParseURL implements the sqlclient.URLParser interface.
func (parser) ParseURL(u *url.URL) *sqlclient.URL {
	v := u.Query()
	v.Set("parseTime", "true")
	u.RawQuery = v.Encode()
	return &sqlclient.URL{URL: u, DSN: dsn(u), Schema: strings.TrimPrefix(u.Path, "/")}
}

// ChangeSchema implements the sqlclient.SchemaChanger interface.
func (parser) ChangeSchema(u *url.URL, s string) *url.URL {
	nu := *u
	nu.Path = "/" + s
	return &nu
}

// dsn returns the MySQL standard DSN for opening
// the sql.DB from the user provided URL.
func dsn(u *url.URL) string {
	var (
		b      strings.Builder
		values = u.Query()
	)
	b.WriteString(u.User.Username())
	if p, ok := u.User.Password(); ok {
		b.WriteByte(':')
		b.WriteString(p)
	}
	if b.Len() > 0 {
		b.WriteByte('@')
	}
	switch {
	case strings.HasSuffix(u.Scheme, "+unix"):
		b.WriteString("unix(")
		// The path is always absolute, and
		// therefore the host should be empty.
		b.WriteString(u.Path)
		b.WriteString(")/")
		if name := values.Get("database"); name != "" {
			b.WriteString(name)
			values.Del("database")
		}
	default:
		if u.Host != "" {
			b.WriteString("tcp(")
			b.WriteString(u.Host)
			b.WriteByte(')')
		}
		if u.Path != "" {
			b.WriteString(u.Path)
		} else {
			b.WriteByte('/')
		}
	}
	if p := values.Encode(); p != "" {
		b.WriteByte('?')
		b.WriteString(p)
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
// https://dev.mysql.com/doc/refman/8.0/en/other-vendor-data-types.html
const (
	TypeBool    = "bool"
	TypeBoolean = "boolean"

	TypeBit       = "bit"       // MYSQL_TYPE_BIT
	TypeInt       = "int"       // MYSQL_TYPE_LONG
	TypeTinyInt   = "tinyint"   // MYSQL_TYPE_TINY
	TypeSmallInt  = "smallint"  // MYSQL_TYPE_SHORT
	TypeMediumInt = "mediumint" // MYSQL_TYPE_INT24
	TypeBigInt    = "bigint"    // MYSQL_TYPE_LONGLONG

	TypeDecimal = "decimal" // MYSQL_TYPE_DECIMAL
	TypeNumeric = "numeric" // MYSQL_TYPE_DECIMAL (numeric_type rule in sql_yacc.yy)
	TypeFloat   = "float"   // MYSQL_TYPE_FLOAT
	TypeDouble  = "double"  // MYSQL_TYPE_DOUBLE
	TypeReal    = "real"    // MYSQL_TYPE_FLOAT or MYSQL_TYPE_DOUBLE (real_type in sql_yacc.yy)

	TypeTimestamp = "timestamp" // MYSQL_TYPE_TIMESTAMP
	TypeDate      = "date"      // MYSQL_TYPE_DATE
	TypeTime      = "time"      // MYSQL_TYPE_TIME
	TypeDateTime  = "datetime"  // MYSQL_TYPE_DATETIME
	TypeYear      = "year"      // MYSQL_TYPE_YEAR

	TypeVarchar    = "varchar"    // MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_VARCHAR
	TypeChar       = "char"       // MYSQL_TYPE_STRING
	TypeVarBinary  = "varbinary"  // MYSQL_TYPE_VAR_STRING + NULL CHARACTER_SET.
	TypeBinary     = "binary"     // MYSQL_TYPE_STRING + NULL CHARACTER_SET.
	TypeBlob       = "blob"       // MYSQL_TYPE_BLOB
	TypeTinyBlob   = "tinyblob"   // MYSQL_TYPE_TINYBLOB
	TypeMediumBlob = "mediumblob" // MYSQL_TYPE_MEDIUM_BLOB
	TypeLongBlob   = "longblob"   // MYSQL_TYPE_LONG_BLOB
	TypeText       = "text"       // MYSQL_TYPE_BLOB + CHARACTER_SET utf8mb4
	TypeTinyText   = "tinytext"   // MYSQL_TYPE_TINYBLOB + CHARACTER_SET utf8mb4
	TypeMediumText = "mediumtext" // MYSQL_TYPE_MEDIUM_BLOB + CHARACTER_SET utf8mb4
	TypeLongText   = "longtext"   // MYSQL_TYPE_LONG_BLOB with + CHARACTER_SET utf8mb4

	TypeEnum = "enum" // MYSQL_TYPE_ENUM
	TypeSet  = "set"  // MYSQL_TYPE_SET
	TypeJSON = "json" // MYSQL_TYPE_JSON

	TypeGeometry           = "geometry"           // MYSQL_TYPE_GEOMETRY
	TypePoint              = "point"              // Geometry_type::kPoint
	TypeMultiPoint         = "multipoint"         // Geometry_type::kMultipoint
	TypeLineString         = "linestring"         // Geometry_type::kLinestring
	TypeMultiLineString    = "multilinestring"    // Geometry_type::kMultilinestring
	TypePolygon            = "polygon"            // Geometry_type::kPolygon
	TypeMultiPolygon       = "multipolygon"       // Geometry_type::kMultipolygon
	TypeGeoCollection      = "geomcollection"     // Geometry_type::kGeometrycollection
	TypeGeometryCollection = "geometrycollection" // Geometry_type::kGeometrycollection

	TypeUUID = "uuid" // MariaDB supported uuid type from 10.7.0+

	TypeInet4 = "inet4" // MariaDB type for storage of IPv4 addresses, from 10.10.0+.
	TypeInet6 = "inet6" // MariaDB type for storage of IPv6 addresses, from 10.10.0+.
)

// Additional common constants in MySQL.
const (
	IndexTypeBTree    = "BTREE"
	IndexTypeHash     = "HASH"
	IndexTypeFullText = "FULLTEXT"
	IndexTypeSpatial  = "SPATIAL"

	IndexParserNGram = "ngram"
	IndexParserMeCab = "mecab"

	EngineInnoDB = "InnoDB"
	EngineMyISAM = "MyISAM"
	EngineMemory = "Memory"
	EngineCSV    = "CSV"
	EngineNDB    = "NDB" // NDBCLUSTER

	currentTS     = "current_timestamp"
	defaultGen    = "default_generated"
	autoIncrement = "auto_increment"

	virtual    = "VIRTUAL"
	stored     = "STORED"
	persistent = "PERSISTENT"
)
