package schema

import "context"

type Commander interface {
	Inspect(ctx context.Context) ([]byte, error)
	Apply(ctx context.Context) error
}
