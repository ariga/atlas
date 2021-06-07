package schemahcl

import (
	"fmt"

	"ariga.io/atlas/sql/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
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
	ctx, err := evalContext(srcHCL)
	if err != nil {
		return err
	}
	if tgt, ok := spec.(*schema.SchemaSpec); ok {
		f := &schemaFile{}
		if diag := gohcl.DecodeBody(srcHCL.Body, ctx, f); diag.HasErrors() {
			return diag
		}
		for _, tbl := range f.Tables {
			spec, err := tbl.spec(ctx)
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
		Schema  *schemaRef `hcl:"schema,optional"`
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

func (t *table) spec(ctx *hcl.EvalContext) (*schema.TableSpec, error) {
	out := &schema.TableSpec{
		Name:       t.Name,
		SchemaName: t.Schema.Name,
	}
	for _, col := range t.Columns {
		cs, err := col.spec(ctx)
		if err != nil {
			return nil, err
		}
		out.Columns = append(out.Columns, cs)
	}
	body, ok := t.Remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(ctx, body.Attributes, skip("schema"))
	if err != nil {
		return nil, err
	}
	out.Attrs = attrs
	return out, nil
}

func (c *column) spec(ctx *hcl.EvalContext) (*schema.ColumnSpec, error) {
	spec := &schema.ColumnSpec{
		Name:     c.Name,
		TypeName: c.TypeName,
		Null:     c.Null,
		Default:  c.Default,
	}
	body, ok := c.Remain.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := toAttrs(ctx, body.Attributes, skip("type", "default", "null"))
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs

	for _, blk := range body.Blocks {
		resource, err := toResource(ctx, blk)
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

func toAttrs(ctx *hcl.EvalContext, hclAttrs hclsyntax.Attributes, skip map[string]struct{}) ([]*schema.SpecAttr, error) {
	var attrs []*schema.SpecAttr
	for _, hclAttr := range hclAttrs {
		if _, ok := skip[hclAttr.Name]; ok {
			continue
		}
		at := &schema.SpecAttr{K: hclAttr.Name}
		value, diag := hclAttr.Expr.Value(ctx)
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
		attrs = append(attrs, at)
	}
	return attrs, nil
}

func toResource(ctx *hcl.EvalContext, block *hclsyntax.Block) (*schema.ResourceSpec, error) {
	spec := &schema.ResourceSpec{
		Type: block.Type,
	}
	if len(block.Labels) > 0 {
		spec.Name = block.Labels[0]
	}
	attrs, err := toAttrs(ctx, block.Body.Attributes, skipNone())
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

// evalContext does an initial pass through the hcl.File f and returns a context with populated
// variables that can be used in the actual file evaluation
func evalContext(f *hcl.File) (*hcl.EvalContext, error) {
	var fi struct {
		Schemas []struct {
			Name string `hcl:",label"`
		} `hcl:"schema,block"`
		Tables []struct {
			Name    string `hcl:",label"`
			Columns []struct {
				Name   string   `hcl:",label"`
				Remain hcl.Body `hcl:",remain"`
			} `hcl:"column,block"`
			Remain hcl.Body `hcl:",remain"`
		} `hcl:"table,block"`
		Remain hcl.Body `hcl:",remain"`
	}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, &fi); diag.HasErrors() {
		return nil, diag
	}
	schemas := make(map[string]cty.Value)
	for _, sch := range fi.Schemas {
		ref, err := toSchemaRef(sch.Name)
		if err != nil {
			return nil, fmt.Errorf("schema: failed creating ref to schema %q", sch.Name)
		}
		schemas[sch.Name] = ref
	}
	tables := make(map[string]cty.Value)
	for _, tab := range fi.Tables {
		cols := make(map[string]cty.Value)
		for _, col := range tab.Columns {
			ref, err := toColumnRef(tab.Name, col.Name)
			if err != nil {
				return nil, fmt.Errorf("schema: failed ref for column %q in table %q", col.Name, tab.Name)
			}
			cols[col.Name] = ref
		}
		tables[tab.Name] = cty.ObjectVal(map[string]cty.Value{
			"column": cty.MapVal(cols),
		})
	}
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"schema": cty.MapVal(schemas),
			"table":  cty.MapVal(tables),
			"reference_option": cty.MapVal(map[string]cty.Value{
				"no_action":   cty.StringVal(string(schema.NoAction)),
				"restrict":    cty.StringVal(string(schema.Restrict)),
				"cascade":     cty.StringVal(string(schema.Cascade)),
				"set_null":    cty.StringVal(string(schema.SetNull)),
				"set_default": cty.StringVal(string(schema.SetDefault)),
			}),
		},
	}, nil
}

type schemaRef struct {
	Name string `cty:"name"`
}

func toSchemaRef(name string) (cty.Value, error) {
	typ := cty.Object(map[string]cty.Type{
		"name": cty.String,
	})
	s := &schemaRef{Name: name}
	return gocty.ToCtyValue(s, typ)
}

type columnRef struct {
	Name  string `cty:"name"`
	Table string `cty:"table"`
}

func toColumnRef(table, column string) (cty.Value, error) {
	typ := cty.Object(map[string]cty.Type{
		"name":  cty.String,
		"table": cty.String,
	})
	c := columnRef{
		Name:  column,
		Table: table,
	}
	return gocty.ToCtyValue(c, typ)
}
