package client

import (
	"context"
)

// Inspect returns the schema description by its name.
func Inspect(ctx context.Context, d Driver) ([]byte, error) {
	return nil, nil
}

// Apply migrates the data source to match the requested schema.
func Apply(ctx context.Context, d Driver, s []byte) error {
	return nil
}
