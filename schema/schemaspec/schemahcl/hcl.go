package schemahcl

import (
	"fmt"
	"sort"
	"strconv"

	schemaspec2 "ariga.io/atlas/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type container struct {
	Body hcl.Body `hcl:",remain"`
}

func decode(body []byte) (*schemaspec2.Resource, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, "")
	if diag.HasErrors() {
		return nil, diag
	}
	c := &container{}
	ctx := &hcl.EvalContext{}
	if diag := gohcl.DecodeBody(srcHCL.Body, ctx, c); diag.HasErrors() {
		return nil, diag
	}
	return extract(ctx, c.Body, nil)
}

func extract(ctx *hcl.EvalContext, remain hcl.Body, skip map[string]struct{}) (*schemaspec2.Resource, error) {
	body, ok := remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(ctx, body.Attributes)
	if err != nil {
		return nil, err
	}
	res := &schemaspec2.Resource{
		Attrs: attrs,
	}
	for _, blk := range body.Blocks {
		resource, err := toResource(ctx, blk)
		if err != nil {
			return nil, err
		}
		res.Children = append(res.Children, resource)
	}
	return res, nil
}

func toAttrs(ctx *hcl.EvalContext, hclAttrs hclsyntax.Attributes) ([]*schemaspec2.Attr, error) {
	var attrs []*schemaspec2.Attr
	for _, hclAttr := range hclAttrs {
		at := &schemaspec2.Attr{K: hclAttr.Name}
		value, diag := hclAttr.Expr.Value(ctx)
		if diag.HasErrors() {
			return nil, diag
		}
		var err error
		if value.CanIterateElements() {
			at.V, err = extractListValue(value)
		} else {
			at.V, err = extractLiteralValue(value)
		}
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, at)
	}
	// hclsyntax.Attributes is an alias for map[string]*Attribute
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].K < attrs[j].K
	})
	return attrs, nil
}

func extractListValue(value cty.Value) (*schemaspec2.ListValue, error) {
	lst := &schemaspec2.ListValue{}
	it := value.ElementIterator()
	for it.Next() {
		_, v := it.Element()
		litv, err := extractLiteralValue(v)
		if err != nil {
			return nil, err
		}
		lst.V = append(lst.V, litv.V)
	}
	return lst, nil
}

func extractLiteralValue(value cty.Value) (*schemaspec2.LiteralValue, error) {
	switch value.Type() {
	case cty.String:
		return &schemaspec2.LiteralValue{V: strconv.Quote(value.AsString())}, nil
	case cty.Number:
		bf := value.AsBigFloat()
		num, _ := bf.Float64()
		return &schemaspec2.LiteralValue{V: strconv.FormatFloat(num, 'f', -1, 64)}, nil
	case cty.Bool:
		return &schemaspec2.LiteralValue{V: strconv.FormatBool(value.True())}, nil
	default:
		return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
	}
}

func toResource(ctx *hcl.EvalContext, block *hclsyntax.Block) (*schemaspec2.Resource, error) {
	spec := &schemaspec2.Resource{
		Type: block.Type,
	}
	if len(block.Labels) > 0 {
		spec.Name = block.Labels[0]
	}
	attrs, err := toAttrs(ctx, block.Body.Attributes)
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs
	for _, blk := range block.Body.Blocks {
		res, err := toResource(ctx, blk)
		if err != nil {
			return nil, err
		}
		spec.Children = append(spec.Children, res)
	}
	return spec, nil
}
