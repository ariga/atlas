// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

// DriverNameDSQL holds the name used for DSQL driver registration.
const DriverNameDSQL = "auroradsql"

type (
	// dsqlDriver wraps Driver for AWS Aurora DSQL which doesn't support advisory locks.
	dsqlDriver struct {
		*Driver
	}

	// dsqlParser implements the sqlclient.URLParser interface for DSQL.
	dsqlParser struct{}

	// dsqlRevisionReadWriter implements migrate.RevisionReadWriter using direct SQL
	// to bypass Ent which uses unsupported features like regoper.
	dsqlRevisionReadWriter struct {
		db     schema.ExecQuerier
		schema string
	}
)

func init() {
	sqlclient.Register(
		DriverNameDSQL,
		sqlclient.OpenerFunc(dsqlOpener),
		sqlclient.RegisterDriverOpener(OpenDSQL),
		sqlclient.RegisterCodec(codec, codec),
		sqlclient.RegisterURLParser(dsqlParser{}),
	)
}

// OpenDSQL opens a new AWS Aurora DSQL driver.
// DSQL is PostgreSQL-compatible but doesn't support JSON types/functions,
// so we use the CockroachDB code path which avoids JSON in queries.
func OpenDSQL(db schema.ExecQuerier) (migrate.Driver, error) {
	c := &conn{ExecQuerier: db}
	rows, err := db.QueryContext(context.Background(), paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("auroradsql: scanning system variables: %w", err)
	}
	var ver, am, crdb sql.NullString
	if err := sqlx.ScanOne(rows, &ver, &am, &crdb); err != nil {
		return nil, fmt.Errorf("auroradsql: scanning system variables: %w", err)
	}
	if c.version, err = strconv.Atoi(ver.String); err != nil {
		return nil, fmt.Errorf("auroradsql: malformed version: %s: %w", ver.String, err)
	}
	c.accessMethod = am.String
	c.crdb = true // Force CRDB code path (no JSON functions)
	return &dsqlDriver{
		Driver: &Driver{
			conn:        c,
			Differ:      &sqlx.Diff{DiffDriver: &crdbDiff{diff{c}}},
			Inspector:   &crdbInspect{inspect{c}},
			PlanApplier: &planApply{c},
		},
	}, nil
}

func dsqlOpener(_ context.Context, u *url.URL) (*sqlclient.Client, error) {
	ur := dsqlParser{}.ParseURL(u)
	db, err := sql.Open(DriverName, ur.DSN)
	if err != nil {
		return nil, err
	}
	drv, err := OpenDSQL(db)
	if err != nil {
		if cerr := db.Close(); cerr != nil {
			err = fmt.Errorf("%w: %v", err, cerr)
		}
		return nil, err
	}
	if d, ok := drv.(*dsqlDriver); ok {
		d.schema = ur.Schema
	}
	return &sqlclient.Client{
		Name:   DriverNameDSQL,
		DB:     db,
		URL:    ur,
		Driver: drv,
	}, nil
}

// ParseURL implements the sqlclient.URLParser interface.
func (dsqlParser) ParseURL(u *url.URL) *sqlclient.URL {
	u.Scheme = "postgres"
	return &sqlclient.URL{URL: u, DSN: u.String(), Schema: u.Query().Get("search_path")}
}

// Lock implements schema.Locker for dsqlDriver.
// Returns a no-op since Aurora DSQL doesn't support advisory locks.
func (d *dsqlDriver) Lock(_ context.Context, _ string, _ time.Duration) (schema.UnlockFunc, error) {
	return func() error { return nil }, nil
}

// RevisionsReadWriter returns a RevisionReadWriter for DSQL that bypasses Ent.
func (d *dsqlDriver) RevisionsReadWriter(_ context.Context, schema string) (migrate.RevisionReadWriter, error) {
	return &dsqlRevisionReadWriter{db: d.ExecQuerier, schema: schema}, nil
}

// Ident returns the table identifier.
func (r *dsqlRevisionReadWriter) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: "atlas_schema_revisions", Schema: r.schema}
}

