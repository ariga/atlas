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
		Tables        []*sqlspec.Table    `spec:"table"`
		Views         []*sqlspec.View     `spec:"view"`
		Materialized  []*sqlspec.View     `spec:"materialized"`
		Enums         []*enum             `spec:"enum"`
		Domains       []*domain           `spec:"domain"`
		Composites    []*composite        `spec:"composite"`
		Sequences     []*sqlspec.Sequence `spec:"sequence"`
		Funcs         []*sqlspec.Func     `spec:"function"`
		Procs         []*sqlspec.Func     `spec:"procedure"`
		Aggregates    []*aggregate        `spec:"aggregate"`
		Triggers      []*sqlspec.Trigger  `spec:"trigger"`
		EventTriggers []*eventTrigger     `spec:"event_trigger"`
		Extensions    []*extension        `spec:"extension"`
		Schemas       []*sqlspec.Schema   `spec:"schema"`
	}

	// Enum holds a specification for an enum type.
	enum struct {
		Name      string         `spec:",name"`
		Qualifier string         `spec:",qualifier"`
		Schema    *schemahcl.Ref `spec:"schema"`
		Values    []string       `spec:"values"`
		schemahcl.DefaultExtension
	}

	// domain holds a specification for a domain type.
	domain struct {
		Name      string           `spec:",name"`
		Qualifier string           `spec:",qualifier"`
		Schema    *schemahcl.Ref   `spec:"schema"`
		Type      *schemahcl.Type  `spec:"type"`
		Null      bool             `spec:"null"`
		Default   cty.Value        `spec:"default"`
		Checks    []*sqlspec.Check `spec:"check"`
		schemahcl.DefaultExtension
	}

	// composite holds a specification for a composite type.
	composite struct {
		Name      string            `spec:",name"`
		Qualifier string            `spec:",qualifier"`
		Schema    *schemahcl.Ref    `spec:"schema"`
		Fields    []*compositeField `spec:"field"`
		schemahcl.DefaultExtension
	}

	// compositeField holds a specification for a field in a composite type.
	// The extension might hold optional attributes such as collation.
	compositeField struct {
		Name string          `spec:",name"`
		Type *schemahcl.Type `spec:"type"`
		schemahcl.DefaultExtension
	}

	// extension holds a specification for a postgres extension.
	// Note, extension names are unique within a realm (database).
	extension struct {
		Name string `spec:",name"`
		// Schema, version and comment are conditionally
		// added to the extension definition.
		schemahcl.DefaultExtension
	}

	// eventTrigger holds a specification for a postgres event trigger.
	// Note, event trigger names are unique within a realm (database).
	eventTrigger struct {
		Name string `spec:",name"`
		// Schema, version and comment are conditionally
		// added to the extension definition.
		schemahcl.DefaultExtension
	}

	// aggregate holds the specification for an aggregation function.
	aggregate struct {
		Name      string             `spec:",name"`
		Qualifier string             `spec:",qualifier"`
		Schema    *schemahcl.Ref     `spec:"schema"`
		Args      []*sqlspec.FuncArg `spec:"arg"`
		// state_type, state_func and rest of the attributes
		// are appended after the function arguments.
		schemahcl.DefaultExtension
	}
)

// merge merges the doc d1 into d.
func (d *doc) merge(d1 *doc) {
	d.Enums = append(d.Enums, d1.Enums...)
	d.Funcs = append(d.Funcs, d1.Funcs...)
	d.Procs = append(d.Procs, d1.Procs...)
	d.Views = append(d.Views, d1.Views...)
	d.Tables = append(d.Tables, d1.Tables...)
	d.Domains = append(d.Domains, d1.Domains...)
	d.Composites = append(d.Composites, d1.Composites...)
	d.Schemas = append(d.Schemas, d1.Schemas...)
	d.Aggregates = append(d.Aggregates, d1.Aggregates...)
	d.Sequences = append(d.Sequences, d1.Sequences...)
	d.Extensions = append(d.Extensions, d1.Extensions...)
	d.Triggers = append(d.Triggers, d1.Triggers...)
	d.EventTriggers = append(d.EventTriggers, d1.EventTriggers...)
	d.Materialized = append(d.Materialized, d1.Materialized...)
}

func (d *doc) ScanDoc() *specutil.ScanDoc {
	return &specutil.ScanDoc{
		Schemas:      d.Schemas,
		Tables:       d.Tables,
		Views:        d.Views,
		Funcs:        d.Funcs,
		Procs:        d.Procs,
		Triggers:     d.Triggers,
		Materialized: d.Materialized,
	}
}

