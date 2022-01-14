package postgres

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// UnmarshalSpec unmarshals an Atlas DDL document using an unmarshaler into v.
func UnmarshalSpec(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error {
	return specutil.Unmarshal(data, unmarshaler, v, convertTable)
}

// MarshalSpec marshals v into an Atlas DDL document using a schemaspec.Marshaler.
func MarshalSpec(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error) {
	return specutil.Marshal(v, marshaler, schemaSpec)
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
	if err := fixDefaultQuotes(spec.Default); err != nil {
		return nil, err
	}
	return specutil.Column(spec, convertColumnType)
}

// fixDefaultQuotes fixes the quotes on the Default field to be single quotes
// instead of double quotes.
func fixDefaultQuotes(value schemaspec.Value) error {
	lv, ok := value.(*schemaspec.LiteralValue)
	if !ok {
		return nil
	}
	if sqlx.IsQuoted(lv.V, '"') {
		uq, err := strconv.Unquote(lv.V)
		if err != nil {
			return err
		}
		lv.V = "'" + uq + "'"
	}
	return nil
}

// convertColumnType converts a sqlspec.Column into a concrete Postgres schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

func parseRawType(typ string) (schema.Type, error) {
	d, err := parseColumn(typ)
	if err != nil {
		return nil, err
	}
	// Normalize PostgreSQL array data types from "CREATE TABLE" format to
	// "INFORMATION_SCHEMA" format (i.e. as it is inspected from the database).
	if t, ok := arrayType(typ); ok {
		d = &columnDesc{typ: TypeArray, udt: t}
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
	return specutil.FromTable(
		tab,
		columnSpec,
		specutil.FromPrimaryKey,
		specutil.FromIndex,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
}

// columnSpec converts from a concrete Postgres schema.Column into a sqlspec.Column.
func columnSpec(col *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	return specutil.FromColumn(col, columnTypeSpec)
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
	specutil.WithFormatter(FormatType),
	specutil.WithParser(parseRawType),
	specutil.WithSpecs(
		specutil.TypeSpec(TypeBit, &schemaspec.TypeAttr{Name: "len", Kind: reflect.Int64}),
		specutil.AliasTypeSpec("bit_varying", TypeBitVar, &schemaspec.TypeAttr{Name: "len", Kind: reflect.Int64}),
		specutil.TypeSpec(TypeVarChar, specutil.SizeTypeAttr(true)),
		specutil.AliasTypeSpec("character_varying", TypeCharVar, &schemaspec.TypeAttr{Name: "size", Kind: reflect.Int}),
		specutil.TypeSpec(TypeChar, specutil.SizeTypeAttr(true)),
		specutil.TypeSpec(TypeCharacter, specutil.SizeTypeAttr(true)),
		specutil.TypeSpec(TypeInt2),
		specutil.TypeSpec(TypeInt4),
		specutil.TypeSpec(TypeInt8),
		specutil.TypeSpec(TypeInt),
		specutil.TypeSpec(TypeInteger),
		specutil.TypeSpec(TypeSmallInt),
		specutil.TypeSpec(TypeBigInt),
		specutil.TypeSpec(TypeText),
		specutil.TypeSpec(TypeBoolean),
		specutil.TypeSpec(TypeBool),
		specutil.TypeSpec(TypeBytea),
		specutil.TypeSpec(TypeCIDR),
		specutil.TypeSpec(TypeInet),
		specutil.TypeSpec(TypeMACAddr),
		specutil.TypeSpec(TypeMACAddr8),
		specutil.TypeSpec(TypeCircle),
		specutil.TypeSpec(TypeLine),
		specutil.TypeSpec(TypeLseg),
		specutil.TypeSpec(TypeBox),
		specutil.TypeSpec(TypePath),
		specutil.TypeSpec(TypePoint),
		specutil.TypeSpec(TypeDate),
		specutil.TypeSpec(TypeTime),
		specutil.AliasTypeSpec("time_with_time_zone", TypeTimeWTZ),
		specutil.AliasTypeSpec("time_without_time_zone", TypeTimeWOTZ),
		specutil.TypeSpec(TypeTimestamp),
		specutil.AliasTypeSpec("timestamp_with_time_zone", TypeTimestampWTZ),
		specutil.AliasTypeSpec("timestamp_without_time_zone", TypeTimestampWOTZ),
		specutil.TypeSpec("enum", &schemaspec.TypeAttr{Name: "values", Kind: reflect.Slice, Required: true}),
		specutil.AliasTypeSpec("double_precision", TypeDouble),
		specutil.TypeSpec(TypeReal),
		specutil.TypeSpec(TypeFloat8),
		specutil.TypeSpec(TypeFloat4),
		specutil.TypeSpec(TypeNumeric),
		specutil.TypeSpec(TypeDecimal),
		specutil.TypeSpec(TypeSmallSerial),
		specutil.TypeSpec(TypeSerial),
		specutil.TypeSpec(TypeBigSerial),
		specutil.TypeSpec(TypeSerial2),
		specutil.TypeSpec(TypeSerial4),
		specutil.TypeSpec(TypeSerial8),
		specutil.TypeSpec(TypeXML),
		specutil.TypeSpec(TypeJSON),
		specutil.TypeSpec(TypeJSONB),
		specutil.TypeSpec(TypeUUID),
		specutil.TypeSpec(TypeMoney),
		specutil.TypeSpec("hstore"),
		specutil.TypeSpec("sql", &schemaspec.TypeAttr{Name: "def", Required: true, Kind: reflect.String}),
	),
)

var (
	hclState = schemahcl.New(schemahcl.WithTypes(TypeRegistry.Specs()))
	// UnmarshalHCL unmarshals an Atlas HCL DDL document into v.
	UnmarshalHCL = schemaspec.UnmarshalerFunc(func(bytes []byte, i interface{}) error {
		return UnmarshalSpec(bytes, hclState, i)
	})
	// MarshalHCL marshals v into an Atlas HCL DDL document.
	MarshalHCL = schemaspec.MarshalerFunc(func(v interface{}) ([]byte, error) {
		return MarshalSpec(v, hclState)
	})
)
