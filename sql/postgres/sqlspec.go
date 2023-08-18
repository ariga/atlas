// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/postgres/internal/postgresop"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type (
	doc struct {
		Tables  []*sqlspec.Table  `spec:"table"`
		Views   []*sqlspec.View   `spec:"view"`
		Enums   []*Enum           `spec:"enum"`
		Schemas []*sqlspec.Schema `spec:"schema"`
	}
	// Enum holds a specification for an enum, that can be referenced as a column type.
	Enum struct {
		Name      string         `spec:",name"`
		Qualifier string         `spec:",qualifier"`
		Schema    *schemahcl.Ref `spec:"schema"`
		Values    []string       `spec:"values"`
		schemahcl.DefaultExtension
	}
)

// Label returns the defaults label used for the enum resource.
func (e *Enum) Label() string { return e.Name }

// QualifierLabel returns the qualifier label used for the enum resource, if any.
func (e *Enum) QualifierLabel() string { return e.Qualifier }

// SetQualifier sets the qualifier label used for the enum resource.
func (e *Enum) SetQualifier(q string) { e.Qualifier = q }

// SchemaRef returns the schema reference for the enum.
func (e *Enum) SchemaRef() *schemahcl.Ref { return e.Schema }

func init() {
	schemahcl.Register("enum", &Enum{})
}

// evalSpec evaluates an Atlas DDL document into v using the input.
func evalSpec(p *hclparse.Parser, v any, input map[string]cty.Value) error {
	switch v := v.(type) {
	case *schema.Realm:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if err := specutil.Scan(v,
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views},
			&specutil.ScanFuncs{Table: convertTable, View: convertView},
		); err != nil {
			return fmt.Errorf("specutil: failed converting to *schema.Realm: %w", err)
		}
		if len(d.Enums) > 0 {
			if err := convertEnums(d.Tables, d.Enums, v); err != nil {
				return err
			}
		}
	case *schema.Schema:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if len(d.Schemas) != 1 {
			return fmt.Errorf("specutil: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		r := &schema.Realm{}
		if err := specutil.Scan(r,
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views},
			&specutil.ScanFuncs{Table: convertTable, View: convertView},
		); err != nil {
			return err
		}
		if err := convertEnums(d.Tables, d.Enums, r); err != nil {
			return err
		}
		*v = *r.Schemas[0]
	case schema.Schema, schema.Realm:
		return fmt.Errorf("postgres: Eval expects a pointer: received %[1]T, expected *%[1]T", v)
	default:
		return hclState.Eval(p, v, input)
	}
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemahcl.Marshaler.
func MarshalSpec(v any, marshaler schemahcl.Marshaler) ([]byte, error) {
	var d doc
	switch s := v.(type) {
	case *schema.Schema:
		var err error
		doc, err := schemaSpec(s)
		if err != nil {
			return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
		}
		d.Tables = doc.Tables
		d.Views = doc.Views
		d.Schemas = doc.Schemas
		d.Enums = doc.Enums
	case *schema.Realm:
		for _, s := range s.Schemas {
			doc, err := schemaSpec(s)
			if err != nil {
				return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
			}
			d.Tables = append(d.Tables, doc.Tables...)
			d.Views = append(d.Views, doc.Views...)
			d.Schemas = append(d.Schemas, doc.Schemas...)
			d.Enums = append(d.Enums, doc.Enums...)
		}
		if err := specutil.QualifyObjects(d.Tables); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Views); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Enums); err != nil {
			return nil, err
		}
		if err := specutil.QualifyReferences(d.Tables, s); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("specutil: failed marshaling spec. %T is not supported", v)
	}
	return marshaler.MarshalSpec(&d)
}

