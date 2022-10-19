// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	"github.com/stretchr/testify/require"
)

func TestReporter_Status(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)

	// Clean.
	dir, err := migrate.NewLocalDir(filepath.Join("testdata", "broken"))
	require.NoError(t, err)
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, (&StatusReporter{
		Client:       c,
		Dir:          dir,
		ReportWriter: &TemplateWriter{T: DefaultStatusTemplate, W: &buf},
	}).Report(ctx))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    1
  -- Executed Files:  0
  -- Pending Files:   3
`, buf.String())

	// Applied one.
	buf.Reset()
	rrw, err := NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	ex, err := migrate.NewExecutor(c.Driver, dir, rrw)
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	require.NoError(t, (&StatusReporter{
		Client:       c,
		Dir:          dir,
		ReportWriter: &TemplateWriter{T: DefaultStatusTemplate, W: &buf},
	}).Report(ctx))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 1
  -- Next Version:    2
  -- Executed Files:  1
  -- Pending Files:   2
`, buf.String())

	// Applied two.
	buf.Reset()
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	require.NoError(t, (&StatusReporter{
		Client:       c,
		Dir:          dir,
		ReportWriter: &TemplateWriter{T: DefaultStatusTemplate, W: &buf},
	}).Report(ctx))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    3
  -- Executed Files:  2
  -- Pending Files:   1
`, buf.String())

	// Partial three.
	buf.Reset()
	require.NoError(t, err)
	require.Error(t, ex.ExecuteN(ctx, 1))
	require.NoError(t, (&StatusReporter{
		Client:       c,
		Dir:          dir,
		ReportWriter: &TemplateWriter{T: DefaultStatusTemplate, W: &buf},
	}).Report(ctx))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 3 (1 statements applied)
  -- Next Version:    3 (1 statements left)
  -- Executed Files:  3 (last one partially)
  -- Pending Files:   1

Last migration attempt had errors:
  -- SQL:   THIS LINE ADDS A SYNTAX ERROR;
  -- ERROR: near "THIS": syntax error
`, buf.String())

	// Fixed three - okay.
	buf.Reset()
	dir2, err := migrate.NewLocalDir(filepath.Join("testdata", "fixed"))
	require.NoError(t, err)
	*dir = *dir2
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	require.NoError(t, (&StatusReporter{
		Client:       c,
		Dir:          dir,
		ReportWriter: &TemplateWriter{T: DefaultStatusTemplate, W: &buf},
	}).Report(ctx))
	require.Equal(t, `Migration Status: OK
  -- Current Version: 3
  -- Next Version:    Already at latest version
  -- Executed Files:  3
  -- Pending Files:   0
`, buf.String())
}
