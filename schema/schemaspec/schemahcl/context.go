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
func evalCtx(f *hcl.File) (*hcl.EvalContext, error) {
	c := &container{}
	if diag := gohcl.DecodeBody(f.Body, &hcl.EvalContext{}, c); diag.HasErrors() {
		return nil, diag
	}
	b, ok := c.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected an hcl body")
	}
	vars, err := blockVars(b, "")
	if err != nil {
		return nil, err
	}
	return &hcl.EvalContext{
		Variables: vars,
	}, nil
}

func blockVars(b *hclsyntax.Body, parentAddr string) (map[string]cty.Value, error) {
	types, err := typeDefs(b)
	if err != nil {
		return nil, fmt.Errorf("schemahcl: failed extracting type definitions: %w", err)
	}
	vars := make(map[string]cty.Value)
	for typeName, typ := range types {
		v := make(map[string]cty.Value)
		for _, blk := range blocksOfType(b.Blocks, typeName) {
			blkName := blockName(blk)
			if blkName == "" {
				continue
			}
			attrs := attrMap(blk.Body.Attributes)
			// Fill missing attributes with zero values.
			for n, t := range typ.AttributeTypes() {
				if _, ok := attrs[n]; !ok {
					attrs[n] = cty.NullVal(t)
				}
			}
			self := addr(parentAddr, typeName, blkName)
			attrs["__ref"] = cty.StringVal(self)
			varMap, err := blockVars(blk.Body, self)
			if err != nil {
				return nil, err
			}
			// Merge children blocks in.
			for k, v := range varMap {
				attrs[k] = v
			}

			v[blkName] = cty.ObjectVal(attrs)
		}
		if len(v) > 0 {
			vars[typeName] = cty.MapVal(v)
		}
	}
	return vars, nil
}

func addr(parentAddr, typeName, blkName string) string {
	var prefixDot string
	if len(parentAddr) > 0 {
		prefixDot = "."
	}
	return fmt.Sprintf("%s%s$%s.%s", parentAddr, prefixDot, typeName, blkName)
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

func typeDefs(b *hclsyntax.Body) (map[string]cty.Type, error) {
	types, err := extractTypes(b)
	if err != nil {
		return nil, err
	}
	objs := make(map[string]cty.Type)
	for k, v := range types {
		objs[k] = cty.Object(v)
	}
	return objs, nil
}

// extractTypes returns a map of block types a block. Types are computed
// as an intersection fields in all instances. If conflicting field types are encountered
// an error is returned.
// TODO(rotemtam): type definitions should be fed into the hcl parser from the plugin system.
func extractTypes(b *hclsyntax.Body) (map[string]map[string]cty.Type, error) {
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
	return types, nil
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
