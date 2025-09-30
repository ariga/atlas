// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package cmdapi

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	cmdmigrate "ariga.io/atlas/cmd/atlas/internal/migrate"
	"ariga.io/atlas/sql/migrate"
	"github.com/spf13/cobra"
)

// migrateApplyRun represents the 'atlas migrate apply' subcommand.
func migrateApplyRun(cmd *cobra.Command, args []string, flags migrateApplyFlags, env *Env, mr *MigrateReport) (err error) {
	var (
		count int
		ctx   = cmd.Context()
	)
	if len(args) > 0 {
		if count, err = strconv.Atoi(args[0]); err != nil {
			if nerr := (&strconv.NumError{}); errors.As(err, &nerr) && nerr.Err != nil {
				err = nerr.Err
			}
			return fmt.Errorf("invalid amount argument %q (%w). Omit the argument or pass a valid integer instead", args[0], err)
		}
		if count < 1 {
			return fmt.Errorf("cannot apply '%d' migration files", count)
		}
	}
	if err := dirFormatBC(flags.dirFormat, &flags.dirURL); err != nil {
		return err
	}
	dirURL, err := url.Parse(flags.dirURL)
	if err != nil {
		return fmt.Errorf("parse dir-url: %w", err)
	}
	// Open and validate the migration directory.
	dir, err := cmdmigrate.DirURL(ctx, dirURL, false)
	if err != nil {
		return err
	}
	if err := migrate.Validate(dir); err != nil {
		printChecksumError(cmd, err)
		return err
	}
	// Open a client to the database.
	if flags.url == "" {
		return errors.New(`required flag "url" not set`)
	}
	client, err := env.openClient(ctx, flags.url)
	if err != nil {
		return err
	}
	defer client.Close()
	// Prevent usage printing after input validation.
	cmd.SilenceUsage = true
	// Acquire a lock.
	unlock, err := client.Driver.Lock(ctx, applyLockValue, flags.lockTimeout)
	if err != nil {
		return fmt.Errorf("acquiring database lock: %w", err)
	}
	// If unlocking fails notify the user about it.
	defer func() { cobra.CheckErr(unlock()) }()
	if err := checkRevisionSchemaClarity(cmd, client, flags.revisionSchema); err != nil {
		return err
	}
	var rrw migrate.RevisionReadWriter
	if rrw, err = entRevisions(ctx, client, flags.revisionSchema); err != nil {
		return err
	}
	mrrw, ok := rrw.(cmdmigrate.RevisionReadWriter)
	if !ok {
		return fmt.Errorf("unexpected revision read-writer type: %T", rrw)
	}
	if err := mrrw.Migrate(ctx); err != nil {
		return err
	}
	// Setup reporting info.
	report := cmdlog.NewMigrateApply(ctx, client, dirURL)
	mr.Init(client, report, mrrw)
	// If cloud reporting is enabled, and we cannot obtain the current
	// target identifier, abort and report it to the user.
	if err := mr.RecordTargetID(cmd.Context()); err != nil {
		return err
	}
	// Determine pending files.
	opts, err := flags.migrateOptions()
	if err != nil {
		return err
	}
	opts = append(opts, migrate.WithOperatorVersion(operatorVersion()), migrate.WithLogger(report))
	ex, err := migrate.NewExecutor(client.Driver, dir, rrw, opts...)
	if err != nil {
		return err
	}
	pending, err := ex.Pending(ctx)
	if err != nil && !errors.Is(err, migrate.ErrNoPendingFiles) {
		mr.RecordPlanError(cmd, flags, err.Error())
		return err
	}
	noPending := errors.Is(err, migrate.ErrNoPendingFiles)
	// Get the pending files before obtaining applied revisions,
	// as the Executor may write a baseline revision in the table.
	applied, err := rrw.ReadRevisions(ctx)
	if err != nil {
		return err
	}
	if noPending {
		migrate.LogNoPendingFiles(report, applied)
		return mr.Done(cmd, flags)
	}
	if l := len(pending); count == 0 || count >= l {
		// Cannot apply more than len(pending) migration files.
		count = l
	}
	pending = pending[:count]
	migrate.LogIntro(report, applied, pending)
	var (
		mux = tx{
			dryRun: flags.dryRun,
			mode:   flags.txMode,
			schema: flags.revisionSchema,
			c:      client,
			rrw:    rrw,
		}
		drv migrate.Driver
	)
	for _, f := range pending {
		if drv, rrw, err = mux.driverFor(ctx, f); err != nil {
			break
		}
		if ex, err = migrate.NewExecutor(drv, dir, rrw, opts...); err != nil {
			return fmt.Errorf("unexpected executor creation error: %w", err)
		}
		if err = mux.mayRollback(ex.Execute(ctx, f)); err != nil {
			break
		}
		if err = mux.mayCommit(); err != nil {
			break
		}
	}
	if err == nil {
		if err = mux.commit(); err == nil {
			report.Log(migrate.LogDone{})
		}
	}
	if err != nil {
		report.Error = err.Error()
	}
	return errors.Join(err, mr.Done(cmd, flags))
}
