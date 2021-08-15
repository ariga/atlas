// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemaspec

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Extension is the interface that should be implemented by extensions to
// the core Spec resources.
//
// To specify the mapping from the extension struct fields to the schemaspec.Resource
// use the `spec` key on the field's tag. To specify that a field should be mapped to
// the corresponding Resource's `Name` specify ",name" to the tag value. For example:
// field annotate
//   type Example struct {
//      Name  string `spec:,name"
//      Value int `spec:"value"`
//   }
//
type Extension interface {
	// Type returns the type name for the Extension, to be set in the
	// Resource.Type field when using Resource.Scan.
	Type() string
}

// As reads the attributes and children resources of the resource into the target Extension.
func (r *Resource) As(target Extension) error {
	var seenName bool
	v := reflect.ValueOf(target).Elem()
	for _, ft := range specFields(target) {
		field := v.FieldByName(ft.field)
		if ft.isName {
			if seenName {
				return errors.New("schemaspec: extension must have only one isName field")
			}
			seenName = true
			if field.Kind() != reflect.String {
				return errors.New("schemaspec: extension isName field must be of type string")
			}
			field.SetString(r.Name)
			continue
		}
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
	v := reflect.ValueOf(ext).Elem()
	for _, ft := range specFields(ext) {
		field := v.FieldByName(ft.field)
		if ft.isName {
			if field.Kind() != reflect.String {
				return errors.New("schemaspec: extension name field must be string")
			}
			r.Name = field.String()
			continue
		}
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
		parts := strings.Split(lookup, ",")
		fields = append(fields, fieldTag{
			field:  f.Name,
			tag:    lookup,
			isName: len(parts) > 1 && parts[1] == "name",
		})
	}
	return fields
}

type fieldTag struct {
	field, tag string
	isName     bool
}
