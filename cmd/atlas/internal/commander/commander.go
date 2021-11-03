package commander

import (
	"context"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema"
)

type (
	Driver interface {
		schema.Differ
		schema.Execer
		schema.Inspector
	}
	atlasDriver struct {
		*mysql.Driver
		schema.Differ
		schema.Execer
	}
)

func NewAtlasDriver(ctx context.Context, dsn string) (*atlasDriver, error) {
	return &atlasDriver{
		nil,
		nil,
		nil,
	}, nil
}
