package specutil

import (
	"errors"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
)

// PrintType returns the string representation of a column type which can be parsed
// by the driver into a schema.Type.
func PrintType(typ *schemaspec.Type, spec *schemaspec.TypeSpec) (string, error) {
	if len(spec.Attributes) == 0 {
		return typ.T, nil
	}
	var (
		args        []string
		mid, suffix string
	)
	for _, arg := range typ.Attributes {
		// TODO(rotemtam): make this part of the TypeSpec
		if arg.K == "unsigned" {
			b, err := arg.Bool()
			if err != nil {
				return "", err
			}
			if b {
				suffix += " unsigned"
			}
			continue
		}
		lit, ok := arg.V.(*schemaspec.LiteralValue)
		if !ok {
			return "", errors.New("expecting literal value")
		}
		args = append(args, lit.V)
	}
	if len(args) > 0 {
		mid = "(" + strings.Join(args, ",") + ")"
	}
	return typ.T + mid + suffix, nil
}
