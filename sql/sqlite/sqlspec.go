// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlite

import (
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type doc struct {
	Tables   []*sqlspec.Table   `spec:"table"`
	Views    []*sqlspec.View    `spec:"view"`
	Triggers []*sqlspec.Trigger `spec:"trigger"`
	Schemas  []*sqlspec.Schema  `spec:"schema"`
}

// evalSpec evaluates an Atlas DDL document using an unmarshaler into v by using the input.
func evalSpec(p *hclparse.Parser, v any, input map[string]cty.Value) error {
	switch v := v.(type) {
	case *schema.Realm:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if err := specutil.Scan(v,
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views, Triggers: d.Triggers},
			scanFuncs,
		); err != nil {
			return fmt.Errorf("specutil: failed converting to *schema.Realm: %w", err)
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
			&specutil.ScanDoc{Schemas: d.Schemas, Tables: d.Tables, Views: d.Views, Triggers: d.Triggers},
			scanFuncs,
		); err != nil {
			return err
		}
		*v = *r.Schemas[0]
	case schema.Schema, schema.Realm:
		return fmt.Errorf("sqlite: Eval expects a pointer: received %[1]T, expected *%[1]T", v)
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

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	t, err := specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, convertIndex, specutil.Check)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("without_rowid"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		if b {
			t.AddAttrs(&WithoutRowID{})
		}
	}
	if attr, ok := spec.Attr("strict"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		if b {
			t.AddAttrs(&Strict{})
		}
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
			return nil, fmt.Errorf("unexpected view index %s.%s", v.Name, i.Name)
		},
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// convertIndex converts a sqlspec.Index into a schema.Index.
func convertIndex(spec *sqlspec.Index, t *schema.Table) (*schema.Index, error) {
	idx, err := specutil.Index(spec, t)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("where"); ok {
		p, err := attr.String()
		if err != nil {
			return nil, err
		}
		idx.Attrs = append(idx.Attrs, &IndexPredicate{P: p})
	}
	return idx, nil
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
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
	return c, nil
}

// convertColumnType converts a sqlspec.Column into a concrete SQLite schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

// schemaSpec converts from a concrete SQLite schema to Atlas specification.
func schemaSpec(s *schema.Schema) (*specutil.SchemaSpec, error) {
	return specutil.FromSchema(s, &specutil.SchemaFuncs{
		Table: tableSpec,
		View:  viewSpec,
	})
}

// tableSpec converts from a concrete SQLite sqlspec.Table to a schema.Table.
func tableSpec(t *schema.Table) (*sqlspec.Table, error) {
	spec, err := specutil.FromTable(
		t,
		columnSpec,
		specutil.FromPrimaryKey,
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
	if err != nil {
		return nil, err
	}
	options := &schemahcl.Resource{}
	if sqlx.Has(t.Attrs, &WithoutRowID{}) {
		options.SetAttr(schemahcl.BoolAttr("without_rowid", true))
	}
	if sqlx.Has(t.Attrs, &Strict{}) {
		options.SetAttr(schemahcl.BoolAttr("strict", true))
	}
	if options != nil {
		spec.Extra.Children = append(spec.Extra.Children, options)
	}
	return spec, nil
}

// viewSpec converts from a concrete SQLite schema.View to a sqlspec.View.
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

func indexSpec(idx *schema.Index) (*sqlspec.Index, error) {
	spec, err := specutil.FromIndex(idx)
	if err != nil {
		return nil, err
	}
	if i := (IndexPredicate{}); sqlx.Has(idx.Attrs, &i) && i.P != "" {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("where", strconv.Quote(i.P)))
	}
	return spec, nil
}

// columnSpec converts from a concrete SQLite schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	s, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if sqlx.Has(c.Attrs, &AutoIncrement{}) {
		s.Extra.Attrs = append(s.Extra.Attrs, schemahcl.BoolAttr("auto_increment", true))
	}
	if x := (schema.GeneratedExpr{}); sqlx.Has(c.Attrs, &x) {
		s.Extra.Children = append(s.Extra.Children, specutil.FromGenExpr(x, storedOrVirtual))
	}
	return s, nil
}

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{Type: st}, nil
}

// TypeRegistry contains the supported TypeSpecs for the sqlite driver.
var TypeRegistry = schemahcl.NewRegistry(
	schemahcl.WithFormatter(FormatType),
	schemahcl.WithParser(ParseType),
	schemahcl.WithSpecs(
		schemahcl.NewTypeSpec(TypeReal, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec(TypeBlob, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeText, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec(TypeInteger, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("int", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("tinyint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("smallint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("mediumint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("bigint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasTypeSpec("unsigned_big_int", "unsigned big int", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("int2", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("int8", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("uint64", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("double", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasTypeSpec("double_precision", "double precision", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("float", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("varchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasTypeSpec("varying_character", "varying character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("nchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasTypeSpec("native_character", "native character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("nvarchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("clob", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.NewTypeSpec("numeric", schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec("decimal", schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
		schemahcl.NewTypeSpec("bool"),
		schemahcl.NewTypeSpec("boolean"),
		schemahcl.NewTypeSpec("date"),
		schemahcl.NewTypeSpec("datetime"),
		schemahcl.NewTypeSpec("json"),
		schemahcl.NewTypeSpec("uuid"),
	),
)

var (
	hclState = schemahcl.New(append(
		specOptions,
		schemahcl.WithTypes("table.column.type", TypeRegistry.Specs()),
		schemahcl.WithTypes("view.column.type", TypeRegistry.Specs()),
		schemahcl.WithScopedEnums("table.column.as.type", stored, virtual),
		schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
	)...)
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

// storedOrVirtual returns a STORED or VIRTUAL
// generated type option based on the given string.
func storedOrVirtual(s string) string {
	if s = strings.ToUpper(s); s == "" {
		return virtual
	}
	return s
}
