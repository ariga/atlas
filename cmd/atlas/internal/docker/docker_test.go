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
		Image: "arigaio/mysql",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	u, err = url.Parse("docker://mysql/8")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "arigaio/mysql:8",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	u, err = url.Parse("docker://mysql/latest/test")
	require.NoError(t, err)
	cfg, err = FromURL(u)
	require.NoError(t, err)
	require.Equal(t, &Config{
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
		Image:    "postgis/postgis:14-3.4",
		Database: "postgres",
		Env:      []string{"POSTGRES_PASSWORD=pass"},
		Port:     "5432",
		Out:      io.Discard,
	}, cfg)
}
