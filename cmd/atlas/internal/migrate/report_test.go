// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestReporter_Status(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)

	// Clean.
	dir, err := migrate.NewLocalDir(filepath.Join("../migrate/testdata", "broken"))
	require.NoError(t, err)
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	rr := &StatusReporter{Client: c, Dir: dir}
	report, err := rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
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
	rr = &StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
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
	rr = &StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
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
	rr = &StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
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
	dir2, err := migrate.NewLocalDir(filepath.Join("../migrate/testdata", "fixed"))
	require.NoError(t, err)
	*dir = *dir2
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	rr = &StatusReporter{Client: c, Dir: dir}
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: OK
  -- Current Version: 3
  -- Next Version:    Already at latest version
  -- Executed Files:  3
  -- Pending Files:   0
`, buf.String())
}

func TestReporter_FromCheckpoint(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)
	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, dir.WriteFile("1.sql", []byte("create table t1(c int);")))
	require.NoError(t, dir.WriteFile("2.sql", []byte("create table t2(c int);")))
	require.NoError(t, dir.WriteCheckpoint("3_checkpoint.sql", "", []byte("create table t1(c int);\ncreate table t2(c int);")))
	sum, err := dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	require.NoError(t, migrate.Validate(dir))
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	rr := &StatusReporter{Client: c, Dir: dir}

	// Clean.
	report, err := rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    3 (checkpoint)
  -- Executed Files:  0
  -- Pending Files:   1
`, buf.String())

	// Clean and revisions table exists.
	buf.Reset()
	rrw, err := NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    3 (checkpoint)
  -- Executed Files:  0
  -- Pending Files:   1
`, buf.String())

	// Execute one.
	buf.Reset()
	ex, err := migrate.NewExecutor(c.Driver, dir, rrw)
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 1))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: OK
  -- Current Version: 3
  -- Next Version:    Already at latest version
  -- Executed Files:  1
  -- Pending Files:   0
`, buf.String())

	// Add a new file.
	buf.Reset()
	require.NoError(t, dir.WriteFile("4.sql", []byte("create table t3(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	require.NoError(t, migrate.Validate(dir))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 3
  -- Next Version:    4
  -- Executed Files:  1
  -- Pending Files:   1
`, buf.String())

	// Execute one.
	buf.Reset()
	require.NoError(t, ex.ExecuteN(ctx, 1))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: OK
  -- Current Version: 4
  -- Next Version:    Already at latest version
  -- Executed Files:  2
  -- Pending Files:   0
`, buf.String())
}

func TestReporter_OutOfOrder(t *testing.T) {
	var (
		buf strings.Builder
		ctx = context.Background()
	)
	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, dir.WriteFile("1.sql", []byte("create table t1(c int);")))
	require.NoError(t, dir.WriteFile("2.sql", []byte("create table t2(c int);")))
	sum, err := dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	c, err := sqlclient.Open(ctx, "sqlite://?mode=memory")
	require.NoError(t, err)
	defer c.Close()
	rr := &StatusReporter{Client: c, Dir: dir}

	rrw, err := NewEntRevisions(ctx, c)
	require.NoError(t, err)
	require.NoError(t, rrw.Migrate(ctx))
	report, err := rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: No migration applied yet
  -- Next Version:    1
  -- Executed Files:  0
  -- Pending Files:   2
`, buf.String())

	ex, err := migrate.NewExecutor(c.Driver, dir, rrw)
	require.NoError(t, err)
	require.NoError(t, ex.ExecuteN(ctx, 2))

	// One file was added out of order.
	buf.Reset()
	require.NoError(t, dir.WriteFile("1.5.sql", []byte("create table t1_5(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   1 (out of order)

  ERROR: migration file 1.5.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())

	// Multiple files were added our of order.
	buf.Reset()
	require.NoError(t, dir.WriteFile("1.6.sql", []byte("create table t1_6(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   2 (out of order)

  ERROR: migration files 1.5.sql, 1.6.sql were added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())

	// A mix of pending and out of order files.
	buf.Reset()
	require.NoError(t, dir.WriteFile("3.sql", []byte("create table t3(c int);")))
	sum, err = dir.Checksum()
	require.NoError(t, err)
	require.NoError(t, migrate.WriteSumFile(dir, sum))
	report, err = rr.Report(ctx)
	require.NoError(t, err)
	require.NoError(t, cmdlog.MigrateStatusTemplate.Execute(&buf, report))
	require.Equal(t, `Migration Status: PENDING
  -- Current Version: 2
  -- Next Version:    UNKNOWN
  -- Executed Files:  2
  -- Pending Files:   3 (2 out of order)

  ERROR: migration files 1.5.sql, 1.6.sql were added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error
`, buf.String())
}
