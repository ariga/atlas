// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// Marshal returns the Atlas HCL encoding of v.
var Marshal = MarshalerFunc(New().MarshalSpec)

type (
	// State is used to evaluate and marshal Atlas HCL documents and stores a configuration for these operations.
	State struct {
		config *Config
	}
	// Evaluator is the interface that wraps the Eval function.
	Evaluator interface {
		// Eval evaluates parsed HCL files using input variables into a schema.Realm.
		Eval(*hclparse.Parser, any, map[string]cty.Value) error
	}
	// EvalFunc is an adapter that allows the use of an ordinary function as an Evaluator.
	EvalFunc func(*hclparse.Parser, any, map[string]cty.Value) error
	// Marshaler is the interface that wraps the MarshalSpec function.
	Marshaler interface {
		// MarshalSpec marshals the provided input into a valid Atlas HCL document.
		MarshalSpec(any) ([]byte, error)
	}
	// MarshalerFunc is the function type that is implemented by the MarshalSpec
	// method of the Marshaler interface.
	MarshalerFunc func(any) ([]byte, error)
)

// MarshalSpec implements Marshaler for Atlas HCL documents.
func (s *State) MarshalSpec(v any) ([]byte, error) {
	r := &Resource{}
	if err := r.Scan(v); err != nil {
		return nil, fmt.Errorf("schemahcl: failed scanning %T to resource: %w", v, err)
	}
	return s.encode(r)
}

// EvalFiles evaluates the files in the provided paths using the input variables and
// populates v with the result.
func (s *State) EvalFiles(paths []string, v any, input map[string]cty.Value) error {
	parser := hclparse.NewParser()
	for _, path := range paths {
		if _, diag := parser.ParseHCLFile(path); diag.HasErrors() {
			return diag
		}
	}
	return s.Eval(parser, v, input)
}

// Eval evaluates the parsed HCL documents using the input variables and populates v
// using the result.
func (s *State) Eval(parsed *hclparse.Parser, v any, input map[string]cty.Value) error {
	ctx := s.config.newCtx()
	reg := &blockDef{
		fields:   make(map[string]struct{}),
		children: make(map[string]*blockDef),
	}
	files := parsed.Files()
	fileNames := make([]string, 0, len(files))
	allBlocks := make([]*hclsyntax.Block, 0, len(files))
	for name, file := range files {
		fileNames = append(fileNames, name)
		if err := s.setInputVals(ctx, file.Body, input); err != nil {
			return err
		}
		body := file.Body.(*hclsyntax.Body)
		if err := s.evalReferences(ctx, body); err != nil {
			return err
		}
		blocks := make(hclsyntax.Blocks, 0, len(body.Blocks))
		for _, b := range body.Blocks {
			switch {
			// Variable blocks are not reachable by reference.
			case b.Type == varBlock:
				continue
			// Semi-evaluate blocks with the for_each meta argument.
			case b.Body != nil && b.Body.Attributes[forEachAttr] != nil:
				nb, err := forEachBlocks(ctx, b)
				if err != nil {
					return err
				}
				blocks = append(blocks, nb...)
			default:
				blocks = append(blocks, b)
			}
			reg.child(extractDef(b, reg))
		}
		body.Blocks = blocks
		allBlocks = append(allBlocks, blocks...)
	}
	vars, err := blockVars(allBlocks, "", reg)
	if err != nil {
		return err
	}
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]cty.Value)
	}
	for k, v := range vars {
		ctx.Variables[k] = v
	}
	spec := &Resource{}
	sort.Slice(fileNames, func(i, j int) bool {
		return fileNames[i] < fileNames[j]
	})
	for _, fn := range fileNames {
		file := files[fn]
		r, err := s.resource(ctx, file)
		if err != nil {
			return err
		}
		spec.Children = append(spec.Children, r.Children...)
		spec.Attrs = append(spec.Attrs, r.Attrs...)
	}
	if err := patchRefs(spec); err != nil {
		return err
	}
	if err := spec.As(v); err != nil {
		return fmt.Errorf("schemahcl: failed reading spec as %T: %w", v, err)
	}
	return nil
}