var (
	hclState = schemahcl.New(append(specOptions,
		schemahcl.WithTypes("table.column.type", TypeRegistry.Specs()),
		schemahcl.WithTypes("view.column.type", TypeRegistry.Specs()),
		schemahcl.WithScopedEnums("view.check_option", schema.ViewCheckOptionLocal, schema.ViewCheckOptionCascaded),
		schemahcl.WithScopedEnums("table.index.type", IndexTypeBTree, IndexTypeBRIN, IndexTypeHash, IndexTypeGIN, IndexTypeGiST, "GiST", IndexTypeSPGiST, "SPGiST"),
		schemahcl.WithScopedEnums("table.partition.type", PartitionTypeRange, PartitionTypeList, PartitionTypeHash),
		schemahcl.WithScopedEnums("table.column.identity.generated", GeneratedTypeAlways, GeneratedTypeByDefault),
		schemahcl.WithScopedEnums("table.column.as.type", "STORED"),
		schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.index.on.ops", func() (ops []string) {
			for _, op := range postgresop.Classes {
				ops = append(ops, op.Name)
			}
			return ops
		}()...))...,
	)
	// MarshalHCL marshals v into an Atlas HCL DDL document.
	MarshalHCL = schemahcl.MarshalerFunc(func(v any) ([]byte, error) {
		return MarshalSpec(v, hclState)
	})
	// EvalHCL implements the schemahcl.Evaluator interface.
	EvalHCL = schemahcl.EvalFunc(evalSpec)

	// EvalHCLBytes is a helper that evaluates an HCL document from a byte slice instead
	// of from an hclparse.Parser instance.
	EvalHCLBytes = specutil.HCLBytesFunc(EvalHCL)
)

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	t, err := specutil.Table(spec, parent, convertColumn, convertPK, convertIndex, specutil.Check)
	if err != nil {
		return nil, err
	}
	if err := convertPartition(spec.Extra, t); err != nil {
		return nil, err
	}
	return t, nil
}

