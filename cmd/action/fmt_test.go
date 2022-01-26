// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	unformatted = `block  "x"  {
 x = 1
    y     = 2
}
`
	formatted = `block "x" {
  x = 1
  y = 2
}
`
)

func TestFmt(t *testing.T) {
	for _, tt := range []struct {
		name          string
		input         string
		expectedFile  string
		expectedPrint bool
	}{
		{
			name:          "unformatted",
			input:         unformatted,
			expectedFile:  formatted,
			expectedPrint: true,
		},
		{
			name:          "formatted",
			input:         formatted,
			expectedFile:  formatted,
			expectedPrint: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			filename := "test.hcl"
			dir := setupFmtTest(t, filename, tt.input)
			FmtCmd.ResetCommands() // Detach from sub-commands and parents, needed to skip input validation done by them.
			FmtCmd.SetOut(&out)
			FmtCmd.SetArgs([]string{dir})
			err := FmtCmd.Execute()
			require.NoError(t, err)
			require.Equal(t, tt.expectedPrint, out.Len() > 0)
			rf, err := os.ReadFile(filepath.Join(dir, filename))
			require.NoError(t, err)
			require.Equal(t, tt.expectedFile, string(rf))
		})
	}
}

func setupFmtTest(t *testing.T, filename, contents string) string {
	dir, err := os.MkdirTemp(os.TempDir(), "fmt-test-")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	file := path.Join(dir, filename)
	err = os.WriteFile(file, []byte(contents), 0644)
	require.NoError(t, err)
	return dir
}
