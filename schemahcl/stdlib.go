// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func varBlockContext(ctx *hcl.EvalContext) *hcl.EvalContext {
	ctx = ctx.NewChild()
	ctx.Variables = map[string]cty.Value{
		"string": cty.CapsuleVal(ctyNilType, &cty.String),
		"bool":   cty.CapsuleVal(ctyNilType, &cty.Bool),
		"number": cty.CapsuleVal(ctyNilType, &cty.Number),
		// Exists for backwards compatibility.
		"int": cty.CapsuleVal(ctyNilType, &cty.Number),
	}
	ctx.Functions = map[string]function.Function{
		"list": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "elem_type", Type: ctyNilType},
			},
			Type: function.StaticReturnType(ctyNilType),
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
				argT := args[0].EncapsulatedValue().(*cty.Type)
				listT := cty.List(*argT)
				return cty.CapsuleVal(ctyNilType, &listT), nil
			},
		}),
		"set": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "elem_type", Type: ctyNilType},
			},
			Type: function.StaticReturnType(ctyNilType),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				argT := args[0].EncapsulatedValue().(*cty.Type)
				setT := cty.Set(*argT)
				return cty.CapsuleVal(ctyNilType, &setT), nil
			},
		}),
		"map": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "elem_type", Type: ctyNilType},
			},
			Type: function.StaticReturnType(ctyNilType),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				argT := args[0].EncapsulatedValue().(*cty.Type)
				mapT := cty.Map(*argT)
				return cty.CapsuleVal(ctyNilType, &mapT), nil
			},
		}),
		"tuple": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "elem_type", Type: cty.List(ctyNilType)},
			},
			Type: function.StaticReturnType(ctyNilType),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				argV := args[0]
				argsT := make([]cty.Type, 0, argV.LengthInt())
				for it := argV.ElementIterator(); it.Next(); {
					_, ev := it.Element()
					argsT = append(argsT, *ev.EncapsulatedValue().(*cty.Type))
				}
				tupleT := cty.Tuple(argsT)
				return cty.CapsuleVal(ctyNilType, &tupleT), nil
			},
		}),
		"object": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "attr_type", Type: cty.Map(ctyNilType)},
			},
			Type: function.StaticReturnType(ctyNilType),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				argV := args[0]
				argsT := make(map[string]cty.Type)
				for it := argV.ElementIterator(); it.Next(); {
					nameV, typeV := it.Element()
					name := nameV.AsString()
					argsT[name] = *typeV.EncapsulatedValue().(*cty.Type)
				}
				objT := cty.Object(argsT)
				return cty.CapsuleVal(ctyNilType, &objT), nil
			},
		}),
	}
	return ctx
}

// standard functions exist in schemahcl language.
func stdFuncs() map[string]function.Function {
	return map[string]function.Function{
		"abs":             stdlib.AbsoluteFunc,
		"ceil":            stdlib.CeilFunc,
		"chomp":           stdlib.ChompFunc,
		"chunklist":       stdlib.ChunklistFunc,
		"coalescelist":    stdlib.CoalesceListFunc,
		"compact":         stdlib.CompactFunc,
		"concat":          stdlib.ConcatFunc,
		"contains":        stdlib.ContainsFunc,
		"csvdecode":       stdlib.CSVDecodeFunc,
		"distinct":        stdlib.DistinctFunc,
		"element":         stdlib.ElementFunc,
		"flatten":         stdlib.FlattenFunc,
		"floor":           stdlib.FloorFunc,
		"format":          stdlib.FormatFunc,
		"formatdate":      stdlib.FormatDateFunc,
		"formatlist":      stdlib.FormatListFunc,
		"indent":          stdlib.IndentFunc,
		"join":            stdlib.JoinFunc,
		"jsondecode":      stdlib.JSONDecodeFunc,
		"jsonencode":      stdlib.JSONEncodeFunc,
		"keys":            stdlib.KeysFunc,
		"log":             stdlib.LogFunc,
		"lower":           stdlib.LowerFunc,
		"max":             stdlib.MaxFunc,
		"merge":           stdlib.MergeFunc,
		"min":             stdlib.MinFunc,
		"parseint":        stdlib.ParseIntFunc,
		"pow":             stdlib.PowFunc,
		"range":           stdlib.RangeFunc,
		"regex":           stdlib.RegexFunc,
		"regexall":        stdlib.RegexAllFunc,
		"reverse":         stdlib.ReverseListFunc,
		"setintersection": stdlib.SetIntersectionFunc,
		"setproduct":      stdlib.SetProductFunc,
		"setsubtract":     stdlib.SetSubtractFunc,
		"setunion":        stdlib.SetUnionFunc,
		"signum":          stdlib.SignumFunc,
		"slice":           stdlib.SliceFunc,
		"sort":            stdlib.SortFunc,
		"split":           stdlib.SplitFunc,
		"strrev":          stdlib.ReverseFunc,
		"substr":          stdlib.SubstrFunc,
		"timeadd":         stdlib.TimeAddFunc,
		"title":           stdlib.TitleFunc,
		"tostring":        makeToFunc(cty.String),
		"tonumber":        makeToFunc(cty.Number),
		"tobool":          makeToFunc(cty.Bool),
		"toset":           makeToFunc(cty.Set(cty.DynamicPseudoType)),
		"tolist":          makeToFunc(cty.List(cty.DynamicPseudoType)),
		"trim":            stdlib.TrimFunc,
		"trimprefix":      stdlib.TrimPrefixFunc,
		"trimspace":       stdlib.TrimSpaceFunc,
		"trimsuffix":      stdlib.TrimSuffixFunc,
		"try":             tryfunc.TryFunc,
		"upper":           stdlib.UpperFunc,
		"values":          stdlib.ValuesFunc,
		"zipmap":          stdlib.ZipmapFunc,
	}
}

