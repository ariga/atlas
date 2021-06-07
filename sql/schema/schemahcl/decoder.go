package schemahcl

import (
	"fmt"

	"ariga.io/atlas/sql/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Decode(body []byte, spec schema.Spec) error {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, "in-memory.hcl")
	if diag.HasErrors() {
		return diag
	}
	if srcHCL == nil {
		return fmt.Errorf("schemahcl: contents is nil")
	}
	switch tgt := spec.(type) {
	case *schema.SchemaSpec:
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
		// TODO:
		// case *schema.MigrationSpec:
	}
	return nil
}

type table struct {
	Name    string    `hcl:",label"`
	Columns []*column `hcl:"column,block"`
	Remain  hcl.Body  `hcl:",remain"`
}

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

type column struct {
	Name     string   `hcl:",label"`
	TypeName string   `hcl:"type"`
	Null     bool     `hcl:"null,optional"`
	Default  *string  `hcl:"default,optional"`
	Remain   hcl.Body `hcl:",remain"`
}

func (c *column) spec() (*schema.ColumnSpec, error) {
	out := &schema.ColumnSpec{
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

	// todo: extract and recurse
	for _, block := range body.Blocks {
		attrs, err := toAttrs(block.Body.Attributes, skipNone())
		if err != nil {
			return nil, err
		}
		var name string
		if len(block.Labels) > 0 {
			name = block.Labels[0]
		}
		spc := &schema.ResourceSpec{
			Type:     block.Type,
			Name:     name,
			Attrs:    attrs,
			Children: nil,
		}
		out.Children = append(out.Children, spc)
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

func skipNone() map[string]struct{} {
	return skip()
}

func toAttrs(attrs hclsyntax.Attributes, skip map[string]struct{}) ([]*schema.SpecAttr, error) {
	var out []*schema.SpecAttr
	for _, attr := range attrs {
		if _, ok := skip[attr.Name]; ok {
			continue
		}
		at := &schema.SpecAttr{K: attr.Name}
		value, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			return nil, diag
		}
		switch value.Type().GoString() {
		case "cty.String":
			at.V = schema.String(value.AsString())
		case "cty.Number":
			bf := value.AsBigFloat()
			num, _ := bf.Float64()
			at.V = schema.Number(num)
		case "cty.Bool":
			at.V = schema.Bool(value.True())
		default:
			return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
		}
		out = append(out, at)
	}
	return out, nil
}

type schemaFile struct {
	Tables []*table `hcl:"table,block"`
	Remain hcl.Body `hcl:",remain"`
}