// ReadRevisions returns all revisions from the table.
func (r *dsqlRevisionReadWriter) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version FROM %s ORDER BY version`,
		r.tableName(),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var revs []*migrate.Revision
	for rows.Next() {
		rev, err := r.scanRevision(rows)
		if err != nil {
			return nil, err
		}
		revs = append(revs, rev)
	}
	return revs, rows.Err()
}

// ReadRevision returns a revision by version.
func (r *dsqlRevisionReadWriter) ReadRevision(ctx context.Context, version string) (*migrate.Revision, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version FROM %s WHERE version = $1`,
		r.tableName(),
	), version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, migrate.ErrRevisionNotExist
	}
	return r.scanRevision(rows)
}

// WriteRevision saves the revision to the storage.
func (r *dsqlRevisionReadWriter) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	partialHashes, err := json.Marshal(rev.PartialHashes)
	if err != nil {
		return fmt.Errorf("marshaling partial hashes: %w", err)
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (version) DO UPDATE SET
		 description = $2, type = $3, applied = $4, total = $5, executed_at = $6, execution_time = $7, error = $8, error_stmt = $9, hash = $10, partial_hashes = $11, operator_version = $12`,
		r.tableName(),
	), rev.Version, rev.Description, rev.Type, rev.Applied, rev.Total, rev.ExecutedAt, rev.ExecutionTime, rev.Error, rev.ErrorStmt, rev.Hash, string(partialHashes), rev.OperatorVersion)
	return err
}

// DeleteRevision deletes a revision by version.
func (r *dsqlRevisionReadWriter) DeleteRevision(ctx context.Context, version string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, r.tableName()), version)
	return err
}

// CurrentRevision returns the current (latest) revision.
func (r *dsqlRevisionReadWriter) CurrentRevision(ctx context.Context) (*migrate.Revision, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version FROM %s ORDER BY version DESC LIMIT 1`,
		r.tableName(),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, migrate.ErrRevisionNotExist
	}
	return r.scanRevision(rows)
}

// Migrate creates the revisions schema and table if they don't exist.
func (r *dsqlRevisionReadWriter) Migrate(ctx context.Context) error {
	if r.schema != "" {
		if _, err := r.db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, r.schema)); err != nil {
			return fmt.Errorf("creating schema %q: %w", r.schema, err)
		}
	}
	// Using TEXT for partial_hashes since DSQL doesn't support JSONB.
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) NOT NULL PRIMARY KEY,
			description VARCHAR(255) NOT NULL,
			type INTEGER NOT NULL DEFAULT 2,
			applied INTEGER NOT NULL DEFAULT 0,
			total INTEGER NOT NULL DEFAULT 0,
			executed_at TIMESTAMPTZ NOT NULL,
			execution_time BIGINT NOT NULL,
			error TEXT,
			error_stmt TEXT,
			hash VARCHAR(255) NOT NULL,
			partial_hashes TEXT,
			operator_version VARCHAR(255) NOT NULL
		)
	`, r.tableName()))
	return err
}

// ID returns an identifier for the connected database.
func (r *dsqlRevisionReadWriter) ID(_ context.Context, _ string) (string, error) {
	if r.schema != "" {
		return r.schema, nil
	}
	return "public", nil
}

func (r *dsqlRevisionReadWriter) tableName() string {
	if r.schema != "" {
		return fmt.Sprintf("%q.atlas_schema_revisions", r.schema)
	}
	return "atlas_schema_revisions"
}

func (r *dsqlRevisionReadWriter) scanRevision(rows *sql.Rows) (*migrate.Revision, error) {
	var (
		rev           migrate.Revision
		partialHashes string
		execTime      int64
	)
	if err := rows.Scan(
		&rev.Version, &rev.Description, &rev.Type, &rev.Applied, &rev.Total,
		&rev.ExecutedAt, &execTime, &rev.Error, &rev.ErrorStmt, &rev.Hash,
		&partialHashes, &rev.OperatorVersion,
	); err != nil {
		return nil, err
	}
	rev.ExecutionTime = time.Duration(execTime)
	if partialHashes != "" {
		if err := json.Unmarshal([]byte(partialHashes), &rev.PartialHashes); err != nil {
			return nil, err
		}
	}
	return &rev, nil
}
