// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestRevisionType_MarshalText(t *testing.T) {
	for _, tt := range []struct {
		r  migrate.RevisionType
		ex string
	}{
		{migrate.RevisionTypeUnknown, "unknown (0000)"},
		{migrate.RevisionTypeBaseline, "baseline"},
		{migrate.RevisionTypeExecute, "applied"},
		{migrate.RevisionTypeResolved, "manually set"},
		{migrate.RevisionTypeExecute | migrate.RevisionTypeResolved, "applied + manually set"},
		{migrate.RevisionTypeExecute | migrate.RevisionTypeBaseline, "unknown (0011)"},
		{1 << 3, "unknown (1000)"},
	} {
		ac, err := tt.r.MarshalText()
		require.NoError(t, err)
		require.Equal(t, tt.ex, string(ac))
	}
}

func TestPlanner_WritePlan(t *testing.T) {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	plan := &migrate.Plan{
		Name: "add_t1_and_t2",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int)", Reverse: "DROP TABLE t1 IF EXISTS"},
			{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2"},
		},
	}

	// DefaultFormatter
	pl := migrate.NewPlanner(nil, d, migrate.PlanWithChecksum(false))
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	v := time.Now().UTC().Format("20060102150405")
	require.Equal(t, countFiles(t, d), 1)
	requireFileEqual(t, d, v+"_add_t1_and_t2.sql", "CREATE TABLE t1(c int);\nCREATE TABLE t2(c int);\n")

	// Custom formatter (creates "up" and "down" migration files).
	fmt, err := migrate.NewTemplateFormatter(
		template.Must(template.New("").Parse("{{ .Name }}.up.sql")),
		template.Must(template.New("").Parse("{{ range .Changes }}{{ println .Cmd }}{{ end }}")),
		template.Must(template.New("").Parse("{{ .Name }}.down.sql")),
		template.Must(template.New("").Parse("{{ range .Changes }}{{ println .Reverse }}{{ end }}")),
	)
	require.NoError(t, err)
	pl = migrate.NewPlanner(nil, d, migrate.PlanFormat(fmt), migrate.PlanWithChecksum(false))
	require.NotNil(t, pl)
	require.NoError(t, pl.WritePlan(plan))
	require.Equal(t, countFiles(t, d), 3)
	requireFileEqual(t, d, "add_t1_and_t2.up.sql", "CREATE TABLE t1(c int)\nCREATE TABLE t2(c int)\n")
	requireFileEqual(t, d, "add_t1_and_t2.down.sql", "DROP TABLE t1 IF EXISTS\nDROP TABLE t2\n")
}

func TestPlanner_WriteCheckpoint(t *testing.T) {
	p := t.TempDir()
	d, err := migrate.NewLocalDir(p)
	require.NoError(t, err)
	plan := &migrate.Plan{
		Name: "checkpoint",
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int)", Reverse: "DROP TABLE t1 IF EXISTS"},
			{Cmd: "CREATE TABLE t2(c int)", Reverse: "DROP TABLE t2"},
		},
	}

	// DefaultFormatter
	pl := migrate.NewPlanner(nil, d)
	require.NotNil(t, pl)
	require.NoError(t, pl.WriteCheckpoint(plan, "v1"))
	files, err := d.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, `-- atlas:checkpoint v1

CREATE TABLE t1(c int);
CREATE TABLE t2(c int);
`, string(files[0].Bytes()))
}

func TestPlanner_Plan(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)

	// nothing to do
	pl := migrate.NewPlanner(drv, d)
	plan, err := pl.Plan(ctx, "empty", migrate.Realm(nil))
	require.ErrorIs(t, err, migrate.ErrNoPlan)
	require.Nil(t, plan)

	// there are changes
	drv.changes = []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c", "int"))},
		&schema.AddTable{T: schema.NewTable("t2").AddColumns(schema.NewIntColumn("c", "int"))},
	}
	drv.plan = &migrate.Plan{
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);"},
			{Cmd: "CREATE TABLE t2(c int);"},
		},
	}
	plan, err = pl.Plan(ctx, "", migrate.Realm(nil))
	require.NoError(t, err)
	require.Equal(t, drv.plan, plan)
}

