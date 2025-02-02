// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/json"
)

func stdTypes(ctx *hcl.EvalContext) *hcl.EvalContext {
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
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
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
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
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
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
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
			Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
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
		"alltrue":         allTrueFunc,
		"can":             tryfunc.CanFunc,
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
		"empty":           emptyFunc,
		"endswith":        endsWithFunc,
		"flatten":         stdlib.FlattenFunc,
		"floor":           stdlib.FloorFunc,
		"format":          stdlib.FormatFunc,
		"formatdate":      stdlib.FormatDateFunc,
		"formatlist":      stdlib.FormatListFunc,
		"indent":          stdlib.IndentFunc,
		"index":           stdlib.IndexFunc,
		"join":            stdlib.JoinFunc,
		"jsondecode":      stdlib.JSONDecodeFunc,
		"jsonencode":      stdlib.JSONEncodeFunc,
		"yamldecode":      ctyyaml.YAMLDecodeFunc,
		"yamlencode":      ctyyaml.YAMLEncodeFunc,
		"keys":            stdlib.KeysFunc,
		"length":          stdlib.LengthFunc,
		"log":             stdlib.LogFunc,
		"lower":           stdlib.LowerFunc,
		"max":             stdlib.MaxFunc,
		"merge":           stdlib.MergeFunc,
		"min":             stdlib.MinFunc,
		"parseint":        stdlib.ParseIntFunc,
		"pow":             stdlib.PowFunc,
		"print":           printFunc,
		"range":           stdlib.RangeFunc,
		"regex":           stdlib.RegexFunc,
		"regexall":        stdlib.RegexAllFunc,
		"regexpescape":    regexpEscape,
		"regexreplace":    stdlib.RegexReplaceFunc,
		"replace":         stdlib.ReplaceFunc,
		"reverse":         stdlib.ReverseListFunc,
		"setintersection": stdlib.SetIntersectionFunc,
		"setproduct":      stdlib.SetProductFunc,
		"setsubtract":     stdlib.SetSubtractFunc,
		"setunion":        stdlib.SetUnionFunc,
		"signum":          stdlib.SignumFunc,
		"slice":           stdlib.SliceFunc,
		"sort":            stdlib.SortFunc,
		"split":           stdlib.SplitFunc,
		"startswith":      startsWithFunc,
		"strrev":          stdlib.ReverseFunc,
		"substr":          stdlib.SubstrFunc,
		"timeadd":         stdlib.TimeAddFunc,
		"title":           stdlib.TitleFunc,
		"tobool":          makeToFunc(cty.Bool),
		"tolist":          makeToFunc(cty.List(cty.DynamicPseudoType)),
		"tonumber":        makeToFunc(cty.Number),
		"toset":           makeToFunc(cty.Set(cty.DynamicPseudoType)),
		"tostring":        makeToFunc(cty.String),
		"trim":            stdlib.TrimFunc,
		"trimprefix":      stdlib.TrimPrefixFunc,
		"trimspace":       stdlib.TrimSpaceFunc,
		"trimsuffix":      stdlib.TrimSuffixFunc,
		"try":             tryfunc.TryFunc,
		"upper":           stdlib.UpperFunc,
		"urlescape":       urlEscapeFunc,
		"urluserinfo":     urlUserinfoFunc,
		"urlqueryset":     urlQuerySetFunc,
		"urlsetpath":      urlSetPathFunc,
		"values":          stdlib.ValuesFunc,
		"zipmap":          stdlib.ZipmapFunc,
		// A patch from the past. Should be moved
		// to specific scopes in the future.
		"sql": rawExprFunc,
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

var (
	// rawExprFunc is a stub function for raw expressions.
	rawExprFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{Name: "def", Type: cty.String, AllowNull: false},
		},
		Type:        function.StaticReturnType(ctyRawExpr),
		Description: "sql is a stub function for raw expressions.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			x := args[0].AsString()
			if len(x) == 0 {
				return cty.NilVal, errors.New("empty expression")
			}
			t := &RawExpr{X: x}
			return cty.CapsuleVal(ctyRawExpr, t), nil
		},
	})

	urlQuerySetFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "url",
				Type: cty.String,
			},
			{
				Name: "key",
				Type: cty.String,
			},
			{
				Name: "value",
				Type: cty.String,
			},
		},
		Type:        function.StaticReturnType(cty.String),
		Description: "urlqueryset sets the query parameter for the URL.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			u, err := url.Parse(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			q := u.Query()
			q.Set(args[1].AsString(), args[2].AsString())
			u.RawQuery = q.Encode()
			return cty.StringVal(u.String()), nil
		},
	})

	urlUserinfoFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "url",
				Type: cty.String,
			},
			{
				Name: "user",
				Type: cty.String,
			},
		},
		VarParam: &function.Parameter{
			Name:      "pass",
			Type:      cty.String,
			AllowNull: true,
		},
		Type:        function.StaticReturnType(cty.String),
		Description: "urluserinfo sets the user for the URL.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			u, err := url.Parse(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			user := args[1].AsString()
			if len(args) > 2 && !args[2].IsNull() {
				u.User = url.UserPassword(user, args[2].AsString())
			} else {
				u.User = url.User(user)
			}
			return cty.StringVal(u.String()), nil
		},
	})

	urlSetPathFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "url",
				Type: cty.String,
			},
			{
				Name: "path",
				Type: cty.String,
			},
		},
		Type:        function.StaticReturnType(cty.String),
		Description: "urlsetpath sets the path for the URL.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			u, err := url.Parse(args[0].AsString())
			if err != nil {
				return cty.NilVal, err
			}
			u.Path = args[1].AsString()
			return cty.StringVal(u.String()), nil
		},
	})

	urlEscapeFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "string",
				Type: cty.String,
			},
		},
		Type:        function.StaticReturnType(cty.String),
		Description: "urlescape escapes the string so it can be safely placed inside a URL query.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			u := url.QueryEscape(args[0].AsString())
			return cty.StringVal(u), nil
		},
	})

	printFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "print",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			return args[0].Type(), nil
		},
		Description: "print prints the value to stdout, and returns the value.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			switch args[0].Type() {
			case cty.String:
				fmt.Println(args[0].AsString())
			case cty.Number:
				fmt.Println(args[0].AsBigFloat().String())
			case cty.Bool:
				fmt.Println(args[0].True())
			default:
				if b, err := json.Marshal(args[0], args[0].Type()); err != nil {
					fmt.Println(args[0].GoString())
				} else {
					fmt.Println(string(b))
				}
			}
			return args[0], nil
		},
	})

	startsWithFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "s",
				Type: cty.String,
			},
			{
				Name: "prefix",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			if s1, s2 := args[0].AsString(), args[1].AsString(); strings.HasPrefix(s1, s2) {
				return cty.True, nil
			}
			return cty.False, nil
		},
	})

	endsWithFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "s",
				Type: cty.String,
			},
			{
				Name: "suffix",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			if s1, s2 := args[0].AsString(), args[1].AsString(); strings.HasSuffix(s1, s2) {
				return cty.True, nil
			}
			return cty.False, nil
		},
	})

	regexpEscape = function.New(&function.Spec{
		Description: `Return a string that escapes all regular expression metacharacters in the provided text.`,
		Params: []function.Parameter{
			{
				Name: "str",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			return cty.StringVal(regexp.QuoteMeta(args[0].AsString())), nil
		},
	})

	refineNonNull = func(b *cty.RefinementBuilder) *cty.RefinementBuilder {
		return b.NotNull()
	}

	emptyFunc = function.New(&function.Spec{
		Description: `Returns a boolean indicating whether the given collection is empty.`,
		Params: []function.Parameter{
			{
				Name:             "collection",
				Type:             cty.DynamicPseudoType,
				AllowDynamicType: true,
				AllowUnknown:     true,
				AllowMarked:      true,
			},
		},
		Type: func(args []cty.Value) (ret cty.Type, err error) {
			collTy := args[0].Type()
			if !(collTy.IsTupleType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType) {
				return cty.NilType, fmt.Errorf("collection must be a list, a map or a tuple")
			}
			return cty.Bool, nil
		},
		RefineResult: refineNonNull,
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			return args[0].Length().Equals(cty.NumberIntVal(0)), nil
		},
	})

	allTrueFunc = function.New(&function.Spec{
		Description: `Returns a boolean indicating whether all elements in the given collection are true.`,
		Params: []function.Parameter{
			{
				Name:             "collection",
				Type:             cty.DynamicPseudoType,
				AllowDynamicType: true,
				AllowUnknown:     true,
				AllowMarked:      true,
			},
		},
		Type: func(args []cty.Value) (ret cty.Type, err error) {
			collTy := args[0].Type()
			if !(collTy.IsTupleType() || collTy.IsListType() || collTy.IsMapType() || collTy.IsSetType() || collTy == cty.DynamicPseudoType) {
				return cty.NilType, fmt.Errorf("collection must be a list, a map or a tuple")
			}
			return cty.Bool, nil
		},
		RefineResult: refineNonNull,
		Impl: func(args []cty.Value, _ cty.Type) (ret cty.Value, err error) {
			for _, v := range args[0].AsValueSlice() {
				if v.Type() != cty.Bool || !v.True() {
					return cty.False, nil
				}
			}
			return cty.True, nil
		},
	})
)

