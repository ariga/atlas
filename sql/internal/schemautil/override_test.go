package schemautil_test

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestOverride(t *testing.T) {
	spec := schemautil.ColSpec("name", "string")
	spec.Overrides = []*schemaspec.Override{
		{
			Dialect: mysql.Name,
			Resource: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					// A string field
					schemautil.StrLitAttr("type", "varchar(123)"),

					// A boolean field
					schemautil.LitAttr("null", "true"),

					// A Literal
					schemautil.StrLitAttr("default", "howdy"),

					// A custom attribute
					schemautil.LitAttr("custom", "1234"),
				},
			},
		},
	}
	err := schemautil.OverrideFor(mysql.Name, spec)
	require.NoError(t, err)
	require.EqualValues(t, "varchar(123)", spec.Type)
	require.EqualValues(t, "howdy", spec.Default.V)
	require.True(t, spec.Null)
	custom, ok := spec.Attr("custom")
	require.True(t, ok)
	i, err := custom.Int()
	require.NoError(t, err)
	require.EqualValues(t, 1234, i)
}
