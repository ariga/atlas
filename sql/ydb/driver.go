// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	ydbSdk "github.com/ydb-platform/ydb-go-sdk/v3"
)

type (
	// Driver represents a YDB driver for introspecting database schemas,
	// generating diff between schema elements and apply migrations changes.
	Driver struct {
		*conn
		schema.Differ
		schema.Inspector
		migrate.PlanApplier
	}

	// conn represents a database connection and its information.
	conn struct {
		schema.ExecQuerier

		// We use native ydb driver in order to introspect database
		nativeDriver *ydbSdk.Driver

		// YDB doesn't have concept of schema in the same meaning as in Postgres for example.
		// Instead, database objects are organized using "directores"
		// and it is possible to have multiple databases in a single distributed cluster.
		// So we just store the database path prefix (e.g., "/local")
		database string
		// Version of YDB server
		version string
	}
)

var _ interface {
	migrate.StmtScanner
	schema.TypeParseFormatter
} = (*Driver)(nil)

// DriverName holds the name used for registration.
const DriverName = "ydb"

func init() {
	sqlclient.Register(
		DriverName,
		sqlclient.OpenerFunc(opener),
		sqlclient.RegisterURLParser(parser{}),
	)
}

func opener(ctx context.Context, dsn *url.URL) (*sqlclient.Client, error) {
	parser := parser{}.ParseURL(dsn)

	nativeDriver, err := ydbSdk.Open(ctx, parser.DSN)
	if err != nil {
		return nil, err
	}

	conn, err := ydbSdk.Connector(
		nativeDriver,
		ydbSdk.WithAutoDeclare(),
		ydbSdk.WithTablePathPrefix(nativeDriver.Name()),
	)
	if err != nil {
		return nil, err
	}

	sqlDriver := sql.OpenDB(conn)
	drv, err := Open(nativeDriver, sqlDriver)
	if err != nil {
		if cerr := sqlDriver.Close(); cerr != nil {
			err = fmt.Errorf("%w: %v", err, cerr)
		}
		return nil, err
	}

	if d, ok := drv.(*Driver); ok {
		d.database = parser.Schema
	}

	return &sqlclient.Client{
		Name:   DriverName,
		DB:     sqlDriver,
		URL:    parser,
		Driver: drv,
	}, nil
}

// Open opens a new YDB driver.
func Open(nativeDriver *ydbSdk.Driver, sqlDriver *sql.DB) (migrate.Driver, error) {
	c := &conn{
		ExecQuerier:  sqlDriver,
		nativeDriver: nativeDriver,
	}

	rows, err := sqlDriver.QueryContext(context.Background(), "SELECT version()")
	if err != nil {
		return nil, fmt.Errorf("ydb: failed to query version: %w", err)
	}

	var version sql.NullString
	if err := sqlx.ScanOne(rows, &version); err != nil {
		return nil, fmt.Errorf("ydb: failed to scan version: %w", err)
	}
	c.version = version.String

	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   newInspect(c),
		PlanApplier: &planApply{c},
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
// YDB doesn't support advisory locks, so this is a no-op.
func (d *Driver) Lock(_ context.Context, _ string, _ time.Duration) (schema.UnlockFunc, error) {
	return func() error { return nil }, nil
}

// Snapshot implements migrate.Snapshoter.
func (d *Driver) Snapshot(ctx context.Context) (migrate.RestoreFunc, error) {
	if d.database != "" {
		dbSchema, err := d.InspectSchema(ctx, d.database, nil)
		if err != nil {
			return nil, err
		}

		if len(dbSchema.Tables) > 0 {
			return nil, &migrate.NotCleanError{
				State:  schema.NewRealm(dbSchema),
				Reason: fmt.Sprintf("found table %q in connected schema", dbSchema.Tables[0].Name),
			}
		}

		return d.SchemaRestoreFunc(dbSchema), nil
	}

	realm, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return nil, err
	}

	if len(realm.Schemas) > 0 && len(realm.Schemas[0].Tables) > 0 {
		return nil, &migrate.NotCleanError{
			State:  realm,
			Reason: fmt.Sprintf("found table %q in schema %q", realm.Schemas[0].Tables[0].Name, realm.Schemas[0].Name),
		}
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
	if revT == nil {
		revT = &migrate.TableIdent{}
	}

	if d.database != "" {
		switch s, err := d.InspectSchema(ctx, d.database, nil); {
		case err != nil:
			return err
		case len(s.Tables) == 0:
			return nil
		case (revT.Schema == "" || s.Name == revT.Schema) && len(s.Tables) == 1 && s.Tables[0].Name == revT.Name:
			return nil
		default:
			return &migrate.NotCleanError{
				State:  schema.NewRealm(s),
				Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name),
			}
		}
	}

	realm, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return err
	}

	for _, s := range realm.Schemas {
		switch {
		case len(s.Tables) == 0:
			continue
		case s.Name != revT.Schema || len(s.Tables) > 1:
			return &migrate.NotCleanError{
				State:  realm,
				Reason: fmt.Sprintf("found multiple tables in schema %q", s.Name),
			}
		case s.Tables[0].Name != revT.Name:
			return &migrate.NotCleanError{
				State:  realm,
				Reason: fmt.Sprintf("found table %q in schema %q", s.Tables[0].Name, s.Name),
			}
		}
	}
	return nil
}

// Version returns the version of the connected database.
func (d *Driver) Version() string {
	return d.conn.version
}

// FormatType converts schema type to its column form in the database.
func (*Driver) FormatType(t schema.Type) (string, error) {
	return FormatType(t)
}

// ParseType returns the schema.Type value represented by the given string.
func (*Driver) ParseType(s string) (schema.Type, error) {
	return ParseType(s)
}

// StmtBuilder is a helper method used to build statements with YDB formatting.
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
			MatchBegin: false,
		},
	}).Scan(input)
}

type parser struct{}

// ParseURL implements the sqlclient.URLParser interface.
func (parser) ParseURL(url *url.URL) *sqlclient.URL {
	// YDB connection string format: grpc://localhost:2136/local
	// The path part becomes the database/schema
	return &sqlclient.URL{
		URL:    url,
		DSN:    url.String(),
		Schema: url.Path, // e.g., "/local"
	}
}

// ChangeSchema implements the sqlclient.SchemaChanger interface.
func (parser) ChangeSchema(url *url.URL, schema string) *url.URL {
	nu := *url
	nu.Path = schema
	return &nu
}
