// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package vercheck

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// New returns a new VerChecker for the endpoint.
func New(endpoint, statePath string) *VerChecker {
	return &VerChecker{endpoint: endpoint, statePath: statePath}
}

type (
	// Latest contains information about the most recent version.
	Latest struct {
		// Version is the new version name.
		Version string
		// Summary contains a brief description of the new version.
		Summary string
		// Link is a URL to a web page describing the new version.
		Link string
	}
	// Advisory contains contents of security advisories.
	Advisory struct {
		Text string `json:"text"`
	}
	// Payload returns information to the client about their existing version of a component.
	Payload struct {
		// Latest is set if there is a newer version to upgrade to.
		Latest *Latest `json:"latest"`
		// Advisory is set if security advisories exist for the current version.
		Advisory *Advisory `json:"advisory"`
	}
	// VerChecker retrieves version information from the vercheck service.
	VerChecker struct {
		endpoint  string
		statePath string
	}
	// State stores information about local runs of VerChecker to limit the
	// frequency in which clients poll the service for information.
	State struct {
		CheckedAt time.Time `json:"checkedat"`
	}
)

var (
	// errSkip is returned when check isn't run because 24 hours haven't passed from the previous run.
	errSkip = errors.New("skip vercheck")
	// Notify is the template for displaying the payload to the user.
	Notify *template.Template
)

// Check makes an HTTP request to endpoint to check if a new version or security advisories
// exist for the current version. Check tries to read the latest time it was run from the
// statePath, if found and 24 hours have not passed the check is skipped. When done, the latest
// time is updated in statePath.
func (v *VerChecker) Check(ctx context.Context, ver string) (*Payload, error) {
	if err := v.verifyTime(); err != nil {
		return nil, err
	}
	endpoint, err := url.JoinPath(v.endpoint, "atlas", ver)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	addHeaders(ctx, req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %s", resp.Status)
	}
	var p Payload
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	if v.statePath != "" {
		s := State{CheckedAt: time.Now()}
		st, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		// Create containing directory if it doesn't exist.
		if err := os.MkdirAll(filepath.Dir(v.statePath), os.ModePerm); err != nil {
			return nil, err
		}
		if err := os.WriteFile(v.statePath, st, 0666); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func (v *VerChecker) verifyTime() error {
	// Skip check if path to state file isn't configured.
	if v.statePath == "" {
		return nil
	}
	var s State
	f, err := os.Open(v.statePath)
	if err != nil {
		return nil
	}
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil
	}
	if time.Since(s.CheckedAt) >= (time.Hour * 24) {
		return nil
	}
	return errSkip
}

//go:embed notification.tmpl
var notifyTmpl string

func init() {
	var err error
	Notify, err = template.New("").Parse(notifyTmpl)
	if err != nil {
		panic(err)
	}
}
