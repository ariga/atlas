// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci_test

import (
	"context"
	"testing"

	"ariga.io/atlas/cmd/atlasci/internal/ci"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestRunner_Run(t *testing.T) {
	ctx := context.Background()
	c, err := sqlclient.Open(ctx, "sqlite://run?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	r := &ci.Runner{
		Scan: testDir{},
		Dev:  c,
		ChangeDetector: testDetector{
			base: []migrate.File{
				testFile{name: "1.sql", content: "CREATE TABLE users (id INT)"},
			},
			feat: []migrate.File{
				testFile{name: "2.sql", content: "CREATE TABLE pets (id INT)\nDROP TABLE users"},
			},
		},
		Analyzer: &testAnalyzer{},
		Reporter: sqlcheck.NopReporter,
	}
	require.NoError(t, r.Run(ctx))

	passes := r.Analyzer.(*testAnalyzer).passes
	require.Len(t, passes, 1)
	changes := passes[0].File.Changes
	require.Len(t, changes, 2)
	require.Equal(t, "CREATE TABLE pets (id INT)", changes[0].Stmt)
	require.Equal(t, "DROP TABLE users", changes[1].Stmt)
}

type testAnalyzer struct {
	passes []*sqlcheck.Pass
}

func (t *testAnalyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	t.passes = append(t.passes, p)
	return nil
}

type testDetector struct {
	base, feat []migrate.File
}

func (t testDetector) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	return t.base, t.feat, nil
}
