// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// evalSpec evaluates an Atlas DDL document into v using the input.
func evalSpec(p *hclparse.Parser, v any, input map[string]cty.Value) error {
	switch v := v.(type) {
	case *schema.Realm:
		var d specutil.Doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if err := specutil.Scan(v,
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views, Funcs: d.Funcs, Procs: d.Procs, Triggers: d.Triggers},
			scanFuncs,
		); err != nil {
			return fmt.Errorf("mysql: failed converting to *schema.Realm: %w", err)
		}
		for _, spec := range d.Schemas {
			s, ok := v.Schema(spec.Name)
			if !ok {
				return fmt.Errorf("could not find schema: %q", spec.Name)
			}
			if err := convertCharset(spec, &s.Attrs); err != nil {
				return err
			}
		}
	case *schema.Schema:
		var d specutil.Doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if len(d.Schemas) != 1 {
			return fmt.Errorf("mysql: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		r := &schema.Realm{}
		if err := specutil.Scan(r,
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views, Funcs: d.Funcs, Procs: d.Procs, Triggers: d.Triggers},
			scanFuncs,
		); err != nil {
			return err
		}
		if err := convertCharset(d.Schemas[0], &r.Schemas[0].Attrs); err != nil {
			return err
		}
		*v = *r.Schemas[0]
	case schema.Schema, schema.Realm:
		return fmt.Errorf("mysql: Eval expects a pointer: received %[1]T, expected *%[1]T", v)
	default:
		return hclState.Eval(p, v, input)
	}
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemahcl.Marshaler.
func MarshalSpec(v any, marshaler schemahcl.Marshaler) ([]byte, error) {
	return specutil.Marshal(v, marshaler, specutil.RealmFuncs{
		Schema:   schemaSpec,
		Triggers: triggersSpec,
	})
}

var (
	hclState = schemahcl.New(
		append(
			specOptions,
			schemahcl.WithTypes("table.column.type", TypeRegistry.Specs()),
			schemahcl.WithTypes("view.column.type", TypeRegistry.Specs()),
			schemahcl.WithScopedEnums("view.check_option", schema.ViewCheckOptionLocal, schema.ViewCheckOptionCascaded),
			schemahcl.WithScopedEnums("table.engine", EngineInnoDB, EngineMyISAM, EngineMemory, EngineCSV, EngineNDB),
			schemahcl.WithScopedEnums("table.index.type", IndexTypeBTree, IndexTypeHash, IndexTypeFullText, IndexTypeSpatial),
			schemahcl.WithScopedEnums("table.index.parser", IndexParserNGram, IndexParserMeCab),
			schemahcl.WithScopedEnums("table.primary_key.type", IndexTypeBTree, IndexTypeHash, IndexTypeFullText, IndexTypeSpatial),
			schemahcl.WithScopedEnums("table.column.as.type", stored, persistent, virtual),
			schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
			schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
		)...,
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
	t, err := specutil.Table(spec, parent, convertColumn, convertPK, convertIndex, convertCheck)
	if err != nil {
		return nil, err
	}
	if err := convertCharset(spec, &t.Attrs); err != nil {
		return nil, err
	}
	// MySQL allows setting the initial AUTO_INCREMENT value
	// on the table definition.
	if attr, ok := spec.Attr("auto_increment"); ok {
		v, err := attr.Int64()
		if err != nil {
			return nil, err
		}
		t.AddAttrs(&AutoIncrement{V: v})
	}
	if attr, ok := spec.Attr("engine"); ok {
		v, err := attr.String()
		if err != nil {
			return nil, err
		}
		t.AddAttrs(&Engine{V: v})
	}
	return t, err
}

// convertView converts a sqlspec.View to a schema.View.
func convertView(spec *sqlspec.View, parent *schema.Schema) (*schema.View, error) {
	v, err := specutil.View(
		spec, parent,
		func(c *sqlspec.Column, _ *schema.View) (*schema.Column, error) {
			return specutil.Column(c, convertColumnType)
		},
		func(i *sqlspec.Index, v *schema.View) (*schema.Index, error) {
			return nil, fmt.Errorf("unexpected view index %s.%s", v.Name, i.Name)
		},
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// convertPK converts a sqlspec.PrimaryKey into a schema.Index.
func convertPK(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	idx, err := specutil.PrimaryKey(spec, parent)
	if err != nil {
		return nil, err
	}
	if err := convertIndexType(spec, idx); err != nil {
		return nil, err
	}
	if err := convertIndexParser(spec, idx); err != nil {
		return nil, err
	}
	return idx, nil
}

// convertIndex converts a sqlspec.Index into a schema.Index.
func convertIndex(spec *sqlspec.Index, parent *schema.Table) (*schema.Index, error) {
	idx, err := specutil.Index(spec, parent, convertPart)
	if err != nil {
		return nil, err
	}
	if err := convertIndexType(spec, idx); err != nil {
		return nil, err
	}
	if err := convertIndexParser(spec, idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func convertIndexType(spec specutil.Attrer, idx *schema.Index) error {
	if attr, ok := spec.Attr("type"); ok {
		t, err := attr.String()
		if err != nil {
			return err
		}
		idx.AddAttrs(&IndexType{T: t})
	}
	return nil
}

func convertIndexParser(spec specutil.Attrer, idx *schema.Index) error {
	if attr, ok := spec.Attr("parser"); ok {
		p, err := attr.String()
		if err != nil {
			return err
		}
		idx.AddAttrs(&IndexParser{P: p})
	}
	return nil
}

func convertPart(spec *sqlspec.IndexPart, part *schema.IndexPart) error {
	if attr, ok := spec.Attr("prefix"); ok {
		if part.X != nil {
			return errors.New("attribute 'on.prefix' cannot be used in functional part")
		}
		p, err := attr.Int()
		if err != nil {
			return err
		}
		part.AddAttrs(&SubPart{Len: p})
	}
	return nil
}

// convertCheck converts a sqlspec.Check into a schema.Check.
func convertCheck(spec *sqlspec.Check) (*schema.Check, error) {
	c, err := specutil.Check(spec)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("enforced"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		c.AddAttrs(&Enforced{V: b})
	}
	return c, nil
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
	}
	if err := convertCharset(spec, &c.Attrs); err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("on_update"); ok {
		x, err := attr.RawExpr()
		if err != nil {
			return nil, fmt.Errorf(`unexpected type %T for attribute "on_update"`, attr.V.Type())
		}
		c.AddAttrs(&OnUpdate{A: x.X})
	}
	if attr, ok := spec.Attr("auto_increment"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		if b {
			c.AddAttrs(&AutoIncrement{})
		}
	}
	if err := specutil.ConvertGenExpr(spec.Remain(), c, storedOrVirtual); err != nil {
		return nil, err
	}
	return c, err
}

// convertColumnType converts a sqlspec.Column into a concrete MySQL schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

// schemaSpec converts from a concrete MySQL schema to Atlas specification.
func schemaSpec(s *schema.Schema) (*specutil.SchemaSpec, error) {
	spec, err := specutil.FromSchema(s, specFuncs)
	if err != nil {
		return nil, err
	}
	if c, ok := sqlx.Charset(s.Attrs, nil); ok {
		spec.Schema.Extra.Attrs = append(spec.Schema.Extra.Attrs, schemahcl.StringAttr("charset", c))
	}
	if c, ok := sqlx.Collate(s.Attrs, nil); ok {
		spec.Schema.Extra.Attrs = append(spec.Schema.Extra.Attrs, schemahcl.StringAttr("collate", c))
	}
	return spec, nil
}

// tableSpec converts from a concrete MySQL sqlspec.Table to a schema.Table.
func tableSpec(t *schema.Table) (*sqlspec.Table, error) {
	ts, err := specutil.FromTable(
		t,
		columnSpec,
		pkSpec,
		indexSpec,
		specutil.FromForeignKey,
		checkSpec,
	)
	if err != nil {
		return nil, err
	}
	if c, ok := sqlx.Charset(t.Attrs, t.Schema.Attrs); ok {
		ts.Extra.Attrs = append(ts.Extra.Attrs, schemahcl.StringAttr("charset", c))
	}
	if c, ok := sqlx.Collate(t.Attrs, t.Schema.Attrs); ok {
		ts.Extra.Attrs = append(ts.Extra.Attrs, schemahcl.StringAttr("collate", c))
	}
	// Marshal the engine attribute only if it is not InnoDB (default).
	if e := (&Engine{}); sqlx.Has(t.Attrs, e) && e.V != "" && !e.Default {
		attr := schemahcl.StringAttr("engine", e.V)
		for _, e1 := range []string{EngineInnoDB, EngineMyISAM, EngineMemory, EngineCSV, EngineNDB} {
			if strings.EqualFold(e.V, e1) {
				attr = specutil.VarAttr("engine", e1)
				break
			}
		}
		ts.Extra.Attrs = append(ts.Extra.Attrs, attr)
	}
	return ts, nil
}

// viewSpec converts from a concrete MySQL schema.View to a sqlspec.View.
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
	spec.Extra.Attrs = indexTypeSpec(idx, spec.Extra.Attrs)
	return spec, nil
}

func indexSpec(idx *schema.Index) (*sqlspec.Index, error) {
	spec, err := specutil.FromIndex(idx, partAttr)
	if err != nil {
		return nil, err
	}
	spec.Extra.Attrs = indexTypeSpec(idx, spec.Extra.Attrs)
	return spec, nil
}

func indexTypeSpec(idx *schema.Index, attrs []*schemahcl.Attr) []*schemahcl.Attr {
	// Avoid printing the index type if it is the default.
	if i := (IndexType{}); sqlx.Has(idx.Attrs, &i) && i.T != IndexTypeBTree {
		attrs = append(attrs, specutil.VarAttr("type", strings.ToUpper(i.T)))
	}
	// Print fulltext index parser. Use the pre-defined parser variables if known.
	if p := (IndexParser{}); sqlx.Has(idx.Attrs, &p) && p.P != "" {
		attr := schemahcl.StringAttr("parser", p.P)
		for _, p1 := range []string{IndexParserNGram, IndexParserMeCab} {
			if strings.EqualFold(p.P, p1) {
				attr = specutil.VarAttr("parser", p1)
				break
			}
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

func partAttr(_ *schema.Index, part *schema.IndexPart, spec *sqlspec.IndexPart) error {
	if p := (SubPart{}); sqlx.Has(part.Attrs, &p) && p.Len > 0 {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.IntAttr("prefix", p.Len))
	}
	return nil
}

// columnSpec converts from a concrete MySQL schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, t *schema.Table) (*sqlspec.Column, error) {
	spec, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if c, ok := sqlx.Charset(c.Attrs, t.Attrs); ok {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.StringAttr("charset", c))
	}
	if c, ok := sqlx.Collate(c.Attrs, t.Attrs); ok {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.StringAttr("collate", c))
	}
	if o := (OnUpdate{}); sqlx.Has(c.Attrs, &o) {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.RawAttr("on_update", o.A))
	}
	if sqlx.Has(c.Attrs, &AutoIncrement{}) {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.BoolAttr("auto_increment", true))
	}
	if x := (schema.GeneratedExpr{}); sqlx.Has(c.Attrs, &x) {
		spec.Extra.Children = append(spec.Extra.Children, specutil.FromGenExpr(x, storedOrVirtual))
	}
	return spec, nil
}

// storedOrVirtual returns a STORED or VIRTUAL
// generated type option based on the given string.
func storedOrVirtual(s string) string {
	switch s = strings.ToUpper(s); s {
	// The default is VIRTUAL if no type is specified.
	case "":
		return virtual
	// In MariaDB, PERSISTENT is synonyms for STORED.
	case persistent:
		return stored
	}
	return s
}

// checkSpec converts from a concrete MySQL schema.Check into a sqlspec.Check.
func checkSpec(s *schema.Check) *sqlspec.Check {
	c := specutil.FromCheck(s)
	if e := (Enforced{}); sqlx.Has(s.Attrs, &e) {
		c.Extra.Attrs = append(c.Extra.Attrs, schemahcl.BoolAttr("enforced", true))
	}
	return c
}

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	c := &sqlspec.Column{Type: st}
	for _, attr := range st.Attrs {
		// TODO(rotemtam): infer this from the TypeSpec
		if attr.K == "unsigned" {
			c.Extra.Attrs = append(c.Extra.Attrs, attr)
		}
	}
	return c, nil
}

