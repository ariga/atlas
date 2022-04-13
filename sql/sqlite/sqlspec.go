package sqlite

import (
	"reflect"
	"strconv"

	"ariga.io/atlas/internal/types"
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
	return specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, convertIndex, specutil.Check)
}

// convertIndex converts a sqlspec.Index into a schema.Index.
func convertIndex(spec *sqlspec.Index, t *schema.Table) (*schema.Index, error) {
	idx, err := specutil.Index(spec, t)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("where"); ok {
		p, err := attr.String()
		if err != nil {
			return nil, err
		}
		idx.Attrs = append(idx.Attrs, &IndexPredicate{P: p})
	}
	return idx, nil
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("auto_increment"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		if b {
			c.AddAttrs(&AutoIncrement{})
		}
	}
	return c, nil
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
		indexSpec,
		specutil.FromForeignKey,
		specutil.FromCheck,
	)
}

func indexSpec(idx *schema.Index) (*sqlspec.Index, error) {
	spec, err := specutil.FromIndex(idx)
	if err != nil {
		return nil, err
	}
	if i := (IndexPredicate{}); sqlx.Has(idx.Attrs, &i) && i.P != "" {
		spec.Extra.Attrs = append(spec.Extra.Attrs, specutil.VarAttr("where", strconv.Quote(i.P)))
	}
	return spec, nil
}

// columnSpec converts from a concrete SQLite schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, _ *schema.Table) (*sqlspec.Column, error) {
	s, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if sqlx.Has(c.Attrs, &AutoIncrement{}) {
		s.Extra.Attrs = append(s.Extra.Attrs, specutil.BoolAttr("auto_increment", true))
	}
	return s, nil
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
var TypeRegistry = types.NewRegistry(
	types.WithFormatter(FormatType),
	types.WithParser(ParseType),
	types.WithSpecs(
		types.Spec(TypeReal, types.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		types.Spec(TypeBlob, types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec(TypeText, types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec(TypeInteger, types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("int", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("tinyint", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("smallint", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("mediumint", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("bigint", types.WithAttributes(types.SizeTypeAttr(false))),
		types.AliasSpec("unsigned_big_int", "unsigned big int", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("int2", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("int8", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("double", types.WithAttributes(types.SizeTypeAttr(false))),
		types.AliasSpec("double_precision", "double precision", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("float", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("character", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("varchar", types.WithAttributes(types.SizeTypeAttr(false))),
		types.AliasSpec("varying_character", "varying character", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("nchar", types.WithAttributes(types.SizeTypeAttr(false))),
		types.AliasSpec("native_character", "native character", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("nvarchar", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("clob", types.WithAttributes(types.SizeTypeAttr(false))),
		types.Spec("numeric", types.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		types.Spec("decimal", types.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		types.Spec("boolean"),
		types.Spec("date"),
		types.Spec("datetime"),
		types.Spec("json"),
		types.Spec("uuid"),
	),
)

var (
	hclState = schemahcl.New(
		schemahcl.WithTypes(TypeRegistry.Specs()),
		schemahcl.WithScopedEnums("table.foreign_key.on_update", specutil.ReferenceVars...),
		schemahcl.WithScopedEnums("table.foreign_key.on_delete", specutil.ReferenceVars...),
	)
	// UnmarshalHCL unmarshals an Atlas HCL DDL document into v.
	UnmarshalHCL = schemaspec.UnmarshalerFunc(func(bytes []byte, i interface{}) error {
		return UnmarshalSpec(bytes, hclState, i)
	})
	// MarshalHCL marshals v into an Atlas HCL DDL document.
	MarshalHCL = schemaspec.MarshalerFunc(func(v interface{}) ([]byte, error) {
		return MarshalSpec(v, hclState)
	})
)
