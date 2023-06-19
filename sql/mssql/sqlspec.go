// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

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
func tableSpec(t *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(
		t,
		columnSpec,
		specutil.FromPrimaryKey,
		func(i *schema.Index) (*sqlspec.Index, error) {
			return specutil.FromIndex(i)
		},
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
