package schemahcl

import (
	"reflect"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
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
			Functions: make(map[string]function.Function),
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
func WithTypes(typeSpecs []*schemaspec.TypeSpec) Option {
	return func(config *Config) {
		for _, typeSpec := range typeSpecs {
			if len(typeSpec.Attributes) == 0 {
				typ := &schemaspec.Type{T: typeSpec.T}
				config.ctx.Variables[typeSpec.Name] = cty.CapsuleVal(ctyTypeSpec, typ)
				continue
			}
			spec := &function.Spec{
				Type: function.StaticReturnType(ctyTypeSpec),
			}
			for _, arg := range typeSpec.Attributes {
				p := function.Parameter{
					Name:      arg.Name,
					AllowNull: !arg.Required,
				}
				switch arg.Kind {
				case reflect.String:
					p.Type = cty.String
				case reflect.Int, reflect.Float32:
					p.Type = cty.Number
				case reflect.Bool:
					p.Type = cty.Bool
				}
				spec.Params = append(spec.Params, p)
				spec.Impl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					t := &schemaspec.Type{
						T: typeSpec.T,
					}
					for i, arg := range args {
						v, err := extractLiteralValue(arg)
						if err != nil {
							return cty.NilVal, err
						}
						t.Attributes = append(t.Attributes, schemaspec.Attr{K: typeSpec.Attributes[i].Name, V: v})
					}
					return cty.CapsuleVal(ctyTypeSpec, t), nil
				}
			}
			config.ctx.Functions[typeSpec.Name] = function.New(spec)
		}
	}
}
