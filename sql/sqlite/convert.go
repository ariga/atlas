package sqlite

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// FormatType converts types to one format. A lowered format.
// This is due to SQLite flexibility to allow any data types
// and use a set of rules to define the type affinity.
// See: https://www.sqlite.org/datatype3.html
func (d *Driver) FormatType(t schema.Type) (string, error) {
	var f string
	switch t := t.(type) {
	case *schema.BoolType:
		f = strings.ToLower(t.T)
	case *schema.BinaryType:
		f = strings.ToLower(t.T)
	case *schema.EnumType:
		f = t.T
	case *schema.IntegerType:
		f = strings.ToLower(t.T)
	case *schema.StringType:
		f = strings.ToLower(t.T)
	case *schema.TimeType:
		strings.ToLower(t.T)
	case *schema.FloatType:
		f = strings.ToLower(t.T)
	case *schema.DecimalType:
		f = strings.ToLower(t.T)
	case *schema.JSONType:
		f = strings.ToLower(t.T)
	case *schema.SpatialType:
		f = strings.ToLower(t.T)
	case *UUIDType:
		f = strings.ToLower(t.T)
	default:
		return "", fmt.Errorf("sqlite: invalid schema type: %T", t)
	}
	return f, nil
}

// mustFormat calls to FormatType and panics in case of error.
func (d *Driver) mustFormat(t schema.Type) string {
	s, err := d.FormatType(t)
	if err != nil {
		panic(err)
	}
	return s
}
