// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	// defaultURL for Atlas Cloud.
	defaultURL = "https://api.atlasgo.cloud/query"
	// DefaultProjectName is the default name for projects.
	DefaultProjectName = "default"
	// DefaultDirName is the default directory for reporting
	// if no directory was specified by the user.
	DefaultDirName = ".atlas"
)

// Client is a client for the Atlas Cloud API.
type Client struct {
	client   *http.Client
	endpoint string
}

// New creates a new Client for the Atlas Cloud API.
func New(endpoint, token string) *Client {
	if endpoint == "" {
		endpoint = defaultURL
	}
	return &Client{
		endpoint: endpoint,
		client: &http.Client{
			Transport: &roundTripper{
				token: token,
			},
			Timeout: time.Second * 30,
		},
	}
}

type clientCtxKey struct{}

// NewContext returns a new context with the given Client attached.
func NewContext(parent context.Context, c *Client) context.Context {
	return context.WithValue(parent, clientCtxKey{}, c)
}

// FromContext returns a Client stored inside a context, or nil if there isn't one.
func FromContext(ctx context.Context) *Client {
	c, _ := ctx.Value(clientCtxKey{}).(*Client)
	return c
}

// DirInput is the input type for retrieving a single directory.
type DirInput struct {
	Slug string `json:"slug,omitempty"`
	Name string `json:"name,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// Dir retrieves a directory from the Atlas Cloud API.
func (c *Client) Dir(ctx context.Context, input DirInput) (migrate.Dir, error) {
	var (
		payload struct {
			Dir struct {
				Content []byte `json:"content"`
			} `json:"dirState"`
		}
		query = `
		query dirState($input: DirStateInput!) {
		   dirState(input: $input) {
		     content
		   }
		}`
		vars = struct {
			Input DirInput `json:"input"`
		}{
			Input: input,
		}
	)
	if err := c.post(ctx, query, vars, &payload); err != nil {
		return nil, err
	}
	return migrate.UnarchiveDir(payload.Dir.Content)
}

type (
	// ReportMigrationSetInput represents the input type for reporting a set of migration deployments.
	ReportMigrationSetInput struct {
		ID        string                 `json:"id"`
		Planned   int                    `json:"planned"`
		StartTime time.Time              `json:"startTime"`
		EndTime   time.Time              `json:"endTime"`
		Error     *string                `json:"error,omitempty"`
		Log       []ReportStep           `json:"log,omitempty"`
		Completed []ReportMigrationInput `json:"completed,omitempty"`
	}

	// ReportMigrationInput represents an input type for reporting a migration deployments.
	ReportMigrationInput struct {
		ProjectName    string              `json:"projectName"`
		EnvName        string              `json:"envName"`
		DirName        string              `json:"dirName"`
		AtlasVersion   string              `json:"atlasVersion"`
		Target         DeployedTargetInput `json:"target"`
		StartTime      time.Time           `json:"startTime"`
		EndTime        time.Time           `json:"endTime"`
		FromVersion    string              `json:"fromVersion"`
		ToVersion      string              `json:"toVersion"`
		CurrentVersion string              `json:"currentVersion"`
		Error          *string             `json:"error,omitempty"`
		Files          []DeployedFileInput `json:"files"`
		Log            string              `json:"log"`
	}

	// DeployedTargetInput represents the input type for a deployed target.
	DeployedTargetInput struct {
		ID     string `json:"id"`
		Schema string `json:"schema"`
		URL    string `json:"url"` // URL string without userinfo.
	}

	// DeployedFileInput represents the input type for a deployed file.
	DeployedFileInput struct {
		Name      string          `json:"name"`
		Content   string          `json:"content"`
		StartTime time.Time       `json:"startTime"`
		EndTime   time.Time       `json:"endTime"`
		Skipped   int             `json:"skipped"`
		Applied   int             `json:"applied"`
		Error     *StmtErrorInput `json:"error,omitempty"`
	}

	// StmtErrorInput represents the input type for a statement error.
	StmtErrorInput struct {
		Stmt string `json:"stmt"`
		Text string `json:"text"`
	}

	// ReportStep is top-level step in a report.
	ReportStep struct {
		Text      string          `json:"text"`
		StartTime time.Time       `json:"startTime"`
		EndTime   time.Time       `json:"endTime"`
		Error     bool            `json:"error,omitempty"`
		Log       []ReportStepLog `json:"log,omitempty"`
	}
	// ReportStepLog is a log entry in a step.
	ReportStepLog struct {
		Text     string          `json:"text,omitempty"`
		Children []ReportStepLog `json:"children,omitempty"`
	}
)

// ReportMigrationSet reports a set of migration deployments to the Atlas Cloud API.
func (c *Client) ReportMigrationSet(ctx context.Context, input ReportMigrationSetInput) (string, error) {
	var (
		payload struct {
			ReportMigrationSet struct {
				URL string `json:"url"`
			} `json:"reportMigrationSet"`
		}
		query = `
		mutation ReportMigrationSet($input: ReportMigrationSetInput!) {
		   reportMigrationSet(input: $input) {
		     url
		   }
		}`
		vars = struct {
			Input ReportMigrationSetInput `json:"input"`
		}{
			Input: input,
		}
	)
	if err := c.post(ctx, query, vars, &payload); err != nil {
		return "", err
	}
	return payload.ReportMigrationSet.URL, nil
}

// ReportMigration reports a migration deployment to the Atlas Cloud API.
func (c *Client) ReportMigration(ctx context.Context, input ReportMigrationInput) (string, error) {
	var (
		payload struct {
			ReportMigration struct {
				URL string `json:"url"`
			} `json:"reportMigration"`
		}
		query = `
		mutation ReportMigration($input: ReportMigrationInput!) {
		   reportMigration(input: $input) {
		     url
		   }
		}`
		vars = struct {
			Input ReportMigrationInput `json:"input"`
		}{
			Input: input,
		}
	)
	if err := c.post(ctx, query, vars, &payload); err != nil {
		return "", err
	}
	return payload.ReportMigration.URL, nil
}

func (c *Client) post(ctx context.Context, query string, vars, data any) error {
	body, err := json.Marshal(struct {
		Query     string `json:"query"`
		Variables any    `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: vars,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	if res.StatusCode != http.StatusOK {
		var v struct {
			Errors errlist `json:"errors,omitempty"`
		}
		if err := json.NewDecoder(res.Body).Decode(&v); err != nil || len(v.Errors) == 0 {
			return fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
		return v.Errors
	}
	var scan = struct {
		Data   any     `json:"data"`
		Errors errlist `json:"errors,omitempty"`
	}{
		Data: data,
	}
	if err := json.NewDecoder(res.Body).Decode(&scan); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(scan.Errors) > 0 {
		return scan.Errors
	}
	return nil
}

type (
	// errlist wraps the gqlerror.List to print errors without
	// extra newlines and prefix info added.
	errlist gqlerror.List
	// roundTripper is a http.RoundTripper that adds the Authorization header.
	roundTripper struct {
		token string
	}
)

func (e errlist) Error() string {
	s := strings.TrimPrefix(gqlerror.List(e).Error(), "input:")
	return strings.TrimSpace(s)
}

// RoundTrip implements http.RoundTripper.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	SetHeader(req, r.token)
	return http.DefaultTransport.RoundTrip(req)
}

// RedactedURL returns a URL string with the userinfo redacted.
func RedactedURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	return u.Redacted(), nil
}

// version of the CLI set by cmdapi.
var version = "development"

// SetVersion allow cmdapi to set the version
// of the CLI provided at build time.
func SetVersion(v, flavor string) {
	version = v
	if flavor != "" {
		version += "-" + flavor
	}
}

// SetHeader sets header fields for cloud requests.
func SetHeader(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", UserAgent())
	req.Header.Set("Content-Type", "application/json")
}

// UserAgent is the value the CLI uses in the User-Agent HTTP header.
func UserAgent() string {
	return fmt.Sprintf("Atlas/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
}
