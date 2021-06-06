package schemahcl

import (
	"fmt"

	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Parse(body []byte, filename string) ([]*schemaspec.Column, error) {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, filename)
	if diag.HasErrors() {
		return nil, diag
	}
	if srcHCL == nil {
		return nil, fmt.Errorf("schema: file %q contents is nil", filename)
	}
	f := &fileHCL{}
	if diag := gohcl.DecodeBody(srcHCL.Body, nil, f); diag.HasErrors() {
		return nil, diag
	}
	var out []*schemaspec.Column
	for _, c := range f.Columns {
		spec, err := c.spec()
		if err != nil {
			return nil, err
		}
		out = append(out, spec)
	}
	return out, nil
}

type colHCL struct {
	Name     string   `hcl:",label"`
	TypeName string   `hcl:"type"`
	Null     bool     `hcl:"null,optional"`
	Default  *string  `hcl:"default,optional"`
	Remain   hcl.Body `hcl:",remain"`
}

func (c *colHCL) spec() (*schemaspec.Column, error) {
	out := &schemaspec.Column{
		Name:     c.Name,
		TypeName: c.TypeName,
		Null:     c.Null,
		Default:  c.Default,
	}
	body, ok := c.Remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(body.Attributes, skip("type", "default", "null"))
	if err != nil {
		return nil, err
	}
	out.Attrs = attrs
	for _, block := range body.Blocks {
		attrs, err := toAttrs(block.Body.Attributes, skip())
		if err != nil {
			return nil, err
		}
		bl := &schemaspec.Block{
			Type:   block.Type,
			Labels: block.Labels,
			Attrs:  attrs,
			Blocks: nil,
		}
		out.Blocks = append(out.Blocks, bl)
	}
	return out, nil
}

func skip(lst ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(lst))
	for _, item := range lst {
		out[item] = struct{}{}
	}
	return out
}

func toAttrs(attrs hclsyntax.Attributes, skip map[string]struct{}) ([]*schemaspec.Attr, error) {
	var out []*schemaspec.Attr
	for _, attr := range attrs {
		if _, ok := skip[attr.Name]; ok {
			continue
		}
		at := &schemaspec.Attr{K: attr.Name}
		value, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			return nil, diag
		}
		switch value.Type().GoString() {
		case "cty.String":
			at.V = schemaspec.String(value.AsString())
		case "cty.Number":
			bf := value.AsBigFloat()
			num, _ := bf.Float64()
			at.V = schemaspec.Number(num)
		case "cty.Bool":
			at.V = schemaspec.Bool(value.True())
		default:
			return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
		}
		out = append(out, at)
	}
	return out, nil
}

type fileHCL struct {
	Columns []*colHCL `hcl:"column,block"`
}
