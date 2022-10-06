// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// blockVar is an HCL resource that defines an input variable to the Atlas DDL document.
type blockVar struct {
	Name    string    `hcl:",label"`
	Type    cty.Value `hcl:"type"`
	Default cty.Value `hcl:"default,optional"`
}

// setInputVals sets the input values into the evaluation context. HCL documents can define
// input variables in the document body by defining "variable" blocks:
//
//	variable "name" {
//	  type = string // also supported: number, bool
//	  default = "rotemtam"
//	}
func (s *State) setInputVals(ctx *hcl.EvalContext, body hcl.Body, input map[string]cty.Value) error {
	var (
		doc struct {
			Vars   []*blockVar `hcl:"variable,block"`
			Remain hcl.Body    `hcl:",remain"`
		}
		nctx = varBlockContext(ctx)
	)
	if diag := gohcl.DecodeBody(body, nctx, &doc); diag.HasErrors() {
		return diag
	}
	ctxVars := make(map[string]cty.Value)
	for _, v := range doc.Vars {
		var vv cty.Value
		switch iv, ok := input[v.Name]; {
		case ok:
			vv = iv
		case v.Default != cty.NilVal:
			vv = v.Default
		default:
			return fmt.Errorf("missing value for required variable %q", v.Name)
		}
		vt := v.Type.EncapsulatedValue().(*cty.Type)
		// In case the input value is a primitive type and the expected type is a list,
		// wrap it as a list because the variable type may not be known to the caller.
		if vt.IsListType() && vv.Type().Equals(vt.ElementType()) {
			vv = cty.ListVal([]cty.Value{vv})
		}
		cv, err := convert.Convert(vv, *vt)
		if err != nil {
			return fmt.Errorf("variable %q: %w", v.Name, err)
		}
		ctxVars[v.Name] = cv
	}
	mergeCtxVar(ctx, ctxVars)
	return nil
}

// evalReferences evaluates data blocks.
func (s *State) evalReferences(ctx *hcl.EvalContext, body *hclsyntax.Body) error {
	type node struct {
		addr  [3]string
		edges func() []hcl.Traversal
		value func() (cty.Value, hcl.Diagnostics)
	}
	var (
		nodes  = make(map[[3]string]*node)
		blocks = make(hclsyntax.Blocks, 0, len(body.Blocks))
	)
	for _, b := range body.Blocks {
		switch b := b; b.Type {
		case dataBlock:
			if len(b.Labels) < 2 {
				return fmt.Errorf("data block %q must have exactly 2 labels", b.Type)
			}
			h, ok := s.config.datasrc[b.Labels[0]]
			if !ok {
				return fmt.Errorf("missing data source handler for %q", b.Labels[0])
			}
			// Data references are combined from
			// "data", "source" and "name" labels.
			addr := [3]string{dataBlock, b.Labels[0], b.Labels[1]}
			nodes[addr] = &node{
				addr:  addr,
				value: func() (cty.Value, hcl.Diagnostics) { return h(ctx, b) },
				edges: func() []hcl.Traversal { return bodyVars(b.Body) },
			}
		case localsBlock:
			for k, v := range b.Body.Attributes {
				k, v := k, v
				// Local references are combined from
				// "local" and "name" labels.
				addr := [3]string{localRef, k, ""}
				nodes[addr] = &node{
					addr:  addr,
					value: func() (cty.Value, hcl.Diagnostics) { return v.Expr.Value(ctx) },
					edges: func() []hcl.Traversal { return hclsyntax.Variables(v.Expr) },
				}
			}
		default:
			blocks = append(blocks, b)
		}
	}
	var (
		visit    func(n *node) error
		visited  = make(map[*node]bool)
		progress = make(map[*node]bool)
	)
	visit = func(n *node) error {
		if visited[n] {
			return nil
		}
		if progress[n] {
			addr := n.addr[:]
			if addr[2] == "" {
				addr = addr[:2]
			}
			return fmt.Errorf("cyclic reference to %q", strings.Join(addr, "."))
		}
		progress[n] = true
		for _, e := range n.edges() {
			var addr [3]string
			switch root := e.RootName(); {
			case root == localRef && len(e) == 2:
				addr = [3]string{localRef, e[1].(hcl.TraverseAttr).Name, ""}
			case root == dataBlock && len(e) > 2:
				addr = [3]string{dataBlock, e[1].(hcl.TraverseAttr).Name, e[2].(hcl.TraverseAttr).Name}
			}
			// Unrecognized reference.
			if nodes[addr] == nil {
				continue
			}
			if err := visit(nodes[addr]); err != nil {
				return err
			}
		}
		delete(progress, n)
		v, diags := n.value()
		if diags.HasErrors() {
			return diags
		}
		switch n.addr[0] {
		case dataBlock:
			data := make(map[string]cty.Value)
			if vv, ok := ctx.Variables[dataBlock]; ok {
				data = vv.AsValueMap()
			}
			src := make(map[string]cty.Value)
			if vv, ok := data[n.addr[1]]; ok {
				src = vv.AsValueMap()
			}
			src[n.addr[2]] = v
			data[n.addr[1]] = cty.ObjectVal(src)
			ctx.Variables[dataBlock] = cty.ObjectVal(data)
		case localRef:
			locals := make(map[string]cty.Value)
			if vv, ok := ctx.Variables[localRef]; ok {
				locals = vv.AsValueMap()
			}
			locals[n.addr[1]] = v
			ctx.Variables[localRef] = cty.ObjectVal(locals)
		}
		return nil
	}
	for _, n := range nodes {
		if err := visit(n); err != nil {
			return err
		}
	}
	body.Blocks = blocks
	return nil
}

