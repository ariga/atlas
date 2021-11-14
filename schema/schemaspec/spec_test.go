package schemaspec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoolVal(t *testing.T) {
	b, err := BoolVal(&LiteralValue{V: "true"})
	require.NoError(t, err)
	require.True(t, b)
}

func TestBools(t *testing.T) {
	a := Attr{
		K: "b",
		V: &ListValue{
			V: []Value{
				&LiteralValue{V: "true"},
				&LiteralValue{V: "false"},
				&LiteralValue{V: "true"},
			},
		},
	}
	bls, err := a.Bools()
	require.NoError(t, err)
	require.EqualValues(t, []bool{true, false, true}, bls)
}
