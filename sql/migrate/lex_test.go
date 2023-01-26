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
		require.Equalf(t, string(buf), strings.Join(stmts, "\n-- end --\n"), "mismatched statements in file %q", f.Name())
	}
}

func TestLocalFile_StmtDecls(t *testing.T) {
	f := `cmd0;
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

-- atlas:nolint
cmd7;
`
	stmts, err := NewLocalFile("f", []byte(f)).StmtDecls()
	require.NoError(t, err)
	require.Len(t, stmts, 8)

	require.Equal(t, "cmd0;", stmts[0].Text)
	require.Equal(t, 0, stmts[0].Pos, "start of the file")

	require.Equal(t, "cmd1;", stmts[1].Text)
	require.Equal(t, strings.Index(f, "cmd1;"), stmts[1].Pos)
	require.Equal(t, []string{"-- test\n"}, stmts[1].Comments)

	require.Equal(t, "cmd2;", stmts[2].Text)
	require.Equal(t, strings.Index(f, "cmd2;"), stmts[2].Pos)
	require.Equal(t, []string{"-- hello\n", "-- world\n"}, stmts[2].Comments)

	require.Equal(t, "cmd3;", stmts[3].Text)
	require.Equal(t, strings.Index(f, "cmd3;"), stmts[3].Pos)
	require.Equal(t, []string{"# command\n"}, stmts[3].Comments)

	require.Equal(t, "cmd4;", stmts[4].Text)
	require.Equal(t, strings.Index(f, "cmd4;"), stmts[4].Pos)
	require.Equal(t, []string{"/* comment1 */", "/* comment2 */"}, stmts[4].Comments)

	require.Equal(t, "cmd5;", stmts[5].Text)
	require.Equal(t, strings.Index(f, "cmd5;"), stmts[5].Pos)
	require.Equal(t, []string{"--atlas:nolint\n", "-- atlas:nolint destructive\n"}, stmts[5].Comments)
	require.Equal(t, []string{"", "destructive"}, stmts[5].Directive("nolint"))

	require.Equal(t, "cmd6;", stmts[6].Text)
	require.Equal(t, strings.Index(f, "cmd6;"), stmts[6].Pos)
	require.Equal(t, []string{"#atlas:lint error\n", "/*atlas:nolint DS101*/", "/* atlas:lint not a directive */", "/*\natlas:lint not a directive\n*/"}, stmts[6].Comments)
	require.Equal(t, []string{"error"}, stmts[6].Directive("lint"))
	require.Equal(t, []string{"DS101"}, stmts[6].Directive("nolint"))

	require.Equal(t, "cmd7;", stmts[7].Text)
	require.Equal(t, []string{""}, stmts[7].Directive("nolint"))
}