// EvalBytes evaluates the data byte-slice as an Atlas HCL document using the input variables
// and stores the result in v.
func (s *State) EvalBytes(data []byte, v any, input map[string]cty.Value) error {
	parser := hclparse.NewParser()
	if _, diag := parser.ParseHCL(data, ""); diag.HasErrors() {
		return diag
	}
	return s.Eval(parser, v, input)
}

// addrRef maps addresses to their referenced resource.
type addrRef map[string]*Resource

// patchRefs recursively searches for schemahcl.Ref under the provided schemahcl.Resource
// and patches any variables with their concrete names.
func patchRefs(spec *Resource) error {
	return make(addrRef).patch(spec)
}

func (r addrRef) patch(resource *Resource) error {
	cp := r.copy().load(resource, "")
	for _, attr := range resource.Attrs {
		if !attr.IsRef() {
			continue
		}
		ref := attr.V.EncapsulatedValue().(*Ref)
		referenced, ok := cp[ref.V]
		if !ok {
			return fmt.Errorf("broken reference to %q", ref.V)
		}
		if name, err := referenced.FinalName(); err == nil {
			ref.V = strings.ReplaceAll(ref.V, referenced.Name, name)
		}
	}
	for _, ch := range resource.Children {
		if err := cp.patch(ch); err != nil {
			return err
		}
	}
	return nil
}

func (r addrRef) copy() addrRef {
	n := make(addrRef)
	for k, v := range r {
		n[k] = v
	}
	return n
}

// load the references from the children of the resource.
func (r addrRef) load(res *Resource, track string) addrRef {
	unlabeled := 0
	for _, ch := range res.Children {
		current := addr("", ch.Type, ch.Name, ch.Qualifier)
		if ch.Name == "" {
			current += "." + strconv.Itoa(unlabeled)
			unlabeled++
		}
		if track != "" {
			current = track + "." + current
		}
		r[current] = ch
		r.load(ch, current)
	}
	return r
}

// resource converts the hcl file to a schemahcl.Resource.
func (s *State) resource(ctx *hcl.EvalContext, file *hcl.File) (*Resource, error) {
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	attrs, err := s.toAttrs(ctx, body.Attributes, nil)
	if err != nil {
		return nil, err
	}
	res := &Resource{
		Attrs: attrs,
	}
	for _, blk := range body.Blocks {
		// variable blocks may be included in the document but are skipped in unmarshaling.
		if blk.Type == varBlock {
			continue
		}
		ctx, err := setBlockVars(ctx.NewChild(), blk.Body)
		if err != nil {
			return nil, err
		}
		resource, err := s.toResource(ctx, blk, []string{blk.Type})
		if err != nil {
			return nil, err
		}
		res.Children = append(res.Children, resource)
	}
	return res, nil
}

// mayScopeContext returns a new limited context for the given scope with access only
// to variables defined by WithScopedEnums and WithTypes and references in the document.
func (s *State) mayScopeContext(ctx *hcl.EvalContext, scope []string) *hcl.EvalContext {
	path := strings.Join(scope, ".")
	vars, ok1 := s.config.pathVars[path]
	funcs, ok2 := s.config.pathFuncs[path]
	if !ok1 && !ok2 {
		return ctx
	}
	nctx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: make(map[string]function.Function),
	}
	for n, v := range vars {
		nctx.Variables[n] = v
	}
	for n, f := range funcs {
		nctx.Functions[n] = f
	}
	// A patch from the past. Should be moved
	// to specific scopes in the future.
	nctx.Functions["sql"] = rawExprImpl()
	for p := ctx; p != nil; p = p.Parent() {
		for k, v := range p.Variables {
			if isRef(v) {
				nctx.Variables[k] = v
			}
		}
	}
	return nctx
}