// convertView converts a sqlspec.View to a schema.View.
func convertView(spec *sqlspec.View, parent *schema.Schema) (*schema.View, error) {
	v, err := specutil.View(spec, parent, func(c *sqlspec.Column, _ *schema.View) (*schema.Column, error) {
		return specutil.Column(c, convertColumnType)
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

// convertPartition converts and appends the partition block into the table attributes if exists.
func convertPartition(spec schemahcl.Resource, table *schema.Table) error {
	r, ok := spec.Resource("partition")
	if !ok {
		return nil
	}
	var p struct {
		Type    string           `spec:"type"`
		Columns []*schemahcl.Ref `spec:"columns"`
		Parts   []*struct {
			Expr   string         `spec:"expr"`
			Column *schemahcl.Ref `spec:"column"`
		} `spec:"by"`
	}
	if err := r.As(&p); err != nil {
		return fmt.Errorf("parsing %s.partition: %w", table.Name, err)
	}
	if p.Type == "" {
		return fmt.Errorf("missing attribute %s.partition.type", table.Name)
	}
	key := &Partition{T: p.Type}
	switch n, m := len(p.Columns), len(p.Parts); {
	case n == 0 && m == 0:
		return fmt.Errorf("missing columns or expressions for %s.partition", table.Name)
	case n > 0 && m > 0:
		return fmt.Errorf(`multiple definitions for %s.partition, use "columns" or "by"`, table.Name)
	case n > 0:
		for _, r := range p.Columns {
			c, err := specutil.ColumnByRef(table, r)
			if err != nil {
				return err
			}
			key.Parts = append(key.Parts, &PartitionPart{C: c})
		}
	case m > 0:
		for i, p := range p.Parts {
			switch {
			case p.Column == nil && p.Expr == "":
				return fmt.Errorf("missing column or expression for %s.partition.by at position %d", table.Name, i)
			case p.Column != nil && p.Expr != "":
				return fmt.Errorf("multiple definitions for  %s.partition.by at position %d", table.Name, i)
			case p.Column != nil:
				c, err := specutil.ColumnByRef(table, p.Column)
				if err != nil {
					return err
				}
				key.Parts = append(key.Parts, &PartitionPart{C: c})
			case p.Expr != "":
				key.Parts = append(key.Parts, &PartitionPart{X: &schema.RawExpr{X: p.Expr}})
			}
		}
	}
	table.AddAttrs(key)
	return nil
}

// fromPartition returns the resource spec for representing the partition block.
func fromPartition(p Partition) *schemahcl.Resource {
	key := &schemahcl.Resource{
		Type: "partition",
		Attrs: []*schemahcl.Attr{
			specutil.VarAttr("type", strings.ToUpper(specutil.Var(p.T))),
		},
	}
	columns, ok := func() ([]*schemahcl.Ref, bool) {
		parts := make([]*schemahcl.Ref, 0, len(p.Parts))
		for _, p := range p.Parts {
			if p.C == nil {
				return nil, false
			}
			parts = append(parts, specutil.ColumnRef(p.C.Name))
		}
		return parts, true
	}()
	if ok {
		key.Attrs = append(key.Attrs, schemahcl.RefsAttr("columns", columns...))
		return key
	}
	for _, p := range p.Parts {
		part := &schemahcl.Resource{Type: "by"}
		switch {
		case p.C != nil:
			part.Attrs = append(part.Attrs, schemahcl.RefAttr("column", specutil.ColumnRef(p.C.Name)))
		case p.X != nil:
			part.Attrs = append(part.Attrs, schemahcl.StringAttr("expr", p.X.(*schema.RawExpr).X))
		}
		key.Children = append(key.Children, part)
	}
	return key
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	if err := fixDefaultQuotes(spec); err != nil {
		return nil, err
	}
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
	}
	if r, ok := spec.Extra.Resource("identity"); ok {
		id, err := convertIdentity(r)
		if err != nil {
			return nil, err
		}
		c.Attrs = append(c.Attrs, id)
	}
	if err := specutil.ConvertGenExpr(spec.Remain(), c, generatedType); err != nil {
		return nil, err
	}
	return c, nil
}

func convertIdentity(r *schemahcl.Resource) (*Identity, error) {
	var spec struct {
		Generation string `spec:"generated"`
		Start      int64  `spec:"start"`
		Increment  int64  `spec:"increment"`
	}
	if err := r.As(&spec); err != nil {
		return nil, err
	}
	id := &Identity{Generation: specutil.FromVar(spec.Generation), Sequence: &Sequence{}}
	if spec.Start != 0 {
		id.Sequence.Start = spec.Start
	}
	if spec.Increment != 0 {
		id.Sequence.Increment = spec.Increment
	}
	return id, nil
}

// fixDefaultQuotes fixes the quotes on the Default field to be single quotes
// instead of double quotes.
func fixDefaultQuotes(spec *sqlspec.Column) error {
	if spec.Default.Type() != cty.String {
		return nil
	}
	if s := spec.Default.AsString(); sqlx.IsQuoted(s, '"') {
		uq, err := strconv.Unquote(s)
		if err != nil {
			return err
		}
		s = "'" + uq + "'"
		spec.Default = cty.StringVal(s)
	}
	return nil
}

// convertPK converts a sqlspec.PrimaryKey into a schema.Index.
func convertPK(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	idx, err := specutil.PrimaryKey(spec, parent)
	if err != nil {
		return nil, err
	}
	if err := convertIndexPK(spec, parent, idx); err != nil {
		return nil, err
	}
	return idx, nil
}

// convertIndex converts a sqlspec.Index into a schema.Index.
func convertIndex(spec *sqlspec.Index, t *schema.Table) (*schema.Index, error) {
	idx, err := specutil.Index(spec, t, convertPart)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("type"); ok {
		t, err := attr.String()
		if err != nil {
			return nil, err
		}
		idx.Attrs = append(idx.Attrs, &IndexType{T: strings.ToUpper(t)})
	}
	if attr, ok := spec.Attr("where"); ok {
		p, err := attr.String()
		if err != nil {
			return nil, err
		}
		idx.Attrs = append(idx.Attrs, &IndexPredicate{P: p})
	}
	if attr, ok := spec.Attr("nulls_distinct"); ok {
		v, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		idx.Attrs = append(idx.Attrs, &IndexNullsDistinct{V: v})
	}
	if err := convertIndexPK(spec, t, idx); err != nil {
		return nil, err
	}
	return idx, nil
}

// convertIndexPK converts the index parameters shared between primary and secondary indexes.
func convertIndexPK(spec specutil.Attrer, t *schema.Table, idx *schema.Index) error {
	if attr, ok := spec.Attr("page_per_range"); ok {
		p, err := attr.Int64()
		if err != nil {
			return err
		}
		idx.Attrs = append(idx.Attrs, &IndexStorageParams{PagesPerRange: p})
	}
	if attr, ok := spec.Attr("include"); ok {
		refs, err := attr.Refs()
		if err != nil {
			return err
		}
		if len(refs) == 0 {
			return fmt.Errorf("unexpected empty INCLUDE in index %q definition", idx.Name)
		}
		include := make([]*schema.Column, len(refs))
		for i, r := range refs {
			if include[i], err = specutil.ColumnByRef(t, r); err != nil {
				return err
			}
		}
		idx.Attrs = append(idx.Attrs, &IndexInclude{Columns: include})
	}
	return nil
}

func convertPart(spec *sqlspec.IndexPart, part *schema.IndexPart) error {
	switch opc, ok := spec.Attr("ops"); {
	case !ok:
	case opc.IsRawExpr():
		expr, err := opc.RawExpr()
		if err != nil {
			return err
		}
		var op IndexOpClass
		if err := op.UnmarshalText([]byte(expr.X)); err != nil {
			return fmt.Errorf("unexpected index.on.ops expression %q: %w", expr.X, err)
		}
		if op.Name != "" {
			part.Attrs = append(part.Attrs, &op)
		}
	case opc.IsRef():
		name, err := opc.Ref()
		if err != nil {
			return err
		}
		part.Attrs = append(part.Attrs, &IndexOpClass{Name: name})
	default:
		name, err := opc.String()
		if err != nil {
			return err
		}
		part.Attrs = append(part.Attrs, &IndexOpClass{Name: name})
	}
	return nil
}

const defaultTimePrecision = 6

// convertColumnType converts a sqlspec.Column into a concrete Postgres schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	typ, err := TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
	if err != nil {
		return nil, err
	}
	// Handle default values for time precision types.
	if t, ok := typ.(*schema.TimeType); ok && strings.HasPrefix(t.T, "time") {
		if _, ok := attr(spec.Type, "precision"); !ok {
			p := defaultTimePrecision
			t.Precision = &p
		}
	}
	return typ, nil
}

// convertEnums converts possibly referenced column types (like enums) to
// an actual schema.Type and sets it on the correct schema.Column.
func convertEnums(tables []*sqlspec.Table, enums []*Enum, r *schema.Realm) error {
	byName := make(map[string]*schema.EnumType)
	for _, e := range enums {
		if byName[e.Name] != nil {
			return fmt.Errorf("duplicate enum %q", e.Name)
		}
		ns, err := specutil.SchemaName(e.Schema)
		if err != nil {
			return fmt.Errorf("extract schema name from enum reference: %w", err)
		}
		es, ok := r.Schema(ns)
		if !ok {
			return fmt.Errorf("schema %q defined on enum %q was not found in realm", ns, e.Name)
		}
		e1 := &schema.EnumType{T: e.Name, Schema: es, Values: e.Values}
		es.Objects = append(es.Objects, e1)
		byName[e.Name] = e1
	}
	for _, t := range tables {
		for _, c := range t.Columns {
			var enum *schema.EnumType
			switch {
			case c.Type.IsRef:
				n, err := enumName(c.Type)
				if err != nil {
					return err
				}
				e, ok := byName[n]
				if !ok {
					return fmt.Errorf("enum %q was not found in realm", n)
				}
				enum = e
			default:
				n, ok := arrayType(c.Type.T)
				if !ok || byName[n] == nil {
					continue
				}
				enum = byName[n]
			}
			schemaT, err := specutil.SchemaName(t.Schema)
			if err != nil {
				return fmt.Errorf("extract schema name from table reference: %w", err)
			}
			ts, ok := r.Schema(schemaT)
			if !ok {
				return fmt.Errorf("schema %q not found in realm for table %q", schemaT, t.Name)
			}
			tt, ok := ts.Table(t.Name)
			if !ok {
				return fmt.Errorf("table %q not found in schema %q", t.Name, ts.Name)
			}
			cc, ok := tt.Column(c.Name)
			if !ok {
				return fmt.Errorf("column %q not found in table %q", c.Name, t.Name)
			}
			switch t := cc.Type.Type.(type) {
			case *ArrayType:
				t.Type = enum
			default:
				cc.Type.Type = enum
			}
		}
	}
	return nil
}

// enumName extracts the name of the referenced Enum from the reference string.
func enumName(ref *schemahcl.Type) (string, error) {
	s := strings.Split(ref.T, "$enum.")
	if len(s) != 2 {
		return "", fmt.Errorf("postgres: failed to extract enum name from %q", ref.T)
	}
	return s[1], nil
}

// enumRef returns a reference string to the given enum name.
func enumRef(n string) *schemahcl.Ref {
	return &schemahcl.Ref{
		V: "$enum." + n,
	}
}

// schemaSpec converts from a concrete Postgres schema to Atlas specification.
func schemaSpec(s *schema.Schema) (*doc, error) {
	spec, err := specutil.FromSchema(s, tableSpec, viewSpec)
	if err != nil {
		return nil, err
	}
	d := &doc{
		Tables:  spec.Tables,
		Views:   spec.Views,
		Schemas: []*sqlspec.Schema{spec.Schema},
		Enums:   make([]*Enum, 0, len(s.Objects)),
	}
	for _, o := range s.Objects {
		if e, ok := o.(*schema.EnumType); ok {
			d.Enums = append(d.Enums, &Enum{
				Name:   e.T,
				Values: e.Values,
				Schema: specutil.SchemaRef(spec.Schema.Name),
			})
		}
	}
	return d, nil
}

// tableSpec converts from a concrete Postgres sqlspec.Table to a schema.Table.
func tableSpec(table *schema.Table) (*sqlspec.Table, error) {
	spec, err := specutil.FromTable(
		table,
		tableColumnSpec,
		pkSpec,
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
	if err != nil {
		return nil, err
	}
	if p := (Partition{}); sqlx.Has(table.Attrs, &p) {
		spec.Extra.Children = append(spec.Extra.Children, fromPartition(p))
	}
	return spec, nil
}

// viewSpec converts from a concrete PostgreSQL schema.View to a sqlspec.View.
func viewSpec(view *schema.View) (*sqlspec.View, error) {
	spec, err := specutil.FromView(view, func(c *schema.Column, _ *schema.View) (*sqlspec.Column, error) {
		return specutil.FromColumn(c, columnTypeSpec)
	})
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func pkSpec(idx *schema.Index) (*sqlspec.PrimaryKey, error) {
	spec, err := specutil.FromPrimaryKey(idx)
	if err != nil {
		return nil, err
	}
	spec.Extra.Attrs = indexPKSpec(idx, spec.Extra.Attrs)
	return spec, nil
}

func indexSpec(idx *schema.Index) (*sqlspec.Index, error) {
	spec, err := specutil.FromIndex(idx, partAttr)
	if err != nil {
		return nil, err
	}
	// Avoid printing the index type if it is the default.
	if i := (IndexType{}); sqlx.Has(idx.Attrs, &i) && strings.ToUpper(i.T) != IndexTypeBTree {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("type", strings.ToUpper(i.T)))
	}
	if i := (IndexPredicate{}); sqlx.Has(idx.Attrs, &i) && i.P != "" {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("where", strconv.Quote(i.P)))
	}
	if i := (IndexNullsDistinct{}); sqlx.Has(idx.Attrs, &i) && !i.V {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.BoolAttr("nulls_distinct", i.V))
	}
	spec.Extra.Attrs = indexPKSpec(idx, spec.Extra.Attrs)
	return spec, nil
}

