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

func TestSecurity_Scan(t *testing.T) {
	c := mockClient(t)
	for _, tt := range []struct {
		name   string
		params *atlasexec.SecurityScanParams
		args   string
	}{
		{
			name:   "url only",
			params: &atlasexec.SecurityScanParams{URL: []string{"postgres://localhost:5432/app"}},
			args:   "security scan --url postgres://localhost:5432/app --format {{ json . }}",
		},
		{
			name: "all flags",
			params: &atlasexec.SecurityScanParams{
				ConfigURL:   "file://atlas.hcl",
				Env:         "prod",
				Vars:        atlasexec.Vars2{"token": "secret"},
				URL:         []string{"postgres://localhost:5432/app", "postgres://localhost:5433/reports"},
				MinSeverity: "HIGH",
				FailOn:      "CRITICAL",
				Ignore:      []string{"CVE-2014-2669", "CVE-2017-18359"},
			},
			args: "security scan --config file://atlas.hcl --env prod --var token=secret " +
				"--url postgres://localhost:5432/app --url postgres://localhost:5433/reports " +
				"--min-severity HIGH --fail-on CRITICAL --ignore CVE-2014-2669 --ignore CVE-2017-18359 " +
				"--format {{ json . }}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDERR", "")
			t.Setenv("TEST_STDOUT", `{"Targets":[{"URL":"postgres://localhost:5432/app","Driver":"postgres","Version":"18.0","Extensions":["pgcrypto"]}]}`)
			scan, err := c.SecurityScan(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, []*atlasexec.SecurityScanTarget{{
				URL:        "postgres://localhost:5432/app",
				Driver:     "postgres",
				Version:    "18.0",
				Extensions: []string{"pgcrypto"},
			}}, scan.Targets)
			require.Zero(t, scan.Count())
			require.Empty(t, scan.Levels())
			require.Empty(t, scan.Failures())
		})
	}
}

// Records the Security Graph reports for a legacy database, next to a database
// that could not be scanned, as the CLI prints them.
const securityScanIssues = `{"Targets":[{"URL":"postgres://localhost:5433/legacy","Driver":"postgres","Version":"9.3.0","Extensions":["hstore","postgis"],` +
	`"Vulnerabilities":[` +
	`{"Name":"hstore","Version":"1.3","ID":"CVE-2014-2669","Level":"ELEVATED","Severity":"MEDIUM",` +
	`"Description":"Multiple integer overflows in contrib/hstore/hstore_io.c in PostgreSQL before 9.3.4 allow remote authenticated users to cause a denial of service.",` +
	`"Suggestion":"Upgrade the database engine to version 9.3.3 or later to resolve CVE-2014-2669: the fix for extension \"hstore\" ships in engine releases"},` +
	`{"Name":"postgis","Version":"2.3.1","ID":"CVE-2017-18359","Level":"HIGH","Severity":"HIGH",` +
	`"Suggestion":"Upgrade extension \"postgis\" to version 2.3.3 or later to resolve CVE-2017-18359"}]},` +
	`{"URL":"postgres://localhost:5434/gone","Error":"connection refused"}]}`

func TestSecurity_ScanIssues(t *testing.T) {
	c := mockClient(t)
	t.Setenv("TEST_ARGS", "security scan --url postgres://localhost:5433/legacy --url postgres://localhost:5434/gone --fail-on HIGH --format {{ json . }}")
	t.Setenv("TEST_STDERR", "")
	t.Setenv("TEST_STDOUT", securityScanIssues)
	// The scan reports its result and fails the command on it.
	t.Setenv("TEST_EXIT_CODE", "1")
	scan, err := c.SecurityScan(context.Background(), &atlasexec.SecurityScanParams{
		URL:    []string{"postgres://localhost:5433/legacy", "postgres://localhost:5434/gone"},
		FailOn: "HIGH",
	})
	require.ErrorIs(t, err, atlasexec.ErrSecurityScan)
	require.Equal(t, 2, scan.Count())
	require.Equal(t, []atlasexec.SecurityScanLevel{
		{Level: "HIGH", Count: 1},
		{Level: "ELEVATED", Count: 1},
	}, scan.Levels())
	require.Equal(t, []*atlasexec.SecurityScanTarget{{
		URL:   "postgres://localhost:5434/gone",
		Error: "connection refused",
	}}, scan.Failures())
	require.Equal(t, []*atlasexec.SecurityVulnerability{
		{
			Name: "hstore", Version: "1.3", ID: "CVE-2014-2669", Level: "ELEVATED", Severity: "MEDIUM",
			Description: "Multiple integer overflows in contrib/hstore/hstore_io.c in PostgreSQL before 9.3.4 allow remote authenticated users to cause a denial of service.",
			Suggestion:  `Upgrade the database engine to version 9.3.3 or later to resolve CVE-2014-2669: the fix for extension "hstore" ships in engine releases`,
		},
		{
			Name: "postgis", Version: "2.3.1", ID: "CVE-2017-18359", Level: "HIGH", Severity: "HIGH",
			Suggestion: `Upgrade extension "postgis" to version 2.3.3 or later to resolve CVE-2017-18359`,
		},
	}, scan.Targets[0].Vulnerabilities)
}