// convertCharset converts spec charset/collation
// attributes to schema element attributes.
func convertCharset(spec specutil.Attrer, attrs *[]schema.Attr) error {
	if attr, ok := spec.Attr("charset"); ok {
		s, err := attr.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Charset{V: s})
	}
	// For backwards compatibility, accepts both "collate" and "collation".
	attr, ok := spec.Attr("collate")
	if !ok {
		attr, ok = spec.Attr("collation")
	}
	if ok {
		s, err := attr.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Collation{V: s})
	}
	return nil
}

// TypeRegistry contains the supported TypeSpecs for the mysql driver.
var TypeRegistry = schemahcl.NewRegistry(
	schemahcl.WithFormatter(FormatType),
	schemahcl.WithParser(ParseType),
	schemahcl.WithSpecs(
		&schemahcl.TypeSpec{
			Name: TypeEnum,
			T:    TypeEnum,
			Attributes: []*schemahcl.TypeAttr{
				{Name: "values", Kind: reflect.Slice, Required: true},
			},
			RType: reflect.TypeOf(schema.EnumType{}),
			FromSpec: func(t *schemahcl.Type) (schema.Type, error) {
				if len(t.Attrs) != 1 || t.Attrs[0].K != "values" {
					return nil, fmt.Errorf("invalid enum type spec: %v", t)
				}
				v, err := t.Attrs[0].Strings()
				if err != nil {
					return nil, err
				}
				return &schema.EnumType{T: "enum", Values: v}, nil
			},
		},
		&schemahcl.TypeSpec{
			Name: TypeSet,
			T:    TypeSet,
			Attributes: []*schemahcl.TypeAttr{
				{Name: "values", Kind: reflect.Slice, Required: true},
			},
			RType: reflect.TypeOf(SetType{}),
			FromSpec: func(t *schemahcl.Type) (schema.Type, error) {
				if len(t.Attrs) != 1 || t.Attrs[0].K != "values" {
					return nil, fmt.Errorf("invalid set type spec: %v", t)
				}
				v, err := t.Attrs[0].Strings()
				if err != nil {
					return nil, err
				}
				return &SetType{Values: v}, nil
			},
		},
		schemahcl.NewTypeSpec(TypeBool),
		schemahcl.NewTypeSpec(TypeBoolean),
		schemahcl.NewTypeSpec(TypeBit, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeInt, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeTinyInt, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeSmallInt, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeMediumInt, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeBigInt, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeDecimal, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeNumeric, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeFloat, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeDouble, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeReal, schemahcl.WithAttributes(unsignedTypeAttr(), schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeTimestamp, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
		schemahcl.NewTypeSpec(TypeDate),
		schemahcl.NewTypeSpec(TypeTime, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
		schemahcl.NewTypeSpec(TypeDateTime, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
		schemahcl.NewTypeSpec(TypeYear, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
		schemahcl.NewTypeSpec(TypeVarchar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(true))),
		schemahcl.NewTypeSpec(TypeChar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeVarBinary, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(true))),
		schemahcl.NewTypeSpec(TypeBinary, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeBlob, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeTinyBlob),
		schemahcl.NewTypeSpec(TypeMediumBlob),
		schemahcl.NewTypeSpec(TypeLongBlob),
		schemahcl.NewTypeSpec(TypeJSON),
		schemahcl.NewTypeSpec(TypeText, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeTinyText),
		schemahcl.NewTypeSpec(TypeMediumText),
		schemahcl.NewTypeSpec(TypeLongText),
		schemahcl.NewTypeSpec(TypeGeometry),
		schemahcl.NewTypeSpec(TypePoint),
		schemahcl.NewTypeSpec(TypeMultiPoint),
		schemahcl.NewTypeSpec(TypeLineString),
		schemahcl.NewTypeSpec(TypeMultiLineString),
		schemahcl.NewTypeSpec(TypePolygon),
		schemahcl.NewTypeSpec(TypeMultiPolygon),
		schemahcl.NewTypeSpec(TypeGeometryCollection),
		schemahcl.NewTypeSpec(TypeInet4),
		schemahcl.NewTypeSpec(TypeInet6),
	),
)

func unsignedTypeAttr() *schemahcl.TypeAttr {
	return &schemahcl.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
