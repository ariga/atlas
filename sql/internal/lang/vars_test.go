package lang

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
	require.EqualValues(t, []*Var{
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