// Label returns the defaults label used for the enum resource.
func (e *enum) Label() string { return e.Name }

// QualifierLabel returns the qualifier label used for the enum resource, if any.
func (e *enum) QualifierLabel() string { return e.Qualifier }

// SetQualifier sets the qualifier label used for the enum resource.
func (e *enum) SetQualifier(q string) { e.Qualifier = q }

// SchemaRef returns the schema reference for the enum.
func (e *enum) SchemaRef() *schemahcl.Ref { return e.Schema }

// Label returns the defaults label used for the domain resource.
func (d *domain) Label() string { return d.Name }

// QualifierLabel returns the qualifier label used for the domain resource, if any.
func (d *domain) QualifierLabel() string { return d.Qualifier }

// SetQualifier sets the qualifier label used for the domain resource.
func (d *domain) SetQualifier(q string) { d.Qualifier = q }

// SchemaRef returns the schema reference for the domain.
func (d *domain) SchemaRef() *schemahcl.Ref { return d.Schema }

// Label returns the defaults label used for the composite resource.
func (c *composite) Label() string { return c.Name }

// QualifierLabel returns the qualifier label used for the composite resource, if any.
func (c *composite) QualifierLabel() string { return c.Qualifier }

// SetQualifier sets the qualifier label used for the composite resource.
func (c *composite) SetQualifier(q string) { c.Qualifier = q }

// SchemaRef returns the schema reference for the composite.
func (c *composite) SchemaRef() *schemahcl.Ref { return c.Schema }

// Label returns the defaults label used for the aggregate resource.
func (a *aggregate) Label() string { return a.Name }

// QualifierLabel returns the qualifier label used for the aggregate resource, if any.
func (a *aggregate) QualifierLabel() string { return a.Qualifier }

// SetQualifier sets the qualifier label used for the aggregate resource.
func (a *aggregate) SetQualifier(q string) { a.Qualifier = q }

// SchemaRef returns the schema reference for the aggregate.
func (a *aggregate) SchemaRef() *schemahcl.Ref { return a.Schema }

func init() {
	schemahcl.Register("enum", &enum{})
	schemahcl.Register("domain", &domain{})
	schemahcl.Register("composite", &composite{})
	schemahcl.Register("aggregate", &aggregate{})
	schemahcl.Register("extension", &extension{})
	schemahcl.Register("event_trigger", &eventTrigger{})
}

// evalSpec evaluates an Atlas DDL document into v using the input.
func evalSpec(p *hclparse.Parser, v any, input map[string]cty.Value) error {
	switch v := v.(type) {
	case *schema.Realm:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if err := specutil.Scan(v, d.ScanDoc(), scanFuncs); err != nil {
			return fmt.Errorf("specutil: failed converting to *schema.Realm: %w", err)
		}
		if err := convertTypes(&d, v); err != nil {
			return err
		}
		if err := convertAggregate(&d, v); err != nil {
			return err
		}
		if err := convertSequences(d.Tables, d.Sequences, v); err != nil {
			return err
		}
		if err := convertExtensions(d.Extensions, v); err != nil {
			return err
		}
		if err := convertEventTriggers(d.EventTriggers, v); err != nil {
			return err
		}
		if err := normalizeRealm(v); err != nil {
			return err
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
		if err := specutil.Scan(r, d.ScanDoc(), scanFuncs); err != nil {
			return err
		}
		if err := convertTypes(&d, r); err != nil {
			return err
		}
		if err := convertAggregate(&d, r); err != nil {
			return err
		}
		if err := convertSequences(d.Tables, d.Sequences, r); err != nil {
			return err
		}
		// Extensions are skipped in schema scope.
		if err := normalizeRealm(r); err != nil {
			return err
		}
		*v = *r.Schemas[0]
	case schema.Schema, schema.Realm:
		return fmt.Errorf("postgres: Eval expects a pointer: received %[1]T, expected *%[1]T", v)
	default:
		return fmt.Errorf("postgres: unexpected type %T", v)
	}
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemahcl.Marshaler.
func MarshalSpec(v any, marshaler schemahcl.Marshaler) ([]byte, error) {
	var (
		d  doc
		ts []*schema.Trigger
	)
	switch rv := v.(type) {
	case *schema.Schema:
		d1, trs, err := schemaSpec(rv)
		if err != nil {
			return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
		}
		ts = trs
		d.merge(d1)
	case *schema.Realm:
		for _, s := range rv.Schemas {
			d1, trs, err := schemaSpec(s)
			if err != nil {
				return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
			}
			d.merge(d1)
			ts = append(ts, trs...)
		}
		if err := realmObjectsSpec(&d, rv); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Tables); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Views); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Materialized); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Aggregates); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Enums); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Domains); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Composites); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Sequences); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Funcs); err != nil {
			return nil, err
		}
		if err := specutil.QualifyObjects(d.Procs); err != nil {
			return nil, err
		}
		if err := specutil.QualifyReferences(d.Tables, rv); err != nil {
			return nil, err
		}
		if err := qualifySeqRefs(d.Sequences, d.Tables, rv); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("specutil: failed marshaling spec. %T is not supported", v)
	}
	if err := triggersSpec(ts, &d); err != nil {
		return nil, err
	}
	return marshaler.MarshalSpec(&d)
}

