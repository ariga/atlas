package schemahcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// evalCtx constructs an *hcl.EvalContext with the Variables field populated with per
// block type reference maps that can be used in the HCL file evaluation. For example,
// if the evaluated file contains blocks such as:
//	greeting "english" {
//		word = "hello"
//	}
//	greeting "hebrew" {
//		word = "shalom"
//	}
//
// They can be then referenced in other blocks:
//	message "welcome_hebrew" {
//		title = "{greeting.hebrew.word}, world!"
//	}
//
// This currently only works for the top level blocks, nested blocks and their
// attributes are not loaded into the eval context.
// TODO(rotemtam): support nested blocks.
func evalCtx(f *hcl.File) (*hcl.EvalContext, error) {
	c := &container{}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, c); diag.HasErrors() {
		return nil, diag
	}
	b, ok := c.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected an hcl body")
	}
	types, err := extractTypes(b)
	if err != nil {
		return nil, fmt.Errorf("schemahcl: failed extracting type definitions: %w", err)
	}
	out := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
	}
	for n, typ := range types {
		v := make(map[string]cty.Value)
		for _, blk := range blocksOfType(b.Blocks, n) {
			name := blockName(blk)
			if name == "" {
				continue
			}
			attrs := attrMap(blk.Body.Attributes)
			// Fill missing attributes with zero values.
			for n, t := range typ.AttributeTypes() {
				if _, ok := attrs[n]; !ok {
					attrs[n] = cty.NullVal(t)
				}
			}
			v[name] = cty.ObjectVal(attrs)
		}
		out.Variables[n] = cty.MapVal(v)
	}
	return out, nil
}

func blockName(blk *hclsyntax.Block) string {
	if len(blk.Labels) == 0 {
		return ""
	}
	return blk.Labels[0]
}

func blocksOfType(blocks hclsyntax.Blocks, typeName string) []*hclsyntax.Block {
	var out []*hclsyntax.Block
	for _, block := range blocks {
		if block.Type == typeName {
			out = append(out, block)
		}
	}
	return out
}

func attrMap(attrs hclsyntax.Attributes) map[string]cty.Value {
	out := make(map[string]cty.Value)
	for _, v := range attrs {
		value, err := v.Expr.Value(nil)
		if err != nil {
			continue
		}
		out[v.Name] = value
	}
	return out
}

// extractTypes returns a map of block types in the document. Types are computed
// as an intersection fields in all instances. If conflicting field types are encountered
// an error is returned.
// TODO(rotemtam): type definitions should be fed into the hcl parser from the plugin system.
func extractTypes(b *hclsyntax.Body) (map[string]cty.Type, error) {
	types := make(map[string]map[string]cty.Type)
	for _, blk := range b.Blocks {
		attrTypes := extractAttrTypes(&hcl.EvalContext{}, blk)
		if _, ok := types[blk.Type]; !ok {
			types[blk.Type] = attrTypes
			continue
		}
		if err := mergeAttrTypes(types[blk.Type], attrTypes); err != nil {
			return nil, err
		}
	}
	out := make(map[string]cty.Type)
	for k, v := range types {
		out[k] = cty.Object(v)
	}
	return out, nil
}

func mergeAttrTypes(target, other map[string]cty.Type) error {
	for k, v := range other {
		targetV, exists := target[k]
		if !exists {
			target[k] = v
			continue
		}
		if !v.Equals(targetV) {
			return fmt.Errorf("schemahcl: field %q must have a constant type. %q!=%q", k, targetV.GoString(), v.GoString())
		}
	}
	return nil
}

func extractAttrTypes(ctx *hcl.EvalContext, blk *hclsyntax.Block) map[string]cty.Type {
	attrTypes := make(map[string]cty.Type)
	for _, attr := range blk.Body.Attributes {
		value, err := attr.Expr.Value(ctx)
		if err != nil {
			continue
		}
		attrTypes[attr.Name] = value.Type()
	}
	return attrTypes
}
