package sqlspec

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
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
		switch v := arg.V.(type) {
		case *schemaspec.LiteralValue:
			args = append(args, v.V)
		case *schemaspec.ListValue:
			for _, li := range v.V {
				lit, ok := li.(*schemaspec.LiteralValue)
				if !ok {
					return "", errors.New("expecting literal value")
				}
				uq, err := strconv.Unquote(lit.V)
				if err != nil {
					return "", fmt.Errorf("expecting list items to be quoted strings: %w", err)
				}

				args = append(args, `'`+uq+`'`)
			}
		default:
			return "", fmt.Errorf("unsupported type %T for PrintType", v)
		}

	}
	if len(args) > 0 {
		mid = "(" + strings.Join(args, ",") + ")"
	}
	return typ.T + mid + suffix, nil
}

// TypeSpec returns a TypeSpec with the provided name.
func TypeSpec(name string, attrs ...*schemaspec.TypeAttr) *schemaspec.TypeSpec {
	return &schemaspec.TypeSpec{
		Name:       name,
		T:          name,
		Attributes: attrs,
	}
}

// SizeTypeAttr returns a TypeAttr for a size attribute.
func SizeTypeAttr(required bool) *schemaspec.TypeAttr {
	return &schemaspec.TypeAttr{
		Name:     "size",
		Kind:     reflect.Int,
		Required: required,
	}
}

// UnsignedTypeAttr returns a TypeAttr for an `unsigned` attribute relevant for integer types.
func UnsignedTypeAttr() *schemaspec.TypeAttr {
	return &schemaspec.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
