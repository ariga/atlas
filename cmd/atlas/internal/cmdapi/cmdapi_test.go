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
	"os/exec"
	"testing"

	"ariga.io/atlas/sql/sqlite"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCLI_Version(t *testing.T) {
	// Required to have a clean "stderr" while running first time.
	tests := []struct {
		name     string
		cmd      *exec.Cmd
		expected string
	}{
		{
			name: "dev mode",
			cmd: exec.Command("go", "run", "ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "release",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlas/internal/cmdapi.version=v1.2.3",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version v1.2.3\nhttps://github.com/ariga/atlas/releases/tag/v1.2.3\n",
		},
		{
			name: "canary",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlas/internal/cmdapi.version=v0.3.0-6539f2704b5d-canary",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas version v0.3.0-6539f2704b5d-canary\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
		{
			name: "flavor",
			cmd: exec.Command("go", "run",
				"-ldflags",
				"-X ariga.io/atlas/cmd/atlas/internal/cmdapi.flavor=flavor",
				"ariga.io/atlas/cmd/atlas",
				"version",
			),
			expected: "atlas flavor version - development\nhttps://github.com/ariga/atlas/releases/latest\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ATLAS_NO_UPDATE_NOTIFIER", "true")
			stdout := bytes.NewBuffer(nil)
			tt.cmd.Stdout = stdout
			tt.cmd.Stderr = os.Stderr
			require.NoError(t, tt.cmd.Run())
			require.Equal(t, tt.expected, stdout.String())
		})
	}
}

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
