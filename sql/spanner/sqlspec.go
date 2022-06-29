// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"reflect"
	"strconv"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// evalSpec evaluates an Atlas DDL document using an unmarshaler into v by using the input.
func evalSpec(data []byte, v interface{}, input map[string]string) error {
	return specutil.Unmarshal(data, hclState, v, input, convertTable)
}

// MarshalSpec marshals v into an Atlas DDL document using a schemaspec.Marshaler.
func MarshalSpec(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error) {
	return specutil.Marshal(v, marshaler, schemaSpec)
}

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, convertIndex, specutil.Check)
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
		panic(err)
		return nil, err
	}
	if err := specutil.ConvertGenExpr(spec.Remain(), c, storedOrVirtual); err != nil {
		panic(err)
		return nil, err
	}
	return c, nil
}

// convertColumnType converts a sqlspec.Column into a concrete  schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

// schemaSpec converts from a concrete  schema to Atlas specification.
func schemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return specutil.FromSchema(schem, tableSpec)
}

// tableSpec converts from a concrete  sqlspec.Table to a schema.Table.
func tableSpec(tab *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(
		tab,
		columnSpec,
		specutil.FromPrimaryKey,
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
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

// columnSpec converts from a concrete  schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	s, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
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

// TypeRegistry contains the supported TypeSpecs for the spanner driver.
var TypeRegistry = schemahcl.NewRegistry(
	schemahcl.WithFormatter(FormatType),
	schemahcl.WithParser(ParseType),
	schemahcl.WithSpecs(
		schemahcl.TypeSpec(TypeString, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.TypeSpec(TypeInt64, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.TypeSpec(TypeNumeric, schemahcl.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		schemahcl.TypeSpec("decimal", schemahcl.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		schemahcl.TypeSpec(TypeBool),
		schemahcl.TypeSpec(TypeTimestamp),
		schemahcl.TypeSpec(TypeDate),
		schemahcl.TypeSpec(TypeJSON),
	),
)

var (
	hclState = schemahcl.New(
		schemahcl.WithTypes(TypeRegistry.Specs()),
		schemahcl.WithScopedEnums("table.column.as.type", stored, virtual),
		schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
	)
	// UnmarshalHCL unmarshals an Atlas HCL DDL document into v.
	UnmarshalHCL = schemaspec.UnmarshalerFunc(func(data []byte, v interface{}) error {
		return evalSpec(data, v, nil)
	})
	// MarshalHCL marshals v into an Atlas HCL DDL document.
	MarshalHCL = schemaspec.MarshalerFunc(func(v interface{}) ([]byte, error) {
		return MarshalSpec(v, hclState)
	})
	// EvalHCL implements the schemahcl.Evaluator interface.
	EvalHCL = schemahcl.EvalFunc(evalSpec)
)

// storedOrVirtual returns a STORED or VIRTUAL
// generated type option based on the given string.
func storedOrVirtual(s string) string {
	if s = strings.ToUpper(s); s == "" {
		return virtual
	}
	return s
}
