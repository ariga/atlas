package schemahcl

import (
	"fmt"
	"strconv"

	"ariga.io/atlas/sql/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Decode implements schema.Decoder. It parses an HCL document describing a schema Spec into spec.
func Decode(body []byte, spec schema.Spec) error {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, "in-memory.hcl")
	if diag.HasErrors() {
		return diag
	}
	if srcHCL == nil {
		return fmt.Errorf("schemahcl: contents is nil")
	}
	if tgt, ok := spec.(*schema.SchemaSpec); ok {
		f := &schemaFile{}
		if diag := gohcl.DecodeBody(srcHCL.Body, nil, f); diag.HasErrors() {
			return diag
		}
		for _, tbl := range f.Tables {
			spec, err := tbl.spec()
			if err != nil {
				return err
			}
			tgt.Tables = append(tgt.Tables, spec)
		}
		return nil
	}
	return fmt.Errorf("schemahcl: unsupported spec type %T", spec)
}

type (
	schemaFile struct {
		Tables []*table `hcl:"table,block"`
		Remain hcl.Body `hcl:",remain"`
	}
	table struct {
		Name    string    `hcl:",label"`
		Columns []*column `hcl:"column,block"`
		Remain  hcl.Body  `hcl:",remain"`
	}
	column struct {
		Name     string   `hcl:",label"`
		TypeName string   `hcl:"type"`
		Null     bool     `hcl:"null,optional"`
		Default  *string  `hcl:"default,optional"`
		Remain   hcl.Body `hcl:",remain"`
	}
)

func (t *table) spec() (*schema.TableSpec, error) {
	out := &schema.TableSpec{
		Name: t.Name,
	}
	for _, col := range t.Columns {
		cs, err := col.spec()
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, cs)
	}
	body, ok := t.Remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(body.Attributes, skipNone())
	if err != nil {
		return nil, err
	}
	out.Attrs = attrs
	return out, nil
}

func (c *column) spec() (*schema.ColumnSpec, error) {
	spec := &schema.ColumnSpec{
		Name:    c.Name,
		Type:    c.TypeName,
		Null:    c.Null,
		Default: c.Default,
	}
	body, ok := c.Remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(body.Attributes, skip("type", "default", "null"))
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs

	for _, blk := range body.Blocks {
		resource, err := toResource(blk)
		if err != nil {
			return nil, err
		}
		spec.Children = append(spec.Children, resource)
	}
	return spec, nil
}

func skip(lst ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(lst))
	for _, item := range lst {
		out[item] = struct{}{}
	}
	return out
}

func skipNone() map[string]struct{} {
	return skip()
}

func toAttrs(hclAttrs hclsyntax.Attributes, skip map[string]struct{}) ([]*schema.SpecAttr, error) {
	var attrs []*schema.SpecAttr
	for _, hclAttr := range hclAttrs {
		if _, ok := skip[hclAttr.Name]; ok {
			continue
		}
		at := &schema.SpecAttr{K: hclAttr.Name}
		value, diag := hclAttr.Expr.Value(nil)
		if diag.HasErrors() {
			return nil, diag
		}
		switch value.Type().GoString() {
		case "cty.String":
			at.V = &schema.SpecLiteral{V: strconv.Quote(value.AsString())}
		case "cty.Number":
			bf := value.AsBigFloat()
			num, _ := bf.Float64()
			at.V = &schema.SpecLiteral{V: fmt.Sprintf("%f", num)}
		case "cty.Bool":
			at.V = &schema.SpecLiteral{V: strconv.FormatBool(value.True())}
		default:
			return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
		}
		attrs = append(attrs, at)
	}
	return attrs, nil
}

func toResource(block *hclsyntax.Block) (*schema.ResourceSpec, error) {
	spec := &schema.ResourceSpec{
		Type: block.Type,
	}
	if len(block.Labels) > 0 {
		spec.Name = block.Labels[0]
	}
	attrs, err := toAttrs(block.Body.Attributes, skipNone())
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs
	for _, blk := range block.Body.Blocks {
		res, err := toResource(blk)
		if err != nil {
			return nil, err
		}
		spec.Children = append(spec.Children, res)
	}
	return spec, nil
}
