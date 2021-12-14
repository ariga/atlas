package mysql

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
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
	s, ok := v.(*schema.Schema)
	if !ok {
		return fmt.Errorf("mysql: failed unmarshaling spec. %T is not supported", v)
	}
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
		return nil, fmt.Errorf("mysql: failed marshaling spec. %T is not supported", v)
	}
	spec, tables, err := schemaSpec(s)
	if err != nil {
		return nil, fmt.Errorf("mysql: failed converting schema to spec: %w", err)
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
	t, err := specutil.Table(spec, parent, convertColumn, convertPrimaryKey, convertIndex)
	if err != nil {
		return nil, err
	}
	if err := convertCharset(spec, &t.Attrs); err != nil {
		return nil, err
	}
	return t, err
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
	switch sqlspec.Type(spec.Type) {
	case sqlspec.TypeInt, sqlspec.TypeInt8, sqlspec.TypeInt16,
		sqlspec.TypeInt64, sqlspec.TypeUint, sqlspec.TypeUint8,
		sqlspec.TypeUint16, sqlspec.TypeUint64:
		return convertInteger(spec)
	case sqlspec.TypeString:
		return convertString(spec)
	case sqlspec.TypeBinary:
		return convertBinary(spec)
	case sqlspec.TypeEnum:
		return convertEnum(spec)
	case sqlspec.TypeBoolean:
		return convertBoolean(spec)
	case sqlspec.TypeDecimal:
		return convertDecimal(spec)
	case sqlspec.TypeFloat:
		return convertFloat(spec)
	case sqlspec.TypeTime:
		return convertTime(spec)
	}
	return parseRawType(spec.Type)
}

func convertInteger(spec *sqlspec.Column) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.Type, "u"),
	}
	switch spec.Type {
	case "int8", "uint8":
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.T = tInt
	case "int64", "uint64":
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.Type)
	}
	return typ, nil
}

func convertBinary(spec *sqlspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		bt.Size = s
	}
	switch {
	case bt.Size == 0:
		bt.T = tBlob
	case bt.Size <= math.MaxUint8:
		bt.T = tTinyBlob
	case bt.Size > math.MaxUint8 && bt.Size <= math.MaxUint16:
		bt.T = tBlob
	case bt.Size > math.MaxUint16 && bt.Size <= 1<<24-1:
		bt.T = tMediumBlob
	case bt.Size > 1<<24-1 && bt.Size <= math.MaxUint32:
		bt.T = tLongBlob
	default:
		return nil, fmt.Errorf("mysql: blob fields can be up to 4GB long")
	}
	return bt, nil
}

func convertString(spec *sqlspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		Size: 255,
	}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	switch {
	case st.Size <= math.MaxUint16:
		st.T = tVarchar
	case st.Size > math.MaxUint16 && st.Size <= (1<<24-1):
		st.T = tMediumText
	case st.Size > (1<<24-1) && st.Size <= math.MaxUint32:
		st.T = tLongText
	default:
		return nil, fmt.Errorf("mysql: string fields can be up to 4GB long")
	}
	return st, nil
}

func convertEnum(spec *sqlspec.Column) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, fmt.Errorf("mysql: expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
}

func convertBoolean(_ *sqlspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(_ *sqlspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
}

func convertDecimal(spec *sqlspec.Column) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: tDecimal,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		dt.Precision = p
	}
	if scale, ok := spec.Attr("scale"); ok {
		s, err := scale.Int()
		if err != nil {
			return nil, err
		}
		dt.Scale = s
	}
	return dt, nil
}

func convertFloat(spec *sqlspec.Column) (schema.Type, error) {
	ft := &schema.FloatType{
		T: tFloat,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	// A precision from 0 to 23 results in a 4-byte single-precision FLOAT column.
	// A precision from 24 to 53 results in an 8-byte double-precision DOUBLE column:
	// https://dev.mysql.com/doc/refman/8.0/en/floating-point-types.html
	if ft.Precision > 23 {
		ft.T = tDouble
	}
	return ft, nil
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
	ts, err := specutil.FromTable(t, columnSpec, specutil.FromPrimaryKey, specutil.FromIndex, specutil.FromForeignKey)
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
	ct, err := columnTypeSpec(c.Type.Type)
	if err != nil {
		return nil, err
	}
	if c, ok := hasCharset(c.Attrs, t.Attrs); ok {
		ct.Extra.Attrs = append(ct.Extra.Attrs, specutil.StrAttr("charset", c))
	}
	if c, ok := hasCollate(c.Attrs, t.Attrs); ok {
		ct.Extra.Attrs = append(ct.Extra.Attrs, specutil.StrAttr("collation", c))
	}
	return &sqlspec.Column{
		Name: c.Name,
		Type: ct.Type,
		Null: c.Type.Null,
		DefaultExtension: schemaspec.DefaultExtension{
			Extra: schemaspec.Resource{Attrs: ct.DefaultExtension.Extra.Attrs},
		},
	}, nil
}

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func columnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	switch t := t.(type) {
	case *schema.EnumType:
		return enumSpec(t)
	case *schema.IntegerType:
		return integerSpec(t)
	case *schema.StringType:
		return stringSpec(t)
	case *schema.DecimalType:
		precision := specutil.LitAttr("precision", strconv.Itoa(t.Precision))
		scale := specutil.LitAttr("scale", strconv.Itoa(t.Scale))
		return specutil.NewCol("", "decimal", precision, scale), nil
	case *schema.BinaryType:
		return binarySpec(t)
	case *schema.BoolType:
		return &sqlspec.Column{Type: "boolean"}, nil
	case *schema.FloatType:
		precision := specutil.LitAttr("precision", strconv.Itoa(t.Precision))
		return specutil.NewCol("", "float", precision), nil
	case *schema.TimeType:
		return &sqlspec.Column{Type: t.T}, nil
	case *schema.JSONType:
		return &sqlspec.Column{Type: t.T}, nil
	case *schema.SpatialType:
		return &sqlspec.Column{Type: t.T}, nil
	case *schema.UnsupportedType:
		return &sqlspec.Column{Type: t.T}, nil
	default:
		return nil, fmt.Errorf("mysql: failed to convert column type %T to spec", t)
	}
}

