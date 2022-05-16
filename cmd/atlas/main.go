// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"os"
	"os/signal"

	"ariga.io/atlas/cmd/atlascmd"
	_ "ariga.io/atlas/cmd/atlascmd/docker"
	_ "ariga.io/atlas/sql/cockroach"
	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/postgres"
	_ "ariga.io/atlas/sql/sqlite"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	atlascmd.Root.SetOut(os.Stdout)
	err := atlascmd.Root.ExecuteContext(ctx)
	if err != nil {
		atlascmd.Root.PrintErrln("Error:", err)
	}
	atlascmd.CheckForUpdate()
	if err != nil {
		os.Exit(1)
	}
}
