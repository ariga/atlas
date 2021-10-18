package schemahcl

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"ariga.io/atlas/schema/schemaspec"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Default is a default implementation of schemaspec.Marshaler and schemaspec.Unmarshaler
// for Atlas HCL documents.
var Default = &defaultCodec{}

type defaultCodec struct {
}

// UnmarshalSpec implements schemaspec.Unmarshaler by invoking Unmarshal.
func (*defaultCodec) UnmarshalSpec(data []byte, v interface{}) error {
	return Unmarshal(data, v)
}

// MarshalSpec implements schemaspec.Marshaler by invoking Marshal.
func (m *defaultCodec) MarshalSpec(v interface{}) ([]byte, error) {
	return Marshal(v)
}

// Marshal returns the Atlas HCL encoding of v.
func Marshal(v interface{}) ([]byte, error) {
	r := &schemaspec.Resource{}
	if err := r.Scan(v); err != nil {
		return nil, fmt.Errorf("schemahcl: failed scanning %T to resource: %w", v, err)
	}
	return encode(r)
}

// Unmarshal parses the Atlas HCL-encoded data and stores the result in the target.
func Unmarshal(data []byte, v interface{}) error {
	spec, err := decode(data)
	if err != nil {
		return fmt.Errorf("schemahcl: failed decoding: %w", err)
	}
	if err := spec.As(v); err != nil {
		return fmt.Errorf("schemahcl: failed reading spec as %T: %w", v, err)
	}
	return nil
}

type container struct {
	Body hcl.Body `hcl:",remain"`
}

// decode decodes the input Atlas HCL document and returns a *schemaspec.Resource representing it.
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
		switch {
		case isRef(value):
			at.V = &schemaspec.Ref{V: value.GetAttr("__ref").AsString()}
		case value.Type().IsTupleType():
			at.V, err = extractListValue(value)
		default:
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

func isRef(v cty.Value) bool {
	return v.Type().IsObjectType() && v.Type().HasAttribute("__ref")
}

func extractListValue(value cty.Value) (*schemaspec.ListValue, error) {
	lst := &schemaspec.ListValue{}
	it := value.ElementIterator()
	for it.Next() {
		_, v := it.Element()
		if isRef(v) {
			lst.V = append(lst.V, &schemaspec.Ref{V: v.GetAttr("__ref").AsString()})
			continue
		}
		litv, err := extractLiteralValue(v)
		if err != nil {
			return nil, err
		}
		lst.V = append(lst.V, litv)
	}
	return lst, nil
}

func extractLiteralValue(value cty.Value) (*schemaspec.LiteralValue, error) {
	switch value.Type() {
	case ctySchemaLit:
		return value.EncapsulatedValue().(*schemaspec.LiteralValue), nil
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

// encode encodes the give *schemaspec.Resource into a byte slice containing an Atlas HCL
// document representing it.
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
	var labels []string
	if b.Name != "" {
		labels = append(labels, b.Name)
	}
	blk := body.AppendNewBlock(b.Type, labels)
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
	case *schemaspec.Ref:
		expr := strings.ReplaceAll(v.V, "$", "")
		body.SetAttributeRaw(attr.K, hclRawTokens(expr))
	case *schemaspec.LiteralValue:
		body.SetAttributeRaw(attr.K, hclRawTokens(v.V))
	case *schemaspec.ListValue:
		lst := make([]string, 0, len(v.V))
		for _, item := range v.V {
			val, err := schemaspec.StrVal(item)
			if err != nil {
				return err
			}
			lst = append(lst, val)
		}
		body.SetAttributeRaw(attr.K, hclRawList(lst))
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
