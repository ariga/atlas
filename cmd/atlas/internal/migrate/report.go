// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
)

// StatusReporter is used to gather information about migration status.
type StatusReporter struct {
	// Client configures the connection to the database to file a MigrateStatus for.
	Client *sqlclient.Client
	// Dir is used for scanning and validating the migration directory.
	Dir migrate.Dir
	// Schema name the revision table resides in.
	Schema string
}

// Report creates and writes a MigrateStatus.
func (r *StatusReporter) Report(ctx context.Context) (*cmdlog.MigrateStatus, error) {
	rep := &cmdlog.MigrateStatus{Env: cmdlog.NewEnv(r.Client, r.Dir)}
	// Check if there already is a revision table in the defined schema.
	// Inspect schema and check if the table does already exist.
	sch, err := r.Client.InspectSchema(ctx, r.Schema, &schema.InspectOptions{Tables: []string{revision.Table}})
	if err != nil && !schema.IsNotExistError(err) {
		return nil, err
	}
	if schema.IsNotExistError(err) || func() bool { _, ok := sch.Table(revision.Table); return !ok }() {
		// Either schema or table does not exist.
		if rep.Available, err = migrate.FilesFromLastCheckpoint(r.Dir); err != nil {
			return nil, err
		}
		rep.Pending = rep.Available
	} else {
		// Both exist, fetch their data.
		rrw, err := RevisionsForClient(ctx, r.Client, r.Schema)
		if err != nil {
			return nil, err
		}
		if err := rrw.Migrate(ctx); err != nil {
			return nil, err
		}
		ex, err := migrate.NewExecutor(r.Client.Driver, r.Dir, rrw)
		if err != nil {
			return nil, err
		}
		rep.Applied, err = rrw.ReadRevisions(ctx)
		if err != nil {
			return nil, err
		}
		if rep.Pending, err = ex.Pending(ctx); err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
			if err1 := (*migrate.HistoryNonLinearError)(nil); errors.As(err, &err1) {
				rep.Error = err1.Error()
				rep.Status = statusPending
				rep.Pending = err1.Pending
				rep.OutOfOrder = err1.OutOfOrder
				// Non-linear error means at least one file was applied.
				rep.Current = rep.Applied[len(rep.Applied)-1].Version
				return rep, nil
			}
			return nil, err
		}
		// If no files were applied, all pending files are
		// available. The first one might be a checkpoint.
		if len(rep.Applied) == 0 {
			rep.Available = rep.Pending
		} else if rep.Available, err = r.Dir.Files(); err != nil {
			return nil, err
		}
	}
	switch len(rep.Pending) {
	case len(rep.Available):
		rep.Current = "No migration applied yet"
	default:
		rep.Current = rep.Applied[len(rep.Applied)-1].Version
	}
	if len(rep.Pending) == 0 {
		rep.Status = statusOK
		rep.Next = "Already at latest version"
	} else {
		rep.Status = statusPending
		rep.Next = rep.Pending[0].Version()
	}
	// If the last one is partially applied (and not manually resolved).
	if len(rep.Applied) != 0 {
		last := rep.Applied[len(rep.Applied)-1]
		if !last.Type.Has(migrate.RevisionTypeResolved) && last.Applied < last.Total {
			rep.SQL = strings.ReplaceAll(last.ErrorStmt, "\n", " ")
			rep.Error = strings.ReplaceAll(last.Error, "\n", " ")
			rep.Count = last.Applied
			idx := migrate.FilesLastIndex(rep.Available, func(f migrate.File) bool {
				return f.Version() == last.Version
			})
			if idx == -1 {
				return nil, fmt.Errorf("migration file with version %q not found", last.Version)
			}
			stmts, err := migrate.FileStmts(r.Client.Driver, rep.Available[idx])
			if err != nil {
				return nil, err
			}
			rep.Total = len(stmts)
		}
	}
	return rep, nil
}

const (
	statusOK      = "OK"
	statusPending = "PENDING"
)
