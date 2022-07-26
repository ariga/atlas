// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate_test

import (
	_ "embed"
	"testing"

	"ariga.io/atlas/sql/migrate"

	"github.com/stretchr/testify/require"
)

func TestDirective(t *testing.T) {
	mode, ok := migrate.Directive("atlas:sum ignore", "", "sum")
	require.True(t, ok)
	require.Equal(t, "ignore", mode)
	mode, ok = migrate.Directive("atlas:sum ignore\n", "", "sum")
	require.True(t, ok)
	require.Equal(t, "ignore", mode)
	mode, ok = migrate.Directive("atlas:sum ignore\n\n", "", "sum")
	require.True(t, ok)
	require.Equal(t, "ignore", mode)

	delimiter, ok := migrate.Directive("-- atlas:delimiter \\n\\n\nCREATE TABLE t(c)\n\n", "-- ", "delimiter")
	require.True(t, ok)
	require.Equal(t, "\\n\\n", delimiter)

	delimiter, ok = migrate.Directive("-- \\n\\n\nCREATE TABLE t(c 'atlas:delimiter')\n\n", "-- ", "delimiter")
	require.False(t, ok)
	require.Empty(t, delimiter)

	baseline, ok := migrate.Directive("-- atlas:baseline\n...\n", "-- ", "baseline")
	require.True(t, ok)
	require.Empty(t, baseline)
	baseline, ok = migrate.Directive("-- atlas:baseline \n...\n", "-- ", "baseline")
	require.True(t, ok)
	require.Empty(t, baseline)
	baseline, ok = migrate.Directive("-- atlas:baseline  \n...\n", "-- ", "baseline")
	require.True(t, ok)
	require.Empty(t, baseline)
}

func TestSplitBaseline(t *testing.T) {
	dir, err := migrate.NewLocalDir("testdata/migrate/baseline1")
	require.NoError(t, err)
	baseline, files, err := migrate.SplitBaseline(dir)
	require.NoError(t, err)
	require.Len(t, baseline, 1)
	require.True(t, baseline[0].Baseline())
	require.Equal(t, "1_baseline.sql", baseline[0].Name())
	require.Len(t, files, 1)
	require.False(t, files[0].Baseline())
	require.Equal(t, "2_initial.sql", files[0].Name())

	dir, err = migrate.NewLocalDir("testdata/migrate/baseline2")
	require.NoError(t, err)
	baseline, files, err = migrate.SplitBaseline(dir)
	require.NoError(t, err)
	require.Len(t, baseline, 3)
	require.Equal(t, "1_baseline.sql", baseline[0].Name())
	require.Equal(t, "2_baseline.sql", baseline[1].Name())
	require.Equal(t, "3_baseline.sql", baseline[2].Name())
	require.Len(t, files, 1)
	require.Equal(t, "4_initial.sql", files[0].Name())
}
