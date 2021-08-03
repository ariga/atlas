// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemautil

import (
	"strconv"

	"ariga.io/atlas/sql/schema/schemaspec"
)

// ColSpec is a helper method for constructing *schemaspec.Column instances.
func ColSpec(name, coltype string, attrs ...*schemaspec.Attr) *schemaspec.Column {
	return &schemaspec.Column{
		Name: name,
		Type: coltype,
		Resource: schemaspec.Resource{
			Attrs: attrs,
		},
	}
}

// LitAttr is a helper method for constructing *schemaspec.Attr instances that contain literal values.
func LitAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: v},
	}
}

// StrLitAttr is a helper method for constructing *schemaspec.Attr instances that contain literal values
// representing string literals.
func StrLitAttr(k, v string) *schemaspec.Attr {
	return LitAttr(k, strconv.Quote(v))
}

// ListAttr is a helper method for constructing *schemaspec.Attr instances that contain list values.
func ListAttr(k string, values ...string) *schemaspec.Attr {
	for i, v := range values {
		values[i] = strconv.Quote(v)
	}
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.ListValue{V: values},
	}
}
