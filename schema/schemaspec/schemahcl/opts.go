package schemahcl

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type (
	// Config configures an unmarshaling.
	Config struct {
		types  []*schemaspec.TypeSpec
		newCtx func() *hcl.EvalContext
	}

	// Option configures a Config.
	Option func(*Config)
)

// New returns a state configured with options.
func New(opts ...Option) *state {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &state{config: cfg}
}

// WithTypes configures the list of given types as identifiers in the unmarshaling context.
func WithTypes(typeSpecs []*schemaspec.TypeSpec) Option {
	newCtx := func() *hcl.EvalContext {
		ctx := &hcl.EvalContext{
			Variables: make(map[string]cty.Value),
			Functions: make(map[string]function.Function),
		}
		for _, ts := range typeSpecs {
			typeSpec := ts
			// If no required args exist, register the type as a variable in the HCL context.
			if len(typeFuncReqArgs(typeSpec)) == 0 {
				typ := &schemaspec.Type{T: typeSpec.T}
				ctx.Variables[typeSpec.Name] = cty.CapsuleVal(ctyTypeSpec, typ)
			}
			// If func args exist, register the type as a function in HCL.
			if len(typeFuncArgs(typeSpec)) > 0 {
				ctx.Functions[typeSpec.Name] = typeFuncSpec(typeSpec)
			}
		}
		return ctx
	}
	return func(config *Config) {
		config.newCtx = newCtx
		config.types = append(config.types, typeSpecs...)
	}
}

// typeFuncSpec returns the HCL function for defining the type in the spec.
func typeFuncSpec(typeSpec *schemaspec.TypeSpec) function.Function {
	spec := &function.Spec{
		Type: function.StaticReturnType(ctyTypeSpec),
	}
	for _, arg := range typeFuncArgs(typeSpec) {
		if arg.Kind == reflect.Slice || !arg.Required {
			spec.VarParam = &function.Parameter{
				Name: "args",
				Type: cty.DynamicPseudoType,
			}
			continue
		}
		p := function.Parameter{
			Name:      arg.Name,
			AllowNull: !arg.Required,
		}
		switch arg.Kind {
		case reflect.String:
			p.Type = cty.String
		case reflect.Int, reflect.Float32, reflect.Int64:
			p.Type = cty.Number
		case reflect.Bool:
			p.Type = cty.Bool
		}
		spec.Params = append(spec.Params, p)
	}
	spec.Impl = typeFuncSpecImpl(spec, typeSpec)
	return function.New(spec)
}

// typeFuncSpecImpl returns the function implementation for the HCL function spec.
func typeFuncSpecImpl(spec *function.Spec, typeSpec *schemaspec.TypeSpec) function.ImplFunc {
	return func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		t := &schemaspec.Type{
			T: typeSpec.T,
		}
		if len(args) > len(typeSpec.Attributes) && typeSpec.Attributes[len(typeSpec.Attributes)-1].Kind != reflect.Slice {
			return cty.NilVal, fmt.Errorf("too many arguments for type definition %q", typeSpec.Name)
		}
		// TypeRegistry enforces that:
		// 1. Required attrs come before optionals
		// 2. Slice attrs can only be last
		for _, attr := range typeFuncArgs(typeSpec) {
			// If the attribute is a slice, read all remaining args into a list value.
			if attr.Kind == reflect.Slice {
				lst := &schemaspec.ListValue{}
				for _, arg := range args {
					v, err := extractLiteralValue(arg)
					if err != nil {
						return cty.NilVal, err
					}
					lst.V = append(lst.V, v)
				}
				t.Attrs = append(t.Attrs, &schemaspec.Attr{K: attr.Name, V: lst})
				break
			}
			if len(args) == 0 {
				break
			}
			// Pop the first arg and add it as a literal to the type.
			var arg cty.Value
			arg, args = args[0], args[1:]
			v, err := extractLiteralValue(arg)
			if err != nil {
				return cty.NilVal, err
			}
			t.Attrs = append(t.Attrs, &schemaspec.Attr{K: attr.Name, V: v})
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
