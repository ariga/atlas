package schemautil

import (
	"strconv"

	"ariga.io/atlas/schema/schemaspec"
)

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
	lst := &schemaspec.ListValue{}
	for _, v := range values {
		lst.V = append(lst.V, &schemaspec.LiteralValue{V: strconv.Quote(v)})
	}
	return &schemaspec.Attr{
		K: k,
		V: lst,
	}
}