func binarySpec(t *schema.BinaryType) (*sqlspec.Column, error) {
	switch t.T {
	case tBlob:
		return &sqlspec.Column{Type: "binary"}, nil
	case tTinyBlob, tMediumBlob, tLongBlob:
		size := specutil.LitAttr("size", strconv.Itoa(t.Size))
		return specutil.NewCol("", "binary", size), nil
	}
	return nil, fmt.Errorf("mysql: schema binary failed to convert %q", t.T)
}

func stringSpec(t *schema.StringType) (*sqlspec.Column, error) {
	switch t.T {
	case tVarchar, tMediumText, tLongText, tTinyText, tText, tChar:
		if t.Size == 0 {
			return &sqlspec.Column{Type: t.T}, nil
		}
		return specutil.NewCol("", "string", specutil.LitAttr("size", strconv.Itoa(t.Size))), nil
	}
	return nil, fmt.Errorf("mysql: schema string failed to convert %q", t.T)
}

func integerSpec(t *schema.IntegerType) (*sqlspec.Column, error) {
	switch t.T {
	case tInt:
		if t.Unsigned {
			return specutil.NewCol("", "uint"), nil
		}
		return &sqlspec.Column{Type: "int"}, nil
	case tTinyInt:
		return &sqlspec.Column{Type: "int8"}, nil
	case tMediumInt:
		return &sqlspec.Column{Type: tMediumInt}, nil
	case tSmallInt:
		return &sqlspec.Column{Type: tSmallInt}, nil
	case tBigInt:
		if t.Unsigned {
			return specutil.NewCol("", "uint64"), nil
		}
		return &sqlspec.Column{Type: "int64"}, nil
	}
	return nil, fmt.Errorf("mysql: schema integer failed to convert %q", t.T)
}

func enumSpec(t *schema.EnumType) (*sqlspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("mysql: schema enum fields to have values")
	}
	quoted := make([]string, 0, len(t.Values))
	for _, v := range t.Values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return specutil.NewCol("", "enum", specutil.ListAttr("values", quoted...)), nil
}

// convertCharset converts spec charset/collation
// attributes to schema element attributes.
func convertCharset(spec interface {
	Attr(string) (*schemaspec.Attr, bool)
}, attrs *[]schema.Attr) error {
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
	specutil.TypeSpec(tEnum, &schemaspec.TypeAttr{Name: "values", Kind: reflect.Slice, Required: true}),
	specutil.TypeSpec(tSet, &schemaspec.TypeAttr{Name: "values", Kind: reflect.Slice, Required: true}),
	specutil.TypeSpec(tBit, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tTinyInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tSmallInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tMediumInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tBigInt, unsignedTypeAttr(), specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tDecimal),
	specutil.TypeSpec(tNumeric),
	specutil.TypeSpec(tFloat, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
	specutil.TypeSpec(tDouble, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
	specutil.TypeSpec(tReal, &schemaspec.TypeAttr{Name: "precision", Kind: reflect.Int, Required: false}, &schemaspec.TypeAttr{Name: "scale", Kind: reflect.Int, Required: false}),
	specutil.TypeSpec(tTimestamp),
	specutil.TypeSpec(tDate),
	specutil.TypeSpec(tTime),
	specutil.TypeSpec(tDateTime),
	specutil.TypeSpec(tYear),
	specutil.TypeSpec(tVarchar, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tChar, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tVarBinary, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tBinary, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tBlob, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tTinyBlob),
	specutil.TypeSpec(tMediumBlob),
	specutil.TypeSpec(tLongBlob),
	specutil.TypeSpec(tText, specutil.SizeTypeAttr(false)),
	specutil.TypeSpec(tTinyText),
	specutil.TypeSpec(tMediumText),
	specutil.TypeSpec(tLongText),
	specutil.TypeSpec(tGeometry),
	specutil.TypeSpec(tPoint),
	specutil.TypeSpec(tMultiPoint),
	specutil.TypeSpec(tLineString),
	specutil.TypeSpec(tMultiLineString),
	specutil.TypeSpec(tPolygon),
	specutil.TypeSpec(tMultiPolygon),
	specutil.TypeSpec(tGeometryCollection),
)

func unsignedTypeAttr() *schemaspec.TypeAttr {
	return &schemaspec.TypeAttr{
		Name: "unsigned",
		Kind: reflect.Bool,
	}
}
