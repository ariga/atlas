package specutil

import (
	"errors"
	"fmt"
	"log"
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
	for _, arg := range typ.Attrs {
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
					return "", fmt.Errorf("expecting literal value. got: %T", li)
				}
				uq, err := strconv.Unquote(lit.V)
				if err != nil {
					return "", fmt.Errorf("expecting list items to be quoted strings: %w", err)
				}
				args = append(args, "'"+uq+"'")
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
		r.r = append(r.r, s)
	}
	return nil
}

// NewRegistry creates a new *TypeRegistry, registers the provided types and panics
// if an error occurs.
func NewRegistry(specs ...*schemaspec.TypeSpec) *TypeRegistry {
	r := &TypeRegistry{}
	if err := r.Register(specs...); err != nil {
		log.Fatalf("failed registering types: %s", err)
	}
	return r
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
	// Iterate the attributes in reverse order, so we can skip zero value and optional attrs.
	for i := len(typeSpec.Attributes) - 1; i >= 0; i-- {
		attr := typeSpec.Attributes[i]
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
			v := int(field.Int())
			if v == 0 && len(s.Attrs) == 0 {
				break
			}
			i := strconv.Itoa(v)
			s.Attrs = append([]*schemaspec.Attr{LitAttr(attr.Name, i)}, s.Attrs...)
		case reflect.Bool:
			v := field.Bool()
			if !v && len(s.Attrs) == 0 {
				break
			}
			b := strconv.FormatBool(v)
			s.Attrs = append([]*schemaspec.Attr{LitAttr(attr.Name, b)}, s.Attrs...)
		case reflect.Slice:
			lits := make([]string, 0, field.Len())
			for i := 0; i < field.Len(); i++ {
				fi := field.Index(i)
				if fi.Kind() != reflect.String {
					return nil, errors.New("specutil: only string slices currently supported")
				}
				lits = append(lits, strconv.Quote(fi.String()))
			}
			s.Attrs = append([]*schemaspec.Attr{ListAttr(attr.Name, lits...)}, s.Attrs...)
		default:
			return nil, fmt.Errorf("specutil: unsupported attr kind %s for attribute %q of %q", attr.Kind, attr.Name, typeSpec.Name)
		}
	}
	return s, nil
}

// Specs returns the TypeSpecs in the registry.
func (r *TypeRegistry) Specs() []*schemaspec.TypeSpec {
	return r.r
}

// Type converts a *schemaspec.Type into a schema.Type.
func (r *TypeRegistry) Type(typ *schemaspec.Type, extra []*schemaspec.Attr, parser func(string) (schema.Type, error)) (schema.Type, error) {
	typeSpec, ok := r.Find(typ.T)
	if !ok {
		return nil, fmt.Errorf("specutil: typespec not found for %s", typ.T)
	}
	nfa := typeNonFuncArgs(typeSpec)
	picked := pickTypeAttrs(extra, nfa)
	typ.Attrs = appendIfNotExist(typ.Attrs, picked)
	printType, err := r.PrintType(typ)
	if err != nil {
		return nil, err
	}
	return parser(printType)
}

// TypeSpec returns a TypeSpec with the provided name.
func TypeSpec(name string, attrs ...*schemaspec.TypeAttr) *schemaspec.TypeSpec {
	return AliasTypeSpec(name, name, attrs...)
}

// AliasTypeSpec returns a TypeSpec with the provided name.
func AliasTypeSpec(name, dbType string, attrs ...*schemaspec.TypeAttr) *schemaspec.TypeSpec {
	return &schemaspec.TypeSpec{
		Name:       name,
		T:          dbType,
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

// typeNonFuncArgs returns the type attributes that are NOT configured via arguments to the
// type definition, `int unsigned`.
func typeNonFuncArgs(spec *schemaspec.TypeSpec) []*schemaspec.TypeAttr {
	var args []*schemaspec.TypeAttr
	for _, attr := range spec.Attributes {
		// TODO(rotemtam): this should be defined on the TypeSpec.
		if attr.Name == "unsigned" {
			args = append(args, attr)
		}
	}
	return args
}

// pickTypeAttrs returns the relevant Attrs matching the wanted TypeAttrs.
func pickTypeAttrs(src []*schemaspec.Attr, wanted []*schemaspec.TypeAttr) []*schemaspec.Attr {
	keys := make(map[string]struct{})
	for _, w := range wanted {
		keys[w.Name] = struct{}{}
	}
	var picked []*schemaspec.Attr
	for _, attr := range src {
		if _, ok := keys[attr.K]; ok {
			picked = append(picked, attr)
		}
	}
	return picked
}

func appendIfNotExist(base []*schemaspec.Attr, additional []*schemaspec.Attr) []*schemaspec.Attr {
	exists := make(map[string]struct{})
	for _, attr := range base {
		exists[attr.K] = struct{}{}
	}
	for _, attr := range additional {
		if _, ok := exists[attr.K]; !ok {
			base = append(base, attr)
		}
	}
	return base
}
