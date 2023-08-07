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
	rep, err := cmdlog.NewMigrateStatus(r.Client, r.Dir)
	if err != nil {
		return nil, err
	}
	// Check if there already is a revision table in the defined schema.
	// Inspect schema and check if the table does already exist.
	sch, err := r.Client.InspectSchema(ctx, r.Schema, &schema.InspectOptions{Tables: []string{revision.Table}})
	if err != nil && !schema.IsNotExistError(err) {
		return nil, err
	}
	if schema.IsNotExistError(err) || func() bool { _, ok := sch.Table(revision.Table); return !ok }() {
		// Either schema or table does not exist.
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
		rep.Pending, err = ex.Pending(ctx)
		if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
			return nil, err
		}
		rep.Applied, err = rrw.ReadRevisions(ctx)
		if err != nil {
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
		rep.Status = "OK"
		rep.Next = "Already at latest version"
	} else {
		rep.Status = "PENDING"
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
			stmts, err := rep.Available[idx].Stmts()
			if err != nil {
				return nil, err
			}
			rep.Total = len(stmts)
		}
	}
	return rep, nil
}
