// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ci_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"

	"ariga.io/atlas/cmd/atlasci/internal/ci"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlclient"

	"github.com/stretchr/testify/require"
)

func TestRunner_Run(t *testing.T) {
	ctx := context.Background()
	b := &bytes.Buffer{}
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
		ReportWriter: &ci.TemplateWriter{
			T: ci.DefaultTemplate,
			W: b,
		},
	}
	require.NoError(t, r.Run(ctx))

	passes := r.Analyzer.(*testAnalyzer).passes
	require.Len(t, passes, 1)
	changes := passes[0].File.Changes
	require.Len(t, changes, 2)
	require.Equal(t, "CREATE TABLE pets (id INT)", changes[0].Stmt)
	require.Equal(t, "DROP TABLE users", changes[1].Stmt)
	require.Equal(t, `Report 1. File "2.sql":

	L1: Diagnostic 1

`, b.String())
}

//go:embed testdata/atlas.sum
var hash []byte

func TestChecksumAnalyzer_Analyze(t *testing.T) {
	d, err := migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	err = (&ci.ChecksumAnalyzer{Dir: d}).Analyze(context.Background(), nil)
	require.NoError(t, err)

	d, err = migrate.NewLocalDir(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, d.WriteFile("atlas.sum", hash))
	err = (&ci.ChecksumAnalyzer{Dir: d}).Analyze(context.Background(), nil)
	require.ErrorIs(t, err, migrate.ErrChecksumMismatch)
}

type testAnalyzer struct {
	passes []*sqlcheck.Pass
}

func (t *testAnalyzer) Analyze(_ context.Context, p *sqlcheck.Pass) error {
	t.passes = append(t.passes, p)
	r := sqlcheck.Report{
		Text: fmt.Sprintf("Report %d. File %q", len(t.passes), p.File.Name()),
	}
	for i := 1; i <= len(t.passes); i++ {
		r.Diagnostics = append(r.Diagnostics, sqlcheck.Diagnostic{
			Pos:  i,
			Text: fmt.Sprintf("Diagnostic %d", i),
		})
	}
	p.Reporter.WriteReport(r)
	return nil
}

type testDetector struct {
	base, feat []migrate.File
}

func (t testDetector) DetectChanges(context.Context) ([]migrate.File, []migrate.File, error) {
	return t.base, t.feat, nil
}