func TestPlanner_PlanSchema(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)

	// Schema is missing in dev connection.
	pl := migrate.NewPlanner(drv, d)
	plan, err := pl.PlanSchema(ctx, "empty", migrate.Realm(nil))
	require.EqualError(t, err, `not found`)
	require.Nil(t, plan)

	drv.realm = *schema.NewRealm(schema.New("test"))
	pl = migrate.NewPlanner(drv, d)
	plan, err = pl.PlanSchema(ctx, "empty", migrate.Realm(schema.NewRealm()))
	require.EqualError(t, err, `no schema was found in desired state`)
	require.Nil(t, plan)

	drv.realm = *schema.NewRealm(schema.New("test"))
	pl = migrate.NewPlanner(drv, d)
	plan, err = pl.PlanSchema(ctx, "empty", migrate.Realm(schema.NewRealm(schema.New("test"), schema.New("dev"))))
	require.EqualError(t, err, `2 schemas were found in desired state; expect 1`)
	require.Nil(t, plan)

	drv.realm = *schema.NewRealm(schema.New("test"))
	pl = migrate.NewPlanner(drv, d)
	plan, err = pl.PlanSchema(ctx, "multi", migrate.Realm(schema.NewRealm(schema.New("test"))))
	require.ErrorIs(t, err, migrate.ErrNoPlan)
	require.Nil(t, plan)
}

func TestPlanner_Checkpoint(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)

	// Nothing to do.
	pl := migrate.NewPlanner(drv, d)
	plan, err := pl.Checkpoint(ctx, "empty")
	require.NoError(t, err)
	require.Equal(t, &migrate.Plan{Name: "empty"}, plan)

	// There are changes.
	drv.changes = []schema.Change{
		&schema.AddTable{T: schema.NewTable("t1").AddColumns(schema.NewIntColumn("c", "int"))},
		&schema.AddTable{T: schema.NewTable("t2").AddColumns(schema.NewIntColumn("c", "int"))},
	}
	drv.plan = &migrate.Plan{
		Changes: []*migrate.Change{
			{Cmd: "CREATE TABLE t1(c int);"},
			{Cmd: "CREATE TABLE t2(c int);"},
		},
	}
	plan, err = pl.Checkpoint(ctx, "checkpoint")
	require.NoError(t, err)
	require.Equal(t, drv.plan, plan)
}

func TestPlanner_CheckpointSchema(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
	)
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)

	// Schema is missing in dev connection.
	pl := migrate.NewPlanner(drv, d)
	plan, err := pl.CheckpointSchema(ctx, "empty")
	require.EqualError(t, err, `not found`)
	require.Nil(t, plan)

	drv.realm = *schema.NewRealm(schema.New("test"))
	pl = migrate.NewPlanner(drv, d)
	plan, err = pl.CheckpointSchema(ctx, "empty")
	require.Equal(t, &migrate.Plan{Name: "empty"}, plan)
}

