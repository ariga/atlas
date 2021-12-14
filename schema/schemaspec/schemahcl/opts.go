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
			// If no required args exist, register the type as a variable in the HCL context.
			if len(typeFuncReqArgs(typeSpec)) == 0 {
				typ := &schemaspec.Type{T: typeSpec.T}
				config.ctx.Variables[typeSpec.Name] = cty.CapsuleVal(ctyTypeSpec, typ)
			}
			// If func args exist, register the type as a function in HCL.
			if len(typeFuncArgs(typeSpec)) > 0 {
				config.ctx.Functions[typeSpec.Name] = typeFuncSpec(typeSpec)
			}
		}
	}
}

// typeFuncSpec returns the HCL function for defining the type in the spec.
func typeFuncSpec(typeSpec *schemaspec.TypeSpec) function.Function {
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
		spec.Impl = typeFuncSpecImpl(spec, typeSpec)
	}
	return function.New(spec)
}

// typeFuncSpecImpl returns the function implementation for the HCL function spec.
func typeFuncSpecImpl(spec *function.Spec, typeSpec *schemaspec.TypeSpec) function.ImplFunc {
	return func(args []cty.Value, retType cty.Type) (cty.Value, error) {
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
			t.Attrs = append(t.Attrs, &schemaspec.Attr{K: spec.VarParam.Name, V: lst})
		} else {
			for i, arg := range args {
				v, err := extractLiteralValue(arg)
				if err != nil {
					return cty.NilVal, err
				}
				attrName := spec.Params[i].Name
				t.Attrs = append(t.Attrs, &schemaspec.Attr{K: attrName, V: v})
			}
		}
		return cty.CapsuleVal(ctyTypeSpec, t), nil
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

// typeFuncReqArgs returns the required type attributes that are configured via arguments.
// for instance, in MySQL a field may be defined as both `int` and `int(10)`, in this case
// it is not a required parameter.
func typeFuncReqArgs(spec *schemaspec.TypeSpec) []*schemaspec.TypeAttr {
	var args []*schemaspec.TypeAttr
	for _, arg := range typeFuncArgs(spec) {
		if arg.Required {
			args = append(args, arg)
		}
	}
	return args
}
