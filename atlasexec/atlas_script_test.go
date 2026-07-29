// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/atlasexec"
	"github.com/stretchr/testify/require"
)

func TestScript_Exec(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.ScriptExecParams
		args   string
		stdout string
	}{
		{
			name:   "url only",
			params: &atlasexec.ScriptExecParams{URL: "sqlite://file?mode=memory"},
			args:   "script exec --format {{ json . }} --url sqlite://file?mode=memory",
			stdout: `{"Scripts":[{"Name":"a"}]}`,
		},
		{
			name: "all flags",
			params: &atlasexec.ScriptExecParams{
				ConfigURL: "file://config.hcl",
				Env:       "dev",
				URL:       "sqlite://file?mode=memory",
				Files:     []string{"file://a.script.hcl", "file://b.script.hcl"},
				Match:     "purge_.*",
				Quiet:     true,
			},
			args:   "script exec --format {{ json . }} --config file://config.hcl --env dev --url sqlite://file?mode=memory --file file://a.script.hcl --file file://b.script.hcl --run purge_.* --quiet",
			stdout: `{"Scripts":[{"Name":"a"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.ScriptExec(context.Background(), tt.params)
			require.NoError(t, err)
			require.Len(t, result.Scripts, 1)
			require.Equal(t, "a", result.Scripts[0].Name)
		})
	}
}

func TestScript_Query(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	t.Setenv("TEST_ARGS", "script query --format {{ json . }} --url sqlite://file?mode=memory")
	t.Setenv("TEST_STDOUT", `{"Scripts":[{"Name":"q","Queries":[{"Index":0,"Out":"1"}]}]}`)
	result, err := c.ScriptQuery(context.Background(), &atlasexec.ScriptQueryParams{URL: "sqlite://file?mode=memory"})
	require.NoError(t, err)
	require.Len(t, result.Scripts, 1)
	require.Equal(t, "1", result.Scripts[0].Queries[0].Out)
}

func TestScript_Loop(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	t.Setenv("TEST_ARGS", "script loop --format {{ json . }} --url sqlite://file?mode=memory")
	t.Setenv("TEST_STDOUT", `{"Scripts":[{"Name":"l","Iterations":[{"Index":0,"Size":10}]}]}`)
	result, err := c.ScriptLoop(context.Background(), &atlasexec.ScriptLoopParams{URL: "sqlite://file?mode=memory"})
	require.NoError(t, err)
	require.Len(t, result.Scripts, 1)
	require.Equal(t, 10, result.Scripts[0].Iterations[0].Size)
}

func TestScript_Test(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.ScriptTestParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.ScriptTestParams{},
			args:   "script test",
			stdout: "test result",
		},
		{
			name: "all flags",
			params: &atlasexec.ScriptTestParams{
				ConfigURL: "file://config.hcl",
				Env:       "dev",
				DevURL:    "docker://postgres/16/dev",
				Run:       "example",
				Paths:     []string{"./foo", "./bar"},
			},
			args:   "script test --config file://config.hcl --env dev --dev-url docker://postgres/16/dev --run example ./foo ./bar",
			stdout: "test result",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.ScriptTest(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.stdout, result)
		})
	}
}

func TestScript_Push(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.ScriptPushParams
		args   string
		stdout string
	}{
		{
			name:   "files only",
			params: &atlasexec.ScriptPushParams{Files: []string{"file://scripts"}},
			args:   "script push --format {{ json . }} --file file://scripts",
			stdout: `{"Name":"my-scripts","Files":1,"Link":"https://a.io/scripts/my-scripts"}`,
		},
		{
			name: "with name and env",
			params: &atlasexec.ScriptPushParams{
				Env:   "prod",
				Name:  "my-scripts",
				Files: []string{"file://a.script.hcl"},
			},
			args:   "script push --format {{ json . }} --env prod --file file://a.script.hcl my-scripts",
			stdout: `{"Name":"my-scripts","Files":1,"Link":"https://a.io/scripts/my-scripts"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.ScriptPush(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "https://a.io/scripts/my-scripts", result.Link)
			require.Equal(t, 1, result.Files)
		})
	}
}
