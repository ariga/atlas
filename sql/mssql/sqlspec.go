// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

type doc struct {
	Tables  []*sqlspec.Table  `spec:"table"`
	Schemas []*sqlspec.Schema `spec:"schema"`
}

// evalSpec evaluates an Atlas DDL document into v using the input.
func evalSpec(p *hclparse.Parser, v any, input map[string]cty.Value) error {
	switch v := v.(type) {
	case *schema.Realm:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		err := specutil.Scan(v, d.Schemas, d.Tables, convertTable)
		if err != nil {
			return fmt.Errorf("mssql: failed converting to *schema.Realm: %w", err)
		}
		for _, spec := range d.Schemas {
			s, ok := v.Schema(spec.Name)
			if !ok {
				return fmt.Errorf("mssql: could not find schema: %q", spec.Name)
			}
			if err := convertCharset(spec, &s.Attrs); err != nil {
				return err
			}
		}
	case *schema.Schema:
		var d doc
		if err := hclState.Eval(p, &d, input); err != nil {
			return err
		}
		if len(d.Schemas) != 1 {
			return fmt.Errorf("mssql: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		var r schema.Realm
		if err := specutil.Scan(&r, d.Schemas, d.Tables, convertTable); err != nil {
			return err
		}
		if err := convertCharset(d.Schemas[0], &r.Schemas[0].Attrs); err != nil {
			return err
		}
		r.Schemas[0].Realm = nil
		*v = *r.Schemas[0]
	case schema.Schema, schema.Realm:
		return fmt.Errorf("mssql: eval expects a pointer: received %[1]T, expected *%[1]T", v)
	default:
		return hclState.Eval(p, v, input)
	}
	return nil
}

// convertCharset converts spec charset/collation
// attributes to schema element attributes.
func convertCharset(spec specutil.Attrer, attrs *[]schema.Attr) error {
	if attr, ok := spec.Attr("collate"); ok {
		s, err := attr.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Collation{V: s})
	}
	return nil
}

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, convertIndex, specutil.Check)
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
	}
	if r, ok := spec.Extra.Resource("identity"); ok {
		id := &Identity{}
		if err := id.fromSpec(r); err != nil {
			return nil, err
		}
		c.Attrs = append(c.Attrs, id)
	}
	// if err := specutil.ConvertGenExpr(spec.Remain(), c, generatedType); err != nil {
	// 	return nil, err
	// }
	return c, nil
}

// convertColumnType converts a sqlspec.Column into a schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
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