var (
	hclState = schemahcl.New(append(specOptions,
		schemahcl.WithTypes("table.column.type", TypeRegistry.Specs()),
		schemahcl.WithTypes("view.column.type", TypeRegistry.Specs()),
		schemahcl.WithTypes("materialized.column.type", TypeRegistry.Specs()),
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
	if err := convertUnique(spec.Extra, t); err != nil {
		return nil, err
	}
	if err := convertExclude(spec.Extra, t); err != nil {
		return nil, err
	}
	if err := convertPartition(spec.Extra, t); err != nil {
		return nil, err
	}
	return t, nil
}

// convertView converts a sqlspec.View to a schema.View.
func convertView(spec *sqlspec.View, parent *schema.Schema) (*schema.View, error) {
	v, err := specutil.View(
		spec, parent,
		func(c *sqlspec.Column, _ *schema.View) (*schema.Column, error) {
			return specutil.Column(c, convertColumnType)
		},
		func(i *sqlspec.Index, v *schema.View) (*schema.Index, error) {
			idx, err := convertIndex(i, v.AsTable())
			if err != nil {
				return nil, err
			}
			idx.Table, idx.View = nil, v
			return idx, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// convertUnique converts the unique constraints into indexes.
func convertUnique(spec schemahcl.Resource, t *schema.Table) error {
	rs := spec.Resources("unique")
	for _, r := range rs {
		var sx sqlspec.Index
		if err := r.As(&sx); err != nil {
			return fmt.Errorf("parse %s.unique constraint: %w", t.Name, err)
		}
		idx, err := convertIndex(&sx, t)
		if err != nil {
			return err
		}
		idx.SetUnique(true).AddAttrs(UniqueConstraint(sx.Name))
		t.AddIndexes(idx)
	}
	return nil
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
	// The following attributes are mutually exclusive.
	if nulls, ok := spec.Attr("nulls_last"); ok {
		b, err := nulls.Bool()
		if err != nil {
			return err
		}
		if b {
			at := sqlx.AttrOr(part.Attrs, &IndexColumnProperty{})
			at.NullsLast = true
			schema.ReplaceOrAppend(&part.Attrs, at)
		}
	}
	if nulls, ok := spec.Attr("nulls_first"); ok {
		b, err := nulls.Bool()
		if err != nil {
			return err
		}
		if b {
			at := sqlx.AttrOr(part.Attrs, &IndexColumnProperty{})
			at.NullsFirst = true
			schema.ReplaceOrAppend(&part.Attrs, at)
		}
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

// enumName extracts the name of the referenced Enum from the reference string.
func enumName(ref *schemahcl.Type) (string, error) {
	s := strings.Split(ref.T, "$enum.")
	if len(s) != 2 {
		return "", fmt.Errorf("postgres: failed to extract enum name from %q", ref.T)
	}
	return s[1], nil
}

// schemaSpec converts from a concrete Postgres schema to Atlas specification.
func schemaSpec(s *schema.Schema) (*doc, []*schema.Trigger, error) {
	spec, err := specutil.FromSchema(s, specFuncs)
	if err != nil {
		return nil, nil, err
	}
	d := &doc{
		Tables:       spec.Tables,
		Views:        spec.Views,
		Materialized: spec.Materialized,
		Funcs:        spec.Funcs,
		Procs:        spec.Procs,
		Schemas:      []*sqlspec.Schema{spec.Schema},
		Enums:        make([]*enum, 0, len(s.Objects)),
		Domains:      make([]*domain, 0, len(s.Objects)),
		Composites:   make([]*composite, 0, len(s.Objects)),
	}
	if err := objectSpec(d, spec, s); err != nil {
		return nil, nil, err
	}
	return d, spec.Triggers, nil
}

// tableSpec converts from a concrete Postgres sqlspec.Table to a schema.Table.
func tableSpec(t *schema.Table) (*sqlspec.Table, error) {
	spec, err := specutil.FromTable(
		t,
		tableColumnSpec,
		pkSpec,
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
	if err != nil {
		return nil, err
	}
	idxs := make([]*sqlspec.Index, 0, len(spec.Indexes))
	for i, idx1 := range spec.Indexes {
		if len(t.Indexes) <= i || t.Indexes[i].Name != idx1.Name {
			return nil, fmt.Errorf("unexpected spec index %q was not found in table %q", idx1.Name, t.Name)
		}
		if c, ok := uniqueConst(t.Indexes[i].Attrs); ok && c.N != "" {
			spec.Extra.Children = append(spec.Extra.Children, &schemahcl.Resource{
				Type: "unique",
				Name: c.N,
				Attrs: append(
					[]*schemahcl.Attr{schemahcl.RefsAttr("columns", idx1.Columns...)},
					idx1.Extra.Attrs...,
				),
			})
		} else if ex, ok := excludeConst(t.Indexes[i].Attrs); !ok {
			idxs = append(idxs, idx1)
		} else if err := excludeSpec(spec, idx1, t.Indexes[i], ex); err != nil {
			return nil, err
		}
	}
	spec.Indexes = idxs
	if p := (Partition{}); sqlx.Has(t.Attrs, &p) {
		spec.Extra.Children = append(spec.Extra.Children, fromPartition(p))
	}
	return spec, nil
}

// viewSpec converts from a concrete PostgreSQL schema.View to a sqlspec.View.
func viewSpec(view *schema.View) (*sqlspec.View, error) {
	spec, err := specutil.FromView(
		view,
		func(c *schema.Column, _ *schema.View) (*sqlspec.Column, error) {
			return specutil.FromColumn(c, columnTypeSpec)
		},
		indexSpec,
	)
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
		var attr *schemahcl.Attr
		switch strings.ToUpper(i.T) {
		case IndexTypeBRIN, IndexTypeHash, IndexTypeGIN, IndexTypeGiST, IndexTypeSPGiST:
			attr = specutil.VarAttr("type", strings.ToUpper(i.T))
		default:
			attr = schemahcl.StringAttr("type", i.T)
		}
		spec.Extra.Attrs = append(spec.Extra.Attrs, attr)
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
	if op := (IndexOpClass{}); sqlx.Has(part.Attrs, &op) {
		switch d, err := op.DefaultFor(idx, part); {
		case err != nil:
			return err
		case d:
		case len(op.Params) > 0:
			spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.RawAttr("ops", op.String()))
		case postgresop.HasClass(op.String()):
			spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("ops", op.String()))
		default:
			spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.StringAttr("ops", op.String()))
		}
	}
	if nulls := (IndexColumnProperty{}); sqlx.Has(part.Attrs, &nulls) {
		switch {
		case part.Desc && nulls.NullsLast:
			spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.BoolAttr("nulls_last", true))
		case !part.Desc && nulls.NullsFirst:
			spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.BoolAttr("nulls_first", true))
		}
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
	switch o := t.(type) {
	case *schema.EnumType:
		return &sqlspec.Column{
			Type: &schemahcl.Type{
				IsRef: true,
				T:     specutil.ObjectRef(o.Schema, o).V},
		}, nil
	case *CompositeType:
		return &sqlspec.Column{
			Type: &schemahcl.Type{
				IsRef: true,
				T:     specutil.ObjectRef(o.Schema, o).V},
		}, nil
	case *DomainType:
		return &sqlspec.Column{
			Type: &schemahcl.Type{
				IsRef: true,
				T:     specutil.ObjectRef(o.Schema, o).V},
		}, nil
	default:
		st, err := TypeRegistry.Convert(t)
		if err != nil {
			return nil, err
		}
		return &sqlspec.Column{Type: st}, nil
	}
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
		schemahcl.NewTypeSpec(TypeBPChar),
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
	),
	// PostgreSQL internal, pseudo, and special types.
	schemahcl.WithSpecs(func() (specs []*schemahcl.TypeSpec) {
		for _, t := range []string{
			typeOID, typeRegClass, typeRegCollation, typeRegConfig, typeRegDictionary, typeRegNamespace,
			typeName, typeRegOper, typeRegOperator, typeRegProc, typeRegProcedure, typeRegRole, typeRegType,
			typeAny, typeAnyElement, typeAnyArray, typeAnyNonArray, typeAnyEnum, typeInternal, typeRecord,
			typeTrigger, typeEventTrigger, typeVoid, typeUnknown,
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
