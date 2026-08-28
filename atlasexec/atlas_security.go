// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type (
	// SecurityScanParams are the parameters for the `security scan` command.
	SecurityScanParams struct {
		ConfigURL string
		Env       string
		Vars      VarArgs

		// URL and Ignore are given to flags the CLI parses as comma-separated
		// lists, hence none of their values may contain a comma.
		URL         []string // Database URL(s) to scan.
		Ignore      []string // CVE identifier(s) not to report.
		MinSeverity string   // Lowest severity to report. One of: NORMAL, ELEVATED, HIGH, CRITICAL.
		FailOn      string   // Lowest severity that fails the command. One of the above.
	}
	// SecurityScan is the result of a 'security scan' run.
	SecurityScan struct {
		Targets []*SecurityScanTarget // Scanned databases, in the order they were given.
		Start   time.Time             // Scan start time.
		End     time.Time             // Scan end time.
	}
	// SecurityScanTarget holds the vulnerabilities reported for a single database.
	SecurityScanTarget struct {
		URL             string                   `json:"URL"`               // Redacted.
		Driver          string                   `json:"Driver,omitempty"`  // e.g., postgres.
		Version         string                   `json:"Version,omitempty"` // e.g., 16.2.
		Extensions      []string                 `json:"Extensions,omitempty"`
		Vulnerabilities []*SecurityVulnerability `json:"Vulnerabilities,omitempty"`
		Error           string                   `json:"Error,omitempty"` // Set if the database was not scanned.
	}
	// SecurityVulnerability is a vulnerability reported for an installed extension.
	SecurityVulnerability struct {
		Name        string `json:"Name"`                  // Extension the vulnerability was reported for.
		Version     string `json:"Version"`               // Its installed version.
		ID          string `json:"ID"`                    // e.g., CVE-2024-10977.
		Level       string `json:"Level"`                 // Severity as the Security Graph grades it. May be empty.
		Severity    string `json:"Severity"`              // CVSS rating of the record. Empty if it carries none.
		Title       string `json:"Title,omitempty"`       // Empty for the records with no assigned title.
		Description string `json:"Description,omitempty"` // Verbatim, may run to a paragraph.
		Suggestion  string `json:"Suggestion,omitempty"`  // e.g., the version to upgrade to.
	}
	// SecurityScanLevel is the number of issues reported for a severity level.
	SecurityScanLevel struct {
		Level string
		Count int
	}
)

var (
	// The severity levels an issue is graded with, lowest first.
	securityLevels = []string{"NORMAL", "ELEVATED", "HIGH", "CRITICAL"}

	// ErrSecurityScan is returned by SecurityScan when the scan ran and its
	// result failed the command: a database could not be scanned, or an issue
	// reached the severity the command fails on. The report is returned with it.
	ErrSecurityScan = errors.New("security scan failed")
)

// SecurityScan runs the 'security scan' command. The command prints its report
// before failing on it, hence the report is returned whenever it was printed,
// next to the error that failed the command: ErrSecurityScan if the scan result
// did, or the command error itself. e.g., a notify block that could not send.
// A nil report means the command printed none.
func (c *Client) SecurityScan(ctx context.Context, params *SecurityScanParams) (*SecurityScan, error) {
	args := []string{"security", "scan"}
	// Global flags
	if params.ConfigURL != "" {
		args = append(args, "--config", params.ConfigURL)
	}
	if params.Env != "" {
		args = append(args, "--env", params.Env)
	}
	if params.Vars != nil {
		args = append(args, params.Vars.AsArgs()...)
	}
	// Flags of the 'security scan' sub-command
	args = append(args, repeatFlag("--url", params.URL)...)
	if params.MinSeverity != "" {
		args = append(args, "--min-severity", params.MinSeverity)
	}
	if params.FailOn != "" {
		args = append(args, "--fail-on", params.FailOn)
	}
	args = append(args, repeatFlag("--ignore", params.Ignore)...)
	args = append(args, "--format", "{{ json . }}")
	r, err := c.runCommand(ctx, args)
	if cliErr := (&Error{}); errors.As(err, &cliErr) {
		scan, jsonErr := firstResult(jsonDecode[SecurityScan](strings.NewReader(cliErr.Stdout), nil))
		switch {
		// The command failed before reporting anything.
		case jsonErr != nil:
			return nil, err
		// A result that failed the command is reported with nothing on stderr,
		// as the command prints it and stays silent about it.
		case cliErr.Stderr != "":
			return scan, err
		default:
			return scan, ErrSecurityScan
		}
	}
	return firstResult(jsonDecode[SecurityScan](r, err))
}

// Count returns the total number of reported issues.
func (s *SecurityScan) Count() int {
	var n int
	for _, t := range s.Targets {
		n += len(t.Vulnerabilities)
	}
	return n
}

// Levels returns the number of issues per severity level, highest first. Levels
// that reported none are omitted, as are the issues left ungraded.
func (s *SecurityScan) Levels() []SecurityScanLevel {
	counts := make(map[string]int)
	for _, t := range s.Targets {
		for _, v := range t.Vulnerabilities {
			if v.Level != "" {
				counts[strings.ToUpper(v.Level)]++
			}
		}
	}
	levels := make([]SecurityScanLevel, 0, len(counts))
	for l, n := range counts {
		levels = append(levels, SecurityScanLevel{Level: l, Count: n})
	}
	// A level this version does not know is sorted last, by name, as the
	// order of the known ones is the only one it can rank.
	slices.SortFunc(levels, func(l1, l2 SecurityScanLevel) int {
		switch i1, i2 := slices.Index(securityLevels, l1.Level), slices.Index(securityLevels, l2.Level); {
		case i1 == -1 && i2 == -1:
			return strings.Compare(l1.Level, l2.Level)
		case i1 == -1:
			return +1
		case i2 == -1:
			return -1
		default:
			return i2 - i1
		}
	})
	return levels
}

// Failures returns the databases that could not be scanned.
func (s *SecurityScan) Failures() []*SecurityScanTarget {
	var targets []*SecurityScanTarget
	for _, t := range s.Targets {
		if t.Error != "" {
			targets = append(targets, t)
		}
	}
	return targets
}

// LevelText describes the vulnerability, graded by the level the Security Graph gave it.
func (v *SecurityVulnerability) LevelText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Extension %q", v.Name)
	if v.Version != "" {
		fmt.Fprintf(&b, " version %q", v.Version)
	}
	fmt.Fprintf(&b, " is vulnerable to %s", v.ID)
	if v.Level != "" {
		fmt.Fprintf(&b, " (%s)", strings.ToUpper(v.Level))
	}
	// More than half of the reported CVEs carry no title.
	if v.Title != "" {
		fmt.Fprintf(&b, ": %s", strings.TrimSuffix(v.Title, "."))
	}
	if v.Suggestion != "" {
		fmt.Fprintf(&b, ". %s", v.Suggestion)
	}
	return b.String()
}