func indexPKSpec(idx *schema.Index, attrs []*schemahcl.Attr) []*schemahcl.Attr {
	if i := (IndexInclude{}); sqlx.Has(idx.Attrs, &i) && len(i.Columns) > 0 {
		refs := make([]*schemahcl.Ref, 0, len(i.Columns))
		for _, c := range i.Columns {
			refs = append(refs, specutil.ColumnRef(c.Name))
		}
		attrs = append(attrs, schemahcl.RefsAttr("include", refs...))
	}
	if p, ok := indexStorageParams(idx.Attrs); ok {
		attrs = append(attrs, schemahcl.Int64Attr("page_per_range", p.PagesPerRange))
	}
	return attrs
}

func partAttr(idx *schema.Index, part *schema.IndexPart, spec *sqlspec.IndexPart) error {
	var op IndexOpClass
	if !sqlx.Has(part.Attrs, &op) {
		return nil
	}
	switch d, err := op.DefaultFor(idx, part); {
	case err != nil:
		return err
	case d:
	case len(op.Params) > 0:
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.RawAttr("ops", op.String()))
	default:
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("ops", op.String()))
	}
	return nil
}

// tableColumnSpec converts from a concrete Postgres schema.Column into a sqlspec.Column.
func tableColumnSpec(c *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	s, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if i := (&Identity{}); sqlx.Has(c.Attrs, i) {
		s.Extra.Children = append(s.Extra.Children, fromIdentity(i))
	}
	if x := (schema.GeneratedExpr{}); sqlx.Has(c.Attrs, &x) {
		s.Extra.Children = append(s.Extra.Children, specutil.FromGenExpr(x, generatedType))
	}
	return s, nil
}

