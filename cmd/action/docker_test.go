// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action_test

import (
	"context"
	"io/ioutil"
	"testing"

	"ariga.io/atlas/cmd/action"
	"github.com/stretchr/testify/require"
)

func TestDockerConfig(t *testing.T) {
	ctx := context.Background()

	// invalid config
	_, err := (&action.DockerConfig{}).Run(ctx)
	require.Error(t, err)

	// MySQL
	cfg, err := action.MySQL("latest", action.Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &action.DockerConfig{
		Image: "mysql:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   ioutil.Discard,
	}, cfg)

	// MariaDB
	cfg, err = action.MariaDB("latest", action.Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &action.DockerConfig{
		Image: "mariadb:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   ioutil.Discard,
	}, cfg)

	// PostgreSQL
	cfg, err = action.PostgreSQL("latest", action.Out(ioutil.Discard))
	require.NoError(t, err)
	require.Equal(t, &action.DockerConfig{
		Image: "postgres:latest",
		Env:   []string{"POSTGRES_PASSWORD=pass"},
		Port:  "5432",
		Out:   ioutil.Discard,
		Setup: []string{"DROP SCHEMA IF EXISTS public CASCADE;"},
	}, cfg)
}
