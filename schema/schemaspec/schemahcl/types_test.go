package schemahcl

import (
	"reflect"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestTypePrint(t *testing.T) {
	intSpec := &schemaspec.TypeSpec{
		Name: "int",
		T:    "int",
		Attributes: []*schemaspec.TypeAttr{
			unsignedTypeAttr(),
		},
	}
	for _, tt := range []struct {
		spec     *schemaspec.TypeSpec
		typ      *schemaspec.Type
		expected string
	}{
		{
			spec:     intSpec,
			typ:      &schemaspec.Type{T: "int"},
			expected: "int",
		},
		{
			spec:     intSpec,
			typ:      &schemaspec.Type{T: "int", Attrs: []*schemaspec.Attr{litAttr("unsigned", "true")}},
			expected: "int unsigned",
		},
		{
			spec: &schemaspec.TypeSpec{
				Name:       "float",
				T:          "float",
				Attributes: []*schemaspec.TypeAttr{unsignedTypeAttr()},
			},
			typ:      &schemaspec.Type{T: "float", Attrs: []*schemaspec.Attr{litAttr("unsigned", "true")}},
			expected: "float unsigned",
		},
		{
			spec: &schemaspec.TypeSpec{
				T:    "varchar",
				Name: "varchar",
				Attributes: []*schemaspec.TypeAttr{
					{Name: "size", Kind: reflect.Int, Required: true},
				},
			},
			typ:      &schemaspec.Type{T: "varchar", Attrs: []*schemaspec.Attr{litAttr("size", "255")}},
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
	text := &schemaspec.TypeSpec{Name: "text", T: "text"}
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
	err := registry.Register(&schemaspec.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "a" is of kind slice but not last`)
	err = registry.Register(&schemaspec.TypeSpec{
		Name: "Z",
		T:    "Z",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "b", Required: true},
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemaspec.TypeSpec{
		Name: "Z2",
		T:    "Z2",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "a", Required: false, Kind: reflect.Slice},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemaspec.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: true},
		},
	})
	require.EqualError(t, err, `specutil: invalid typespec "X": attr "b" required after optional attr`)
	err = registry.Register(&schemaspec.TypeSpec{
		Name: "X",
		T:    "X",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "a", Required: true},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
	err = registry.Register(&schemaspec.TypeSpec{
		Name: "Y",
		T:    "Y",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "a", Required: false},
			{Name: "b", Required: false},
		},
	})
	require.NoError(t, err)
}

func TestRegistryConvert(t *testing.T) {
	r := &TypeRegistry{}
	err := r.Register(
		Spec("varchar", WithAttributes(SizeTypeAttr(true))),
		Spec("int", WithAttributes(unsignedTypeAttr())),
		Spec(
			"decimal",
			WithAttributes(
				&schemaspec.TypeAttr{
					Name:     "precision",
					Kind:     reflect.Int,
					Required: false,
				},
				&schemaspec.TypeAttr{
					Name:     "scale",
					Kind:     reflect.Int,
					Required: false,
				},
			),
		),
		Spec("enum", WithAttributes(&schemaspec.TypeAttr{
			Name:     "values",
			Kind:     reflect.Slice,
			Required: true,
		})),
	)
	require.NoError(t, err)
	for _, tt := range []struct {
		typ         schema.Type
		expected    *schemaspec.Type
		expectedErr string
	}{
		{
			typ:      &schema.StringType{T: "varchar", Size: 255},
			expected: &schemaspec.Type{T: "varchar", Attrs: []*schemaspec.Attr{litAttr("size", "255")}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemaspec.Type{T: "int", Attrs: []*schemaspec.Attr{litAttr("unsigned", "true")}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemaspec.Type{T: "int", Attrs: []*schemaspec.Attr{litAttr("unsigned", "true")}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2}, // decimal(10,2)
			expected: &schemaspec.Type{T: "decimal", Attrs: []*schemaspec.Attr{
				litAttr("precision", "10"),
				litAttr("scale", "2"),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10}, // decimal(10)
			expected: &schemaspec.Type{T: "decimal", Attrs: []*schemaspec.Attr{
				litAttr("precision", "10"),
			}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Scale: 2}, // decimal(0,2)
			expected: &schemaspec.Type{T: "decimal", Attrs: []*schemaspec.Attr{
				litAttr("precision", "0"),
				litAttr("scale", "2"),
			}},
		},
		{
			typ:      &schema.DecimalType{T: "decimal"}, // decimal
			expected: &schemaspec.Type{T: "decimal"},
		},
		{
			typ: &schema.EnumType{T: "enum", Values: []string{"on", "off"}},
			expected: &schemaspec.Type{T: "enum", Attrs: []*schemaspec.Attr{
				listAttr("values", `"on"`, `"off"`),
			}},
		},
		{
			typ:         nil,
			expected:    &schemaspec.Type{},
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

func unsignedTypeAttr() *schemaspec.TypeAttr {
	return &schemaspec.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
