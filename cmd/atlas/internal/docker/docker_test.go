// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package docker

import (
	"context"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerConfig(t *testing.T) {
	ctx := context.Background()

	// invalid config
	_, err := (&Config{}).Run(ctx)
	require.Error(t, err)

	// MySQL
	cfg, err := MySQL("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "arigaio/mysql:latest",
		User:  url.UserPassword("root", pass),
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// MariaDB
	cfg, err = MariaDB("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "arigaio/mariadb:latest",
		User:  url.UserPassword("root", pass),
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// PostgreSQL
	cfg, err = PostgreSQL("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image:    "postgres:latest",
		User:     url.UserPassword("postgres", pass),
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		Database: "postgres",
		Port:     "5432",
		Out:      io.Discard,
	}, cfg)

	// SQL Server
	cfg, err = SQLServer("2022-latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image:    "mcr.microsoft.com/mssql/server:2022-latest",
		User:     url.UserPassword("sa", passSQLServer),
		Port:     "1433",
		Database: "master",
		Out:      io.Discard,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_PID=Developer",
			"MSSQL_SA_PASSWORD=" + passSQLServer,
		},
	}, cfg)

	// ClickHouse
	cfg, err = ClickHouse("23.11", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "clickhouse/clickhouse-server:23.11",
		User:  url.UserPassword("default", pass),
		Port:  "9000",
		Out:   io.Discard,
		Env: []string{
			"CLICKHOUSE_PASSWORD=pass",
		},
	}, cfg)
}

func TestFromURL(t *testing.T) {
	u, err := url.Parse("docker://mysql")
	require.NoError(t, err)
	cfg, err := FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver: "mysql",
		Image:  "arigaio/mysql",
		User:   url.UserPassword("root", pass),
		Env:    []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:   "3306",
		Out:    io.Discard,
	}, cfg)

	u, err = url.Parse("docker://mysql/8")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver: "mysql",
		Image:  "arigaio/mysql:8",
		User:   url.UserPassword("root", pass),
		Env:    []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:   "3306",
		Out:    io.Discard,
	}, cfg)

	u, err = url.Parse("docker://mysql/latest/test")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "mysql",
		Image:    "arigaio/mysql:latest",
		Database: "test",
		Env:      []string{"MYSQL_ROOT_PASSWORD=pass", "MYSQL_DATABASE=test"},
		User:     url.UserPassword("root", pass),
		Port:     "3306",
		Out:      io.Discard,
		setup:    []string{"CREATE DATABASE IF NOT EXISTS `test`"},
	}, cfg)

	u, err = url.Parse("docker://postgres/13")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "postgres",
		Image:    "postgres:13",
		Database: "postgres",
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		User:     url.UserPassword("postgres", pass),
		Port:     "5432",
		Out:      io.Discard,
	}, cfg)

	u, err = url.Parse("docker://postgis/14-3.4")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "postgres",
		Image:    "postgis/postgis:14-3.4",
		Database: "postgres",
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		User:     url.UserPassword("postgres", pass),
		Port:     "5432",
		Out:      io.Discard,
	}, cfg)

	u, err = url.Parse("docker://postgis/14-3.4/dev")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "postgres",
		Image:    "postgis/postgis:14-3.4",
		Database: "dev",
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		User:     url.UserPassword("postgres", pass),
		Port:     "5432",
		Out:      io.Discard,
		setup:    []string{`CREATE DATABASE "dev"`},
	}, cfg)

	// SQL Server
	u, err = url.Parse("docker://sqlserver")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "sqlserver",
		Image:    "mcr.microsoft.com/mssql/server",
		Database: "master",
		User:     url.UserPassword("sa", passSQLServer),
		Port:     "1433",
		Out:      io.Discard,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_PID=Developer",
			"MSSQL_SA_PASSWORD=" + passSQLServer,
		},
	}, cfg)

	u, err = url.Parse("docker://sqlserver/2022-latest")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "sqlserver",
		Image:    "mcr.microsoft.com/mssql/server:2022-latest",
		Database: "master",
		User:     url.UserPassword("sa", passSQLServer),
		Port:     "1433",
		Out:      io.Discard,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_PID=Developer",
			"MSSQL_SA_PASSWORD=" + passSQLServer,
		},
	}, cfg)

	u, err = url.Parse("docker://sqlserver/2019-latest/foo")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "sqlserver",
		setup:    []string{"CREATE DATABASE [foo]"},
		Image:    "mcr.microsoft.com/mssql/server:2019-latest",
		Database: "foo",
		User:     url.UserPassword("sa", passSQLServer),
		Port:     "1433",
		Out:      io.Discard,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_PID=Developer",
			"MSSQL_SA_PASSWORD=" + passSQLServer,
		},
	}, cfg)

	// Azure SQL Edge
	u, err = url.Parse("docker+sqlserver://mcr.microsoft.com/azure-sql-edge:1.0.7/foo")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver:   "sqlserver",
		setup:    []string{"CREATE DATABASE [foo]"},
		Image:    "mcr.microsoft.com/azure-sql-edge:1.0.7",
		Database: "foo",
		User:     url.UserPassword("sa", passSQLServer),
		Port:     "1433",
		Out:      io.Discard,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_PID=Developer",
			"MSSQL_SA_PASSWORD=" + passSQLServer,
		},
	}, cfg)

	// ClickHouse
	u, err = url.Parse("docker://clickhouse")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver: "clickhouse",
		Image:  "clickhouse/clickhouse-server",
		Env:    []string{"CLICKHOUSE_PASSWORD=pass"},
		User:   url.UserPassword("default", pass),
		Port:   "9000",
		Out:    io.Discard,
	}, cfg)

	// ClickHouse with tag
	u, err = url.Parse("docker://clickhouse/23.11")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		driver: "clickhouse",
		Image:  "clickhouse/clickhouse-server:23.11",
		User:   url.UserPassword("default", pass),
		Env:    []string{"CLICKHOUSE_PASSWORD=pass"},
		Port:   "9000",
		Out:    io.Discard,
	}, cfg)
}

