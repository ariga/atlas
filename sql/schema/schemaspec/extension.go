package schemaspec

import (
	"fmt"
	"reflect"
	"strconv"
)

// ExtensionSpec is the interface that should be implemented by extensions to
// the core Spec resources.
type ExtensionSpec interface {
	Name() string
	Type() string
}

// Scan reads the attributes and children resources of the resource into the target ExtensionSpec.
func (r *Resource) Scan(target ExtensionSpec) error {
	v := reflect.ValueOf(target).Elem()
	for tag, name := range specFields(target) {
		field := v.FieldByName(name)
		attr, ok := r.Attr(tag)
		if !ok {
			return fmt.Errorf("schemaspec: resource does not have attr %q", tag)
		}
		switch field.Kind() {
		case reflect.String:
			s, err := attr.String()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as string: %w", name, err)
			}
			field.SetString(s)
		case reflect.Int:
			i, err := attr.Int()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as integer: %w", name, err)
			}
			field.SetInt(int64(i))
		case reflect.Bool:
			b, err := attr.Bool()
			if err != nil {
				return fmt.Errorf("schemaspec: value of attr %q cannot be read as bool: %w", name, err)
			}
			field.SetBool(b)
		default:
			return fmt.Errorf("schemaspec: unsupported field kind %q", field.Kind())
		}
	}
	return nil
}

// ExtAsResource writes the ExtensionSpec as a Resource.
func ExtAsResource(ext ExtensionSpec) *Resource {
	out := &Resource{
		Type: ext.Type(),
		Name: ext.Name(),
	}
	v := reflect.ValueOf(ext).Elem()
	for tag, name := range specFields(ext) {
		field := v.FieldByName(name)
		var lit string
		switch field.Kind() {
		case reflect.String:
			lit = strconv.Quote(field.String())
		case reflect.Int:
			lit = fmt.Sprintf("%d", field.Int())
		case reflect.Bool:
			lit = strconv.FormatBool(field.Bool())
		}
		attr := &Attr{
			K: tag,
			V: &LiteralValue{V: lit},
		}
		out.SetAttr(attr)
	}
	return out
}

// specFields uses reflection to find struct fields that are tagged with "spec"
// and returns a map from the tag to the field name.
func specFields(ext ExtensionSpec) map[string]string {
	t := reflect.TypeOf(ext)
	fields := make(map[string]string)
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		lookup, ok := f.Tag.Lookup("spec")
		if !ok {
			continue
		}
		fields[lookup] = f.Name
	}
	return fields
}
