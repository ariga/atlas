// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"database/sql"
	"testing"

	"ariga.io/atlas/cmd/atlascmd/migrate/ent/revision"
	"ariga.io/atlas/sql/sqlite"
	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestEntRevisions_Init(t *testing.T) {
	db, err := sql.Open(dialect.SQLite, "file:revisions?cache=shared&mode=memory&_fk=true")
	require.NoError(t, err)
	r := NewEntRevisions(db, dialect.SQLite)

	require.NoError(t, r.Init(context.Background()))

	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	s, err := drv.InspectSchema(context.Background(), "main", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	_, ok := s.Table(revision.Table)
	require.True(t, ok)
}
