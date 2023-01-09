// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"testing"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestNewEntRevisions(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	r, err := NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, r.Migrate(ctx))

	s, err := c.Driver.InspectSchema(ctx, "", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	_, ok := s.Table(revision.Table)
	require.True(t, ok)

	cur, err := r.CurrentRevision(ctx)
	require.True(t, ent.IsNotFound(err))
	require.Nil(t, cur)

	err = r.WriteRevision(ctx, &migrate.Revision{
		Version:         "1",
		Description:     "desc",
		Type:            migrate.RevisionTypeResolved,
		ExecutedAt:      time.Now(),
		Hash:            "hash",
		OperatorVersion: "0.1.0",
	})
	require.NoError(t, err)
	cur, err = r.CurrentRevision(ctx)
	require.NoError(t, err)
	require.Equal(t, "1", cur.Version)

	next := *cur
	next.Version = "2"
	require.NoError(t, r.WriteRevision(ctx, &next))
	cur, err = r.CurrentRevision(ctx)
	require.NoError(t, err)
	require.Equal(t, "2", cur.Version)

	revs, err := r.ReadRevisions(ctx)
	require.NoError(t, err)
	require.Len(t, revs, 2)
	require.Equal(t, "1", revs[0].Version)
	require.Equal(t, "2", revs[1].Version)
}
