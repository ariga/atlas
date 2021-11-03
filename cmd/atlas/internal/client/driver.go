package client

import (
	"context"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
)

type (
	// Driver provides to diff, execute and inspect against a Data Source.
	Driver interface {
		schema.Differ
		schema.Execer
		schema.Inspector
	}
	// AtlasDriver implements the Driver interface using Atlas.
	atlasDriver struct {
		*mysql.Driver
		schema.Differ
		schema.Execer
	}
)

// NewAtlasDriver connects a new Atlas Driver.
func NewAtlasDriver(ctx context.Context, dsn string) (*atlasDriver, error) {
	return &atlasDriver{
		nil,
		nil,
		nil,
	}, nil
}
