// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"fmt"

	"ariga.io/atlas/sql/internal/sqlx"

	"ariga.io/atlas/sql/schema"
)

type (
	// Driver represents a SQLite driver for introspecting database schemas,
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
		fkEnabled  bool
		version    string
		collations []string
	}
)

// Open opens a new SQLite driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	c := conn{ExecQuerier: db}
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&c.version); err != nil {
		return nil, fmt.Errorf("sqlite: scanning database version: %w", err)
	}
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&c.fkEnabled); err != nil {
		return nil, fmt.Errorf("sqlite: check foreign_keys pragma: %w", err)
	}
	rows, err := db.Query("SELECT name FROM pragma_collation_list()")
	if err != nil {
		return nil, fmt.Errorf("sqlite: check collation_list pragma: %w", err)
	}
	if c.collations, err = sqlx.ScanStrings(rows); err != nil {
		return nil, fmt.Errorf("sqlite: scanning database collations: %w", err)
	}
	return &Driver{
		conn:      c,
		Differ:    &sqlx.Diff{DiffDriver: &diff{c}},
		Execer:    &planApply{c},
		Inspector: &inspect{c},
	}, nil
}

// SQLite standard data types as defined in its codebase and documentation.
// https://www.sqlite.org/datatype3.html
// https://github.com/sqlite/sqlite/blob/master/src/global.c
const (
	tInteger = "integer" // SQLITE_TYPE_INTEGER
	tReal    = "real"    // SQLITE_TYPE_REAL
	tText    = "text"    // SQLITE_TYPE_TEXT
	tBlob    = "blob"    // SQLITE_TYPE_BLOB
)
