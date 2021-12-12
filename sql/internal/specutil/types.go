package specutil

import (
	"errors"
	"fmt"
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

// TypeRegistry is a collection of *schemaspec.TypeSpec.
type TypeRegistry struct {
	r []*schemaspec.TypeSpec
}

// Register adds one or more TypeSpec to the registry.
func (r *TypeRegistry) Register(specs ...*schemaspec.TypeSpec) error {
	for _, s := range specs {
		if _, exists := r.Find(s.T); exists {
			return fmt.Errorf("specutil: type with T of %q already registered", s.T)
		}
		if _, exists := r.FindByName(s.Name); exists {
			return fmt.Errorf("specutil: type with name of %q already registered", s.T)
		}
	}
	r.r = append(r.r, specs...)
	return nil
}

// FindByName searches the registry for types that have the provided name.
func (r *TypeRegistry) FindByName(name string) (*schemaspec.TypeSpec, bool) {
	for _, current := range r.r {
		if current.Name == name {
			return current, true
		}
	}
	return nil, false
}

// Find searches the registry for types that have the provided T.
func (r *TypeRegistry) Find(t string) (*schemaspec.TypeSpec, bool) {
	for _, current := range r.r {
		if current.T == t {
			return current, true
		}
	}
	return nil, false
}