func TestExecutor_Replay(t *testing.T) {
	ctx := context.Background()
	d, err := migrate.NewLocalDir(filepath.FromSlash("testdata/migrate"))
	require.NoError(t, err)

	drv := &mockDriver{}
	ex, err := migrate.NewExecutor(drv, d, migrate.NopRevisionReadWriter{})
	require.NoError(t, err)

	// No options.
	_, err = ex.Replay(ctx, migrate.RealmConn(drv, nil))
	require.NoError(t, err)
	require.Equal(t, []string{"DROP TABLE IF EXISTS t;", "CREATE TABLE t(c int);"}, drv.executed)

	// ReplayVersion option.
	*drv = mockDriver{}
	dir, err := migrate.NewLocalDir(filepath.FromSlash("testdata/migrate/sub"))
	require.NoError(t, err)
	*d = *dir
	_, err = ex.Replay(ctx, migrate.RealmConn(drv, nil), migrate.ReplayToVersion("0"))
	require.ErrorContains(t, err, "migration with version \"0\" not found")
	require.Zero(t, drv.executed)

	*drv = mockDriver{}
	_, err = ex.Replay(ctx, migrate.RealmConn(drv, nil), migrate.ReplayToVersion("1.a"))
	require.NoError(t, err)
	require.Equal(t, []string{"CREATE TABLE t_sub(c int);", "ALTER TABLE t_sub ADD c1 int;"}, drv.executed)

	*drv = mockDriver{}
	_, err = ex.Replay(ctx, migrate.RealmConn(drv, nil), migrate.ReplayToVersion("3"))
	require.NoError(t, err)
	require.Equal(t, []string{
		"CREATE TABLE t_sub(c int);",
		"ALTER TABLE t_sub ADD c1 int;",
		"ALTER TABLE t_sub ADD c2 int;",
		"ALTER TABLE t_sub ADD c3 int;",
		"ALTER TABLE t_sub ADD c4 int;",
	}, drv.executed)

	// Does not work if database is not clean.
	drv.dirty = true
	drv.realm = schema.Realm{Schemas: []*schema.Schema{{Name: "schema"}}}
	_, err = ex.Replay(ctx, migrate.RealmConn(drv, nil))
	require.ErrorAs(t, err, new(*migrate.NotCleanError))
}

func TestExecutor_Pending(t *testing.T) {
	var (
		ctx  = context.Background()
		drv  = &mockDriver{}
		rrw  = &mockRevisionReadWriter{}
		log  = &mockLogger{}
		rev1 = &migrate.Revision{
			Version:     "1.a",
			Description: "sub.up",
			Applied:     2,
			Total:       2,
			Hash:        "nXyZR020M/mH7LxkoTkJr7BcQkipVg90imQ9I4595dw=",
		}
		rev2 = &migrate.Revision{
			Version:     "2.10.x-20",
			Description: "description",
			Applied:     1,
			Total:       1,
			Hash:        "wQB3Vh3PHVXQg9OD3Gn7TBxbZN3r1Qb7TtAE1g3q9mQ=",
		}
		rev3 = &migrate.Revision{
			Version:     "3",
			Description: "partly",
			Applied:     1,
			Total:       2,
			Error:       "this is an migration error",
			Hash:        "+O40cAXHgvMClnynHd5wggPAeZAk7zSEaNgzXCZOfmY=",
		}
	)
	dir, err := migrate.NewLocalDir(filepath.Join("testdata", "migrate", "sub"))
	require.NoError(t, err)
	ex, err := migrate.NewExecutor(drv, dir, rrw, migrate.WithLogger(log))
	require.NoError(t, err)

	// All are pending
	p, err := ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 3)

	// 2 are pending.
	*rrw = []*migrate.Revision{rev1}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 2)

	// Only the last one is pending (in full).
	*rrw = []*migrate.Revision{rev1, rev2}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 1)

	// First statement of last one is marked as applied, second isn't. Third file is still pending.
	*rrw = []*migrate.Revision{rev1, rev2, rev3}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 1)

	// Nothing to do if all migrations are applied.
	rev3.Applied = rev3.Total
	*rrw = []*migrate.Revision{rev1, rev2, rev3}
	p, err = ex.Pending(ctx)
	require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
	require.Len(t, p, 0)

	// If there is a revision in the past with no existing migration, we don't care.
	rev3.Applied = rev3.Total
	*rrw = []*migrate.Revision{{Version: "2.11"}, rev3}
	p, err = ex.Pending(ctx)
	require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
	require.Len(t, p, 0)

	// If only the last migration file is applied, we expect there are no pending files.
	*rrw = []*migrate.Revision{rev3}
	p, err = ex.Pending(ctx)
	require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
	require.Len(t, p, 0)

	// If the last applied revision has no matching migration file, and the last
	// migration version precedes that revision, there is nothing to do.
	*rrw = []*migrate.Revision{{Version: "5"}}
	p, err = ex.Pending(ctx)
	require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
	require.Len(t, p, 0)

	// The applied revision precedes every migration file. Expect all files pending.
	*rrw = []*migrate.Revision{{Version: "1.1"}}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 3)

	// All except one file are applied. The latest revision does not exist and its version is smaller than the last
	// migration file. Expect the last file pending.
	*rrw = []*migrate.Revision{rev1, rev2, {Version: "2.11"}}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 1)
	require.Equal(t, rev3.Version, p[0].Version())

	// If the last revision is partially applied and the matching migration file does not exist, we have a problem.
	*rrw = []*migrate.Revision{{Version: "deleted", Description: "desc", Total: 1}}
	p, err = ex.Pending(ctx)
	require.EqualError(t, err, migrate.MissingMigrationError{Version: "deleted", Description: "desc"}.Error())
	require.Len(t, p, 0)

	// If the last revision is a partially applied checkpoint, expect it to be part of
	// the pending files. Checkpoint files following the failed one are not included.
	dir, err = migrate.NewLocalDir(filepath.Join("testdata", "partial-checkpoint"))
	require.NoError(t, err)
	ex, err = migrate.NewExecutor(drv, dir, rrw, migrate.WithLogger(log))
	require.NoError(t, err)
	*rrw = []*migrate.Revision{{Version: "3", Description: "checkpoint", Total: 2, Applied: 1}}
	p, err = ex.Pending(ctx)
	require.NoError(t, err)
	require.Len(t, p, 3)
	require.Equal(t, "3", p[0].Version())
	require.Equal(t, "4", p[1].Version())
	require.Equal(t, "6", p[2].Version())
}

