package docker

import (
	"context"
	"database/sql"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_Up(t *testing.T) {
	c, err := NewClient()
	require.NoError(t, err)
	ctx := context.Background()

	// invalid config
	_, err = c.Up(ctx, &Config{})
	require.Error(t, err)

	// MySQL
	cfg, err := MySQL("latest")
	require.NoError(t, err)
	ct, err := c.Up(context.Background(), cfg)
	require.NoError(t, err)
	defer func(t *testing.T, ctx context.Context, id string) {
		require.NoError(t, ct.Down(ctx))
		require.Error(t, exec.Command("docker", "inspect", ct.ID).Run()) //nolint:gosec
	}(t, ctx, ct.ID)
	require.NoError(t, ct.Wait(ctx, time.Minute))
	db, err := sql.Open(ct.Driver(), ct.DSN())
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, exec.Command("docker", "inspect", ct.ID).Run()) //nolint:gosec
}
