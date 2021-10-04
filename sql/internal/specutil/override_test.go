// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil_test

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/sqlspec"
	"github.com/stretchr/testify/require"
)

func TestOverride(t *testing.T) {
	spec := specutil.ColSpec("name", "string")
	override := &sqlspec.Override{
		Dialect: "mysql",
		DefaultExtension: schemaspec.DefaultExtension{
			Extra: schemaspec.Resource{
				Attrs: []*schemaspec.Attr{
					// A string field
					specutil.StrLitAttr("type", "varchar(123)"),

					// A boolean field
					specutil.LitAttr("null", "true"),

					// A Literal
					specutil.StrLitAttr("default", "howdy"),

					// A custom attribute
					specutil.LitAttr("custom", "1234"),
				},
			},
		},
	}

	err := specutil.Override(spec, override)
	require.NoError(t, err)
	require.EqualValues(t, "varchar(123)", spec.TypeName)
	require.EqualValues(t, `"howdy"`, spec.Default.V)
	require.True(t, spec.Null)
	custom, ok := spec.DefaultExtension.Extra.Attr("custom")
	require.True(t, ok)
	i, err := custom.Int()
	require.NoError(t, err)
	require.EqualValues(t, 1234, i)
}
