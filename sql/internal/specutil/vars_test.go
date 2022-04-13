// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestVars(t *testing.T) {
	h := `variable "age" {
	type = int
	default = 42
}

variable "happy" {
	type = bool
	default = true
}

variable "price" {
	type = float
	default = 3.14
}
`
	vars, err := ExtractVarsHCL([]byte(h))
	require.NoError(t, err)
	require.Len(t, vars, 3)
	require.EqualValues(t, []*InputVar{
		{
			Name: "age",
			Type: &schemaspec.Type{
				T: "int",
			},
			Default: &schemaspec.LiteralValue{
				V: "42",
			},
		},
		{
			Name: "happy",
			Type: &schemaspec.Type{
				T: "bool",
			},
			Default: &schemaspec.LiteralValue{
				V: "true",
			},
		},
		{
			Name: "price",
			Type: &schemaspec.Type{
				T: "float",
			},
			Default: &schemaspec.LiteralValue{
				V: "3.14",
			},
		},
	}, vars)
}
