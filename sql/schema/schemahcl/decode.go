package schemahcl

import (
	"fmt"

	"ariga.io/atlas/sql/schema/schemacodec"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func Decode(body []byte, spec schemacodec.Spec) error {
	parser := hclparse.NewParser()
	srcHCL, diag := parser.ParseHCL(body, "")
	if diag.HasErrors() {
		return diag
	}
	if srcHCL == nil {
		return fmt.Errorf("schema: contents is nil")
	}
	switch tgt := spec.(type) {
	case *schemacodec.SchemaSpec:
		f := &schemaFileHCL{}
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
		// case *schemacodec.MigrationSpec:
	}
	return nil
}

type tableHCL struct {
	Name    string    `hcl:",label"`
	Columns []*colHCL `hcl:"column,block"`
}

func (t *tableHCL) spec() (*schemacodec.TableSpec, error) {
	out := &schemacodec.TableSpec{
		Name: t.Name,
	}
	for _, col := range t.Columns {
		cs, err := col.spec()
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, cs)
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

func (c *colHCL) spec() (*schemacodec.ColumnSpec, error) {
	out := &schemacodec.ColumnSpec{
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
		attrs, err := toAttrs(block.Body.Attributes, skip())
		if err != nil {
			return nil, err
		}
		bl := &schemacodec.Block{
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

func toAttrs(attrs hclsyntax.Attributes, skip map[string]struct{}) ([]*schemacodec.Attr, error) {
	var out []*schemacodec.Attr
	for _, attr := range attrs {
		if _, ok := skip[attr.Name]; ok {
			continue
		}
		at := &schemacodec.Attr{K: attr.Name}
		value, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			return nil, diag
		}
		switch value.Type().GoString() {
		case "cty.String":
			at.V = schemacodec.String(value.AsString())
		case "cty.Number":
			bf := value.AsBigFloat()
			num, _ := bf.Float64()
			at.V = schemacodec.Number(num)
		case "cty.Bool":
			at.V = schemacodec.Bool(value.True())
		default:
			return nil, fmt.Errorf("schemahcl: unsupported type %q", value.Type().GoString())
		}
		out = append(out, at)
	}
	return out, nil
}

type schemaFileHCL struct {
	Tables []*tableHCL `hcl:"table,block"`
}
