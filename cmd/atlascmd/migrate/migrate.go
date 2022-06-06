// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"

	"ariga.io/atlas/cmd/atlascmd/migrate/ent"
	"ariga.io/atlas/cmd/atlascmd/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
)

type (
	// A EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
	EntRevisions struct {
		ac           *sqlclient.Client  // underlying Atlas client
		sc           *sqlclient.Client  // underlying Atlas client connected to the named schema
		ec           *ent.Client        // underlying Ent client
		schema       string             // name of the schema the revision table resides in
		cache        []migrate.Revision // cache stores writes to Ent for blocked connections (like in SQLite).
		cacheEnabled bool               // if to store writes in the cache and only write on a call to Flush.
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
		r.schema = "atlas_schema_revisions"
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

// WithCache enables the cache (e.g. for SQLite).
func WithCache() Option {
	return func(r *EntRevisions) error {
		r.cacheEnabled = true
		return nil
	}
}

// Init makes sure the revision table does exist in the connected database.
func (r *EntRevisions) Init(ctx context.Context) error {
	// Try to open a connection to the schema we are storing the revision table in.
	var err error
	r.sc, err = sqlclient.OpenURL(ctx, r.ac.URL.URL, sqlclient.OpenSchema(r.schema))
	// If the driver does not support changing the schema (most likely SQLite) use the existing connection.
	if err != nil && errors.Is(err, sqlclient.ErrUnsupported) {
		r.ec = ent.NewClient(ent.Driver(sql.OpenDB(r.ac.Name, r.ac.DB)))
		return r.ec.Schema.Create(ctx, entschema.WithAtlas(true))
	}
	// Driver does support changing schemas. Make sure the schema does exist before proceeding.
	_, err2 := r.ac.InspectSchema(ctx, r.schema, &schema.InspectOptions{Mode: schema.InspectSchemas})
	if err2 != nil && !schema.IsNotExistError(err2) {
		return err2
	}
	if schema.IsNotExistError(err2) {
		if err := r.ac.ApplyChanges(ctx, []schema.Change{
			&schema.AddSchema{S: &schema.Schema{Name: r.schema}},
		}); err != nil {
			return err
		}
	}
	// If the previous connection attempt was unsuccessful, re-try with the schema present.
	if r.sc == nil {
		r.sc, err = sqlclient.OpenURL(ctx, r.ac.URL.URL, sqlclient.OpenSchema(r.schema))
		if err != nil {
			return err
		}
	}
	r.ac.AddClosers(r.sc)
	r.ec = ent.NewClient(ent.Driver(sql.OpenDB(r.sc.Name, r.sc.DB)))
	return r.ec.Schema.Create(ctx, entschema.WithAtlas(true))
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
	for i, r := range revs {
		ret[i] = &migrate.Revision{
			Version:         r.ID,
			Description:     r.Description,
			ExecutionState:  string(r.ExecutionState),
			ExecutedAt:      r.ExecutedAt,
			ExecutionTime:   r.ExecutionTime,
			Hash:            r.Hash,
			OperatorVersion: r.OperatorVersion,
			Meta:            r.Meta,
		}
	}
	return ret, nil
}

// WriteRevision writes a revision to the revisions table.
func (r *EntRevisions) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	if r.cacheEnabled {
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
	if !r.cacheEnabled {
		return nil
	}
	for _, rev := range r.cache {
		if err := r.write(ctx, &rev); err != nil {
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
		SetExecutionState(revision.ExecutionState(rev.ExecutionState)).
		SetExecutedAt(rev.ExecutedAt).
		SetExecutionTime(rev.ExecutionTime).
		SetHash(rev.Hash).
		SetOperatorVersion(rev.OperatorVersion).
		SetMeta(rev.Meta).
		OnConflict(sql.ConflictColumns(revision.FieldID)).
		UpdateNewValues().
		Exec(ctx)
}

var _ migrate.RevisionReadWriter = (*EntRevisions)(nil)
