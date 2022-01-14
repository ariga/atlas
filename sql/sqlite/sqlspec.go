package sqlite

import (
	"reflect"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
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
	return specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, specutil.Index, specutil.Check)
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	return specutil.Column(spec, convertColumnType)
}

// convertColumnType converts a sqlspec.Column into a concrete SQLite schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

// schemaSpec converts from a concrete SQLite schema to Atlas specification.
func schemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return specutil.FromSchema(schem, tableSpec)
}

// tableSpec converts from a concrete SQLite sqlspec.Table to a schema.Table.
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

// columnSpec converts from a concrete SQLite schema.Column into a sqlspec.Column.
func columnSpec(col *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	return specutil.FromColumn(col, columnTypeSpec)
}

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{Type: st}, nil
}

// TypeRegistry contains the supported TypeSpecs for the sqlite driver.
var TypeRegistry = specutil.NewRegistry(
	specutil.WithFormatter(FormatType),
	specutil.WithParser(ParseType),
	specutil.WithSpecs(
		specutil.TypeSpec(TypeReal, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeBlob, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeText, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeInteger, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("int", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("tinyint", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("smallint", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("mediumint", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("bigint", specutil.SizeTypeAttr(false)),
		specutil.AliasTypeSpec("unsigned_big_int", "unsigned big int", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("int2", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("int8", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("double", specutil.SizeTypeAttr(false)),
		specutil.AliasTypeSpec("double_precision", "double precision", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("float", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("character", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("varchar", specutil.SizeTypeAttr(false)),
		specutil.AliasTypeSpec("varying_character", "varying character", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("nchar", specutil.SizeTypeAttr(false)),
		specutil.AliasTypeSpec("native_character", "native character", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("nvarchar", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("clob", specutil.SizeTypeAttr(false)),
		specutil.TypeSpec("numeric", &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec("decimal", &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec("boolean"),
		specutil.TypeSpec("date"),
		specutil.TypeSpec("datetime"),
		specutil.TypeSpec("json"),
		specutil.TypeSpec("uuid"),
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