func mergeCtxVar(ctx *hcl.EvalContext, vals map[string]cty.Value) {
	v, ok := ctx.Variables[varRef]
	if ok {
		v.ForEachElement(func(key cty.Value, val cty.Value) (stop bool) {
			vals[key.AsString()] = val
			return false
		})
	}
	ctx.Variables[varRef] = cty.ObjectVal(vals)
}

func setBlockVars(ctx *hcl.EvalContext, b *hclsyntax.Body) (*hcl.EvalContext, error) {
	defs := defRegistry(b)
	vars, err := blockVars(b.Blocks, "", defs)
	if err != nil {
		return nil, err
	}
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]cty.Value)
	}
	for k, v := range vars {
		ctx.Variables[k] = v
	}
	return ctx, nil
}

func blockVars(blocks hclsyntax.Blocks, parentAddr string, defs *blockDef) (map[string]cty.Value, error) {
	vars := make(map[string]cty.Value)
	for name, def := range defs.children {
		v := make(map[string]cty.Value)
		qv := make(map[string]map[string]cty.Value)
		blocks := blocksOfType(blocks, name)
		if len(blocks) == 0 {
			vars[name] = cty.NullVal(def.asCty())
			continue
		}
		var unlabeled int
		for _, blk := range blocks {
			qualifier, blkName := blockName(blk)
			if blkName == "" {
				blkName = strconv.Itoa(unlabeled)
				unlabeled++
			}
			attrs := attrMap(blk.Body.Attributes)
			// Fill missing attributes with zero values.
			for n := range def.fields {
				if _, ok := attrs[n]; !ok {
					attrs[n] = cty.NullVal(ctySchemaLit)
				}
			}
			self := addr(parentAddr, name, blkName, qualifier)
			attrs["__ref"] = cty.StringVal(self)
			varMap, err := blockVars(blk.Body.Blocks, self, def)
			if err != nil {
				return nil, err
			}
			// Merge children blocks in.
			for k, v := range varMap {
				attrs[k] = v
			}
			switch {
			case qualifier != "":
				obj := cty.ObjectVal(attrs)
				if _, ok := qv[qualifier]; !ok {
					qv[qualifier] = make(map[string]cty.Value)
				}
				qv[qualifier][blkName] = obj
				obj = cty.ObjectVal(qv[qualifier])
				v[qualifier] = obj
			default:
				v[blkName] = cty.ObjectVal(attrs)
			}
		}
		if len(v) > 0 {
			vars[name] = cty.ObjectVal(v)
		}
	}
	return vars, nil
}

func addr(parentAddr, typeName, blkName, qualifier string) string {
	var prefixDot string
	if len(parentAddr) > 0 {
		prefixDot = "."
	}
	suffix := blkName
	if qualifier != "" {
		suffix = qualifier + "." + blkName
	}
	return fmt.Sprintf("%s%s$%s.%s", parentAddr, prefixDot, typeName, suffix)
}

