package schema

import (
	"context"

	"ariga.io/atlas/sql/schema"
)

// Inspect returns the schema description by its name.
func Inspect(ctx context.Context, i schema.Inspector, name string) (*schema.Realm, error) {
	return nil, nil
}

// Apply migrates the data source to match the requested schema.
func Apply(ctx context.Context, s []byte, i schema.Inspector, d schema.Differ, e schema.Execer) error {
	return nil
}
