package specutil

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
			typ:      &schemaspec.Type{T: "int", Attributes: []*schemaspec.Attr{LitAttr("unsigned", "true")}},
			expected: "int unsigned",
		},
		{
			spec: &schemaspec.TypeSpec{
				T:    "varchar",
				Name: "varchar",
				Attributes: []*schemaspec.TypeAttr{
					{Name: "size", Kind: reflect.Int, Required: true},
				},
			},
			typ:      &schemaspec.Type{T: "varchar", Attributes: []*schemaspec.Attr{LitAttr("size", "255")}},
			expected: "varchar(255)",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			s, err := PrintType(tt.typ, tt.spec)
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
	spec, ok := r.FindByName("text")
	require.True(t, ok)
	require.EqualValues(t, spec, text)
}

func TestRegistryConvert(t *testing.T) {
	r := &TypeRegistry{}
	err := r.Register(
		TypeSpec("varchar", SizeTypeAttr(true)),
		TypeSpec("int", unsignedTypeAttr()),
		TypeSpec(
			"decimal",
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
		TypeSpec("enum", &schemaspec.TypeAttr{
			Name:     "values",
			Kind:     reflect.Slice,
			Required: true,
		}),
	)
	require.NoError(t, err)
	for _, tt := range []struct {
		typ         schema.Type
		expected    *schemaspec.Type
		expectedErr string
	}{
		{
			typ:      &schema.StringType{T: "varchar", Size: 255},
			expected: &schemaspec.Type{T: "varchar", Attributes: []*schemaspec.Attr{LitAttr("size", "255")}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemaspec.Type{T: "int", Attributes: []*schemaspec.Attr{LitAttr("unsigned", "true")}},
		},
		{
			typ:      &schema.IntegerType{T: "int", Unsigned: true},
			expected: &schemaspec.Type{T: "int", Attributes: []*schemaspec.Attr{LitAttr("unsigned", "true")}},
		},
		{
			typ: &schema.DecimalType{T: "decimal", Precision: 10, Scale: 2},
			expected: &schemaspec.Type{T: "decimal", Attributes: []*schemaspec.Attr{
				LitAttr("precision", "10"),
				LitAttr("scale", "2"),
			}},
		},
		{
			typ: &schema.EnumType{T: "enum", Values: []string{"on", "off"}},
			expected: &schemaspec.Type{T: "enum", Attributes: []*schemaspec.Attr{
				ListAttr("values", `"on"`, `"off"`),
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
