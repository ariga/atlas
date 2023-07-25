//go:build !ent

package cloudapi

import "context"

// Visualize a schema and return a URL to the visualization.
func (c *Client) Visualize(ctx context.Context, input VisualizeInput) (string, error) {
	return "", nil // unimplemented
}
