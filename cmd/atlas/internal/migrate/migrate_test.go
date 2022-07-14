// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestEntRevisions_Init(t *testing.T) {
	c, err := sqlclient.Open(
		context.Background(),
		fmt.Sprintf("sqlite://%s?cache=shared&mode=memory&_fk=true", filepath.Join(t.TempDir(), "revision")),
	)
	require.NoError(t, err)
	r, err := NewEntRevisions(c)
	require.NoError(t, err)

	require.NoError(t, r.Init(context.Background()))

	s, err := c.Driver.InspectSchema(context.Background(), "", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	_, ok := s.Table(revision.Table)
	require.True(t, ok)
}

func TestEntRevisions_Flush(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(
		context.Background(),
		fmt.Sprintf("sqlite://%s?cache=shared&mode=memory&_fk=true", filepath.Join(t.TempDir(), "revision")),
	)
	require.NoError(t, err)
	r, err := NewEntRevisions(c)
	require.NoError(t, err)
	require.True(t, r.useCache())
	require.NoError(t, r.Init(ctx))

	// Writing will only fill the cache.
	require.NoError(t, r.WriteRevision(ctx, &migrate.Revision{
		Version:    "version",
		ExecutedAt: time.Now(),
		Applied:    1,
		Total:      2,
	}))
	require.Len(t, r.cache, 1)
	revs, err := r.ReadRevisions(ctx)
	require.NoError(t, err)
	require.Len(t, revs, 0)

	// Flushing will save the cached revision.
	require.NoError(t, r.Flush(ctx))
	revs, err = r.ReadRevisions(ctx)
	require.NoError(t, err)
	require.Len(t, revs, 1)
}
