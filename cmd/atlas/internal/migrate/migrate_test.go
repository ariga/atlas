// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/sqlclient"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestNewEntRevisions(t *testing.T) {
	c, err := sqlclient.Open(
		context.Background(),
		fmt.Sprintf("sqlite://%s?cache=shared&mode=memory&_fk=true", filepath.Join(t.TempDir(), "revision")),
	)
	require.NoError(t, err)
	_, err = NewEntRevisions(context.Background(), &Config{drv: c.Driver, db: c.DB, dialect: c.Name})
	require.NoError(t, err)

	s, err := c.Driver.InspectSchema(context.Background(), "", nil)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)
	_, ok := s.Table(revision.Table)
	require.True(t, ok)
}
