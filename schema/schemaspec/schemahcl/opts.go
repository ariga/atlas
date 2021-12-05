package schemahcl

import (
	"ariga.io/atlas/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type (
	// Config configures an unmarshaling.
	Config struct {
		ctx *hcl.EvalContext
	}

	// Option configures a Config.
	Option func(*Config)
)

// NewUnmarshaler returns a schemaspec.Unmarshaler configured with options.
func NewUnmarshaler(opts ...Option) *Unmarshaler {
	cfg := &Config{
		ctx: &hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Unmarshaler{config: cfg}
}

// EvalContext configures an unmarshaler to decode with an *hcl.EvalContext.
func EvalContext(ctx *hcl.EvalContext) Option {
	return func(config *Config) {
		config.ctx = ctx
	}
}

// WithTypes configures the list of given types as identifiers in the unmarshaling context.
func WithTypes(types []*schemaspec.Type) Option {
	return func(config *Config) {
		for _, t := range types {
			config.ctx.Variables[t.Name] = cty.CapsuleVal(ctySchemaType, t)
		}
	}
}
