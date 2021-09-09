package schemahcl

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"ariga.io/atlas/schema/schemaspec"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type container struct {
	Body hcl.Body `hcl:",remain"`
}

func decode(body []byte) (*schemaspec.Resource, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, "")
	if diag.HasErrors() {
		return nil, diag
	}
	if srcHCL == nil {
		return nil, fmt.Errorf("schemahcl: no HCL syntax found in body")
	}
	c := &container{}
	ctx, err := evalCtx(srcHCL)
	if err != nil {
		return nil, err
	}
	if diag := gohcl.DecodeBody(srcHCL.Body, ctx, c); diag.HasErrors() {
		return nil, diag
	}
	return extract(ctx, c.Body)
}

func extract(ctx *hcl.EvalContext, remain hcl.Body) (*schemaspec.Resource, error) {
	body, ok := remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(ctx, body.Attributes)
	if err != nil {
		return nil, err
	}
	res := &schemaspec.Resource{
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

func toAttrs(ctx *hcl.EvalContext, hclAttrs hclsyntax.Attributes) ([]*schemaspec.Attr, error) {
	var attrs []*schemaspec.Attr
	for _, hclAttr := range hclAttrs {
		at := &schemaspec.Attr{K: hclAttr.Name}
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

func extractListValue(value cty.Value) (*schemaspec.ListValue, error) {
	lst := &schemaspec.ListValue{}
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

func extractLiteralValue(value cty.Value) (*schemaspec.LiteralValue, error) {
	switch value.Type() {
	case cty.String:
		return &schemaspec.LiteralValue{V: strconv.Quote(value.AsString())}, nil
	case cty.Number:
		bf := value.AsBigFloat()
		num, _ := bf.Float64()
		return &schemaspec.LiteralValue{V: strconv.FormatFloat(num, 'f', -1, 64)}, nil
	case cty.Bool:
		return &schemaspec.LiteralValue{V: strconv.FormatBool(value.True())}, nil
	default:
		return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
	}
}

func toResource(ctx *hcl.EvalContext, block *hclsyntax.Block) (*schemaspec.Resource, error) {
	spec := &schemaspec.Resource{
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

func encode(r *schemaspec.Resource) ([]byte, error) {
	f := hclwrite.NewFile()
	body := f.Body()
	for _, attr := range r.Attrs {
		if err := writeAttr(attr, body); err != nil {
			return nil, err
		}
	}
	for _, res := range r.Children {
		if err := writeResource(res, body); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func writeResource(b *schemaspec.Resource, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, []string{b.Name})
	nb := blk.Body()
	for _, attr := range b.Attrs {
		if err := writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, b := range b.Children {
		if err := writeResource(b, nb); err != nil {
			return err
		}
	}
	return nil
}

func writeAttr(attr *schemaspec.Attr, body *hclwrite.Body) error {
	switch v := attr.V.(type) {
	case *schemaspec.LiteralValue:
		body.SetAttributeRaw(attr.K, hclRawTokens(v.V))
	case *schemaspec.ListValue:
		body.SetAttributeRaw(attr.K, hclRawList(v.V))
	default:
		return fmt.Errorf("schemacl: unknown literal type %T", v)
	}
	return nil
}

func hclRawTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(s),
		},
	}
}

func hclRawList(items []string) hclwrite.Tokens {
	t := hclwrite.Tokens{&hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte("["),
	}}
	for _, item := range items {
		t = append(t, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(item),
		}, &hclwrite.Token{
			Type:  hclsyntax.TokenComma,
			Bytes: []byte(","),
		})
	}
	t = append(t, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte("]"),
	})
	return t
}
