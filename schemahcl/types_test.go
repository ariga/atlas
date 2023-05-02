// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl_test

import (
	"reflect"
	"testing"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestTypePrint(t *testing.T) {
	intSpec := &schemahcl.TypeSpec{
		Name: "int",
		T:    "int",
		Attributes: []*schemahcl.TypeAttr{
			unsignedTypeAttr(),
		},
	}
	for _, tt := range []struct {
		spec     *schemahcl.TypeSpec
		typ      *schemahcl.Type
		expected string
	}{
		{
			spec:     intSpec,
			typ:      &schemahcl.Type{T: "int"},
			expected: "int",
		},
		{
			spec:     intSpec,
			typ:      &schemahcl.Type{T: "int", Attrs: []*schemahcl.Attr{schemahcl.BoolAttr("unsigned", true)}},
			expected: "int unsigned",
		},
		{
			spec: &schemahcl.TypeSpec{
				Name:       "float",
				T:          "float",
				Attributes: []*schemahcl.TypeAttr{unsignedTypeAttr()},
			},
			typ:      &schemahcl.Type{T: "float", Attrs: []*schemahcl.Attr{schemahcl.BoolAttr("unsigned", true)}},
			expected: "float unsigned",
		},
		{
			spec: &schemahcl.TypeSpec{
				T:    "varchar",
				Name: "varchar",
				Attributes: []*schemahcl.TypeAttr{
					{Name: "size", Kind: reflect.Int, Required: true},
				},
			},
			typ:      &schemahcl.Type{T: "varchar", Attrs: []*schemahcl.Attr{schemahcl.IntAttr("size", 255)}},
			expected: "varchar(255)",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			r := &schemahcl.TypeRegistry{}
			err := r.Register(tt.spec)
			require.NoError(t, err)
			s, err := r.PrintType(tt.typ)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, s)
		})
	}
}

func TestRegistry(t *testing.T) {
	r := &schemahcl.TypeRegistry{}
	text := &schemahcl.TypeSpec{Name: "text", T: "text"}
	err := r.Register(text)
	require.NoError(t, err)
	err = r.Register(text)
	require.EqualError(t, err, `specutil: type with T of "text" already registered`)
	spec, ok := r.ByName("text")
	require.True(t, ok)
	require.EqualValues(t, spec, text)
}

func TestValidSpec(t *testing.T) {
	registry := &schemahcl.TypeRegistry{}
	err := registry.Register(&schemahcl.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "a" is of kind slice but not last`)
	err = registry.Register(&schemahcl.TypeSpec{
		Name: "Z",
		T:    "Z",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "b", Required: true},
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemahcl.TypeSpec{
		Name: "Z2",
		T:    "Z2",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemahcl.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "b" required after optional attr`)
	err = registry.Register(&schemahcl.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "a", Required: true},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemahcl.TypeSpec{
		Name: "Y",
		T:    "Y",
		Attributes: []*schemahcl.TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
}

func TestRegistryConvert(t *testing.T) {
	r := &schemahcl.TypeRegistry{}
	err := r.Register(
		schemahcl.NewTypeSpec("varchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(true))),
		schemahcl.NewTypeSpec("int", schemahcl.WithAttributes(unsignedTypeAttr())),
		schemahcl.NewTypeSpec(
			"decimal",
			schemahcl.WithAttributes(
				&schemahcl.TypeAttr{
					Name:     "precision",
					Kind:     reflect.Int,
					Required: false,
				},
				&schemahcl.TypeAttr{
					Name:     "scale",
					Kind:     reflect.Int,
					Required: false,
				},
			),
		),
		schemahcl.NewTypeSpec("enum", schemahcl.WithAttributes(&schemahcl.TypeAttr{
			Name:     "values",
			Kind:     reflect.Slice,
			Required: true,
		})),
	)
	require.NoError(t, err)
	for _, tt := range []struct {
		typ         schema.Type
		expected    *schemahcl.Type
		expectedErr string
	}{
		{
			typ:      &schema.StringType{T: "varchar", Size: 255},
			expected: &schemahcl.Type{T: "varchar", Attrs: []*schemahcl.Attr{schemahcl.IntAttr("size", 255)}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemahcl.Type{T: "int", Attrs: []*schemahcl.Attr{schemahcl.BoolAttr("unsigned", true)}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemahcl.Type{T: "int", Attrs: []*schemahcl.Attr{schemahcl.BoolAttr("unsigned", true)}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}, // decimal(10,2)
			expected: &schemahcl.Type{T: "decimal", Attrs: []*schemahcl.Attr{
				schemahcl.IntAttr("precision", 10),
				schemahcl.IntAttr("scale", 2),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10}, // decimal(10)
			expected: &schemahcl.Type{T: "decimal", Attrs: []*schemahcl.Attr{
				schemahcl.IntAttr("precision", 10),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Scale: 2}, // decimal(0,2)
			expected: &schemahcl.Type{T: "decimal", Attrs: []*schemahcl.Attr{
				schemahcl.IntAttr("precision", 0),
				schemahcl.IntAttr("scale", 2),
			}},
		},
		{
			typ:      &schema.DecimalType{T: "decimal"}, // decimal
			expected: &schemahcl.Type{T: "decimal"},
		},
		{
			typ: &schema.EnumType{T: "enum", Values: []string{"on", "off"}},
			expected: &schemahcl.Type{T: "enum", Attrs: []*schemahcl.Attr{
				schemahcl.StringsAttr("values", "on", "off"),
			}},
		},
		{
			typ:         nil,
			expected:    &schemahcl.Type{},
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

func unsignedTypeAttr() *schemahcl.TypeAttr {
	return &schemahcl.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