// The command prints the report before the steps that follow it can fail, e.g. a
// notify block that could not send. Its findings are returned with that error.
func TestSecurity_ScanReportedThenFailed(t *testing.T) {
	c := mockClient(t)
	t.Setenv("TEST_ARGS", "security scan --url postgres://localhost:5433/legacy --format {{ json . }}")
	t.Setenv("TEST_STDOUT", securityScanIssues)
	t.Setenv("TEST_STDERR", `Error: security.notify.http "slack": unexpected status 500`)
	scan, err := c.SecurityScan(context.Background(), &atlasexec.SecurityScanParams{
		URL: []string{"postgres://localhost:5433/legacy"},
	})
	require.EqualError(t, err, `Error: security.notify.http "slack": unexpected status 500`)
	require.NotErrorIs(t, err, atlasexec.ErrSecurityScan)
	require.Equal(t, 2, scan.Count())
}

// The command is Pro-gated: without the plan it reports on stderr and scans nothing.
func TestSecurity_ScanError(t *testing.T) {
	c := mockClient(t)
	t.Setenv("TEST_ARGS", "security scan --url postgres://localhost:5432/app --format {{ json . }}")
	t.Setenv("TEST_STDERR", "Abort: atlas security scan is not enabled for your plan.")
	scan, err := c.SecurityScan(context.Background(), &atlasexec.SecurityScanParams{
		URL: []string{"postgres://localhost:5432/app"},
	})
	require.EqualError(t, err, "Abort: atlas security scan is not enabled for your plan.")
	require.Nil(t, scan)
}

func TestSecurityVulnerability_LevelText(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    *atlasexec.SecurityVulnerability
		text string
	}{
		{
			name: "reported record",
			v: &atlasexec.SecurityVulnerability{
				Name: "postgis", Version: "2.3.1", ID: "CVE-2017-18359", Level: "HIGH", Severity: "HIGH",
				Suggestion: `Upgrade extension "postgis" to version 2.3.3 or later to resolve CVE-2017-18359`,
			},
			text: `Extension "postgis" version "2.3.1" is vulnerable to CVE-2017-18359 (HIGH). ` +
				`Upgrade extension "postgis" to version 2.3.3 or later to resolve CVE-2017-18359`,
		},
		{
			name: "title, with its trailing period trimmed",
			v: &atlasexec.SecurityVulnerability{
				Name: "ext", Version: "1.0", ID: "CVE-0000-0001", Level: "critical",
				Title: "Arbitrary code execution.", Suggestion: "Upgrade to 1.1",
			},
			text: `Extension "ext" version "1.0" is vulnerable to CVE-0000-0001 (CRITICAL): Arbitrary code execution. Upgrade to 1.1`,
		},
		{
			name: "no version or suggestion",
			v:    &atlasexec.SecurityVulnerability{Name: "ext", ID: "CVE-0000-0002", Level: "ELEVATED"},
			text: `Extension "ext" is vulnerable to CVE-0000-0002 (ELEVATED)`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.text, tt.v.LevelText())
		})
	}
}

func mockClient(t *testing.T) *atlasexec.Client {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)
	return c
}
