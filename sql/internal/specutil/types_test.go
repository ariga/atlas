package specutil

import (
	"reflect"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestTypePrint(t *testing.T) {
	intSpec := &schemaspec.TypeSpec{
		Name: "int",
		T:    "int",
		Attributes: []*schemaspec.TypeAttr{
			{Name: "unsigned", Kind: reflect.Bool, Required: false},
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
