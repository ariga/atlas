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

type (
	// Config configures an unmarshaling.
	Config struct {
		types            []*TypeSpec
		vars             map[string]cty.Value
		funcs            map[string]function.Function
		pathVars         map[string]map[string]cty.Value
		pathFuncs        map[string]map[string]function.Function
		validator        func() SchemaValidator
		datasrc, initblk map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error)
		typedblk         map[string]map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error)
	}
	// Option configures a Config.
	Option func(*Config)
)

// New returns a State configured with options.
func New(opts ...Option) *State {
	cfg := &Config{
		vars:      make(map[string]cty.Value),
		funcs:     make(map[string]function.Function),
		pathVars:  make(map[string]map[string]cty.Value),
		pathFuncs: make(map[string]map[string]function.Function),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	for n, f := range stdFuncs() {
		cfg.funcs[n] = f
	}
	return &State{
		config: cfg,
		newCtx: func() *hcl.EvalContext {
			return stdTypes(&hcl.EvalContext{
				Variables: cfg.vars,
				Functions: cfg.funcs,
			})
		},
	}
}

// WithScopedEnums configured a list of allowed ENUMs to be used in
// the given context, block or attribute. For example, the following
// option allows setting HASH or BTREE to the "using" attribute in
// "index" block.
//
//	WithScopedEnums("table.index.type", "HASH", "BTREE")
//
//	table "t" {
//		...
//		index "i" {
//			type = HASH     // Allowed.
//			type = INVALID  // Not Allowed.
//		}
//	}
func WithScopedEnums[T interface{ ~string }](path string, enums ...T) Option {
	return func(c *Config) {
		vars := make(map[string]cty.Value, len(enums))
		for i := range enums {
			vars[string(enums[i])] = cty.StringVal(string(enums[i]))
		}
		c.pathVars[path] = vars
	}
}

// WithVariables registers a list of variables to be injected into the context.
func WithVariables(vars map[string]cty.Value) Option {
	return func(c *Config) {
		if c.vars == nil {
			c.vars = make(map[string]cty.Value)
		}
		for n, v := range vars {
			c.vars[n] = v
		}
	}
}

// WithFunctions registers a list of functions to be injected into the context.
func WithFunctions(funcs map[string]function.Function) Option {
	return func(c *Config) {
		if c.funcs == nil {
			c.funcs = make(map[string]function.Function)
		}
		for n, f := range funcs {
			c.funcs[n] = f
		}
	}
}

// WithDataSource registers a data source name and its corresponding handler.
// e.g., the example below registers a data source named "text" that returns
// the string defined in the data source block.
//
//	WithDataSource("text", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, hcl.Diagnostics) {
//		attrs, diags := b.Body.JustAttributes()
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		v, diags := attrs["value"].Expr.Value(ctx)
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		return cty.ObjectVal(map[string]cty.Value{"output": v}), nil
//	})
//
//	data "text" "hello" {
//	  value = "hello world"
//	}
func WithDataSource(name string, h func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error)) Option {
	return func(c *Config) {
		if c.datasrc == nil {
			c.datasrc = make(map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error))
		}
		c.datasrc[name] = h
	}
}

// WithTypeLabelBlock registers a type-block and its label along with the corresponding handler.
// e.g., the example below registers a typed block named "driver" with the label "remote" that
// returns the string defined in the token attribute.
//
//	WithTypeLabelBlock("driver", "remote", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, hcl.Diagnostics) {
//		attrs, diags := b.Body.JustAttributes()
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		v, diags := attrs["token"].Expr.Value(ctx)
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		return cty.ObjectVal(map[string]cty.Value{"url": v}), nil
//	})
//
//	driver "remote" "hello" {
//	  token = "hello world"
//	}
func WithTypeLabelBlock(name, label string, h func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error)) Option {
	return func(c *Config) {
		if c.typedblk == nil {
			c.typedblk = make(map[string]map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error))
		}
		if c.typedblk[name] == nil {
			c.typedblk[name] = make(map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error))
		}
		c.typedblk[name][label] = h
	}
}

