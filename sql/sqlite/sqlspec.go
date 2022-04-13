package sqlite

import (
	"reflect"
	"strconv"

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
var TypeRegistry = schemahcl.NewRegistry(
	schemahcl.WithFormatter(FormatType),
	schemahcl.WithParser(ParseType),
	schemahcl.WithSpecs(
		schemahcl.Spec(TypeReal, schemahcl.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		schemahcl.Spec(TypeBlob, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec(TypeText, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec(TypeInteger, schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("int", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("tinyint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("smallint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("mediumint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("bigint", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasSpec("unsigned_big_int", "unsigned big int", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("int2", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("int8", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("double", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasSpec("double_precision", "double precision", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("float", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("varchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasSpec("varying_character", "varying character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("nchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.AliasSpec("native_character", "native character", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("nvarchar", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("clob", schemahcl.WithAttributes(schemahcl.SizeTypeAttr(false))),
		schemahcl.Spec("numeric", schemahcl.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		schemahcl.Spec("decimal", schemahcl.WithAttributes(&schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false})),
		schemahcl.Spec("boolean"),
		schemahcl.Spec("date"),
		schemahcl.Spec("datetime"),
		schemahcl.Spec("json"),
		schemahcl.Spec("uuid"),
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
