package specutil

import (
	"fmt"
	"strconv"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// StrAttr is a helper method for constructing *schemaspec.Attr of type string.
func StrAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: strconv.Quote(v)},
	}
}

// BoolAttr is a helper method for constructing *schemaspec.Attr of type bool.
func BoolAttr(k string, v bool) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: strconv.FormatBool(v)},
	}
}

// IntAttr is a helper method for constructing *schemaspec.Attr with the numeric value of v.
func IntAttr(k string, v int) *schemaspec.Attr {
	return Int64Attr(k, int64(v))
}

// Int64Attr is a helper method for constructing *schemaspec.Attr with the numeric value of v.
func Int64Attr(k string, v int64) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: strconv.FormatInt(v, 10)},
	}
}

// LitAttr is a helper method for constructing *schemaspec.Attr instances that contain literal values.
func LitAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: v},
	}
}

// RawAttr is a helper method for constructing *schemaspec.Attr instances that contain sql expressions.
func RawAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.RawExpr{X: v},
	}
}

// VarAttr is a helper method for constructing *schemaspec.Attr instances that contain a variable reference.
func VarAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.Ref{V: v},
	}
}

// ListAttr is a helper method for constructing *schemaspec.Attr instances that contain list values.
func ListAttr(k string, litValues ...string) *schemaspec.Attr {
	lv := &schemaspec.ListValue{}
	for _, v := range litValues {
		lv.V = append(lv.V, &schemaspec.LiteralValue{V: v})
	}
	return &schemaspec.Attr{
		K: k,
		V: lv,
	}
}

type doc struct {
	Tables  []*sqlspec.Table  `spec:"table"`
	Schemas []*sqlspec.Schema `spec:"schema"`
}

// Marshal marshals v into an Atlas DDL document using a schemaspec.Marshaler. Marshal uses the given
// schemaSpec function to convert a *schema.Schema into *sqlspec.Schema and []*sqlspec.Table.
func Marshal(v interface{}, marshaler schemaspec.Marshaler, schemaSpec func(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error)) ([]byte, error) {
	d := &doc{}
	switch s := v.(type) {
	case *schema.Schema:
		spec, tables, err := schemaSpec(s)
		if err != nil {
			return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
		}
		d.Tables = tables
		d.Schemas = []*sqlspec.Schema{spec}
	case *schema.Realm:
		for _, s := range s.Schemas {
			spec, tables, err := schemaSpec(s)
			if err != nil {
				return nil, fmt.Errorf("specutil: failed converting schema to spec: %w", err)
			}
			d.Tables = append(d.Tables, tables...)
			d.Schemas = append(d.Schemas, spec)
		}
	default:
		return nil, fmt.Errorf("specutil: failed marshaling spec. %T is not supported", v)
	}
	if err := QualifyDuplicates(d.Tables); err != nil {
		return nil, err
	}
	return marshaler.MarshalSpec(d)
}

// QualifyDuplicates sets the Qualified field equal to the schema name in any tables
// with duplicate names in the provided table specs.
func QualifyDuplicates(tableSpecs []*sqlspec.Table) error {
	seen := make(map[string]*sqlspec.Table, len(tableSpecs))
	for _, tbl := range tableSpecs {
		if s, ok := seen[tbl.Name]; ok {
			schemaName, err := SchemaName(s.Schema)
			if err != nil {
				return err
			}
			s.Qualifier = schemaName
			schemaName, err = SchemaName(tbl.Schema)
			if err != nil {
				return err
			}
			tbl.Qualifier = schemaName
		}
		seen[tbl.Name] = tbl
	}
	return nil
}

// Unmarshal unmarshals an Atlas DDL document using an unmarshaler into v. Unmarshal uses the
// given convertTable function to convert a *sqlspec.Table into a *schema.Table.
func Unmarshal(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{},
	convertTable ConvertTableFunc) error {
	var d doc
	if err := unmarshaler.UnmarshalSpec(data, &d); err != nil {
		return err
	}
	switch v := v.(type) {
	case *schema.Realm:
		err := Scan(v, d.Schemas, d.Tables, convertTable)
		if err != nil {
			return fmt.Errorf("specutil: failed converting to *schema.Realm: %w", err)
		}
	case *schema.Schema:
		if len(d.Schemas) != 1 {
			return fmt.Errorf("specutil: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		var r schema.Realm
		if err := Scan(&r, d.Schemas, d.Tables, convertTable); err != nil {
			return err
		}
		r.Schemas[0].Realm = nil
		*v = *r.Schemas[0]
	default:
		return fmt.Errorf("specutil: failed unmarshaling spec. %T is not supported", v)
	}
	return nil
}
