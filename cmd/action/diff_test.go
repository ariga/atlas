// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"ariga.io/atlas/sql/sqlite"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDiffCmd_Diff(t *testing.T) {
	from := openSQLite(t, "")
	to := openSQLite(t, "create table t1 (id int);")
	cmd := newDiffCmd()
	s, err := runCmd(cmd, "schema", "diff", "--from", from, "--to", to)
	require.NoError(t, err)
	require.EqualValues(t, "-- Create \"t1\" table\nCREATE TABLE `t1` (`id` int NULL)\n", s)
}

func TestDiffCmd_Synced(t *testing.T) {
	from := openSQLite(t, "")
	to := openSQLite(t, "")
	cmd := newDiffCmd()
	s, err := runCmd(cmd, "schema", "diff", "--from", from, "--to", to)
	require.NoError(t, err)
	require.EqualValues(t, "Schemas are synced, no changes to be made.\n", s)
}

// openSQLite creates a sqlite db, seeds it with the seed query and returns the url to it.
func openSQLite(t *testing.T, seed string) string {
	f, err := ioutil.TempFile("", "sqlite.db")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Remove(f.Name())
	})
	dsn := fmt.Sprintf("file:%s?cache=shared&_fk=1", f.Name())
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})
	drv, err := sqlite.Open(db)
	require.NoError(t, err)
	if len(seed) > 0 {
		_, err := drv.ExecContext(context.Background(), seed)
		require.NoError(t, err)
	}
	return fmt.Sprintf("sqlite://%s", dsn)
}

func runCmd(cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
