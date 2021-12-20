package sqlite

import (
	"fmt"
	"reflect"

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
		return fmt.Errorf("sqlite: failed unmarshaling spec. %T is not supported", v)
	}
	if len(d.Schemas) != 1 {
		return fmt.Errorf("sqlite: expecting document to contain a single schema, got %d", len(d.Schemas))
	}
	conv, err := specutil.Schema(d.Schemas[0], d.Tables, convertTable)
	if err != nil {
		return fmt.Errorf("sqlite: failed converting to *schema.Schema: %w", err)
	}
	*s = *conv
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemaspec.Marshaler.
func MarshalSpec(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error) {
	s, ok := v.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("sqlite: failed marshaling spec. %T is not supported", v)
	}
	spec, tables, err := schemaSpec(s)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed converting schema to spec: %w", err)
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

// convertColumnType converts a sqlspec.Column into a concrete SQLite schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs, parseRawType)
}

// schemaSpec converts from a concrete SQLite schema to Atlas specification.
func schemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return specutil.FromSchema(schem, tableSpec)
}

// tableSpec converts from a concrete SQLite sqlspec.Table to a schema.Table.
func tableSpec(tab *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(tab, columnSpec, specutil.FromPrimaryKey, specutil.FromIndex, specutil.FromForeignKey)
}

// columnSpec converts from a concrete SQLite schema.Column into a sqlspec.Column.
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
	specutil.TypeSpec(tReal, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
	specutil.TypeSpec(tBlob, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tText, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec("integer", specutil.SizeTypeAttr(false)),
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
)