func TestExecutor_ExecOrderLinear(t *testing.T) {
	var (
		drv = &mockDriver{}
		ctx = context.Background()
		rrw = &mockRevisionReadWriter{{Version: "1"}, {Version: "2"}, {Version: "3"}}
		dir = func(names ...string) migrate.Dir {
			m := &migrate.MemDir{}
			for _, n := range names {
				require.NoError(t, m.WriteFile(n, nil))
			}
			h, err := m.Checksum()
			require.NoError(t, err)
			require.NoError(t, migrate.WriteSumFile(m, h))
			return m
		}
	)
	t.Run("Linear", func(t *testing.T) {
		ex, err := migrate.NewExecutor(drv, dir(), rrw)
		require.NoError(t, err)
		files, err := ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "3.sql"), rrw)
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "3.sql"), rrw)
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorAs(t, err, new(*migrate.HistoryNonLinearError))
		require.EqualError(t, err, "migration file 2.5.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error")

		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "2.6.sql", "3.sql"), rrw)
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorAs(t, err, new(*migrate.HistoryNonLinearError))
		require.EqualError(t, err, "migration files 2.5.sql, 2.6.sql were added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error")

		// The first file executed as checkpoint, therefore, 1.sql is not pending nor skipped.
		rrw = &mockRevisionReadWriter{{Version: "2"}, {Version: "3"}}
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2_checkpoint.sql", "3.sql"), rrw)
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		// The first file executed as checkpoint, therefore, 1.sql is not pending nor skipped.
		rrw = &mockRevisionReadWriter{{Version: "2"}, {Version: "3"}}
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2_checkpoint.sql", "2.5.sql", "3.sql"), rrw)
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorAs(t, err, new(*migrate.HistoryNonLinearError))
		require.EqualError(t, err, "migration file 2.5.sql was added out of order. See: https://atlasgo.io/versioned/apply#non-linear-error")
	})

	t.Run("LinearSkipped", func(t *testing.T) {
		ex, err := migrate.NewExecutor(drv, dir(), rrw, migrate.WithExecOrder(migrate.ExecOrderLinearSkip))
		require.NoError(t, err)
		files, err := ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "3.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderLinearSkip))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		// File 2.5.sql is skipped.
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "3.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderLinearSkip))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		// Files 2.5.sql and 2.6.sql are skipped.
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "2.6.sql", "3.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderLinearSkip))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)
	})

	t.Run("NonLinear", func(t *testing.T) {
		ex, err := migrate.NewExecutor(drv, dir(), rrw, migrate.WithExecOrder(migrate.ExecOrderNonLinear))
		require.NoError(t, err)
		files, err := ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "3.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderNonLinear))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
		require.Empty(t, files)

		// File 2.5.sql is pending.
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "3.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderNonLinear))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, "2.5.sql", files[0].Name())

		// Files 2.5.sql, 2.6.sql and 4.sql are pending.
		ex, err = migrate.NewExecutor(drv, dir("1.sql", "2.sql", "2.5.sql", "2.6.sql", "3.sql", "4.sql"), rrw, migrate.WithExecOrder(migrate.ExecOrderNonLinear))
		require.NoError(t, err)
		files, err = ex.Pending(ctx)
		require.NoError(t, err)
		require.Len(t, files, 3)
		require.Equal(t, "2.5.sql", files[0].Name())
		require.Equal(t, "2.6.sql", files[1].Name())
		require.Equal(t, "4.sql", files[2].Name())
	})
}

