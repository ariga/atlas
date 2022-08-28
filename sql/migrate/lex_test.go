// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalFile_Stmts(t *testing.T) {
	path := filepath.Join("testdata", "lex")
	dir, err := NewLocalDir(path)
	require.NoError(t, err)
	files, err := dir.Files()
	require.NoError(t, err)
	for _, f := range files {
		stmts, err := f.Stmts()
		require.NoError(t, err)
		buf, err := os.ReadFile(filepath.Join(path, f.Name()+".golden"))
		require.NoError(t, err)
		require.Equal(t, string(buf), strings.Join(stmts, "\n-- end --\n"))
	}
}
