// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/s-sokolko/atlas/sql/sqlite"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestVars_String(t *testing.T) {
	var vs Vars
	require.Equal(t, "[]", vs.String())
	require.NoError(t, vs.Set("a=b"))
	require.Equal(t, "[a:b]", vs.String())
	require.NoError(t, vs.Set("b=c"))
	require.Equal(t, "[a:b, b:c]", vs.String())
	require.NoError(t, vs.Set("a=d"))
	require.Equal(t, "[a:[b, d], b:c]", vs.String(), "multiple values of the same key: --var url=<one> --var url=<two>")
}

func runCmd(cmd *cobra.Command, args ...string) (string, error) {
	return runCmdContext(context.Background(), cmd, args...)
}

func runCmdContext(ctx context.Context, cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// Cobra checks for the args to equal nil and if so uses os.Args[1:].
	// In tests, this leads to go tooling arguments being part of the command arguments.
	if args == nil {
		args = []string{}
	}
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(ctx)
	return out.String(), err
}

// openSQLite creates a sqlite db, seeds it with the seed query and returns the url to it.
func openSQLite(t *testing.T, seed string) string {
	f, err := os.CreateTemp("", "sqlite.db")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Remove(f.Name()))
	})
	dsn := fmt.Sprintf("file:%s?cache=shared&_fk=1", f.Name())
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	if seed != "" {
		_, err := drv.ExecContext(context.Background(), seed)
		require.NoError(t, err)
	}
	return fmt.Sprintf("sqlite://%s", dsn)
}
