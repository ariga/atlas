// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	sch "ariga.io/atlas/cmd/atlas/internal/migrate/ent/schema"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
)

type (
	// EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
	EntRevisions struct {
		ac     *sqlclient.Client // underlying Atlas client
		ec     *ent.Client       // underlying Ent client
		schema string            // name of the schema the revision table resides in
	}

	// Option allows to configure EntRevisions by using functional arguments.
	Option func(*EntRevisions) error
)

// NewEntRevisions creates a new EntRevisions with the given sqlclient.Client.
func NewEntRevisions(ctx context.Context, ac *sqlclient.Client, opts ...Option) (*EntRevisions, error) {
	r := &EntRevisions{ac: ac}
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	if r.schema == "" {
		r.schema = sch.DefaultRevisionSchema
	}
	// Create the connection with the underlying migrate.Driver to have it inside a possible transaction.
	entopts := []ent.Option{ent.Driver(sql.NewDriver(r.ac.Name, sql.Conn{ExecQuerier: r.ac.Driver}))}
	// SQLite does not support multiple schema, therefore schema-config is only needed for other dialects.
	if r.ac.Name != dialect.SQLite {
		// Make sure the schema to store the revisions table in does exist.
		_, err := r.ac.InspectSchema(ctx, r.schema, &schema.InspectOptions{Mode: schema.InspectSchemas})
		if err != nil && !schema.IsNotExistError(err) {
			return nil, err
		}
		if schema.IsNotExistError(err) {
			if err := r.ac.ApplyChanges(ctx, []schema.Change{
				&schema.AddSchema{S: &schema.Schema{Name: r.schema}},
			}); err != nil {
				return nil, err
			}
		}
		// Tell Ent to operate on that schema.
		entopts = append(entopts, ent.AlternateSchema(ent.SchemaConfig{Revision: r.schema}))
	}
	// Instantiate the Ent client and migrate the revision schema.
	r.ec = ent.NewClient(entopts...)
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
	return rev.AtlasRevision(), nil
}

// ReadRevisions reads the revisions from the revisions table.
//
// ReadRevisions will not return results only saved to cache.
func (r *EntRevisions) ReadRevisions(ctx context.Context) ([]*migrate.Revision, error) {
	revs, err := r.ec.Revision.Query().Order(ent.Asc(revision.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]*migrate.Revision, len(revs))
	for i, rev := range revs {
		ret[i] = rev.AtlasRevision()
	}
	return ret, nil
}

// WriteRevision writes a revision to the revisions table.
func (r *EntRevisions) WriteRevision(ctx context.Context, rev *migrate.Revision) error {
	return r.ec.Revision.Create().
		SetRevision(rev).
		OnConflict(sql.ConflictColumns(revision.FieldID)).
		UpdateNewValues().
		Exec(ctx)
}

// Migrate attempts to create / update the revisions table. This is separated since Ent attempts to wrap the migration
// execution in a transaction and assumes the underlying connection is of type *sql.DB, which is not true for actually
// reading and writing revisions.
func (r *EntRevisions) Migrate(ctx context.Context) error {
	return ent.NewClient(ent.Driver(sql.OpenDB(r.ac.Name, r.ac.DB))).Schema.Create(ctx,
		entschema.WithDropColumn(true),
		entschema.WithDiffHook(func(next entschema.Differ) entschema.Differ {
			return entschema.DiffFunc(func(current, desired *schema.Schema) ([]schema.Change, error) {
				t, ok := desired.Table(revision.Table)
				if !ok {
					return nil, errors.New("revisions table not found in desired state")
				}
				t.SetSchema(schema.New(r.schema))
				return next.Diff(current, desired)
			})
		}),
	)
}

var _ migrate.RevisionReadWriter = (*EntRevisions)(nil)