func TestExecutor(t *testing.T) {
	// Passing nil raises error.
	ex, err := migrate.NewExecutor(nil, nil, nil)
	require.EqualError(t, err, "sql/migrate: no driver given")
	require.Nil(t, ex)

	ex, err = migrate.NewExecutor(&mockDriver{}, nil, nil)
	require.EqualError(t, err, "sql/migrate: no dir given")
	require.Nil(t, ex)

	dir, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	ex, err = migrate.NewExecutor(&mockDriver{}, dir, nil)
	require.EqualError(t, err, "sql/migrate: no revision storage given")
	require.Nil(t, ex)

	// Does not operate on invalid migration dir.
	dir, err = migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, dir.WriteFile("atlas.sum", hash))
	ex, err = migrate.NewExecutor(&mockDriver{}, dir, &mockRevisionReadWriter{}, migrate.WithOperatorVersion("op"))
	require.NoError(t, err)
	require.NotNil(t, ex)
	require.ErrorIs(t, ex.ExecuteN(context.Background(), 0), migrate.ErrChecksumMismatch)
	require.ErrorIs(t, ex.ExecuteTo(context.Background(), ""), migrate.ErrChecksumMismatch)

	// Prerequisites.
	var (
		drv  = &mockDriver{}
		rrw  = &mockRevisionReadWriter{}
		log  = &mockLogger{}
		rev1 = &migrate.Revision{
			Version:         "1.a",
			Description:     "sub.up",
			Type:            migrate.RevisionTypeExecute,
			Applied:         2,
			Total:           2,
			Hash:            "nXyZR020M/mH7LxkoTkJr7BcQkipVg90imQ9I4595dw=",
			OperatorVersion: "op",
		}
		rev2 = &migrate.Revision{
			Version:         "2.10.x-20",
			Description:     "description",
			Type:            migrate.RevisionTypeExecute,
			Applied:         1,
			Total:           1,
			Hash:            "wQB3Vh3PHVXQg9OD3Gn7TBxbZN3r1Qb7TtAE1g3q9mQ=",
			OperatorVersion: "op",
		}
	)
	dir, err = migrate.NewLocalDir(filepath.Join("testdata", "migrate", "sub"))
	require.NoError(t, err)
	ex, err = migrate.NewExecutor(drv, dir, rrw, migrate.WithLogger(log), migrate.WithOperatorVersion("op"))
	require.NoError(t, err)

	// Applies two of them.
	require.NoError(t, ex.ExecuteN(context.Background(), 2))
	require.Equal(t, drv.executed, []string{
		"CREATE TABLE t_sub(c int);", "ALTER TABLE t_sub ADD c1 int;", "ALTER TABLE t_sub ADD c2 int;",
	})
	requireEqualRevisions(t, []*migrate.Revision{rev1, rev2}, *rrw)
	require.Len(t, *log, 7)
	require.IsType(t, migrate.LogExecution{}, (*log)[0])
	require.Equal(t, "2.10.x-20", (*log)[0].(migrate.LogExecution).To)
	require.Len(t, (*log)[0].(migrate.LogExecution).Files, 2)
	require.Equal(t, "1.a_sub.up.sql", (*log)[0].(migrate.LogExecution).Files[0].Name())
	require.Equal(t, "2.10.x-20_description.sql", (*log)[0].(migrate.LogExecution).Files[1].Name())
	require.IsType(t, migrate.LogFile{}, (*log)[1])
	require.Equal(t, migrate.LogStmt{SQL: "CREATE TABLE t_sub(c int);"}, (*log)[2])
	require.Equal(t, migrate.LogStmt{SQL: "ALTER TABLE t_sub ADD c1 int;"}, (*log)[3])
	require.IsType(t, migrate.LogFile{}, (*log)[4])
	require.Equal(t, migrate.LogStmt{SQL: "ALTER TABLE t_sub ADD c2 int;"}, (*log)[5])
	require.Equal(t, migrate.LogDone{}, (*log)[6])

	// Partly is pending.
	p, err := ex.Pending(context.Background())
	require.NoError(t, err)
	require.Len(t, p, 1)
	require.Equal(t, "3_partly.sql", p[0].Name())

	// Apply one by one.
	*rrw = mockRevisionReadWriter{}
	*drv = mockDriver{}

	require.NoError(t, ex.ExecuteN(context.Background(), 1))
	require.Equal(t, []string{"CREATE TABLE t_sub(c int);", "ALTER TABLE t_sub ADD c1 int;"}, drv.executed)
	requireEqualRevisions(t, []*migrate.Revision{rev1}, *rrw)

	require.NoError(t, ex.ExecuteN(context.Background(), 1))
	require.Equal(t, []string{
		"CREATE TABLE t_sub(c int);", "ALTER TABLE t_sub ADD c1 int;", "ALTER TABLE t_sub ADD c2 int;",
	}, drv.executed)
	requireEqualRevisions(t, []*migrate.Revision{rev1, rev2}, *rrw)

	// Partly is pending.
	p, err = ex.Pending(context.Background())
	require.NoError(t, err)
	require.Len(t, p, 1)
	require.Equal(t, "3_partly.sql", p[0].Name())

	// Suppose first revision is already executed, only execute second migration file.
	*rrw = []*migrate.Revision{rev1}
	*drv = mockDriver{}

	require.NoError(t, ex.ExecuteN(context.Background(), 1))
	require.Equal(t, []string{"ALTER TABLE t_sub ADD c2 int;"}, drv.executed)
	requireEqualRevisions(t, []*migrate.Revision{rev1, rev2}, *rrw)

	// Partly is pending.
	p, err = ex.Pending(context.Background())
	require.NoError(t, err)
	require.Len(t, p, 1)
	require.Equal(t, "3_partly.sql", p[0].Name())

	// Failing, counter will be correct.
	*rrw = []*migrate.Revision{rev1, rev2}
	*drv = mockDriver{}
	drv.failOn(2, errors.New("this is an error"))
	require.ErrorContains(t, ex.ExecuteN(context.Background(), 1), "this is an error")
	revs, err := rrw.ReadRevisions(context.Background())
	require.NoError(t, err)
	requireEqualRevision(t, &migrate.Revision{
		Version:         "3",
		Description:     "partly",
		Type:            migrate.RevisionTypeExecute,
		Applied:         1,
		Total:           2,
		Error:           "this is an error",
		ErrorStmt:       "ALTER TABLE t_sub ADD c4 int;",
		OperatorVersion: "op",
	}, revs[len(revs)-1])

	// Will fail if applied contents hash has changed (like when editing a partially applied file to fix an error).
	h := revs[len(revs)-1].PartialHashes[0]
	revs[len(revs)-1].PartialHashes[0] += h
	require.ErrorAs(t, ex.ExecuteN(context.Background(), 1), &migrate.HistoryChangedError{})

	// Re-attempting to migrate will pick up where the execution was left off.
	revs[len(revs)-1].PartialHashes[0] = h
	*drv = mockDriver{}
	require.NoError(t, ex.ExecuteN(context.Background(), 1))
	require.Equal(t, []string{"ALTER TABLE t_sub ADD c4 int;"}, drv.executed)

	// Everything is applied.
	require.ErrorIs(t, ex.ExecuteN(context.Background(), 0), migrate.ErrNoPendingFiles)

	// Test ExecuteTo.
	*rrw = []*migrate.Revision{}
	*drv = mockDriver{}
	require.EqualError(t, ex.ExecuteTo(context.Background(), ""), "sql/migrate: migration with version \"\" not found")
	require.NoError(t, ex.ExecuteTo(context.Background(), "2.10.x-20"))
	requireEqualRevisions(t, []*migrate.Revision{rev1, rev2}, *rrw)
}

