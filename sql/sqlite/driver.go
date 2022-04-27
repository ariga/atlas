// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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

func init() {
	sqlclient.Register(
		"sqlite3",
		sqlclient.DriverOpener(Open, func(u *url.URL) string {
			return strings.TrimPrefix(u.String(), u.Scheme+"://")
		}),
		sqlclient.RegisterCodec(MarshalHCL, UnmarshalHCL),
		sqlclient.RegisterFlavours("sqlite"),
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
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
	}, nil
}

// IsClean implements the inlined IsClean interface to override what to consider a clean database.
func (d *Driver) IsClean(ctx context.Context) (bool, error) {
	r, err := d.InspectRealm(ctx, nil)
	if err != nil {
		return false, err
	}
	return r == nil || len(r.Schemas) == 1 && r.Schemas[0].Name == "main" && len(r.Schemas[0].Tables) == 0, nil
}

// Clean implements the inlined migrate.Clean interface to override the "emptying" behavior.
func (d *Driver) Clean(ctx context.Context) error {
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
