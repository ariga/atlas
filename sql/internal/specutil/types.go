package specutil

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"github.com/go-openapi/inflect"
)

// PrintType returns the string representation of a column type which can be parsed
// by the driver into a schema.Type.
func (r *TypeRegistry) PrintType(typ *schemaspec.Type) (string, error) {
	spec, ok := r.Find(typ.T)
	if !ok {
		return "", fmt.Errorf("specutil: type %q not found in registry", typ.T)
	}
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

// Convert converts the schema.Type to a *schemaspec.Type.
func (r *TypeRegistry) Convert(typ schema.Type) (*schemaspec.Type, error) {
	s := &schemaspec.Type{}
	rv := reflect.ValueOf(typ)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, errors.New("specutil: invalid schema.Type on Convert")
	}
	tf := rv.FieldByName("T")
	if !tf.IsValid() {
		return nil, fmt.Errorf("specutil: cannot convert schema.Type without T field for %T", typ)
	}
	if tf.Kind() != reflect.String {
		return nil, fmt.Errorf("specutil: cannot convert non-string T field for %T", typ)
	}
	s.T = tf.String()
	typeSpec, ok := r.Find(s.T)
	if !ok {
		return nil, fmt.Errorf("specutil: type %q not found in registry", s.T)
	}
	for _, attr := range typeSpec.Attributes {
		n := inflect.Camelize(attr.Name)
		field := rv.FieldByName(n)
		if !field.IsValid() {
			return nil, fmt.Errorf("invalid field name %q for attr %q on type spec %q", n, attr.Name, typeSpec.T)
		}
		if field.Kind() != attr.Kind {
			return nil, errors.New("incompatible kinds on typespec attr and typefield")
		}
		switch attr.Kind {
		case reflect.Int:
			i := strconv.Itoa(int(field.Int()))
			s.Attributes = append(s.Attributes, LitAttr(attr.Name, i))
		case reflect.Bool:
			b := strconv.FormatBool(field.Bool())
			s.Attributes = append(s.Attributes, LitAttr(attr.Name, b))
		case reflect.Slice:
			lits := make([]string, 0, field.Len())
			for i := 0; i < field.Len(); i++ {
				fi := field.Index(i)
				if fi.Kind() != reflect.String {
					return nil, errors.New("specutil: only string slices currently supported")
				}
				lits = append(lits, strconv.Quote(fi.String()))
			}
			s.Attributes = append(s.Attributes, ListAttr(attr.Name, lits...))
		default:
			return nil, fmt.Errorf("specutil: unsupported attr kind %s for attribute %q of %q", attr.Kind, attr.Name, typeSpec.Name)
		}
	}
	return s, nil
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
