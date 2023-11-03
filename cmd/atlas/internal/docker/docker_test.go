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
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// MariaDB
	cfg, err = MariaDB("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "arigaio/mariadb:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// PostgreSQL
	cfg, err = PostgreSQL("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image:    "postgres:latest",
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		Database: "postgres",
		Port:     "5432",
		Out:      io.Discard,
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
		Port:     "5432",
		Out:      io.Discard,
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
