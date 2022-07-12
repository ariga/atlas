// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

type (
	// Driver represents a SQLite driver for introspecting database schemas,
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
		fkEnabled  bool
		version    string
		collations []string
	}
)

// DriverName holds the name used for registration.
const DriverName = "sqlite3"

func init() {
	sqlclient.Register(
		DriverName,
		sqlclient.DriverOpener(Open),
		sqlclient.RegisterCodec(MarshalHCL, EvalHCL),
		sqlclient.RegisterFlavours("sqlite"),
		sqlclient.RegisterURLParser(sqlclient.URLParserFunc(func(u *url.URL) *sqlclient.URL {
			uc := &sqlclient.URL{URL: u, DSN: strings.TrimPrefix(u.String(), u.Scheme+"://"), Schema: mainFile}
			if mode := u.Query().Get("mode"); mode == "memory" {
				// The "file:" prefix is mandatory for memory modes.
				uc.DSN = "file:" + uc.DSN
			}
			return uc
		})),
	)
}

// Open opens a new SQLite driver.
func Open(db schema.ExecQuerier) (migrate.Driver, error) {
	var (
		c   = conn{ExecQuerier: db}
		ctx = context.Background()
	)
	rows, err := db.QueryContext(ctx, "SELECT sqlite_version(), foreign_keys from pragma_foreign_keys")
	if err != nil {
		return nil, fmt.Errorf("sqlite: query version and foreign_keys pragma: %w", err)
	}
	if err := sqlx.ScanOne(rows, &c.version, &c.fkEnabled); err != nil {
		return nil, fmt.Errorf("sqlite: scan version and foreign_keys pragma: %w", err)
	}
	if rows, err = db.QueryContext(ctx, "SELECT name FROM pragma_collation_list()"); err != nil {
		return nil, fmt.Errorf("sqlite: query foreign_keys pragma: %w", err)
	}
	if c.collations, err = sqlx.ScanStrings(rows); err != nil {
		return nil, fmt.Errorf("sqlite: scanning database collations: %w", err)
	}
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &Diff{}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

// Snapshot implements migrate.Snapshoter.
func (d *Driver) Snapshot(ctx context.Context) (migrate.RestoreFunc, error) {
	r, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return nil, err
	}
	if !(r == nil || (len(r.Schemas) == 1 && r.Schemas[0].Name == mainFile && len(r.Schemas[0].Tables) == 0)) {
		return nil, migrate.NotCleanError{Reason: fmt.Sprintf("found table %q", r.Schemas[0].Tables[0].Name)}
	}
	return func(ctx context.Context) error {
		for _, stmt := range []string{
			"PRAGMA writable_schema = 1;",
			"DELETE FROM sqlite_master WHERE type IN ('table', 'index', 'trigger');",
			"PRAGMA writable_schema = 0;",
			"VACUUM;",
		} {
			if _, err := d.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

// Lock implements the schema.Locker interface.
func (d *Driver) Lock(_ context.Context, name string, timeout time.Duration) (schema.UnlockFunc, error) {
	path := filepath.Join(os.TempDir(), name+".lock")
	c, err := ioutil.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return acquireLock(path, timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("sql/sqlite: reading lock dir: %w", err)
	}
	expires, err := strconv.ParseInt(string(c), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sql/sqlite: invalid lock file format: parsing expiration date: %w", err)
	}
	if time.Unix(0, expires).After(time.Now()) {
		// Lock is still valid.
		return nil, fmt.Errorf("sql/sqlite: lock on %q already taken", name)
	}
	return acquireLock(path, timeout)
}

func acquireLock(path string, timeout time.Duration) (schema.UnlockFunc, error) {
	lock, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("sql/sqlite: creating lockfile %q: %w", path, err)
	}
	if _, err := lock.Write([]byte(strconv.FormatInt(time.Now().Add(timeout).UnixNano(), 10))); err != nil {
		return nil, fmt.Errorf("sql/sqlite: writing to lockfile %q: %w", path, err)
	}
	defer lock.Close()
	return func() error { return os.Remove(path) }, nil
}

// SQLite standard data types as defined in its codebase and documentation.
// https://www.sqlite.org/datatype3.html
// https://github.com/sqlite/sqlite/blob/master/src/global.c
const (
	TypeInteger = "integer" // SQLITE_TYPE_INTEGER
	TypeReal    = "real"    // SQLITE_TYPE_REAL
	TypeText    = "text"    // SQLITE_TYPE_TEXT
	TypeBlob    = "blob"    // SQLITE_TYPE_BLOB
)

// SQLite generated columns types.
const (
	virtual = "VIRTUAL"
	stored  = "STORED"
)
