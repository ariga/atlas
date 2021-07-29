package schemaspec

import (
	"fmt"
	"reflect"
	"strconv"
)

// Extension is the interface that should be implemented by extensions to
// the core Spec resources.
type Extension interface {
	Name() string
	Type() string
}

// As reads the attributes and children resources of the resource into the target Extension.
func (r *Resource) As(target Extension) error {
	v := reflect.ValueOf(target).Elem()
	for _, ft := range specFields(target) {
		field := v.FieldByName(ft.field)
		attr, ok := r.Attr(ft.tag)
		if !ok {
			return fmt.Errorf("schemaspec: resource does not have attr %q", ft.tag)
		}
		switch field.Kind() {
		case reflect.String:
			s, err := attr.String()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as string: %w", ft.tag, err)
			}
			field.SetString(s)
		case reflect.Int:
			i, err := attr.Int()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as integer: %w", ft.tag, err)
			}
			field.SetInt(int64(i))
		case reflect.Bool:
			b, err := attr.Bool()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as bool: %w", ft.tag, err)
			}
			field.SetBool(b)
		default:
			return fmt.Errorf("schemaspec: unsupported field kind %q", field.Kind())
		}
	}
	return nil
}

// Scan reads the Extension into the Resource. Scan will override the Resource
// name or type if they are set for the extension.
func (r *Resource) Scan(ext Extension) error {
	if ext.Type() != "" {
		r.Type = ext.Type()
	}
	if ext.Name() != "" {
		r.Name = ext.Name()
	}
	v := reflect.ValueOf(ext).Elem()
	for _, ft := range specFields(ext) {
		field := v.FieldByName(ft.field)
		var lit string
		switch field.Kind() {
		case reflect.String:
			lit = strconv.Quote(field.String())
		case reflect.Int:
			lit = fmt.Sprintf("%d", field.Int())
		case reflect.Bool:
			lit = strconv.FormatBool(field.Bool())
		default:
			return fmt.Errorf("schemaspec: unsupported field kind %q", field.Kind())
		}
		attr := &Attr{
			K: ft.tag,
			V: &LiteralValue{V: lit},
		}
		r.SetAttr(attr)
	}
	return nil
}

// specFields uses reflection to find struct fields that are tagged with "spec"
// and returns a list of mappings from the tag to the field name.
func specFields(ext Extension) []fieldTag {
	t := reflect.TypeOf(ext)
	var fields []fieldTag
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		lookup, ok := f.Tag.Lookup("spec")
		if !ok {
			continue
		}
		fields = append(fields, fieldTag{field: f.Name, tag: lookup})
	}
	return fields
}

type fieldTag struct {
	field, tag string
}