func TestFromURL_CustomImage(t *testing.T) {
	for _, tt := range []struct {
		url, image, db, dialect string
	}{
		// PostgreSQL (local and official images).
		{
			url:     "docker+postgres://local",
			image:   "local",
			db:      "postgres",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://_/local/dev",
			image:   "local",
			db:      "dev",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres:///local:tag/dev",
			image:   "local:tag",
			db:      "dev",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://postgres",
			image:   "postgres",
			db:      "postgres",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://_/postgres/dev",
			image:   "postgres",
			db:      "dev",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres:///postgres:16/dev",
			image:   "postgres:16",
			db:      "dev",
			dialect: "postgres",
		},
		// User images.
		{
			url:     "docker+postgres://postgis/postgis:16",
			image:   "postgis/postgis:16",
			db:      "postgres",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://postgis/postgis:16/dev",
			image:   "postgis/postgis:16",
			db:      "dev",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://ghcr.io/namespace/image:tag",
			image:   "ghcr.io/namespace/image:tag",
			db:      "postgres",
			dialect: "postgres",
		},
		{
			url:     "docker+postgres://ghcr.io/namespace/image:tag/dev",
			image:   "ghcr.io/namespace/image:tag",
			db:      "dev",
			dialect: "postgres",
		},
		// MySQL.
		{
			url:     "docker+mysql://local",
			image:   "local",
			dialect: "mysql",
		},
		{
			url:     "docker+mysql:///local/dev",
			image:   "local",
			db:      "dev",
			dialect: "mysql",
		},
		{
			url:     "docker+mysql://user/image",
			image:   "user/image",
			dialect: "mysql",
		},
		{
			url:     "docker+mysql://user/image:tag/dev",
			image:   "user/image:tag",
			db:      "dev",
			dialect: "mysql",
		},
		{
			url:     "docker+mysql://_/mariadb:latest/dev",
			image:   "mariadb:latest",
			db:      "dev",
			dialect: "mysql",
		},
		// SQL Server.
		{
			url:     "docker+sqlserver://mcr.microsoft.com/mssql/server:2022-latest",
			image:   "mcr.microsoft.com/mssql/server:2022-latest",
			db:      "master",
			dialect: "sqlserver",
		},
		{
			url:     "docker+sqlserver://mcr.microsoft.com/mssql/server:2022-latest/dev",
			image:   "mcr.microsoft.com/mssql/server:2022-latest",
			db:      "dev",
			dialect: "sqlserver",
		},
		{
			url:     "docker+sqlserver://mcr.microsoft.com/mssql/server:latest",
			image:   "mcr.microsoft.com/mssql/server:latest",
			db:      "master",
			dialect: "sqlserver",
		},
		// ClickHouse.
		{
			url:     "docker+clickhouse://clickhouse/clickhouse-server:23.11",
			image:   "clickhouse/clickhouse-server:23.11",
			dialect: "clickhouse",
		},
		{
			url:     "docker+clickhouse://clickhouse/clickhouse-server:23.11/dev",
			image:   "clickhouse/clickhouse-server:23.11",
			db:      "dev",
			dialect: "clickhouse",
		},
	} {
		u, err := url.Parse(tt.url)
		require.NoError(t, err)
		cfg, err := FromURL(u)
		require.NoError(t, err)
		require.Equal(t, tt.image, cfg.Image)
		require.Equal(t, tt.db, cfg.Database)
		require.Equal(t, tt.dialect, cfg.driver)
	}
}

func TestImageURL(t *testing.T) {
	for img, u := range map[string]string{
		"postgres:15":                    "docker+postgres://_/postgres:15",
		"postgres":                       "docker+postgres://_/postgres",
		"postgis/postgis:14-3.4":         "docker+postgres://postgis/postgis:14-3.4",
		"ghcr.io/namespace/postgres:tag": "docker+postgres://ghcr.io/namespace/postgres:tag",
	} {
		got, err := ImageURL(DriverPostgres, img)
		require.NoError(t, err)
		require.Equal(t, u, got.String())
	}
	for img, u := range map[string]string{
		"mcr.microsoft.com/azure-sql-edge:1.0.7":     "docker+sqlserver://mcr.microsoft.com/azure-sql-edge:1.0.7",
		"mcr.microsoft.com/mssql/server:2022-latest": "docker+sqlserver://mcr.microsoft.com/mssql/server:2022-latest",
	} {
		got, err := ImageURL(DriverSQLServer, img)
		require.NoError(t, err)
		require.Equal(t, u, got.String())
	}
}

func TestContainerURL(t *testing.T) {
	c := &Container{
		Config: Config{
			driver: "postgres",
			User:   url.UserPassword("postgres", "pass"),
		},
		Port: "5432",
	}
	u, err := c.URL()
	require.NoError(t, err)
	require.Equal(t, "postgres://postgres:pass@localhost:5432/?sslmode=disable", u.String())

	// With DOCKER_HOST
	t.Setenv("DOCKER_HOST", "tcp://host.docker.internal:2375")
	u, err = c.URL()
	require.NoError(t, err)
	require.Equal(t, "postgres://postgres:pass@host.docker.internal:5432/?sslmode=disable", u.String())
}