func TestExecutor_Baseline(t *testing.T) {
	var (
		rrw mockRevisionReadWriter
		drv = &mockDriver{dirty: true}
		log = &mockLogger{}
	)
	dir, err := migrate.NewLocalDir(filepath.Join("testdata/migrate", "sub"))
	require.NoError(t, err)
	ex, err := migrate.NewExecutor(drv, dir, &rrw, migrate.WithLogger(log))
	require.NoError(t, err)

	// Require baseline-version or explicit flag to work on a dirty workspace.
	files, err := ex.Pending(context.Background())
	require.EqualError(t, err, "sql/migrate: connected database is not clean: found table. baseline version or allow-dirty is required")
	require.Nil(t, files)

	rrw = mockRevisionReadWriter{}
	ex, err = migrate.NewExecutor(drv, dir, &rrw, migrate.WithLogger(log), migrate.WithAllowDirty(true))
	require.NoError(t, err)
	files, err = ex.Pending(context.Background())
	require.NoError(t, err)
	require.Len(t, files, 3)

	rrw = mockRevisionReadWriter{}
	ex, err = migrate.NewExecutor(drv, dir, &rrw, migrate.WithLogger(log), migrate.WithBaselineVersion("2.10.x-20"))
	require.NoError(t, err)
	files, err = ex.Pending(context.Background())
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Len(t, rrw, 1)
	require.Equal(t, "2.10.x-20", rrw[0].Version)
	require.Equal(t, "description", rrw[0].Description)
	require.Equal(t, migrate.RevisionTypeBaseline, rrw[0].Type)

	rrw = mockRevisionReadWriter{}
	ex, err = migrate.NewExecutor(drv, dir, &rrw, migrate.WithLogger(log), migrate.WithBaselineVersion("3"))
	require.NoError(t, err)
	files, err = ex.Pending(context.Background())
	require.ErrorIs(t, err, migrate.ErrNoPendingFiles)
	require.Len(t, rrw, 1)
	require.Equal(t, "3", rrw[0].Version)
	require.Equal(t, "partly", rrw[0].Description)
	require.Equal(t, migrate.RevisionTypeBaseline, rrw[0].Type)
}

