package schemahcl

import (
	"fmt"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type (
	Config struct {
		ctx *hcl.EvalContext
	}
	Option func(*Config)
)

// UnmarshalWith returns a schemaspec.UnmarshalerFunc configured with options.
func UnmarshalWith(opts ...Option) schemaspec.UnmarshalerFunc {
	cfg := &Config{
		ctx: &hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(data []byte, v interface{}) error {
		spec, err := decode(cfg.ctx, data)
		if err != nil {
			return fmt.Errorf("schemahcl: failed decoding: %w", err)
		}
		if err := spec.As(v); err != nil {
			return fmt.Errorf("schemahcl: failed reading spec as %T: %w", v, err)
		}
		return nil
	}
}

// EvalContext configures an unmarshaler to decode with an *hcl.EvalContext.
func EvalContext(ctx *hcl.EvalContext) Option {
	return func(config *Config) {
		config.ctx = ctx
	}
}
