package postgres

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

type doc struct {
	Tables  []*sqlspec.Table  `spec:"table"`
	Schemas []*sqlspec.Schema `spec:"schema"`
}

// UnmarshalSpec unmarshals an Atlas DDL document using an unmarshaler into v.
func UnmarshalSpec(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error {
	var d doc
	if err := unmarshaler.UnmarshalSpec(data, &d); err != nil {
		return err
	}
	s, ok := v.(*schema.Schema)
	if !ok {
		return fmt.Errorf("postgres: failed unmarshaling spec. %T is not supported", v)
	}
	if len(d.Schemas) != 1 {
		return fmt.Errorf("postgres: expecting document to contain a single schema, got %d", len(d.Schemas))
	}
	conv, err := specutil.Schema(d.Schemas[0], d.Tables, convertTable)
	if err != nil {
		return fmt.Errorf("postgres: failed converting to *schema.Schema: %w", err)
	}
	*s = *conv
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemaspec.Marshaler.
func MarshalSpec(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error) {
	var (
		s  *schema.Schema
		ok bool
	)
	if s, ok = v.(*schema.Schema); !ok {
		return nil, fmt.Errorf("failed marshaling spec. %T is not supported", v)
	}
	spec, tables, err := schemaSpec(s)
	if err != nil {
		return nil, fmt.Errorf("failed converting schema to spec: %w", err)
	}
	return marshaler.MarshalSpec(&doc{
		Tables:  tables,
		Schemas: []*sqlspec.Schema{spec},
	})
}

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	return specutil.Table(spec, parent, convertColumn, convertPrimaryKey, convertIndex)
}

// convertPrimaryKey converts a sqlspec.PrimaryKey to a schema.Index.
func convertPrimaryKey(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	return specutil.PrimaryKey(spec, parent)
}

// convertIndex converts an sqlspec.Index to a schema.Index.
func convertIndex(spec *sqlspec.Index, parent *schema.Table) (*schema.Index, error) {
	return specutil.Index(spec, parent)
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	return specutil.Column(spec, convertColumnType)
}

// convertColumnType converts a sqlspec.Column into a concrete Postgres schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs, parseRawType)
}

func parseRawType(typ string) (schema.Type, error) {
	d, err := parseColumn(typ)
	if err != nil {
		return nil, err
	}
	// Normalize PostgreSQL array data types from "CREATE TABLE" format to
	// "INFORMATION_SCHEMA" format (i.e. as it is inspected from the database).
	if t, ok := arrayType(typ); ok {
		d = &columnDesc{typ: tArray, udt: t}
	}

	t := columnType(d)
	// If the type is unknown (to us), we fallback to user-defined but expect
	// to improve this in future versions by ensuring this against the database.
	if ut, ok := t.(*schema.UnsupportedType); ok {
		t = &UserDefinedType{T: ut.T}
	}
	return t, nil
}

// schemaSpec converts from a concrete Postgres schema to Atlas specification.
func schemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return specutil.FromSchema(schem, tableSpec)
}

// tableSpec converts from a concrete Postgres sqlspec.Table to a schema.Table.
func tableSpec(tab *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(tab, columnSpec, specutil.FromPrimaryKey, specutil.FromIndex, specutil.FromForeignKey)
}

// columnSpec converts from a concrete Postgres schema.Column into a sqlspec.Column.
func columnSpec(col *schema.Column, t *schema.Table) (*sqlspec.Column, error) {
	ct, err := columnTypeSpec(col.Type.Type)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{
		Name: col.Name,
		Type: ct.Type,
		Null: col.Type.Null,
		DefaultExtension: schemaspec.DefaultExtension{
			Extra: schemaspec.Resource{Attrs: ct.DefaultExtension.Extra.Attrs},
		},
	}, nil
}

// columnTypeSpec converts from a concrete Postgres schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{Type: st}, nil
}

// arrayType reports if the given string is an array type (e.g. int[], text[2]),
// and returns its "udt_name" as it was inspected from the database.
func arrayType(t string) (string, bool) {
	i, j := strings.LastIndexByte(t, '['), strings.LastIndexByte(t, ']')
	if i == -1 || j == -1 {
		return "", false
	}
	for _, r := range t[i+1 : j] {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	return t[:strings.IndexByte(t, '[')], true
}

// TypeRegistry contains the supported TypeSpecs for the Postgres driver.
var TypeRegistry = specutil.NewRegistry(
	specutil.TypeSpec(tBit, &schemaspec.TypeAttr{Name: "len", Kind: reflect.Int64}),
	specutil.AliasTypeSpec("bit_varying", tBitVar, &schemaspec.TypeAttr{Name: "len", Kind: reflect.Int64}),
	specutil.TypeSpec(tVarChar, specutil.SizeTypeAttr(true)),
	specutil.TypeSpec(tChar, specutil.SizeTypeAttr(true)),
	specutil.TypeSpec(tCharacter, specutil.SizeTypeAttr(true)),
	specutil.TypeSpec(tInt2),
	specutil.TypeSpec(tInt4),
	specutil.TypeSpec(tInt8),
	specutil.TypeSpec(tInt),
	specutil.TypeSpec(tInteger),
	specutil.TypeSpec(tSmallInt),
	specutil.TypeSpec(tBigInt),
	specutil.TypeSpec(tText),
	specutil.TypeSpec(tBoolean),
	specutil.TypeSpec(tBool),
	specutil.TypeSpec(tBytea),
	specutil.TypeSpec(tCIDR),
	specutil.TypeSpec(tInet),
	specutil.TypeSpec(tMACAddr),
	specutil.TypeSpec(tMACAddr8),
	specutil.TypeSpec(tCircle),
	specutil.TypeSpec(tLine),
	specutil.TypeSpec(tLseg),
	specutil.TypeSpec(tBox),
	specutil.TypeSpec(tPath),
	specutil.TypeSpec(tPoint),
	specutil.TypeSpec(tDate),
	specutil.TypeSpec(tTime),
	specutil.AliasTypeSpec("time_with_time_zone", tTimeWTZ),
	specutil.AliasTypeSpec("time_without_time_zone", tTimeWOTZ),
	specutil.TypeSpec(tTimestamp),
	specutil.AliasTypeSpec("timestamp_with_time_zone", tTimestampWTZ),
	specutil.AliasTypeSpec("timestamp_without_time_zone", tTimestampWOTZ),
	specutil.TypeSpec("enum", &schemaspec.TypeAttr{Name: "values", Kind: reflect.Slice, Required: true}),
	specutil.AliasTypeSpec("double_precision", tDouble),
	specutil.TypeSpec(tReal),
	specutil.TypeSpec(tFloat8),
	specutil.TypeSpec(tFloat4),
	specutil.TypeSpec(tNumeric),
	specutil.TypeSpec(tDecimal),
	specutil.TypeSpec(tSmallSerial),
	specutil.TypeSpec(tSerial),
	specutil.TypeSpec(tBigSerial),
	specutil.TypeSpec(tSerial2),
	specutil.TypeSpec(tSerial4),
	specutil.TypeSpec(tSerial8),
	specutil.TypeSpec(tXML),
	specutil.TypeSpec(tJSON),
	specutil.TypeSpec(tJSONB),
	specutil.TypeSpec(tUUID),
	specutil.TypeSpec(tMoney),
	specutil.TypeSpec("hstore"),
)
