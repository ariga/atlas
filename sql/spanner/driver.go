// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

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
	_ "github.com/googleapis/go-sql-spanner"
)

type (
	// Driver represents a Spanner driver for introspecting database schemas,
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
		databaseDialect string
	}
)

func init() {
	sqlclient.Register(
		"spanner",
		sqlclient.DriverOpener(Open),
		sqlclient.RegisterCodec(MarshalHCL, EvalHCL),
		sqlclient.RegisterURLParser(sqlclient.URLParserFunc(func(u *url.URL) *sqlclient.URL {
			uc := &sqlclient.URL{URL: u, DSN: strings.TrimPrefix(u.String(), u.Scheme+"://")}
			return uc
		})),
	)
}

// Open opens a new Spanner driver.
func Open(db schema.ExecQuerier) (migrate.Driver, error) {
	var (
		c   = conn{ExecQuerier: db}
		ctx = context.Background()
	)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("spanner: query database options: %w", err)
	}
	if err := sqlx.ScanOne(rows, &c.databaseDialect); err != nil {
		return nil, fmt.Errorf("spanner: query database options: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	return &Driver{
		conn:        c,
		Differ:      &sqlx.Diff{DiffDriver: &diff{c}},
		Inspector:   &inspect{c},
		PlanApplier: &planApply{c},
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
		return nil, fmt.Errorf("sql/spanner: reading lock dir: %w", err)
	}
	expires, err := strconv.ParseInt(string(c), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sql/spanner: invalid lock file format: parsing expiration date: %w", err)
	}
	if time.Unix(0, expires).After(time.Now()) {
		// Lock is still valid.
		return nil, fmt.Errorf("sql/spanner: lock on %q already taken", name)
	}
	return acquireLock(path, timeout)
}

func acquireLock(path string, timeout time.Duration) (schema.UnlockFunc, error) {
	lock, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("sql/spanner: creating lockfile %q: %w", path, err)
	}
	if _, err := lock.Write([]byte(strconv.FormatInt(time.Now().Add(timeout).UnixNano(), 10))); err != nil {
		return nil, fmt.Errorf("sql/spanner: writing to lockfile %q: %w", path, err)
	}
	defer lock.Close()
	return func() error { return os.Remove(path) }, nil
}

// Standard column types (and their aliases) as defined by Spanner.
const (
	TypeString         = "STRING"
	TypeStringArray    = "ARRAY<STRING>"
	TypeBytes          = "BYTES"
	TypeBytesArray     = "ARRAY<BYTES>"
	TypeInt64          = "INT64"
	TypeInt64Array     = "ARRAY<INT64>"
	TypeBool           = "BOOL"
	TypeBoolArray      = "ARRAY<BOOL>"
	TypeFloat64        = "FLOAT64"
	TypeFloat64Array   = "ARRAY<FLOAT64>"
	TypeNumeric        = "NUMERIC"
	TypeNumericArray   = "ARRAY<NUMERIC>"
	TypeTimestamp      = "TIMESTAMP"
	TypeTimestampArray = "ARRAY<TIMESTAMP>"
	TypeDate           = "DATE"
	TypeDateArray      = "ARRAY<DATE>"
	TypeStruct         = "ARRAY<STRUCT>"
	TypeJSON           = "JSON"
)

// SQLite generated columns types.
const (
	virtual = "VIRTUAL"
	stored  = "STORED"
)
