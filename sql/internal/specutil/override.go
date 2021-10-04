// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/sqlspec"
)

// Override overrides the Element fields and attributes using the provided
// sqlspec.Override. It is used to modify the Element to a specific form
// relevant for a particular dialect.
//
// Override maps the sqlspec.Override attributes to the element fields
// by looking at `override` struct tags on the target element struct definition.
// If no field with a matching `override` tag is found, the
// Overrider's relevant attribute is created/replaced.
func Override(element *sqlspec.Column, override *sqlspec.Override) error {
	val := reflect.ValueOf(element).Elem()
	names := nameToField(element)
	for _, attr := range override.Extra.Attrs {
		fld, ok := names[attr.K]
		if !ok {
			element.Extra.SetAttr(attr)
			continue
		}
		field := val.FieldByName(fld)
		if !field.IsValid() || !field.CanSet() {
			return fmt.Errorf("sqlspec: cannot set field %s on type %T", fld, element)
		}
		if err := setField(field, attr); err != nil {
			return err
		}
	}
	return nil
}

func nameToField(element *sqlspec.Column) map[string]string {
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
		if field.Type() == reflect.TypeOf((*schemaspec.LiteralValue)(nil)) {
			field.Set(reflect.ValueOf(attr.V))
			return nil
		}
		fallthrough
	default:
		return fmt.Errorf("schema: unsupported field type %q", field.Type())
	}
	return nil
}
