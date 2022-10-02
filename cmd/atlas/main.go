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
	"sync"
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
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	cmdapi.Root.SetOut(os.Stdout)
	checkForUpdate()
	err := cmdapi.Root.ExecuteContext(ctx)
	if u := update(); u != "" {
		fmt.Println(u)
	}
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

var (
	updateMessage string
	wgVercheck    sync.WaitGroup
)

// update waits for results from checkForUpdate to be ready and returns them. If no result
// is ready by the timeout an empty string is returned.
func update() string {
	update := make(chan string)
	go func() {
		wgVercheck.Wait()
		update <- updateMessage
	}()
	select {
	case u := <-update:
		return u
	case <-time.After(time.Millisecond * 500):
		return ""
	}
}

// checkForUpdate checks for version updates and security advisories for Atlas.
func checkForUpdate() {
	version := cmdapi.Version()
	// Users may skip update checking behavior.
	if v := os.Getenv(envNoUpdate); v != "" {
		return
	}
	// Skip if the current binary version isn't set (dev mode).
	if !semver.IsValid(version) {
		return
	}
	path, err := homedir.Expand(versionFile)
	if err != nil {
		return
	}
	wgVercheck.Add(1)
	go func() {
		defer wgVercheck.Done()
		vc := vercheck.New(vercheckURL, path)
		payload, err := vc.Check(version)
		if err != nil {
			return
		}
		var b bytes.Buffer
		if err := vercheck.Notify.Execute(&b, payload); err != nil {
			return
		}
		updateMessage = b.String()
	}()
	return
}
