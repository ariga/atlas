package spectest

import (
	"reflect"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"
)

func RegistrySanityTest(t *testing.T,
	registry *specutil.TypeRegistry,
	formatter func(schema.Type) (string, error),
	parser func(string) (schema.Type, error),
) {
	for _, ts := range registry.Specs() {
		t.Run(ts.Name, func(t *testing.T) {
			spec := dummyType(t, ts)
			styp, err := registry.Type(spec, nil, parser)
			require.NoError(t, err)
			//_, err = formatter(styp)
			require.NoErrorf(t, err, "failed formatting: %styp", err)
			convert, err := registry.Convert(styp)
			require.NoError(t, err)
			after, err := registry.Type(convert, nil, parser)
			require.NoError(t, err)
			require.EqualValues(t, styp, after)
		})
	}
}

func dummyType(t *testing.T, ts *schemaspec.TypeSpec) *schemaspec.Type {
	spec := &schemaspec.Type{T: ts.T}
	for _, attr := range ts.Attributes {
		var a *schemaspec.Attr
		switch attr.Kind {
		case reflect.Int, reflect.Int64:
			a = specutil.LitAttr(attr.Name, "2")
		case reflect.String:
			a = specutil.LitAttr(attr.Name, `"a"`)
		case reflect.Slice:
			a = specutil.ListAttr(attr.Name, `"a"`, `"b"`)
		case reflect.Bool:
			a = specutil.LitAttr(attr.Name, "false")
		default:
			t.Fatalf("unsupported kind: %s", attr.Kind)
		}
		spec.Attrs = append(spec.Attrs, a)
	}
	return spec
}
