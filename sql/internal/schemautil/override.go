package schemautil

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/sql/schema/schemaspec"
)

// OverrideAttributer joins schemaspec.Overrider and schemaspec.Attributer.
type OverrideAttributer interface {
	schemaspec.Overrider
	schemaspec.Attributer
}

// OverrideFor overrides the Element fields and attributes for the dialect.
// It is used to modify the Element to a specific form relevant for a particular
// dialect.
//
// OverrideFor maps the schemaspec.Override attributes to the element fields
// by looking at `override` struct tags on the target element struct definition.
// If no field with a matching `override` tag is found, the
// Overrider's relevant attribute is created/replaced.
func OverrideFor(dialect string, element OverrideAttributer) error {
	override := element.Override(dialect)
	if override == nil {
		return nil
	}
	val := reflect.ValueOf(element).Elem()
	names := nameToField(element)
	for _, attr := range override.Attrs {
		fld, ok := names[attr.K]
		if !ok {
			element.SetAttr(attr)
			continue
		}
		field := val.FieldByName(fld)
		if !field.IsValid() || !field.CanSet() {
			return fmt.Errorf("schema: cannot set field %s on type %T", fld, element)
		}
		if err := setField(field, attr); err != nil {
			return err
		}
	}
	return nil
}

func nameToField(element OverrideAttributer) map[string]string {
	names := make(map[string]string)
	t := reflect.ValueOf(element).Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		key := f.Tag.Get("override")
		if key != "" {
			names[key] = f.Name
		}
	}
	return names
}

func setField(field reflect.Value, attr *schemaspec.Attr) error {
	switch field.Kind() {
	case reflect.String:
		s, err := attr.String()
		if err != nil {
			return err
		}
		field.SetString(s)
	case reflect.Bool:
		s, err := attr.Bool()
		if err != nil {
			return err
		}
		field.SetBool(s)
	case reflect.Ptr:
		if field.Type() == reflect.ValueOf(&schemaspec.LiteralValue{}).Type() {
			field.Set(reflect.ValueOf(attr.V.(*schemaspec.LiteralValue)))
		} else {
			return fmt.Errorf("schema: unsupported %s field type", field.Type().Name())
		}
	default:
		return fmt.Errorf("schema: unsupported kind %s", field.Kind().String())
	}
	return nil
}
