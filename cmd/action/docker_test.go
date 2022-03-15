// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action_test

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os/exec"
	"testing"
	"time"

	"ariga.io/atlas/cmd/action"
	"github.com/stretchr/testify/require"
)

func TestClient_Up(t *testing.T) {
	ctx := context.Background()

	// invalid config
	_, err := (&action.DockerConfig{}).Run(ctx)
	require.Error(t, err)

	// MySQL
	cfg, err := action.MySQL("latest", action.Out(ioutil.Discard))
	require.NoError(t, err)
	ct, err := cfg.Run(ctx)
	require.NoError(t, err)
	defer func(t *testing.T, ctx context.Context, id string) {
		require.NoError(t, ct.Down(ctx))
		require.Error(t, exec.Command("docker", "inspect", ct.ID).Run()) //nolint:gosec
	}(t, ctx, ct.ID)
	require.NoError(t, ct.Wait(ctx, time.Minute))
	dsn, err := ct.DSN()
	require.NoError(t, err)
	db, err := sql.Open(ct.Driver(), dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, exec.Command("docker", "inspect", ct.ID).Run()) //nolint:gosec
}
