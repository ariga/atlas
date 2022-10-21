package spanner

import (
	"ariga.io/atlas/sql/schema"
)

// ParseType returns the schema.Type value represented by the given raw type.
func ParseType(typ string) (schema.Type, error) {
	t, err := columnType(typ)
	if err != nil {
		return nil, err
	}
	return t, nil
}