// WithInitBlock registers a block that evaluates (first) to a cty.Value,
// has no labels, and can be defined only once. For example:
//
//	WithInitBlock("atlas", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, hcl.Diagnostics) {
//		attrs, diags := b.Body.JustAttributes()
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		v, diags := attrs["modules"].Expr.Value(ctx)
//		if diags.HasErrors() {
//			return cty.NilVal, diags
//		}
//		return cty.ObjectVal(map[string]cty.Value{"modules": v}), nil
//	})
func WithInitBlock(name string, h func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error)) Option {
	return func(c *Config) {
		if c.initblk == nil {
			c.initblk = make(map[string]func(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error))
		}
		c.initblk[name] = h
	}
}

// WithTypes configures the given types as identifiers in the unmarshal
// context. The path controls where the usage of this type is allowed.
func WithTypes(path string, typeSpecs []*TypeSpec) Option {
	vars := make(map[string]cty.Value)
	funcs := make(map[string]function.Function)
	for _, ts := range typeSpecs {
		typeSpec := ts
		// If no required args exist, register the type as a variable in the HCL context.
		if len(typeFuncReqArgs(typeSpec)) == 0 {
			typ := &Type{T: typeSpec.T}
			vars[typeSpec.Name] = cty.CapsuleVal(ctyTypeSpec, typ)
		}
		// If func args exist, register the type as a function in HCL.
		if len(typeFuncArgs(typeSpec)) > 0 {
			funcs[typeSpec.Name] = typeFuncSpec(typeSpec)
		}
	}
	return func(c *Config) {
		c.types = append(c.types, typeSpecs...)
		c.pathVars[path] = vars
		c.pathFuncs[path] = funcs
	}
}

type (
	// SchemaValidator is the interface used for validating HCL documents.
	SchemaValidator interface {
		Err() error
		ValidateBody(*hcl.EvalContext, *hclsyntax.Body) (func() error, error)
		ValidateBlock(*hcl.EvalContext, *hclsyntax.Block) (func() error, error)
		ValidateAttribute(*hcl.EvalContext, *hclsyntax.Attribute, cty.Value) error
	}
	nopValidator struct{}
)

func (nopValidator) Err() error { return nil }
func (nopValidator) ValidateBody(*hcl.EvalContext, *hclsyntax.Body) (func() error, error) {
	return func() error { return nil }, nil
}
func (nopValidator) ValidateBlock(*hcl.EvalContext, *hclsyntax.Block) (func() error, error) {
	return func() error { return nil }, nil
}
func (nopValidator) ValidateAttribute(*hcl.EvalContext, *hclsyntax.Attribute, cty.Value) error {
	return nil
}

// WithSchemaValidator registers a schema validator to be used during unmarshaling.
func WithSchemaValidator(v func() SchemaValidator) Option {
	return func(c *Config) {
		c.validator = v
	}
}

// Marshal returns the Atlas HCL encoding of v.
var Marshal = MarshalerFunc(New().MarshalSpec)

