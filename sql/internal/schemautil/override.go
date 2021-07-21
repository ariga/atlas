package schemautil

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/sql/schema/schemaspec"
	"github.com/go-openapi/inflect"
)

// OverrideFor overrides the Element fields and attributes for the dialect.
// It is used to modify the Element to a specific form relevant for a particular
// dialect.
func OverrideFor(dialect string, element schemaspec.Overrider) error {
	override := element.Override(dialect)
	if override == nil {
		return nil
	}
	val := reflect.ValueOf(element).Elem()
	for _, attr := range override.Attrs {
		n := inflect.Camelize(attr.K) // TODO: infer the field name more intelligently
		field := val.FieldByName(n)
		if !field.IsValid() {
			element.SetAttr(attr)
			continue
		}
		if !field.CanSet() {
			return fmt.Errorf("schema: cannot set field %s on type %T", n, element)
		}
		if err := setField(field, attr); err != nil {
			return err
		}
	}
	return nil
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
