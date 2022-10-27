// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
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
		dialect string
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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("spanner: query database options: %w", err)
	}
	if err := sqlx.ScanOne(rows, &c.dialect); err != nil {
		return nil, fmt.Errorf("spanner: scan database options: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("spanner: failed to execute query: %w", err)
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
	return func() error { return nil }, nil
}

// Standard column types (and their aliases) as defined by Spanner.
const (
	TypeString    = "STRING"
	TypeArray     = "ARRAY"
	TypeBytes     = "BYTES"
	TypeInt64     = "INT64"
	TypeBool      = "BOOL"
	TypeFloat64   = "FLOAT64"
	TypeNumeric   = "NUMERIC"
	TypeTimestamp = "TIMESTAMP"
	TypeDate      = "DATE"
	TypeStruct    = "STRUCT"
	TypeJSON      = "JSON"
)

const (
	stored = "STORED"
)
