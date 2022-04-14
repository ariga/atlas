// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
)

// InputVar is an input variable to an Atlas DDL document.
type InputVar struct {
	Name    string                   `spec:",name"`
	Type    *schemaspec.Type         `spec:"type"`
	Default *schemaspec.LiteralValue `spec:"default"`
}

// ExtractVarsHCL extracts the variable definitions from an Atlas DDL HCL document.
func ExtractVarsHCL(body []byte) ([]*InputVar, error) {
	var d struct {
		Vars []*InputVar `spec:"variable"`
	}
	if err := hclState.UnmarshalSpec(body, &d); err != nil {
		return nil, err
	}
	return d.Vars, nil
}

var (
	hclState = schemahcl.New(schemahcl.WithTypes(
		schemahcl.NewRegistry(
			schemahcl.WithSpecs(
				schemahcl.TypeSpec("int"),
				schemahcl.TypeSpec("bool"),
				schemahcl.TypeSpec("float"),
			),
		).Specs(),
	))
)

func init() {
	schemaspec.Register("variable", &InputVar{})
}
