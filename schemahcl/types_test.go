// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"reflect"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestTypePrint(t *testing.T) {
	intSpec := &TypeSpec{
		Name: "int",
		T:    "int",
		Attributes: []*TypeAttr{
			unsignedTypeAttr(),
		},
	}
	for _, tt := range []struct {
		spec     *TypeSpec
		typ      *Type
		expected string
	}{
		{
			spec:     intSpec,
			typ:      &Type{T: "int"},
			expected: "int",
		},
		{
			spec:     intSpec,
			typ:      &Type{T: "int", Attrs: []*Attr{BoolAttr("unsigned", true)}},
			expected: "int unsigned",
		},
		{
			spec: &TypeSpec{
				Name:       "float",
				T:          "float",
				Attributes: []*TypeAttr{unsignedTypeAttr()},
			},
			typ:      &Type{T: "float", Attrs: []*Attr{BoolAttr("unsigned", true)}},
			expected: "float unsigned",
		},
		{
			spec: &TypeSpec{
				T:    "varchar",
				Name: "varchar",
				Attributes: []*TypeAttr{
					{Name: "size", Kind: reflect.Int, Required: true},
				},
			},
			typ:      &Type{T: "varchar", Attrs: []*Attr{IntAttr("size", 255)}},
			expected: "varchar(255)",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			r := &TypeRegistry{}
			err := r.Register(tt.spec)
			require.NoError(t, err)
			s, err := r.PrintType(tt.typ)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, s)
		})
	}
}

func TestRegistry(t *testing.T) {
	r := &TypeRegistry{}
	text := &TypeSpec{Name: "text", T: "text"}
	err := r.Register(text)
	require.NoError(t, err)
	err = r.Register(text)
	require.EqualError(t, err, `specutil: type with T of "text" already registered`)
	spec, ok := r.findName("text")
	require.True(t, ok)
	require.EqualValues(t, spec, text)
}

func TestValidSpec(t *testing.T) {
	registry := &TypeRegistry{}
	err := registry.Register(&TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "a" is of kind slice but not last`)
	err = registry.Register(&TypeSpec{
		Name: "Z",
		T:    "Z",
		Attributes: []*TypeAttr{
			{Name: "b", Required: true},
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&TypeSpec{
		Name: "Z2",
		T:    "Z2",
		Attributes: []*TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "b" required after optional attr`)
	err = registry.Register(&TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*TypeAttr{
			{Name: "a", Required: true},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&TypeSpec{
		Name: "Y",
		T:    "Y",
		Attributes: []*TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
}

func TestRegistryConvert(t *testing.T) {
	r := &TypeRegistry{}
	err := r.Register(
		NewTypeSpec("varchar", WithAttributes(SizeTypeAttr(true))),
		NewTypeSpec("int", WithAttributes(unsignedTypeAttr())),
		NewTypeSpec(
			"decimal",
			WithAttributes(
				&TypeAttr{
					Name:     "precision",
					Kind:     reflect.Int,
					Required: false,
				},
				&TypeAttr{
					Name:     "scale",
					Kind:     reflect.Int,
					Required: false,
				},
			),
		),
		NewTypeSpec("enum", WithAttributes(&TypeAttr{
			Name:     "values",
			Kind:     reflect.Slice,
			Required: true,
		})),
	)
	require.NoError(t, err)
	for _, tt := range []struct {
		typ         schema.Type
		expected    *Type
		expectedErr string
	}{
		{
			typ:      &schema.StringType{T: "varchar", Size: 255},
			expected: &Type{T: "varchar", Attrs: []*Attr{IntAttr("size", 255)}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &Type{T: "int", Attrs: []*Attr{BoolAttr("unsigned", true)}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &Type{T: "int", Attrs: []*Attr{BoolAttr("unsigned", true)}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}, // decimal(10,2)
			expected: &Type{T: "decimal", Attrs: []*Attr{
				IntAttr("precision", 10),
				IntAttr("scale", 2),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10}, // decimal(10)
			expected: &Type{T: "decimal", Attrs: []*Attr{
				IntAttr("precision", 10),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Scale: 2}, // decimal(0,2)
			expected: &Type{T: "decimal", Attrs: []*Attr{
				IntAttr("precision", 0),
				IntAttr("scale", 2),
			}},
		},
		{
			typ:      &schema.DecimalType{T: "decimal"}, // decimal
			expected: &Type{T: "decimal"},
		},
		{
			typ: &schema.EnumType{T: "enum", Values: []string{"on", "off"}},
			expected: &Type{T: "enum", Attrs: []*Attr{
				StringsAttr("values", "on", "off"),
			}},
		},
		{
			typ:         nil,
			expected:    &Type{},
			expectedErr: "specutil: invalid schema.Type on Convert",
		},
	} {
		t.Run(tt.expected.T, func(t *testing.T) {
			convert, err := r.Convert(tt.typ)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, convert)
		})
	}
}

func unsignedTypeAttr() *TypeAttr {
	return &TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
