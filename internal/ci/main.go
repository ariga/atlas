// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"bytes"
	_ "embed"
	"log"
	"os"
	"text/template"
)

//go:generate go run main.go

// Job defines an integration job to run.
type Job struct {
	Version string   // version to test (passed to go test as flag which database dialect/version)
	Image   string   // name of service
	Regex   string   // run regex
	Env     []string // env of service
	Ports   []string // port mappings
	Options []string // other options
}

var (
	//go:embed ci.tmpl
	t string

	mysqlOptions = []string{
		`--health-cmd "mysqladmin ping -ppass"`,
		`--health-interval 10s`,
		`--health-start-period 10s`,
		`--health-timeout 5s`,
		`--health-retries 10`,
	}
	mysqlEnv = []string{
		"MYSQL_DATABASE: test",
		"MYSQL_ROOT_PASSWORD: pass",
	}
	pgOptions = []string{
		"--health-cmd pg_isready",
		"--health-interval 10s",
		"--health-timeout 5s",
		"--health-retries 5",
	}
	pgEnv = []string{
		"POSTGRES_DB: test",
		"POSTGRES_PASSWORD: pass",
	}
	jobs = []Job{
		{
			Version: "mysql56",
			Image:   "mysql:5.6.35",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"3306:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "mysql57",
			Image:   "mysql:5.7.26",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"3307:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "mysql8",
			Image:   "mysql:8",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"3308:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "maria107",
			Image:   "mariadb:10.7",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"4306:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "maria102",
			Image:   "mariadb:10.2.32",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"4307:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "maria103",
			Image:   "mariadb:10.3.13",
			Regex:   "MySQL",
			Env:     mysqlEnv,
			Ports:   []string{"4308:3306"},
			Options: mysqlOptions,
		},
		{
			Version: "postgres-ext-postgis",
			Image:   "postgis/postgis:latest",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5429:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres10",
			Image:   "postgres:10",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5430:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres11",
			Image:   "postgres:11",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5431:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres12",
			Image:   "postgres:12.3",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5432:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres13",
			Image:   "postgres:13.1",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5433:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres14",
			Image:   "postgres:14",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5434:5432"},
			Options: pgOptions,
		},
		{
			Version: "postgres15",
			Image:   "postgres:15",
			Regex:   "Postgres",
			Env:     pgEnv,
			Ports:   []string{"5435:5432"},
			Options: pgOptions,
		},
		{
			Version: "tidb5",
			Image:   "pingcap/tidb:v5.4.0",
			Regex:   "TiDB",
			Ports:   []string{"4309:4000"},
		},
		{
			Version: "tidb6",
			Image:   "pingcap/tidb:v6.0.0",
			Regex:   "TiDB",
			Ports:   []string{"4310:4000"},
		},
		{
			Version: "sqlite",
			Regex:   "SQLite.*",
		},
		{
			Version: "cockroach",
			Image:   "ghcr.io/ariga/cockroachdb-single-node:v21.2.11",
			Regex:   "Cockroach",
			Ports:   []string{"26257:26257"},
		},
	}
)

func main() {
	var buf bytes.Buffer
	if err := template.Must(template.New("").Parse(t)).Execute(&buf, jobs); err != nil {
		log.Fatalln(err)
	}
	if err := os.WriteFile("../../.github/workflows/ci.yml", buf.Bytes(), 0600); err != nil {
		log.Fatalln(err)
	}
}
