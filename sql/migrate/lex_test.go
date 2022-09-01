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

func TestLocalFile_StmtDecls(t *testing.T) {
	f := NewLocalFile("f", []byte(`
-- test
cmd1;

-- hello
-- world
cmd2;

-- skip
-- this
# comment

/* Skip this as well */

# Skip this
/* one */

# command
cmd3;

/* comment1 */
/* comment2 */
cmd4;

--atlas:nolint
-- atlas:nolint destructive
cmd5;

#atlas:lint error
/*atlas:nolint DS101*/
/* atlas:lint not a directive */
/*
atlas:lint not a directive
*/
cmd6;
`))
	stmts, err := f.StmtDecls()
	require.NoError(t, err)
	require.Len(t, stmts, 6)
	require.Equal(t, "cmd1;", stmts[0].Text)
	require.Equal(t, []string{"-- test\n"}, stmts[0].Comments)
	require.Equal(t, "cmd2;", stmts[1].Text)
	require.Equal(t, []string{"-- hello\n", "-- world\n"}, stmts[1].Comments)
	require.Equal(t, "cmd3;", stmts[2].Text)
	require.Equal(t, []string{"# command\n"}, stmts[2].Comments)
	require.Equal(t, "cmd4;", stmts[3].Text)
	require.Equal(t, []string{"/* comment1 */", "/* comment2 */"}, stmts[3].Comments)
	require.Equal(t, "cmd5;", stmts[4].Text)
	require.Equal(t, []string{"--atlas:nolint\n", "-- atlas:nolint destructive\n"}, stmts[4].Comments)
	require.Equal(t, []string{"", "destructive"}, stmts[4].Directive("nolint"))
	require.Equal(t, "cmd6;", stmts[5].Text)
	require.Equal(t, []string{"#atlas:lint error\n", "/*atlas:nolint DS101*/", "/* atlas:lint not a directive */", "/*\natlas:lint not a directive\n*/"}, stmts[5].Comments)
	require.Equal(t, []string{"error"}, stmts[5].Directive("lint"))
	require.Equal(t, []string{"DS101"}, stmts[5].Directive("nolint"))
}