func (s *State) toAttrs(ctx *hcl.EvalContext, hclAttrs hclsyntax.Attributes, scope []string) ([]*Attr, error) {
	var attrs []*Attr
	for _, hclAttr := range hclAttrs {
		scope := append(scope, hclAttr.Name)
		value, diag := hclAttr.Expr.Value(s.mayScopeContext(ctx, scope))
		if diag.HasErrors() {
			return nil, s.typeError(diag, scope)
		}
		at := &Attr{K: hclAttr.Name}
		switch t := value.Type(); {
		case isRef(value):
			at.V = cty.CapsuleVal(ctyRefType, &Ref{V: value.GetAttr("__ref").AsString()})
		case (t.IsTupleType() || t.IsListType() || t.IsSetType()) && value.LengthInt() > 0:
			values := make([]cty.Value, 0, value.LengthInt())
			for it := value.ElementIterator(); it.Next(); {
				_, v := it.Element()
				if isRef(v) {
					v = cty.CapsuleVal(ctyRefType, &Ref{V: v.GetAttr("__ref").AsString()})
				}
				values = append(values, v)
			}
			at.V = cty.ListVal(values)
		default:
			at.V = value
		}
		attrs = append(attrs, at)
	}
	// hclsyntax.Attrs is an alias for map[string]*Attribute
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].K < attrs[j].K
	})
	return attrs, nil
}

// typeError improves diagnostic reporting in case of parse error.
func (s *State) typeError(diag hcl.Diagnostics, scope []string) error {
	path := strings.Join(scope, ".")
	for _, d := range diag {
		switch e := d.Expression.(type) {
		case *hclsyntax.FunctionCallExpr:
			if d.Summary != "Call to unknown function" {
				continue
			}
			if t, ok := s.findTypeSpec(e.Name); ok && len(t.Attributes) == 0 {
				d.Detail = fmt.Sprintf("Type %q does not accept attributes", t.Name)
			}
		case *hclsyntax.ScopeTraversalExpr:
			if d.Summary != "Unknown variable" {
				continue
			}
			if t, ok := s.findTypeSpec(e.Traversal.RootName()); ok && len(t.Attributes) > 0 {
				d.Detail = fmt.Sprintf("Type %q requires at least 1 argument", t.Name)
			} else if n := len(scope); n > 1 && (s.config.pathVars[path] != nil || s.config.pathFuncs[path] != nil) {
				d.Summary = strings.Replace(d.Summary, "variable", fmt.Sprintf("%s.%s", scope[n-2], scope[n-1]), 1)
				d.Detail = strings.Replace(d.Detail, "variable", scope[n-1], 1)
			}
		}
	}
	return diag
}

func isRef(v cty.Value) bool {
	t := v.Type()
	if !t.IsObjectType() {
		return false
	}
	if t.HasAttribute("__ref") {
		return true
	}
	it := v.ElementIterator()
	for it.Next() {
		if _, v := it.Element(); isRef(v) {
			return true
		}
	}
	return false
}

func (s *State) toResource(ctx *hcl.EvalContext, block *hclsyntax.Block, scope []string) (*Resource, error) {
	spec := &Resource{
		Type: block.Type,
	}
	switch len(block.Labels) {
	case 0:
	case 1:
		spec.Name = block.Labels[0]
	case 2:
		spec.Qualifier = block.Labels[0]
		spec.Name = block.Labels[1]
	default:
		return nil, fmt.Errorf("too many labels for block: %s", block.Labels)
	}
	ctx = s.mayScopeContext(ctx, scope)
	attrs, err := s.toAttrs(ctx, block.Body.Attributes, scope)
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs
	for _, blk := range block.Body.Blocks {
		r, err := s.toResource(ctx, blk, append(scope, blk.Type))
		if err != nil {
			return nil, err
		}
		spec.Children = append(spec.Children, r)
	}
	return spec, nil
}

