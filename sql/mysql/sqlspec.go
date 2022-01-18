package mysql

import (
	"fmt"
	"reflect"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
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
	switch v := v.(type) {
	case *schema.Realm:
		realm, err := specutil.Realm(d.Schemas, d.Tables, convertTable)
		if err != nil {
			return fmt.Errorf("mysql: failed converting to *schema.Realm: %w", err)
		}
		for _, schemaSpec := range d.Schemas {
			schm, ok := realm.Schema(schemaSpec.Name)
			if !ok {
				return fmt.Errorf("could not find schema: %q", schemaSpec.Name)
			}
			if err := convertCharset(schemaSpec, &schm.Attrs); err != nil {
				return err
			}
		}
		*v = *realm
	case *schema.Schema:
		if len(d.Schemas) != 1 {
			return fmt.Errorf("mysql: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		conv, err := specutil.Schema(d.Schemas[0], d.Tables, convertTable)
		if err != nil {
			return fmt.Errorf("mysql: failed converting to *schema.Schema: %w", err)
		}
		if err := convertCharset(d.Schemas[0], &conv.Attrs); err != nil {
			return err
		}
		*v = *conv
	default:
		return fmt.Errorf("mysql: failed unmarshaling spec. %T is not supported", v)
	}
	return nil
}

// MarshalSpec marshals v into an Atlas DDL document using a schemaspec.Marshaler.
func MarshalSpec(v interface{}, marshaler schemaspec.Marshaler) ([]byte, error) {
	return specutil.Marshal(v, marshaler, schemaSpec)
}

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

// convertTable converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the convertSchema function.
func convertTable(spec *sqlspec.Table, parent *schema.Schema) (*schema.Table, error) {
	t, err := specutil.Table(spec, parent, convertColumn, specutil.PrimaryKey, specutil.Index, convertCheck)
	if err != nil {
		return nil, err
	}
	if err := convertCharset(spec, &t.Attrs); err != nil {
		return nil, err
	}
	return t, err
}

// convertCheck converts a sqlspec.Check into a schema.Check.
func convertCheck(spec *sqlspec.Check) (*schema.Check, error) {
	c, err := specutil.Check(spec)
	if err != nil {
		return nil, err
	}
	if attr, ok := spec.Attr("enforced"); ok {
		b, err := attr.Bool()
		if err != nil {
			return nil, err
		}
		if b {
			c.AddAttrs(&Enforced{})
		}
	}
	return c, nil
}

// convertColumn converts a sqlspec.Column into a schema.Column.
func convertColumn(spec *sqlspec.Column, _ *schema.Table) (*schema.Column, error) {
	c, err := specutil.Column(spec, convertColumnType)
	if err != nil {
		return nil, err
	}
	if err := convertCharset(spec, &c.Attrs); err != nil {
		return nil, err
	}
	return c, err
}

// convertColumnType converts a sqlspec.Column into a concrete MySQL schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	return TypeRegistry.Type(spec.Type, spec.Extra.Attrs)
}

// schemaSpec converts from a concrete MySQL schema to Atlas specification.
func schemaSpec(s *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	sc, t, err := specutil.FromSchema(s, tableSpec)
	if err != nil {
		return nil, nil, err
	}
	if c, ok := hasCharset(s.Attrs, nil); ok {
		sc.Extra.Attrs = append(sc.Extra.Attrs, specutil.StrAttr("charset", c))
	}
	if c, ok := hasCollate(s.Attrs, nil); ok {
		sc.Extra.Attrs = append(sc.Extra.Attrs, specutil.StrAttr("collation", c))
	}
	return sc, t, nil
}

// tableSpec converts from a concrete MySQL sqlspec.Table to a schema.Table.
func tableSpec(t *schema.Table) (*sqlspec.Table, error) {
	ts, err := specutil.FromTable(
		t,
		columnSpec,
		specutil.FromPrimaryKey,
		specutil.FromIndex,
		specutil.FromForeignKey,
		checkSpec,
	)
	if err != nil {
		return nil, err
	}
	if c, ok := hasCharset(t.Attrs, t.Schema.Attrs); ok {
		ts.Extra.Attrs = append(ts.Extra.Attrs, specutil.StrAttr("charset", c))
	}
	if c, ok := hasCollate(t.Attrs, t.Schema.Attrs); ok {
		ts.Extra.Attrs = append(ts.Extra.Attrs, specutil.StrAttr("collation", c))
	}
	return ts, nil
}

// columnSpec converts from a concrete MySQL schema.Column into a sqlspec.Column.
func columnSpec(c *schema.Column, t *schema.Table) (*sqlspec.Column, error) {
	col, err := specutil.FromColumn(c, columnTypeSpec)
	if err != nil {
		return nil, err
	}
	if c, ok := hasCharset(c.Attrs, t.Attrs); ok {
		col.Extra.Attrs = append(col.Extra.Attrs, specutil.StrAttr("charset", c))
	}
	if c, ok := hasCollate(c.Attrs, t.Attrs); ok {
		col.Extra.Attrs = append(col.Extra.Attrs, specutil.StrAttr("collation", c))
	}
	return col, nil
}

// checkSpec converts from a concrete MySQL schema.Check into a sqlspec.Check.
func checkSpec(s *schema.Check) *sqlspec.Check {
	c := specutil.FromCheck(s)
	if e := (Enforced{}); sqlx.Has(s.Attrs, &e) {
		c.Extra.Attrs = append(c.Extra.Attrs, specutil.BoolAttr("enforced", true))
	}
	return c
}

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	st, err := TypeRegistry.Convert(t)
	if err != nil {
		return nil, err
	}
	c := &sqlspec.Column{Type: st}
	for _, attr := range st.Attrs {
		// TODO(rotemtam): infer this from the TypeSpec
		if attr.K == "unsigned" {
			c.Extra.Attrs = append(c.Extra.Attrs, attr)
		}
	}
	return c, nil
}

