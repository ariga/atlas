package sqlspec_test

import (
	"reflect"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/sqlspec"
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
			typ:      &schemaspec.Type{T: "int", Attributes: []*schemaspec.Attr{specutil.LitAttr("unsigned", "true")}},
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
			typ:      &schemaspec.Type{T: "varchar", Attributes: []*schemaspec.Attr{specutil.LitAttr("size", "255")}},
			expected: "varchar(255)",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			s, err := sqlspec.PrintType(tt.typ, tt.spec)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, s)
		})
	}
}