type (
	mockDriver struct {
		migrate.Driver
		plan        *migrate.Plan
		changes     []schema.Change
		applied     []schema.Change
		realm       schema.Realm
		executed    []string
		failCounter int
		failWith    error
		dirty       bool
	}
)

// the nth call to ExecContext will fail with the given error.
func (m *mockDriver) failOn(n int, err error) {
	m.failCounter = n
	m.failWith = err
}

func (m *mockDriver) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	if m.failCounter > 0 {
		m.failCounter--
		if m.failCounter == 0 {
			return nil, m.failWith
		}
	}
	m.executed = append(m.executed, query)
	return nil, nil
}

func (m *mockDriver) InspectSchema(context.Context, string, *schema.InspectOptions) (*schema.Schema, error) {
	if len(m.realm.Schemas) == 0 {
		return nil, schema.NotExistError{Err: errors.New("not found")}
	}
	return m.realm.Schemas[0], nil
}

func (m *mockDriver) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return &m.realm, nil
}

func (m *mockDriver) SchemaDiff(_, _ *schema.Schema, _ ...schema.DiffOption) ([]schema.Change, error) {
	return m.changes, nil
}

func (m *mockDriver) RealmDiff(_, _ *schema.Realm, _ ...schema.DiffOption) ([]schema.Change, error) {
	return m.changes, nil
}

