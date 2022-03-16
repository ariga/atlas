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
	Name    string   // name of job
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
			Name:    "mysql56",
			Image:   "mysql:5.6.35",
			Regex:   "MySQL.*/56",
			Env:     mysqlEnv,
			Ports:   []string{"3306:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "mysql57",
			Image:   "mysql:5.7.26",
			Regex:   "MySQL.*/57",
			Env:     mysqlEnv,
			Ports:   []string{"3307:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "mysql8",
			Image:   "mysql:8",
			Regex:   "MySQL.*/8",
			Env:     mysqlEnv,
			Ports:   []string{"3308:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "maria107",
			Image:   "mariadb:10.7",
			Regex:   "MySQL.*/Maria107",
			Env:     mysqlEnv,
			Ports:   []string{"4306:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "maria102",
			Image:   "mariadb:10.2.32",
			Regex:   "MySQL.*/Maria102",
			Env:     mysqlEnv,
			Ports:   []string{"4307:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "maria103",
			Image:   "mariadb:10.3.13",
			Regex:   "MySQL.*/Maria103",
			Env:     mysqlEnv,
			Ports:   []string{"4308:3306"},
			Options: mysqlOptions,
		},
		{
			Name:    "postgres10",
			Image:   "postgres:10",
			Regex:   "Postgres.*/10",
			Env:     pgEnv,
			Ports:   []string{"5430:5432"},
			Options: pgOptions,
		},
		{
			Name:    "postgres11",
			Image:   "postgres:11",
			Regex:   "Postgres.*/11",
			Env:     pgEnv,
			Ports:   []string{"5431:5432"},
			Options: pgOptions,
		},
		{
			Name:    "postgres12",
			Image:   "postgres:12.3",
			Regex:   "Postgres.*/12",
			Env:     pgEnv,
			Ports:   []string{"5432:5432"},
			Options: pgOptions,
		},
		{
			Name:    "postgres13",
			Image:   "postgres:13.1",
			Regex:   "Postgres.*/13",
			Env:     pgEnv,
			Ports:   []string{"5433:5432"},
			Options: pgOptions,
		},
		{
			Name:    "postgres14",
			Image:   "postgres:14",
			Regex:   "Postgres.*/14",
			Env:     pgEnv,
			Ports:   []string{"5434:5432"},
			Options: pgOptions,
		},
		{
			Name:  "tidb5",
			Image: "pingcap/tidb:v5.4.0",
			Regex: "TiDB.*/5",
			Ports: []string{"4309:4000"},
		},
		{
			Name:  "sqlite",
			Regex: "SQLite.*",
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