// MakeFileFunc returns a function that reads a file
// from the given base directory.
func MakeFileFunc(base string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "path",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			if !filepath.IsAbs(base) {
				return cty.NilVal, fmt.Errorf("base directory must be an absolute path. got: %s", base)
			}
			path := args[0].AsString()
			if !filepath.IsAbs(path) {
				path = filepath.Clean(filepath.Join(base, path))
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return cty.NilVal, err
			}
			return cty.StringVal(string(src)), nil
		},
	})
}

// MakeGlobFunc returns a function that returns the names of all files
// matching pattern or nil if there is no matching file
// Deprecated: use fileset function instead.
func MakeGlobFunc(base string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "pattern",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.List(cty.String)),
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			if !filepath.IsAbs(base) {
				return cty.NilVal, fmt.Errorf("base directory must be an absolute path. got: %s", base)
			}
			path := args[0].AsString()
			if !filepath.IsAbs(path) {
				path = filepath.Clean(filepath.Join(base, path))
			}
			matches, err := filepath.Glob(path)
			switch {
			case err != nil:
				return cty.NilVal, err
			case len(matches) == 0:
				return cty.ListValEmpty(cty.String), nil
			default:
				vals := make([]cty.Value, len(matches))
				for i, match := range matches {
					vals[i] = cty.StringVal(match)
				}
				return cty.ListVal(vals), nil
			}
		},
	})
}

// MakeFileSetFunc returns a function that returns the names of all files
// matching pattern or nil if there is no matching file
func MakeFileSetFunc(base string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:        "pattern",
				Type:        cty.String,
				Description: "A file glob pattern to match against files in the base directory.",
			},
		},
		Type:        function.StaticReturnType(cty.List(cty.String)),
		Description: "fileset returns a list of file paths matching the given glob pattern in the base directory.",
		Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
			if !filepath.IsAbs(base) {
				return cty.NilVal, fmt.Errorf("base directory must be an absolute path. got: %s", base)
			}
			path := args[0].AsString()
			if !filepath.IsAbs(path) {
				path = filepath.Clean(filepath.Join(base, path))
			}
			matches, err := doublestar.Glob(path)
			switch {
			case err != nil:
				return cty.NilVal, err
			case len(matches) == 0:
				return cty.ListValEmpty(cty.String), nil
			default:
				vals := make([]cty.Value, len(matches))
				for i, match := range matches {
					vals[i] = cty.StringVal(match)
				}
				return cty.ListVal(vals), nil
			}
		},
	})
}