// encode the given *schemahcl.Resource into a byte slice containing an Atlas HCL
// document representing it.
func (s *State) encode(r *Resource) ([]byte, error) {
	f := hclwrite.NewFile()
	body := f.Body()
	// If the resource has a Type then it is rendered as an HCL block.
	if r.Type != "" {
		blk := body.AppendNewBlock(r.Type, labels(r))
		body = blk.Body()
	}
	for _, attr := range r.Attrs {
		if err := s.writeAttr(attr, body); err != nil {
			return nil, err
		}
	}
	for _, res := range r.Children {
		if err := s.writeResource(res, body); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	_, err := f.WriteTo(&buf)
	return buf.Bytes(), err
}

func (s *State) writeResource(b *Resource, body *hclwrite.Body) error {
	blk := body.AppendNewBlock(b.Type, labels(b))
	nb := blk.Body()
	for _, attr := range b.Attrs {
		if err := s.writeAttr(attr, nb); err != nil {
			return err
		}
	}
	for _, b := range b.Children {
		if err := s.writeResource(b, nb); err != nil {
			return err
		}
	}
	return nil
}

func labels(r *Resource) []string {
	var l []string
	if r.Qualifier != "" {
		l = append(l, r.Qualifier)
	}
	if r.Name != "" {
		l = append(l, r.Name)
	}
	return l
}

func (s *State) writeAttr(attr *Attr, body *hclwrite.Body) error {
	switch {
	case attr.IsRef():
		v, err := attr.Ref()
		if err != nil {
			return err
		}
		ts, err := hclRefTokens(v)
		if err != nil {
			return err
		}
		body.SetAttributeRaw(attr.K, ts)
	case attr.IsType():
		t, err := attr.Type()
		if err != nil {
			return err
		}
		if t.IsRef {
			ts, err := hclRefTokens(t.T)
			if err != nil {
				return err
			}
			body.SetAttributeRaw(attr.K, ts)
			break
		}
		spec, ok := s.findTypeSpec(t.T)
		if !ok {
			v := fmt.Sprintf("sql(%q)", t.T)
			body.SetAttributeRaw(attr.K, hclRawTokens(v))
			break
		}
		st, err := hclType(spec, t)
		if err != nil {
			return err
		}
		body.SetAttributeRaw(attr.K, hclRawTokens(st))
	case attr.IsRawExpr():
		v, err := attr.RawExpr()
		if err != nil {
			return err
		}
		// TODO(rotemtam): the func name should be decided on contextual basis.
		fnc := fmt.Sprintf("sql(%q)", v.X)
		body.SetAttributeRaw(attr.K, hclRawTokens(fnc))
	case attr.V.Type().IsListType():
		// Skip scanning nil slices ([]T(nil)) by default. Users that
		// want to print empty lists, should use make([]T, 0) instead.
		if attr.V.LengthInt() == 0 {
			return nil
		}
		tokens := make([]hclwrite.Tokens, 0, attr.V.LengthInt())
		for _, v := range attr.V.AsValueSlice() {
			if v.Type().IsCapsuleType() {
				ref, ok := v.EncapsulatedValue().(*Ref)
				if !ok {
					return fmt.Errorf("unsupported capsule type: %v", v.Type())
				}
				ts, err := hclRefTokens(ref.V)
				if err != nil {
					return err
				}
				tokens = append(tokens, ts)
			} else {
				tokens = append(tokens, hclwrite.TokensForValue(v))
			}
		}
		body.SetAttributeRaw(attr.K, hclList(tokens))
	default:
		body.SetAttributeValue(attr.K, attr.V)
	}
	return nil
}

func (s *State) findTypeSpec(t string) (*TypeSpec, bool) {
	for _, v := range s.config.types {
		if v.T == t {
			return v, true
		}
	}
	return nil, false
}

func hclType(spec *TypeSpec, typ *Type) (string, error) {
	if spec.Format != nil {
		return spec.Format(typ)
	}
	if len(typeFuncArgs(spec)) == 0 {
		return spec.Name, nil
	}
	args := make([]string, 0, len(spec.Attributes))
	for _, param := range typeFuncArgs(spec) {
		arg, ok := findAttr(typ.Attrs, param.Name)
		if !ok {
			continue
		}
		args = append(args, valueArgs(param, arg.V)...)
	}
	// If no args were chosen and the type can be described without a function.
	if len(args) == 0 && len(typeFuncReqArgs(spec)) == 0 {
		return spec.Name, nil
	}
	return fmt.Sprintf("%s(%s)", spec.Name, strings.Join(args, ",")), nil
}

func valueArgs(spec *TypeAttr, v cty.Value) []string {
	switch {
	case v.Type().IsListType(), v.Type().IsTupleType(), v.Type().IsSetType(), v.Type().IsCollectionType():
		args := make([]string, 0, v.LengthInt())
		for _, v := range v.AsValueSlice() {
			args = append(args, valueArgs(spec, v)...)
		}
		return args
	case v.Type() == cty.String:
		return []string{strconv.Quote(v.AsString())}
	case v.Type() == cty.Number && spec.Kind == reflect.Int:
		iv, _ := v.AsBigFloat().Int64()
		return []string{strconv.FormatInt(iv, 10)}
	case v.Type() == cty.Number:
		fv, _ := v.AsBigFloat().Float64()
		return []string{strconv.FormatFloat(fv, 'f', -1, 64)}
	case v.Type() == cty.Bool:
		return []string{strconv.FormatBool(v.True())}
	}
	return nil
}

func findAttr(attrs []*Attr, k string) (*Attr, bool) {
	for _, attr := range attrs {
		if attr.K == k {
			return attr, true
		}
	}
	return nil, false
}

func hclRefTokens(v string) (t hclwrite.Tokens, err error) {
	// If it is a reference to a type or an enum.
	if !strings.HasPrefix(v, "$") {
		return []*hclwrite.Token{{Type: hclsyntax.TokenIdent, Bytes: []byte(v)}}, nil
	}
	path, err := (&Ref{V: v}).Path()
	if err != nil {
		return nil, err
	}
	for i, p := range path {
		if i > 0 {
			t = append(t, &hclwrite.Token{Type: hclsyntax.TokenDot, Bytes: []byte{'.'}})
		}
		t = append(t, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(p.T)})
		for _, v := range p.V {
			switch {
			case validIdent(v):
				t = append(t,
					&hclwrite.Token{Type: hclsyntax.TokenDot, Bytes: []byte{'.'}},
					&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(v)},
				)
			default:
				t = append(t, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte{'['}})
				t = append(t, hclwrite.TokensForValue(cty.StringVal(v))...)
				t = append(t, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte{']'}})
			}
		}
	}
	return t, nil
}

func hclRawTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(s),
		},
	}
}

func hclList(items []hclwrite.Tokens) hclwrite.Tokens {
	t := hclwrite.Tokens{&hclwrite.Token{
		Type:  hclsyntax.TokenOBrack,
		Bytes: []byte("["),
	}}
	for i, item := range items {
		if i > 0 {
			t = append(t, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		}
		t = append(t, item...)
	}
	t = append(t, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrack,
		Bytes: []byte("]"),
	})
	return t
}

func forEachBlocks(ctx *hcl.EvalContext, b *hclsyntax.Block) ([]*hclsyntax.Block, error) {
	forEach, diags := b.Body.Attributes[forEachAttr].Expr.Value(ctx)
	if diags.HasErrors() {
		return nil, diags
	}
	if t := forEach.Type(); !t.IsSetType() && !t.IsObjectType() {
		return nil, fmt.Errorf("schemahcl: for_each does not support %s type", t.FriendlyName())
	}
	delete(b.Body.Attributes, forEachAttr)
	blocks := make([]*hclsyntax.Block, 0, forEach.LengthInt())
	for it := forEach.ElementIterator(); it.Next(); {
		k, v := it.Element()
		nctx := ctx.NewChild()
		nctx.Variables = map[string]cty.Value{
			eachRef: cty.ObjectVal(map[string]cty.Value{
				"key":   k,
				"value": v,
			}),
		}
		nb, err := copyBlock(nctx, b)
		if err != nil {
			return nil, fmt.Errorf("schemahcl: evaluate block for value %q: %w", v, err)
		}
		blocks = append(blocks, nb)
	}
	return blocks, nil
}

func copyBlock(ctx *hcl.EvalContext, b *hclsyntax.Block) (*hclsyntax.Block, error) {
	nb := &hclsyntax.Block{
		Type:   b.Type,
		Labels: b.Labels,
		Body: &hclsyntax.Body{
			Attributes: make(map[string]*hclsyntax.Attribute),
			Blocks:     make([]*hclsyntax.Block, 0, len(b.Body.Blocks)),
		},
	}
	for k, v := range b.Body.Attributes {
		x, diags := v.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, diags
		}
		nv := *v
		nv.Expr = &hclsyntax.LiteralValueExpr{Val: x}
		nb.Body.Attributes[k] = &nv
	}
	for _, v := range b.Body.Blocks {
		nv, err := copyBlock(ctx, v)
		if err != nil {
			return nil, err
		}
		nb.Body.Blocks = append(nb.Body.Blocks, nv)
	}
	return nb, nil
}

// Eval implements the Evaluator interface.
func (f EvalFunc) Eval(p *hclparse.Parser, i any, input map[string]cty.Value) error {
	return f(p, i, input)
}