// fromIdentity returns the resource spec for representing the identity attributes.
func fromIdentity(i *Identity) *schemahcl.Resource {
	id := &schemahcl.Resource{
		Type: "identity",
		Attrs: []*schemahcl.Attr{
			specutil.VarAttr("generated", strings.ToUpper(specutil.Var(i.Generation))),
		},
	}
	if s := i.Sequence; s != nil {
		if s.Start != 1 {
			id.Attrs = append(id.Attrs, schemahcl.Int64Attr("start", s.Start))
		}
		if s.Increment != 1 {
			id.Attrs = append(id.Attrs, schemahcl.Int64Attr("increment", s.Increment))
		}
	}
	return id
}

// columnTypeSpec converts from a concrete Postgres schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	// Handle postgres enum types. They cannot be put into the TypeRegistry since their name is dynamic.
	if e, ok := t.(*schema.EnumType); ok {
		return &sqlspec.Column{Type: &schemahcl.Type{
			T:     enumRef(e.T).V,
			IsRef: true,
		}}, nil
	}
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{Type: st}, nil
}

// TypeRegistry contains the supported TypeSpecs for the Postgres driver.
var TypeRegistry = schemahcl.NewRegistry(
	schemahcl.WithSpecFunc(typeSpec),
	schemahcl.WithParser(ParseType),
	schemahcl.WithSpecs(
		schemahcl.NewTypeSpec(TypeBit, schemahcl.WithAttributes(&schemahcl.TypeAttr{Name: "len", Kind: reflect.Int64})),
		schemahcl.AliasTypeSpec("bit_varying", TypeBitVar, schemahcl.WithAttributes(&schemahcl.TypeAttr{Name: "len", Kind: reflect.Int64})),
		schemahcl.NewTypeSpec(TypeVarChar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasTypeSpec("character_varying", TypeCharVar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeChar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeCharacter, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeInt2),
		schemahcl.NewTypeSpec(TypeInt4),
		schemahcl.NewTypeSpec(TypeInt8),
		schemahcl.NewTypeSpec(TypeInt),
		schemahcl.NewTypeSpec(TypeInteger),
		schemahcl.NewTypeSpec(TypeSmallInt),
		schemahcl.NewTypeSpec(TypeBigInt),
		schemahcl.NewTypeSpec(TypeText),
		schemahcl.NewTypeSpec(TypeBoolean),
		schemahcl.NewTypeSpec(TypeBool),
		schemahcl.NewTypeSpec(TypeBytea),
		schemahcl.NewTypeSpec(TypeCIDR),
		schemahcl.NewTypeSpec(TypeInet),
		schemahcl.NewTypeSpec(TypeMACAddr),
		schemahcl.NewTypeSpec(TypeMACAddr8),
		schemahcl.NewTypeSpec(TypeCircle),
		schemahcl.NewTypeSpec(TypeLine),
		schemahcl.NewTypeSpec(TypeLseg),
		schemahcl.NewTypeSpec(TypeBox),
		schemahcl.NewTypeSpec(TypePath),
		schemahcl.NewTypeSpec(TypePoint),
		schemahcl.NewTypeSpec(TypePolygon),
		schemahcl.NewTypeSpec(TypeDate),
		schemahcl.NewTypeSpec(TypeTime, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr()), formatTime()),
		schemahcl.NewTypeSpec(TypeTimeTZ, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr()), formatTime()),
		schemahcl.NewTypeSpec(TypeTimestampTZ, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr()), formatTime()),
		schemahcl.NewTypeSpec(TypeTimestamp, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr()), formatTime()),
		schemahcl.AliasTypeSpec("double_precision", TypeDouble),
		schemahcl.NewTypeSpec(TypeReal),
		schemahcl.NewTypeSpec(TypeFloat, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
		schemahcl.NewTypeSpec(TypeFloat8),
		schemahcl.NewTypeSpec(TypeFloat4),
		schemahcl.NewTypeSpec(TypeNumeric, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeDecimal, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeSmallSerial),
		schemahcl.NewTypeSpec(TypeSerial),
		schemahcl.NewTypeSpec(TypeBigSerial),
		schemahcl.NewTypeSpec(TypeSerial2),
		schemahcl.NewTypeSpec(TypeSerial4),
		schemahcl.NewTypeSpec(TypeSerial8),
		schemahcl.NewTypeSpec(TypeXML),
		schemahcl.NewTypeSpec(TypeJSON),
		schemahcl.NewTypeSpec(TypeJSONB),
		schemahcl.NewTypeSpec(TypeUUID),
		schemahcl.NewTypeSpec(TypeMoney),
		schemahcl.NewTypeSpec(TypeTSVector),
		schemahcl.NewTypeSpec(TypeTSQuery),
		schemahcl.NewTypeSpec(TypeInt4Range),
		schemahcl.NewTypeSpec(TypeInt4MultiRange),
		schemahcl.NewTypeSpec(TypeInt8Range),
		schemahcl.NewTypeSpec(TypeInt8MultiRange),
		schemahcl.NewTypeSpec(TypeNumRange),
		schemahcl.NewTypeSpec(TypeNumMultiRange),
		schemahcl.NewTypeSpec(TypeTSRange),
		schemahcl.NewTypeSpec(TypeTSMultiRange),
		schemahcl.NewTypeSpec(TypeTSTZRange),
		schemahcl.NewTypeSpec(TypeTSTZMultiRange),
		schemahcl.NewTypeSpec(TypeDateRange),
		schemahcl.NewTypeSpec(TypeDateMultiRange),
		schemahcl.NewTypeSpec("hstore"),
		schemahcl.NewTypeSpec("sql", schemahcl.WithAttributes(&schemahcl.TypeAttr{Name: "def", Required: true, Kind: reflect.String})),
	),
	// PostgreSQL internal and special types.
	schemahcl.WithSpecs(func() (specs []*schemahcl.TypeSpec) {
		for _, t := range []string{
			typeOID, typeRegClass, typeRegCollation, typeRegConfig, typeRegDictionary, typeRegNamespace,
			typeName, typeRegOper, typeRegOperator, typeRegProc, typeRegProcedure, typeRegRole, typeRegType,
		} {
			specs = append(specs, schemahcl.NewTypeSpec(t))
		}
		return specs
	}()...),
	schemahcl.WithSpecs(func() (specs []*schemahcl.TypeSpec) {
		opts := []schemahcl.TypeSpecOption{
			schemahcl.WithToSpec(func(t schema.Type) (*schemahcl.Type, error) {
				i, ok := t.(*IntervalType)
				if !ok {
					return nil, fmt.Errorf("postgres: unexpected interval type %T", t)
				}
				spec := &schemahcl.Type{T: TypeInterval}
				if i.F != "" {
					spec.T = specutil.Var(strings.ToLower(i.F))
				}
				if p := i.Precision; p != nil && *p != defaultTimePrecision {
					spec.Attrs = []*schemahcl.Attr{schemahcl.IntAttr("precision", *p)}
				}
				return spec, nil
			}),
			schemahcl.WithFromSpec(func(t *schemahcl.Type) (schema.Type, error) {
				i := &IntervalType{T: TypeInterval}
				if t.T != TypeInterval {
					i.F = specutil.FromVar(t.T)
				}
				if a, ok := attr(t, "precision"); ok {
					p, err := a.Int()
					if err != nil {
						return nil, fmt.Errorf(`postgres: parsing attribute "precision": %w`, err)
					}
					if p != defaultTimePrecision {
						i.Precision = &p
					}
				}
				return i, nil
			}),
		}
		for _, f := range []string{"interval", "second", "day to second", "hour to second", "minute to second"} {
			specs = append(specs, schemahcl.NewTypeSpec(specutil.Var(f), append(opts, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr()))...))
		}
		for _, f := range []string{"year", "month", "day", "hour", "minute", "year to month", "day to hour", "day to minute", "hour to minute"} {
			specs = append(specs, schemahcl.NewTypeSpec(specutil.Var(f), opts...))
		}
		return specs
	}()...),
)

