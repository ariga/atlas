// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdapi/vercheck"
	_ "ariga.io/atlas/cmd/atlas/internal/docker"
	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/mysql/mysqlcheck"
	_ "ariga.io/atlas/sql/postgres"
	_ "ariga.io/atlas/sql/postgres/postgrescheck"
	_ "ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqlite/sqlitecheck"
	"golang.org/x/mod/semver"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	cmdapi.Root.SetOut(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// On first signal seen, cancel the context. On the second signal, force stop immediately.
		stop := make(chan os.Signal, 2)
		defer close(stop)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer signal.Stop(stop)
		<-stop   // wait for first interrupt
		cancel() // cancel context to gracefully stop
		fmt.Fprintln(cmdapi.Root.OutOrStdout(), "\ninterrupt received, wait for exit or ^C to terminate")
		// Wait for the context to be canceled. Issuing a second interrupt will cause the process to force stop.
		<-stop // will not block if no signal received due to main routine exiting
		os.Exit(1)
	}()
	ctx = extendContext(context.Background())
	ctx, done := initialize(ctx)
	update := checkForUpdate(ctx)
	err := cmdapi.Root.ExecuteContext(ctx)
	if u := update(); u != "" {
		fmt.Println(u)
	}
	done()
	if err != nil {
		os.Exit(1)
	}
}

const (
	// envNoUpdate when enabled it cancels checking for update
	envNoUpdate = "ATLAS_NO_UPDATE_NOTIFIER"
	vercheckURL = "https://vercheck.ariga.io"
)

func noText() string { return "" }

// checkForUpdate checks for version updates and security advisories for Atlas.
func checkForUpdate(ctx context.Context) func() string {
	done := make(chan struct{})
	version := cmdapi.Version()
	// Users may skip update checking behavior.
	if v := os.Getenv(envNoUpdate); v != "" {
		return noText
	}
	// Skip if the current binary version isn't set (dev mode).
	if !semver.IsValid(version) {
		return noText
	}
	var message string
	go func() {
		defer close(done)
		endpoint := vercheckEndpoint(ctx)
		vc := vercheck.New(endpoint)
		payload, err := vc.Check(ctx, version)
		if err != nil {
			return
		}
		var b bytes.Buffer
		if err := vercheck.Notify.Execute(&b, payload); err != nil {
			return
		}
		message = b.String()
	}()
	return func() string {
		select {
		case <-done:
		case <-time.After(time.Millisecond * 500):
		}
		return message
	}
}
