// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/migrate/ent"
	"ariga.io/atlas/sql/migrate/ent/revision"
	"ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect/sql"
)

// A EntRevisions provides implementation for the migrate.RevisionReadWriter interface.
type EntRevisions struct {
	c *ent.Client
	// sc is the function signature of the Ent migration engine.
	// Due to cyclic dependencies between Ent and Atlas, this value is stitched in at runtime.
	sc func(context.Context) error
}

// NewEntRevisions creates a new EntRevisions with the given ent.Client.
func NewEntRevisions(db schema.ExecQuerier, dialect string) *EntRevisions {
	return &EntRevisions{c: ent.NewClient(ent.Driver(sql.NewDriver(sql.Conn{ExecQuerier: db}, dialect)))}
}

// InitSchemaMigrator stitches in the Ent migration engine to the EntRevisions at runtime. This is necessary
// because the Ent migration engine imports atlas and therefore would introduce a cyclic dependency.
func (r *EntRevisions) InitSchemaMigrator(sc func(context.Context) error) {
	r.sc = sc
}

// Init makes sure the revisions table does exist in the connected database.
func (r *EntRevisions) Init(ctx context.Context) error {
	return r.sc(ctx)
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