func attr(typ *schemahcl.Type, key string) (*schemahcl.Attr, bool) {
	for _, a := range typ.Attrs {
		if a.K == key {
			return a, true
		}
	}
	return nil, false
}

func typeSpec(t schema.Type) (*schemahcl.Type, error) {
	if t, ok := t.(*schema.TimeType); ok && t.T != TypeDate {
		spec := &schemahcl.Type{T: timeAlias(t.T)}
		if p := t.Precision; p != nil && *p != defaultTimePrecision {
			spec.Attrs = []*schemahcl.Attr{schemahcl.IntAttr("precision", *p)}
		}
		return spec, nil
	}
	s, err := FormatType(t)
	if err != nil {
		return nil, err
	}
	return &schemahcl.Type{T: s}, nil
}

// formatTime overrides the default printing logic done by schemahcl.hclType.
func formatTime() schemahcl.TypeSpecOption {
	return schemahcl.WithTypeFormatter(func(t *schemahcl.Type) (string, error) {
		a, ok := attr(t, "precision")
		if !ok {
			return t.T, nil
		}
		p, err := a.Int()
		if err != nil {
			return "", fmt.Errorf(`postgres: parsing attribute "precision": %w`, err)
		}
		return FormatType(&schema.TimeType{T: t.T, Precision: &p})
	})
}

// generatedType returns the default and only type for a generated column.
func generatedType(string) string { return "STORED" }
