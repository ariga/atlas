// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package docker

import (
	"context"
	"io"
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
		Image: "mysql:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// MariaDB
	cfg, err = MariaDB("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "mariadb:latest",
		Env:   []string{"MYSQL_ROOT_PASSWORD=pass"},
		Port:  "3306",
		Out:   io.Discard,
	}, cfg)

	// PostgreSQL
	cfg, err = PostgreSQL("latest", Out(io.Discard))
	require.NoError(t, err)
	require.Equal(t, &Config{
		Image: "postgres:latest",
		Env:   []string{"POSTGRES_PASSWORD=pass"},
		Port:  "5432",
		Out:   io.Discard,
	}, cfg)
}
