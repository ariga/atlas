// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cloudapi

import (
	"context"
	"net/http"
	"time"

	"ariga.io/atlas/sql/migrate"
)

// defaultURL for Atlas Cloud.
const defaultURL = "https://api.atlasgo.cloud/query"

// Client is a client for the Atlas Cloud API.
type Client struct {
	client   gqlClient
	endpoint string
}

// New creates a new Client for the Atlas Cloud API.
func New(endpoint, token string) *Client {
	if endpoint == "" {
		endpoint = defaultURL
	}
	return &Client{
		client: newGQLClient(endpoint, &http.Client{
			Transport: &roundTripper{
				token: token,
			},
			Timeout: time.Second * 30,
		}),
	}
}

// DirInput is the input type for retrieving a single directory.
type DirInput struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

// Dir retrieves a directory from the Atlas Cloud API.
func (c *Client) Dir(ctx context.Context, input DirInput) (migrate.Dir, error) {
	var payload struct {
		Dir struct {
			Content []byte `json:"content"`
		} `json:"dir"`
	}
	if err := c.client.MakeRequest(
		ctx,
		&Request{
			Query: `
				query getDir($input: DirInput!) {
					dir(input: $input) {
						content
					}
				}`,
			Variables: struct {
				Input DirInput `json:"input"`
			}{
				Input: input,
			},
		},
		&Response{
			Data: &payload,
		}); err != nil {
		return nil, err
	}
	return migrate.UnarchiveDir(payload.Dir.Content)
}

// roundTripper is a http.RoundTripper that adds the Authorization header.
type roundTripper struct {
	token string
}

// RoundTrip implements http.RoundTripper.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("User-Agent", "atlas-cli")
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(req)
}
