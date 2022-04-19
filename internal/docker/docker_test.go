// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package docker

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerConfig(t *testing.T) {
	ctx := context.Background()

	// invalid config
	_, err := (&DockerConfig{}).Run(ctx)
	require.Error(t, err)

	// MySQL
	cfg, err := MySQL("latest", Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &DockerConfig{
		Image: "mysql:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   ioutil.Discard,
	}, cfg)

	// MariaDB
	cfg, err = MariaDB("latest", Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &DockerConfig{
		Image: "mariadb:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   ioutil.Discard,
	}, cfg)

	// PostgreSQL
	cfg, err = PostgreSQL("latest", Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &DockerConfig{
		Image: "postgres:latest",
		Env:   []string{"POSTGRES_PASSWORD=pass"},
		Port:  "5432",
		Out:   ioutil.Discard,
		setup: []string{"DROP SCHEMA IF EXISTS public CASCADE;"},
	}, cfg)
}