type (
	// State is used to evaluate and marshal Atlas HCL documents and stores a configuration for these operations.
	State struct {
		config *Config
		newCtx func() *hcl.EvalContext
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
	var (
		ctx          = s.newCtx()
		files        = parsed.Files()
		fileNames    = make([]string, 0, len(files))
		metaBlocks   = make(map[string][]*hclsyntax.Block, len(files))
		staticBlocks = make([]*hclsyntax.Block, 0, len(files))
		reg          = &blockDef{
			fields:   make(map[string]struct{}),
			children: make(map[string]*blockDef),
		}
	)
	if ctx.Variables == nil {
		ctx.Variables = make(map[string]cty.Value)
	}
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
			case b.Type == BlockVariable:
			case b.Body != nil && b.Body.Attributes[forEachAttr] != nil:
				metaBlocks[name] = append(metaBlocks[name], b)
			default:
				blocks = append(blocks, b)
				reg.child(extractDef(b, reg))
			}
		}
		body.Blocks = blocks
		staticBlocks = append(staticBlocks, blocks...)
	}
	vars, err := blockVars(staticBlocks, "", reg)
	if err != nil {
		return err
	}
	for k, v := range vars {
		ctx.Variables[k] = v
	}
	// Semi-evaluate blocks with the for_each meta argument.
	if len(metaBlocks) > 0 {
		blocks := make([]*hclsyntax.Block, 0, len(metaBlocks))
		for name, bs := range metaBlocks {
			for _, b := range bs {
				nb, err := forEachBlocks(ctx, b)
				if err != nil {
					return err
				}
				// Extract the definition of the top-level is enough.
				reg.child(extractDef(b, reg))
				blocks = append(blocks, nb...)
				files[name].Body.(*hclsyntax.Body).Blocks = append(files[name].Body.(*hclsyntax.Body).Blocks, nb...)
			}
		}
		if vars, err = blockVars(blocks, "", reg); err != nil {
			return err
		}
		for k, v := range vars {
			if v.IsNull() {
				continue
			}
			if bs, ok := ctx.Variables[k]; !ok || bs.IsNull() {
				ctx.Variables[k] = v
			} else {
				vs := bs.AsValueMap()
				for k1, v1 := range v.AsValueMap() {
					vs[k1] = v1
				}
				ctx.Variables[k] = cty.ObjectVal(vs)
			}
		}
	}
	spec := &Resource{}
	sort.Slice(fileNames, func(i, j int) bool {
		return fileNames[i] < fileNames[j]
	})
	vr := SchemaValidator(&nopValidator{})
	if s.config.validator != nil {
		vr = s.config.validator()
	}
	for _, name := range fileNames {
		file := files[name]
		r, err := s.resource(ctx, vr, file)
		if err != nil {
			return err
		}
		spec.Children = append(spec.Children, r.Children...)
		spec.Attrs = append(spec.Attrs, r.Attrs...)
	}
	// Validators can fail fast or accumulate errors.
	if err := vr.Err(); err != nil {
		return err
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
	r.load(resource, "")
	for _, attr := range resource.Attrs {
		if !attr.IsRef() {
			continue
		}
		ref := attr.V.EncapsulatedValue().(*Ref)
		referenced, ok := r[ref.V]
		if !ok {
			return fmt.Errorf("broken reference to %q", ref.V)
		}
		if name, err := referenced.FinalName(); err == nil {
			ref.V = strings.ReplaceAll(ref.V, referenced.Name, name)
		}
	}
	for _, ch := range resource.Children {
		if err := r.patch(ch); err != nil {
			return err
		}
	}
	return nil
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
func (s *State) resource(ctx *hcl.EvalContext, vr SchemaValidator, file *hcl.File) (*Resource, error) {
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("schemahcl: expected remainder to be of type *hclsyntax.Body")
	}
	closeScope, err := vr.ValidateBody(ctx, body)
	if err != nil {
		return nil, err
	}
	attrs, err := s.toAttrs(ctx, vr, body.Attributes, nil)
	if err != nil {
		return nil, err
	}
	res := &Resource{
		Attrs:    attrs,
		Children: make([]*Resource, 0, len(body.Blocks)),
	}
	for _, blk := range body.Blocks {
		// variable blocks may be included in the document but are skipped in unmarshaling.
		if blk.Type == BlockVariable {
			continue
		}
		ctx, err := setBlockVars(ctx.NewChild(), blk.Body)
		if err != nil {
			return nil, err
		}
		resource, err := s.toResource(ctx, vr, blk, []string{blk.Type})
		if err != nil {
			return nil, err
		}
		res.Children = append(res.Children, resource)
	}
	if err := closeScope(); err != nil {
		return nil, err
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
	nctx := ctx.NewChild()
	// Use the same variables/functions maps to avoid copying per scope, but return a
	// another child context to prevent writes from different blocks to the same maps.
	nctx.Variables, nctx.Functions = vars, funcs
	return nctx.NewChild()
}

func (s *State) toAttrs(ctx *hcl.EvalContext, vr SchemaValidator, hclAttrs hclsyntax.Attributes, scope []string) ([]*Attr, error) {
	attrs := make([]*Attr, 0, len(hclAttrs))
	for _, hclAttr := range hclAttrs {
		var (
			scope = append(scope, hclAttr.Name)
			nctx  = s.mayScopeContext(ctx, scope)
		)
		value, diag := hclAttr.Expr.Value(nctx)
		if diag.HasErrors() {
			return nil, s.typeError(diag, scope)
		}
		if err := vr.ValidateAttribute(ctx, hclAttr, value); err != nil {
			return nil, err
		}
		at := &Attr{K: hclAttr.Name}
		switch t := value.Type(); {
		case isRef(value):
			if !value.Type().HasAttribute("__ref") {
				return nil, fmt.Errorf("%s: invalid reference used in %s", hclAttr.SrcRange, hclAttr.Name)
			}
			at.V = cty.CapsuleVal(ctyRefType, &Ref{V: value.GetAttr("__ref").AsString()})
		case (t.IsTupleType() || t.IsListType() || t.IsSetType()) && value.LengthInt() > 0:
			var (
				vt     cty.Type
				values = make([]cty.Value, 0, value.LengthInt())
			)
			for it := value.ElementIterator(); it.Next(); {
				_, v := it.Element()
				if isRef(v) {
					v = cty.CapsuleVal(ctyRefType, &Ref{V: v.GetAttr("__ref").AsString()})
				}
				if vt != cty.NilType && vt != v.Type() {
					return nil, fmt.Errorf("%s: mixed list types used in %q attribute", hclAttr.SrcRange, hclAttr.Name)
				}
				vt = v.Type()
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
			switch root := e.Traversal.RootName(); {
			case root == RefData, s.config.typedblk[root] != nil:
				var b strings.Builder
				b.WriteString(root)
				for _, t := range e.Traversal[1:] {
					if v, ok := t.(hcl.TraverseAttr); ok {
						b.WriteString(".")
						b.WriteString(v.Name)
					}
				}
				d.Summary = "Unknown data source"
				if s.config.typedblk[root] != nil {
					d.Summary = "Unknown block type"
				}
				d.Detail = fmt.Sprintf("%s does not exist", b.String())
			case root == RefLocal:
				d.Summary = "Unknown local"
			default:
				if t, ok := s.findTypeSpec(root); ok && len(t.Attributes) > 0 {
					d.Detail = fmt.Sprintf("Type %q requires at least 1 argument", t.Name)
				} else if n := len(scope); n > 1 && (s.config.pathVars[path] != nil || s.config.pathFuncs[path] != nil) {
					d.Summary = strings.Replace(d.Summary, "variable", fmt.Sprintf("%s.%s", scope[n-2], scope[n-1]), 1)
					d.Detail = strings.Replace(d.Detail, "variable", scope[n-1], 1)
				}
			}
		}
	}
	return diag
}

// isRef checks if the given value is a reference or a list of references.
// Exists here for backward compatibility, use isOneRef and isRefList instead.
func isRef(v cty.Value) bool {
	if !v.Type().IsObjectType() {
		return false
	}
	if isOneRef(v) {
		return true
	}
	for it := v.ElementIterator(); it.Next(); {
		if _, v := it.Element(); isRef(v) {
			return true
		}
	}
	return false
}

func isOneRef(v cty.Value) bool {
	t := v.Type()
	return t.IsObjectType() && t.HasAttribute("__ref")
}

func (s *State) toResource(ctx *hcl.EvalContext, vr SchemaValidator, block *hclsyntax.Block, scope []string) (spec *Resource, err error) {
	closeScope, err := vr.ValidateBlock(ctx, block)
	if err != nil {
		return nil, err
	}
	spec = &Resource{Type: block.Type}
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
	attrs, err := s.toAttrs(ctx, vr, block.Body.Attributes, scope)
	if err != nil {
		return nil, err
	}
	spec.Attrs = attrs
	for _, blk := range block.Body.Blocks {
		ctx, err := setBlockVars(ctx.NewChild(), blk.Body)
		if err != nil {
			return nil, err
		}
		r, err := s.toResource(ctx, vr, blk, append(scope, blk.Type))
		if err != nil {
			return nil, err
		}
		spec.Children = append(spec.Children, r)
	}
	if err := closeScope(); err != nil {
		return nil, err
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
	// Anonymous resources are treated as embedded blocks.
	if b.Type != "" {
		blk := body.AppendNewBlock(b.Type, labels(b))
		body = blk.Body()
	}
	for _, attr := range b.Attrs {
		if err := s.writeAttr(attr, body); err != nil {
			return err
		}
	}
	for _, b := range b.Children {
		if err := s.writeResource(b, body); err != nil {
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
	// Heredoc is a special case that currently is not handled by hclwrite:
	// https://github.com/hashicorp/hcl/blob/main/hclwrite/generate.go#L218-L219.
	case attr.V.Type() == cty.String && strings.Count(attr.V.AsString(), "\n") > 1 && strings.HasPrefix(attr.V.AsString(), "<<"):
		v := strings.TrimLeft(attr.V.AsString(), "<-")
		// Heredoc begins with << (or <<-), followed by a token that
		// specifies the terminator and ends with the \n + terminator.
		if lines := strings.Split(v, "\n"); len(lines) > 2 && strings.TrimSpace(lines[0]) == strings.TrimSpace(lines[len(lines)-1]) {
			body.SetAttributeRaw(attr.K, hclwrite.Tokens{
				&hclwrite.Token{
					Type:  hclsyntax.TokenOHeredoc,
					Bytes: []byte(attr.V.AsString()),
				},
			})
		} else {
			body.SetAttributeValue(attr.K, attr.V)
		}
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
	if t := forEach.Type(); !t.IsSetType() && !t.IsObjectType() && !t.IsTupleType() {
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

// typeFuncSpec returns the HCL function for defining the type in the spec.
func typeFuncSpec(typeSpec *TypeSpec) function.Function {
	spec := &function.Spec{
		Type: function.StaticReturnType(ctyTypeSpec),
	}
	for _, arg := range typeFuncArgs(typeSpec) {
		if arg.Kind == reflect.Slice || !arg.Required {
			spec.VarParam = &function.Parameter{
				Name: "args",
				Type: cty.DynamicPseudoType,
			}
			continue
		}
		p := function.Parameter{
			Name:      arg.Name,
			AllowNull: !arg.Required,
		}
		switch arg.Kind {
		case reflect.String:
			p.Type = cty.String
		case reflect.Int, reflect.Float32, reflect.Int64:
			p.Type = cty.Number
		case reflect.Bool:
			p.Type = cty.Bool
		}
		spec.Params = append(spec.Params, p)
	}
	spec.Impl = typeFuncSpecImpl(spec, typeSpec)
	return function.New(spec)
}

// typeFuncSpecImpl returns the function implementation for the HCL function spec.
func typeFuncSpecImpl(_ *function.Spec, typeSpec *TypeSpec) function.ImplFunc {
	return func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		t := &Type{
			T: typeSpec.T,
		}
		if len(args) > len(typeSpec.Attributes) && typeSpec.Attributes[len(typeSpec.Attributes)-1].Kind != reflect.Slice {
			return cty.NilVal, fmt.Errorf("too many arguments for type definition %q", typeSpec.Name)
		}
		// TypeRegistry enforces that:
		// 1. Required attrs come before optionals
		// 2. Slice attrs can only be last
		for _, attr := range typeFuncArgs(typeSpec) {
			// If the attribute is a slice, read all remaining args into a list value.
			if attr.Kind == reflect.Slice {
				t.Attrs = append(t.Attrs, &Attr{K: attr.Name, V: cty.ListVal(args)})
				break
			}
			if len(args) == 0 {
				break
			}
			t.Attrs = append(t.Attrs, &Attr{K: attr.Name, V: args[0]})
			args = args[1:]
		}
		return cty.CapsuleVal(ctyTypeSpec, t), nil
	}
}

// typeFuncArgs returns the type attributes that are configured via arguments to the
// type definition, for example precision and scale in a decimal definition, i.e `decimal(10,2)`.
func typeFuncArgs(spec *TypeSpec) []*TypeAttr {
	var args []*TypeAttr
	for _, attr := range spec.Attributes {
		// TODO(rotemtam): this should be defined on the TypeSpec.
		if attr.Name == "unsigned" {
			continue
		}
		args = append(args, attr)
	}
	return args
}

// typeFuncReqArgs returns the required type attributes that are configured via arguments.
// for instance, in MySQL a field may be defined as both `int` and `int(10)`, in this case
// it is not a required parameter.
func typeFuncReqArgs(spec *TypeSpec) []*TypeAttr {
	var args []*TypeAttr
	for _, arg := range typeFuncArgs(spec) {
		if arg.Required {
			args = append(args, arg)
		}
	}
	return args
}