func (m *mockDriver) PlanChanges(context.Context, string, []schema.Change, ...migrate.PlanOption) (*migrate.Plan, error) {
	return m.plan, nil
}

func (m *mockDriver) ApplyChanges(_ context.Context, changes []schema.Change, _ ...migrate.PlanOption) error {
	m.applied = changes
	return nil
}

func (m *mockDriver) Snapshot(context.Context) (migrate.RestoreFunc, error) {
	if m.dirty {
		return nil, &migrate.NotCleanError{}
	}
	realm := m.realm
	return func(context.Context) error {
		m.realm = realm
		return nil
	}, nil
}

func (m *mockDriver) CheckClean(context.Context, *migrate.TableIdent) error {
	if m.dirty {
		return &migrate.NotCleanError{Reason: "found table"}
	}
	return nil
}

type mockRevisionReadWriter []*migrate.Revision

func (*mockRevisionReadWriter) Ident() *migrate.TableIdent {
	return nil
}

func (*mockRevisionReadWriter) Exists(_ context.Context) (bool, error) {
	return true, nil
}

func (*mockRevisionReadWriter) Init(_ context.Context) error {
	return nil
}

func (rrw *mockRevisionReadWriter) WriteRevision(_ context.Context, r *migrate.Revision) error {
	for i, rev := range *rrw {
		if rev.Version == r.Version {
			(*rrw)[i] = r
			return nil
		}
	}
	*rrw = append(*rrw, r)
	return nil
}

func (rrw *mockRevisionReadWriter) ReadRevision(_ context.Context, v string) (*migrate.Revision, error) {
	for _, r := range *rrw {
		if r.Version == v {
			return r, nil
		}
	}
	return nil, migrate.ErrRevisionNotExist
}

func (rrw *mockRevisionReadWriter) DeleteRevision(_ context.Context, v string) error {
	i := -1
	for j, r := range *rrw {
		if r.Version == v {
			i = j
			break
		}
	}
	if i == -1 {
		return nil
	}
	copy((*rrw)[i:], (*rrw)[i+1:])
	*rrw = (*rrw)[:len(*rrw)-1]
	return nil
}

func (rrw *mockRevisionReadWriter) ReadRevisions(context.Context) ([]*migrate.Revision, error) {
	return *rrw, nil
}

func (rrw *mockRevisionReadWriter) clean() {
	*rrw = []*migrate.Revision{}
}

type mockLogger []migrate.LogEntry

func (m *mockLogger) Log(e migrate.LogEntry) { *m = append(*m, e) }

func requireEqualRevisions(t *testing.T, expected, actual []*migrate.Revision) {
	require.Equal(t, len(expected), len(actual))
	for i := range expected {
		requireEqualRevision(t, expected[i], actual[i])
	}
}

func requireEqualRevision(t *testing.T, expected, actual *migrate.Revision) {
	require.Equal(t, expected.Version, actual.Version)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.Applied, actual.Applied)
	require.Equal(t, expected.Total, actual.Total)
	require.Equal(t, expected.Error, actual.Error)
	if expected.Hash != "" {
		require.Equal(t, expected.Hash, actual.Hash)
	}
	require.Equal(t, expected.OperatorVersion, actual.OperatorVersion)
}

func countFiles(t *testing.T, d migrate.Dir) int {
	files, err := fs.ReadDir(d, "")
	require.NoError(t, err)
	return len(files)
}

func requireFileEqual(t *testing.T, d migrate.Dir, name, contents string) {
	c, err := fs.ReadFile(d, name)
	require.NoError(t, err)
	require.Equal(t, contents, string(c))
}