// makeToFunc constructs a "to..." function, like "tostring", which converts
// its argument to a specific type or type kind. Code was copied from:
// github.com/hashicorp/terraform/blob/master/internal/lang/funcs/conversion.go
func makeToFunc(wantTy cty.Type) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "v",
				// We use DynamicPseudoType rather than wantTy here so that
				// all values will pass through the function API verbatim and
				// we can handle the conversion logic within the Type and
				// Impl functions. This allows us to customize the error
				// messages to be more appropriate for an explicit type
				// conversion, whereas the cty function system produces
				// messages aimed at _implicit_ type conversions.
				Type:             cty.DynamicPseudoType,
				AllowNull:        true,
				AllowMarked:      true,
				AllowDynamicType: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			gotTy := args[0].Type()
			if gotTy.Equals(wantTy) {
				return wantTy, nil
			}
			conv := convert.GetConversionUnsafe(args[0].Type(), wantTy)
			if conv == nil {
				// We'll use some specialized errors for some trickier cases,
				// but most we can handle in a simple way.
				switch {
				case gotTy.IsTupleType() && wantTy.IsTupleType():
					return cty.NilType, function.NewArgErrorf(0, "incompatible tuple type for conversion: %s", convert.MismatchMessage(gotTy, wantTy))
				case gotTy.IsObjectType() && wantTy.IsObjectType():
					return cty.NilType, function.NewArgErrorf(0, "incompatible object type for conversion: %s", convert.MismatchMessage(gotTy, wantTy))
				default:
					return cty.NilType, function.NewArgErrorf(0, "cannot convert %s to %s", gotTy.FriendlyName(), wantTy.FriendlyNameForConstraint())
				}
			}
			// If a conversion is available then everything is fine.
			return wantTy, nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// We didn't set "AllowUnknown" on our argument, so it is guaranteed
			// to be known here but may still be null.
			ret, err := convert.Convert(args[0], retType)
			if err != nil {
				val, _ := args[0].UnmarkDeep()
				// Because we used GetConversionUnsafe above, conversion can
				// still potentially fail in here. For example, if the user
				// asks to convert the string "a" to bool then we'll
				// optimistically permit it during type checking but fail here
				// once we note that the value isn't either "true" or "false".
				gotTy := val.Type()
				switch {
				case gotTy == cty.String && wantTy == cty.Bool:
					what := "string"
					if !val.IsNull() {
						what = strconv.Quote(val.AsString())
					}
					return cty.NilVal, function.NewArgErrorf(0, `cannot convert %s to bool; only the strings "true" or "false" are allowed`, what)
				case gotTy == cty.String && wantTy == cty.Number:
					what := "string"
					if !val.IsNull() {
						what = strconv.Quote(val.AsString())
					}
					return cty.NilVal, function.NewArgErrorf(0, `cannot convert %s to number; given string must be a decimal representation of a number`, what)
				default:
					return cty.NilVal, function.NewArgErrorf(0, "cannot convert %s to %s", gotTy.FriendlyName(), wantTy.FriendlyNameForConstraint())
				}
			}
			return ret, nil
		},
	})
}