func blockName(blk *hclsyntax.Block) (qualifier string, name string) {
	switch len(blk.Labels) {
	case 0:
	case 1:
		name = blk.Labels[0]
	default:
		qualifier = blk.Labels[0]
		name = blk.Labels[1]
	}
	return
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
		value, diag := v.Expr.Value(nil)
		if diag.HasErrors() {
			continue
		}
		literalValue, err := extractValue(value)
		if err != nil {
			continue
		}
		out[v.Name] = cty.CapsuleVal(ctySchemaLit, literalValue)
	}
	return out
}

// ctySchemaLit is a cty.Capsule type the encapsulates a schemahcl.LiteralValue.
var (
	ctySchemaLit = cty.CapsuleWithOps("lit", reflect.TypeOf(LiteralValue{}), &cty.CapsuleOps{
		// ConversionFrom facilitates reading the encapsulated type as a string, as is needed, for example,
		// when interpolating it in a string expression.
		ConversionFrom: func(src cty.Type) func(any, cty.Path) (cty.Value, error) {
			if src != cty.String {
				return nil
			}
			return func(i any, path cty.Path) (cty.Value, error) {
				lit, ok := i.(*LiteralValue)
				if !ok {
					return cty.Value{}, fmt.Errorf("schemahcl: expected *schemahcl.LiteralValue got %T", i)
				}
				uq, err := strconv.Unquote(lit.V)
				if err != nil {
					return cty.StringVal(lit.V), nil
				}
				return cty.StringVal(uq), nil
			}
		},
	})
	ctyNilType  = cty.Capsule("type", reflect.TypeOf(cty.NilType))
	ctyTypeSpec = cty.Capsule("type", reflect.TypeOf(Type{}))
	ctyRawExpr  = cty.Capsule("raw_expr", reflect.TypeOf(RawExpr{}))
)

// Built-in blocks.
const (
	varBlock     = "variable"
	dataBlock    = "data"
	localsBlock  = "locals"
	varRef       = "var"
	localRef     = "local"
	dynamicBlock = "dynamic"
	forEachAttr  = "for_each"
)

// defRegistry returns a tree of blockDef structs representing the schema of the
// blocks in the *hclsyntax.Body. The returned fields and children of each type
// are an intersection of all existing blocks of the same type.
func defRegistry(b *hclsyntax.Body) *blockDef {
	reg := &blockDef{
		fields:   make(map[string]struct{}),
		children: make(map[string]*blockDef),
	}
	for _, blk := range b.Blocks {
		// variable definition blocks are available in the HCL source but not reachable by reference.
		if blk.Type == varBlock {
			continue
		}
		reg.child(extractDef(blk, reg))
	}
	return reg
}

// blockDef describes a type of block in the HCL document.
type blockDef struct {
	name     string
	fields   map[string]struct{}
	parent   *blockDef
	children map[string]*blockDef
}

// child updates the definition for the child type of the blockDef.
func (t *blockDef) child(c *blockDef) {
	ex, ok := t.children[c.name]
	if !ok {
		t.children[c.name] = c
		return
	}
	for f := range c.fields {
		ex.fields[f] = struct{}{}
	}
	for _, c := range c.children {
		ex.child(c)
	}
}

// asCty returns a cty.Type representing the blockDef.
func (t *blockDef) asCty() cty.Type {
	f := make(map[string]cty.Type)
	for attr := range t.fields {
		f[attr] = ctySchemaLit
	}
	f["__ref"] = cty.String
	for _, c := range t.children {
		f[c.name] = c.asCty()
	}
	return cty.Object(f)
}

func extractDef(blk *hclsyntax.Block, parent *blockDef) *blockDef {
	cur := &blockDef{
		name:     blk.Type,
		parent:   parent,
		fields:   make(map[string]struct{}),
		children: make(map[string]*blockDef),
	}
	for _, a := range blk.Body.Attributes {
		cur.fields[a.Name] = struct{}{}
	}
	for _, c := range blk.Body.Blocks {
		cur.child(extractDef(c, cur))
	}
	return cur
}

func bodyVars(b *hclsyntax.Body) (vars []hcl.Traversal) {
	for _, attr := range b.Attributes {
		vars = append(vars, hclsyntax.Variables(attr.Expr)...)
	}
	for _, b := range b.Blocks {
		vars = append(vars, bodyVars(b.Body)...)
	}
	return
}
