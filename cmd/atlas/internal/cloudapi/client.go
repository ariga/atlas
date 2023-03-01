package cloudapi

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi/graphql"
	"ariga.io/atlas/sql/migrate"
)

const (
	UserAgent = "atlas-cli"
)

// Client is a client for the Atlas Cloud API.
type Client struct {
	client   graphql.Client
	endpoint string
}

// New creates a new Client for the Atlas Cloud API.
func New(endpoint, token string) *Client {
	return &Client{
		client: graphql.NewClient(endpoint, &http.Client{
			Transport: &roundTripper{
				token: token,
			},
			Timeout: time.Second * 30,
		}),
	}
}

// GetDir retrieves a directory from the Atlas Cloud API.
func (c *Client) GetDir(ctx context.Context, input DirInput) (migrate.Dir, error) {
	var payload struct {
		Dir struct {
			Content string `json:"content"`
		} `json:"dir"`
	}

	if err := c.client.MakeRequest(
		ctx,
		&graphql.Request{
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
		&graphql.Response{
			Data: &payload,
		}); err != nil {
		return nil, err
	}
	dec, err := base64.StdEncoding.DecodeString(payload.Dir.Content)
	if err != nil {
		return nil, err
	}
	return migrate.UnarchiveDir(dec)
}

// roundTripper is a http.RoundTripper that adds the authorization header.
type roundTripper struct {
	token string
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(req)
}

// Input type for retrieving a single directory.
type DirInput struct {
	Name string `json:"name"`
}
