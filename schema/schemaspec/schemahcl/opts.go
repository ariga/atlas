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
		ctx   *hcl.EvalContext
		types []*schemaspec.TypeSpec
	}

	// Option configures a Config.
	Option func(*Config)
)

// New returns a state configured with options.
func New(opts ...Option) *state {
	cfg := &Config{
		ctx: &hcl.EvalContext{
			Variables: make(map[string]cty.Value),
			Functions: make(map[string]function.Function),
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &state{config: cfg}
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
		for _, ts := range typeSpecs {
			typeSpec := ts
			config.types = append(config.types, typeSpec)
			if len(typeFuncArgs(typeSpec)) == 0 {
				typ := &schemaspec.Type{T: typeSpec.T}
				config.ctx.Variables[typeSpec.Name] = cty.CapsuleVal(ctyTypeSpec, typ)
				continue
			}
			spec := &function.Spec{
				Type: function.StaticReturnType(ctyTypeSpec),
			}
			for _, arg := range typeFuncArgs(typeSpec) {
				p := function.Parameter{
					Name:      arg.Name,
					AllowNull: !arg.Required,
				}
				switch arg.Kind {
				case reflect.Slice:
					p.Type = cty.DynamicPseudoType
					spec.VarParam = &p
				case reflect.String:
					p.Type = cty.String
					spec.Params = append(spec.Params, p)
				case reflect.Int, reflect.Float32:
					p.Type = cty.Number
					spec.Params = append(spec.Params, p)
				case reflect.Bool:
					p.Type = cty.Bool
					spec.Params = append(spec.Params, p)
				}
				spec.Impl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					t := &schemaspec.Type{
						T: typeSpec.T,
					}
					if spec.VarParam != nil {
						lst := &schemaspec.ListValue{}
						for _, arg := range args {
							v, err := extractLiteralValue(arg)
							if err != nil {
								return cty.NilVal, err
							}
							lst.V = append(lst.V, v)
						}
						t.Attributes = append(t.Attributes, &schemaspec.Attr{K: spec.VarParam.Name, V: lst})
					} else {
						for i, arg := range args {
							v, err := extractLiteralValue(arg)
							if err != nil {
								return cty.NilVal, err
							}
							attrName := typeSpec.Attributes[i].Name
							t.Attributes = append(t.Attributes, &schemaspec.Attr{K: attrName, V: v})
						}
					}
					return cty.CapsuleVal(ctyTypeSpec, t), nil
				}
			}
			config.ctx.Functions[typeSpec.Name] = function.New(spec)
		}
	}
}

// typeFuncArgs returns the type attributes that are configured via arguments to the
// type definition, for example precision and scale in a decimal definition, i.e `decimal(10,2)`.
func typeFuncArgs(spec *schemaspec.TypeSpec) []*schemaspec.TypeAttr {
	var args []*schemaspec.TypeAttr
	for _, attr := range spec.Attributes {
		// TODO(rotemtam): this should be defined on the TypeSpec.
		if attr.Name == "unsigned" {
			continue
		}
		args = append(args, attr)
	}
	return args
}

// typeVariadicFuncArgs returns the type attributes that are configured via a variadic
// type definition, for example in an enum("a", "b", "c").
func typeVariadicFuncArgs(spec *schemaspec.TypeSpec) []*schemaspec.TypeAttr {
	var args []*schemaspec.TypeAttr
	for _, attr := range spec.Attributes {
		if attr.Kind == reflect.Slice {
			continue
		}
		args = append(args, attr)
	}
	return args
}
