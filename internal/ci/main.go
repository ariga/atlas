// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"bytes"
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

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
	//go:embed *.tmpl
	tplFS embed.FS
	tpl   = template.Must(template.ParseFS(tplFS, "*.tmpl"))

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
			Version: "sqlite",
			Regex:   "SQLite.*",
		},
	}
)

type (
	goVersions  []string
	concurrency struct {
		group  string
		cancel bool
	}
)

var data = struct {
	Jobs                         []Job
	Flavor, Tags, Runner, Suffix string
	GoVersions                   goVersions
	Concurrency                  concurrency
}{
	Concurrency: concurrency{
		group:  "${{ github.workflow }}-${{ github.head_ref || github.run_id }}",
		cancel: true,
	},
}

func main() {
	flag.StringVar(&data.Flavor, "flavor", "", "")
	flag.StringVar(&data.Tags, "tags", "", "")
	flag.StringVar(&data.Runner, "runner", "ubuntu-latest", "")
	flag.StringVar(&data.Suffix, "suffix", "", "")
	flag.Parse()
	for _, n := range []string{"dialect", "go", "revisions"} {
		var (
			buf bytes.Buffer
			g   = data.Concurrency.group
		)
		if n == "dialect" {
			// Dialect jobs are running after go jobs, putting them in the same concurrency group crates deadlock.
			data.Concurrency.group = fmt.Sprintf("%s-dialect", g)
		}
		if err := tpl.ExecuteTemplate(&buf, fmt.Sprintf("ci_%s.tmpl", n), data); err != nil {
			log.Fatalln(err)
		}
		data.Concurrency.group = g
		err := os.WriteFile(filepath.Clean(fmt.Sprintf("../../.github/workflows/ci-%s_%s.yaml", n, data.Suffix)), buf.Bytes(), 0600)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func (v goVersions) String() (s string) {
	for i := range v {
		v[i] = strconv.Quote(v[i])
	}
	return fmt.Sprintf("[ %s ]", strings.Join(v, ", "))
}

func (c concurrency) String() (s string) {
	return fmt.Sprintf("concurrency:\n  group: %s\n  cancel-in-progress: %t", c.group, c.cancel)
}
