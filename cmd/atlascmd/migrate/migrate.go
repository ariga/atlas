// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"

	"ariga.io/atlas/cmd/atlascmd/migrate/ent"
	"ariga.io/atlas/cmd/atlascmd/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
)

// A EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
type EntRevisions struct{ c *ent.Client }

// NewEntRevisions creates a new EntRevisions with the given ent.Client.
func NewEntRevisions(db schema.ExecQuerier, dialect string, opts ...ent.Option) *EntRevisions {
	opts = append(opts, ent.Driver(sql.NewDriver(dialect, sql.Conn{ExecQuerier: db})))
	return &EntRevisions{c: ent.NewClient(opts...)}
}

// Init makes sure the revisions table does exist in the connected database.
func (r *EntRevisions) Init(ctx context.Context) error {
	return r.c.Schema.Create(ctx, entschema.WithAtlas(true))
}

// ReadRevisions reads the revisions from the revisions table.
func (r *EntRevisions) ReadRevisions(ctx context.Context) (migrate.Revisions, error) {
	revs, err := r.c.Revision.Query().Order(ent.Asc(revision.FieldID)).All(ctx)
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

// WriteRevisions writes the revisions to the revisions table.
func (r *EntRevisions) WriteRevisions(ctx context.Context, rs migrate.Revisions) error {
	bulk := make([]*ent.RevisionCreate, len(rs))
	for i, rev := range rs {
		bulk[i] = r.c.Revision.Create().
			SetID(rev.Version).
			SetDescription(rev.Description).
			SetExecutionState(revision.ExecutionState(rev.ExecutionState)).
			SetExecutedAt(rev.ExecutedAt).
			SetExecutionTime(rev.ExecutionTime).
			SetHash(rev.Hash).
			SetOperatorVersion(rev.OperatorVersion).
			SetMeta(rev.Meta)
	}
	return r.c.Revision.CreateBulk(bulk...).
		OnConflict(
			sql.ConflictColumns(revision.FieldID),
		).
		UpdateNewValues().
		Exec(ctx)
}

var _ migrate.RevisionReadWriter = (*EntRevisions)(nil)
