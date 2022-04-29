// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/migrate/ent"
	"ariga.io/atlas/sql/migrate/ent/revision"
	"entgo.io/ent/dialect/sql"
)

// A RevisionStorage provides implementation for the migrate.RevisionReadWriter interface.
type RevisionStorage struct {
	c *ent.Client
	// sc is the function signature of the Ent migration engine.
	// Due to cyclic dependencies between Ent and Atlas, this value is stitched in at runtime.
	sc func(context.Context) error
}

// NewRevisionStorage creates a new RevisionStorage with the given ent.Client.
func NewRevisionStorage(c *ent.Client) *RevisionStorage {
	return &RevisionStorage{c: c}
}

// InitSchemaMigrator stitches in the Ent migration engine to the mysql.Driver at runtime. This is necessary
// because the Ent migration engine imports atlas and therefore would introduce a cyclic dependency.
func (r *RevisionStorage) InitSchemaMigrator(sc func(context.Context) error) {
	r.sc = sc
}

// Init makes sure the revisions table does exist in the connected database.
func (r *RevisionStorage) Init(ctx context.Context) error {
	return r.sc(ctx)
}

// ReadRevisions read the revisions from the revisions table.
func (r *RevisionStorage) ReadRevisions(ctx context.Context) (migrate.Revisions, error) {
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

// WriteRevisions stores the revisions in the revisions table.
func (r *RevisionStorage) WriteRevisions(ctx context.Context, rs migrate.Revisions) error {
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
	return r.c.Revision.CreateBulk(bulk...).OnConflict().UpdateNewValues().Exec(ctx)
}

var _ migrate.RevisionReadWriter = (*RevisionStorage)(nil)