// convertCharset converts spec charset/collation
// attributes to schema element attributes.
func convertCharset(spec specutil.Attrer, attrs *[]schema.Attr) error {
	if attr, ok := spec.Attr("charset"); ok {
		s, err := attr.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Charset{V: s})
	}
	if attr, ok := spec.Attr("collation"); ok {
		s, err := attr.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Collation{V: s})
	}
	return nil
}

// hasCharset reports if the attribute contains the "charset" attribute,
// and it needs to be defined explicitly on the schema. This is true, in
// case the element charset is different from its parent charset.
func hasCharset(attr []schema.Attr, parent []schema.Attr) (string, bool) {
	var c, p schema.Charset
	if sqlx.Has(attr, &c) && (parent == nil || sqlx.Has(parent, &p) && c.V != p.V) {
		return c.V, true
	}
	return "", false
}

// hasCollate reports if the attribute contains the "collation" attribute,
// and it needs to be defined explicitly on the schema. This is true, in
// case the element collation is different from its parent collation.
func hasCollate(attr []schema.Attr, parent []schema.Attr) (string, bool) {
	var c, p schema.Collation
	if sqlx.Has(attr, &c) && (parent == nil || sqlx.Has(parent, &p) && c.V != p.V) {
		return c.V, true
	}
	return "", false
}

// TypeRegistry contains the supported TypeSpecs for the mysql driver.
var TypeRegistry = specutil.NewRegistry(
	specutil.WithFormatter(FormatType),
	specutil.WithParser(ParseType),
	specutil.WithSpecs(
		&schemaspec.TypeSpec{
			Name: TypeEnum,
			T:    TypeEnum,
			Attributes: []*schemaspec.TypeAttr{
				{Name: "values", Kind: reflect.Slice, Required: true},
			},
			RType: reflect.TypeOf(schema.EnumType{}),
		},
		&schemaspec.TypeSpec{
			Name: TypeSet,
			T:    TypeSet,
			Attributes: []*schemaspec.TypeAttr{
				{Name: "values", Kind: reflect.Slice, Required: true},
			},
			RType: reflect.TypeOf(SetType{}),
		},
		specutil.TypeSpec(TypeBit, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeTinyInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeSmallInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeMediumInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeBigInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeDecimal, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeNumeric, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeFloat, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeDouble, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeReal, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
		specutil.TypeSpec(TypeTimestamp),
		specutil.TypeSpec(TypeDate),
		specutil.TypeSpec(TypeTime),
		specutil.TypeSpec(TypeDateTime),
		specutil.TypeSpec(TypeYear),
		specutil.TypeSpec(TypeVarchar, specutil.SizeTypeAttr(true)),
		specutil.TypeSpec(TypeChar, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeVarBinary, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeBinary, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeBlob, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeTinyBlob),
		specutil.TypeSpec(TypeMediumBlob),
		specutil.TypeSpec(TypeLongBlob),
		specutil.TypeSpec(TypeJSON),
		specutil.TypeSpec(TypeText, specutil.SizeTypeAttr(false)),
		specutil.TypeSpec(TypeTinyText),
		specutil.TypeSpec(TypeMediumText),
		specutil.TypeSpec(TypeLongText),
		specutil.TypeSpec(TypeGeometry),
		specutil.TypeSpec(TypePoint),
		specutil.TypeSpec(TypeMultiPoint),
		specutil.TypeSpec(TypeLineString),
		specutil.TypeSpec(TypeMultiLineString),
		specutil.TypeSpec(TypePolygon),
		specutil.TypeSpec(TypeMultiPolygon),
		specutil.TypeSpec(TypeGeometryCollection),
	),
)

func unsignedTypeAttr() *schemaspec.TypeAttr {
	return &schemaspec.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
