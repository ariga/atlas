// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
)

// DefaultRevisionSchema is the default schema for storing revisions table.
const DefaultRevisionSchema = "atlas_schema_revisions"

type (
	// A EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
	EntRevisions struct {
		ac     *sqlclient.Client  // underlying Atlas client
		ec     *ent.Client        // underlying Ent client
		schema string             // name of the schema the revision table resides in
		cache  []migrate.Revision // cache stores writes to Ent for blocked connections (like in SQLite).
	}

	// Option allows to configure EntRevisions by using functional arguments.
	Option func(*EntRevisions) error
)

// NewEntRevisions creates a new EntRevisions with the given sqlclient.Client. It is important to call
// EntRevisions.Init to initialize the underlying Ent client.
func NewEntRevisions(ac *sqlclient.Client, opts ...Option) (*EntRevisions, error) {
	r := &EntRevisions{ac: ac}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	if r.schema == "" {
		r.schema = DefaultRevisionSchema
	}
	return r, nil
}

// WithSchema configures the schema to use for the revision table.
func WithSchema(s string) Option {
	return func(r *EntRevisions) error {
		r.schema = s
		return nil
	}
}

// Ident returns the table identifier.
func (r *EntRevisions) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{Name: revision.Table, Schema: r.schema}
}

// Init runs migration for the revisions' table in the connected database.
func (r *EntRevisions) Init(ctx context.Context) error {
	// Try to open a connection to the schema we are storing the revision table in.
	sc, err := r.openSchema(ctx)
	// If the driver does not support changing the schema (SQLite) use the existing connection.
	if err != nil && errors.Is(err, sqlclient.ErrUnsupported) {
		r.ec = ent.NewClient(ent.Driver(sql.OpenDB(r.ac.Name, r.ac.DB)))
		return r.ec.Schema.Create(ctx)
	}
	// Driver does support changing schemas. Make sure the schema does exist before proceeding.
	_, err = r.ac.InspectSchema(ctx, r.schema, &schema.InspectOptions{Mode: schema.InspectSchemas})
	if err != nil && !schema.IsNotExistError(err) {
		return err
	}
	if schema.IsNotExistError(err) {
		if err := r.ac.ApplyChanges(ctx, []schema.Change{
			&schema.AddSchema{S: &schema.Schema{Name: r.schema}},
		}); err != nil {
			return err
		}
	}
	// If the previous connection attempt was unsuccessful, re-try with the schema present.
	if sc == nil {
		sc, err = r.openSchema(ctx)
		if err != nil {
			return err
		}
	}
	r.ac.AddClosers(sc)
	r.ec = ent.NewClient(ent.Driver(sql.OpenDB(sc.Name, sc.DB)))
	return r.ec.Schema.Create(ctx, entschema.WithDropColumn(true))
}

// ReadRevision reads a revision from the revisions table.
//
// ReadRevision will not return results only saved in cache.
func (r *EntRevisions) ReadRevision(ctx context.Context, v string) (*migrate.Revision, error) {
	rev, err := r.ec.Revision.Get(ctx, v)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if ent.IsNotFound(err) {
		return nil, migrate.ErrRevisionNotExist
	}
	return fromEnt(rev), nil
}

// ReadRevisions reads the revisions from the revisions table.
//
// ReadRevisions will not return results only saved to cache.
func (r *EntRevisions) ReadRevisions(ctx context.Context) (migrate.Revisions, error) {
	revs, err := r.ec.Revision.Query().Order(ent.Asc(revision.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	ret := make(migrate.Revisions, len(revs))
	for i, rev := range revs {
		ret[i] = fromEnt(rev)
	}
	return ret, nil
}

// WriteRevision writes a revision to the revisions table.
func (r *EntRevisions) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	if r.useCache() {
		// Do not store the pointer since we want to maintain the order for writes to the database.
		r.cache = append(r.cache, *rev)
		return nil
	}
	return r.write(ctx, rev)
}

// Flush writes the changes saved in memory to the database.
//
// This method exists to support both execution of migration in a transaction and saving revision for SQLite flavors,
// since attempting to write to the database while in a transaction will fail there.
func (r *EntRevisions) Flush(ctx context.Context) error {
	if !r.useCache() {
		return nil
	}
	for i := range r.cache {
		if err := r.write(ctx, &r.cache[i]); err != nil {
			return err
		}
	}
	return nil
}

// write attempts to write the given revision to the database.
func (r *EntRevisions) write(ctx context.Context, rev *migrate.Revision) error {
	return r.ec.Revision.Create().
		SetID(rev.Version).
		SetDescription(rev.Description).
		SetType(rev.Type).
		SetApplied(rev.Applied).
		SetTotal(rev.Total).
		SetExecutedAt(rev.ExecutedAt).
		SetExecutionTime(rev.ExecutionTime).
		SetError(rev.Error).
		SetHash(rev.Hash).
		SetPartialHashes(rev.PartialHashes).
		SetOperatorVersion(rev.OperatorVersion).
		OnConflict(sql.ConflictColumns(revision.FieldID)).
		UpdateNewValues().
		Exec(ctx)
}

func fromEnt(r *ent.Revision) *migrate.Revision {
	return &migrate.Revision{
		Version:         r.ID,
		Description:     r.Description,
		Type:            r.Type,
		Applied:         r.Applied,
		Total:           r.Total,
		ExecutedAt:      r.ExecutedAt,
		ExecutionTime:   r.ExecutionTime,
		Error:           r.Error,
		Hash:            r.Hash,
		PartialHashes:   r.PartialHashes,
		OperatorVersion: r.OperatorVersion,
	}
}

func (r *EntRevisions) useCache() bool {
	// For SQLite dialect and flavors we have to enable the revision write cache to postpone writing to
	// the database until the transaction wrapping the migration execution has been committed.
	return r.ac.Name == dialect.SQLite
}

func (r *EntRevisions) openSchema(ctx context.Context) (*sqlclient.Client, error) {
	u := r.ac.URL.URL
	if r.ac.Name == dialect.MySQL && !u.Query().Has("parseTime") {
		v := *u
		q := v.Query()
		q.Set("parseTime", "true")
		v.RawQuery = q.Encode()
		u = &v
	}
	return sqlclient.OpenURL(ctx, u, sqlclient.OpenSchema(r.schema))
}

var _ migrate.RevisionReadWriter = (*EntRevisions)(nil)
