// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci_test

import (
	"context"
	"testing"

	"ariga.io/atlas/internal/ci"
	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestGitChangeDetector(t *testing.T) {
	cs, err := ci.NewGitChangeDetector("", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: root cannot be \"\"")

	cs, err = ci.NewGitChangeDetector("testdata", nil)
	require.Nil(t, cs)
	require.EqualError(t, err, "internal/ci: dir cannot be <nil>")

	// OK
	d, err := migrate.NewLocalDir("testdata/migrations")
	require.NoError(t, err)

	cs, err = ci.NewGitChangeDetector("testdata", d)
	require.NoError(t, err)
	require.NotNil(t, cs)

	base, feat, err := cs.DetectChanges(context.Background())
	require.NoError(t, err)
	require.Len(t, base, 1)
	require.Len(t, feat, 2)
	require.Equal(t, "1_applied.sql", base[0].Name())
	require.Equal(t, "2_new.sql", feat[0].Name())
	require.Equal(t, "3_new_the_second.sql", feat[1].Name())
}