var (
	// TypeRegistry contains the supported TypeSpecs for the mssql driver.
	TypeRegistry = schemahcl.NewRegistry(
		schemahcl.WithFormatter(FormatType),
		schemahcl.WithParser(ParseType),
		schemahcl.WithSpecs(
			schemahcl.NewTypeSpec(TypeBigInt),
			schemahcl.NewTypeSpec(TypeBinary, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeBit),
			schemahcl.NewTypeSpec(TypeChar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeDate),
			schemahcl.NewTypeSpec(TypeDateTime),
			schemahcl.NewTypeSpec(TypeDateTime2, schemahcl.WithAttributes(schemahcl.ScaleTypeAttr())),
			schemahcl.NewTypeSpec(TypeDateTimeOffset, schemahcl.WithAttributes(schemahcl.ScaleTypeAttr())),
			schemahcl.NewTypeSpec(TypeDecimal, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
			schemahcl.NewTypeSpec(TypeFloat, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
			schemahcl.NewTypeSpec(TypeGeography),
			schemahcl.NewTypeSpec(TypeGeometry),
			schemahcl.NewTypeSpec(TypeHierarchyID),
			schemahcl.NewTypeSpec(TypeInt),
			schemahcl.NewTypeSpec(TypeMoney),
			schemahcl.NewTypeSpec(TypeNChar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeNumeric, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr(), schemahcl.ScaleTypeAttr())),
			schemahcl.NewTypeSpec(TypeNVarchar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeReal, schemahcl.WithAttributes(schemahcl.PrecisionTypeAttr())),
			schemahcl.NewTypeSpec(TypeRowVersion),
			schemahcl.NewTypeSpec(TypeSmallDateTime),
			schemahcl.NewTypeSpec(TypeSmallInt),
			schemahcl.NewTypeSpec(TypeSmallMoney),
			schemahcl.NewTypeSpec(TypeSQLVariant),
			schemahcl.NewTypeSpec(TypeTime, schemahcl.WithAttributes(schemahcl.ScaleTypeAttr())),
			schemahcl.NewTypeSpec(TypeTinyInt),
			schemahcl.NewTypeSpec(TypeUniqueIdentifier),
			schemahcl.NewTypeSpec(TypeVarBinary, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeVarchar, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
			schemahcl.NewTypeSpec(TypeXML),
			// Deprecated types
			schemahcl.NewTypeSpec(TypeText), schemahcl.NewTypeSpec(TypeNText), schemahcl.NewTypeSpec(TypeImage),
		),
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

	hclState = schemahcl.New(
		schemahcl.WithTypes("table.column.type", TypeRegistry.Specs()),
		schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
	)
)

// MarshalSpec marshals v into an Atlas DDL document using a schemahcl.Marshaler.
func MarshalSpec(v any, marshaler schemahcl.Marshaler) ([]byte, error) {
	return specutil.Marshal(v, marshaler, func(s *schema.Schema) (*specutil.SchemaSpec, error) {
		return specutil.FromSchema(s, tableSpec, viewSpec)
	})
}

// tableSpec converts from a concrete MSSQL sqlspec.Table to a schema.Table.
func tableSpec(table *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(
		table,
		columnSpec,
		specutil.FromPrimaryKey,
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
}

// columnSpec converts from a concrete MSSQL schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, t *schema.Table) (*sqlspec.Column, error) {
	spec, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if c, ok := sqlx.Collate(c.Attrs, t.Attrs); ok {
		spec.Extra.Attrs = append(spec.Extra.Attrs, schemahcl.StringAttr("collate", c))
	}
	if i := (&Identity{}); sqlx.Has(c.Attrs, i) {
		spec.Extra.Children = append(spec.Extra.Children, i.toSpec())
	}
	return spec, nil
}

// viewSpec converts from a concrete MSSQL schema.View to a sqlspec.View.
func viewSpec(view *schema.View) (*sqlspec.View, error) {
	return specutil.FromView(view, func(c *schema.Column, _ *schema.View) (*sqlspec.Column, error) {
		return specutil.FromColumn(c, columnTypeSpec)
	})
}

// columnTypeSpec converts from a concrete MSSQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{Type: st}, nil
}

// indexSpec converts from a concrete MSSQL schema.Index into a sqlspec.Index.
func indexSpec(idx *schema.Index) (*sqlspec.Index, error) {
	spec, err := specutil.FromIndex(idx)
	if err != nil {
		return nil, err
	}
	// Avoid printing the index type if it is the default.
	if a := (&IndexType{}); sqlx.Has(idx.Attrs, a) && strings.ToUpper(a.T) != IndexTypeClustered {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("type", strings.ToUpper(a.T)))
	}
	if a := (&IndexPredicate{}); sqlx.Has(idx.Attrs, a) && a.P != "" {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("where", strconv.Quote(a.P)))
	}
	return spec, nil
}

// fromSpec converts from a sqlspec.Resource into an Identity.
func (i *Identity) fromSpec(r *schemahcl.Resource) error {
	var spec struct {
		Seed      int64 `spec:"seed"`
		Increment int64 `spec:"increment"`
	}
	if err := r.As(&spec); err != nil {
		return err
	}
	if spec.Seed != 0 {
		i.Seed = spec.Seed
	}
	if spec.Increment != 0 {
		i.Increment = spec.Increment
	}
	return nil
}

// toSpec returns the resource spec for representing the identity attributes.
func (i *Identity) toSpec() *schemahcl.Resource {
	return &schemahcl.Resource{
		Type: "identity",
		Attrs: []*schemahcl.Attr{
			schemahcl.Int64Attr("seed", i.Seed),
			schemahcl.Int64Attr("increment", i.Increment),
		},
	}
}
