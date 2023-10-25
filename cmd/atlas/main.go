// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
	"github.com/mitchellh/go-homedir"
	"golang.org/x/mod/semver"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/libsql/libsql-client-go/libsql"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cmdapi.Root.SetOut(os.Stdout)
	ctx := newContext()
	done := initialize(ctx)
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
	versionFile = "~/.atlas/release.json"
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
	path, err := homedir.Expand(versionFile)
	if err != nil {
		return noText
	}
	var message string
	go func() {
		defer close(done)
		endpoint := vercheckEndpoint(ctx)
		vc := vercheck.New(endpoint, path)
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
