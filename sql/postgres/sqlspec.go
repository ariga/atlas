package postgres

import (
	"errors"
	"fmt"
	"strconv"
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
	switch sqlspec.Type(spec.TypeName) {
	case sqlspec.TypeInt, sqlspec.TypeInt8, sqlspec.TypeInt16,
		sqlspec.TypeInt64, sqlspec.TypeUint, sqlspec.TypeUint8,
		sqlspec.TypeUint16, sqlspec.TypeUint64:
		return convertInteger(spec)
	case sqlspec.TypeString:
		return convertString(spec)
	case sqlspec.TypeEnum:
		return convertEnum(spec)
	case sqlspec.TypeDecimal:
		return convertDecimal(spec)
	case sqlspec.TypeFloat:
		return convertFloat(spec)
	case sqlspec.TypeTime:
		return &schema.TimeType{T: tTimestamp}, nil
	case sqlspec.TypeBinary:
		return &schema.BinaryType{T: tBytea}, nil
	case sqlspec.TypeBoolean:
		return &schema.BoolType{T: tBoolean}, nil
	default:
		return parseRawType(spec)
	}
}

func convertInteger(spec *sqlspec.Column) (schema.Type, error) {
	if strings.HasPrefix(spec.TypeName, "u") {
		return nil, fmt.Errorf("unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{}
	switch sqlspec.Type(spec.TypeName) {
	case sqlspec.TypeInt8:
		return nil, fmt.Errorf("8-bit integers not supported")
	case sqlspec.TypeInt16:
		typ.T = tSmallInt
	case sqlspec.TypeInt:
		typ.T = tInteger
	case sqlspec.TypeInt64:
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("unknown integer column type %q", spec.TypeName)
	}
	return typ, nil
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
	case st.Size < maxCharSize:
		st.T = tVarChar
	default:
		st.T = tText
	}
	return st, nil
}

func convertEnum(spec *sqlspec.Column) (schema.Type, error) {
	attr, ok := spec.Attr("values")
	if !ok {
		return nil, errors.New("expected enum fields to have values")
	}
	list, err := attr.Strings()
	if err != nil {
		return nil, err
	}
	return &schema.EnumType{Values: list}, nil
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
		T: tReal,
	}
	if precision, ok := spec.Attr("precision"); ok {
		p, err := precision.Int()
		if err != nil {
			return nil, err
		}
		ft.Precision = p
	}
	if ft.Precision > 23 {
		ft.T = tDouble
	}
	return ft, nil
}

func parseRawType(spec *sqlspec.Column) (schema.Type, error) {
	d, err := parseColumn(spec.TypeName)
	if err != nil {
		return nil, err
	}
	// Normalize PostgreSQL array data types from "CREATE TABLE" format to
	// "INFORMATION_SCHEMA" format (i.e. as it is inspected from the database).
	if t, ok := arrayType(spec.TypeName); ok {
		d = &columnDesc{typ: tArray, udt: t}
	}
	return columnType(d), nil
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
func columnSpec(col *schema.Column) (*sqlspec.Column, error) {
	ct, err := columnTypeSpec(col.Type.Type)
	if err != nil {
		return nil, err
	}
	return &sqlspec.Column{
		Name:     col.Name,
		TypeName: ct.TypeName,
		Null:     ct.Null,
		DefaultExtension: schemaspec.DefaultExtension{
			Extra: schemaspec.Resource{Attrs: ct.DefaultExtension.Extra.Attrs},
		},
	}, nil
}

// columnTypeSpec converts from a concrete Postgres schema.Type into sqlspec.Column Type.
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
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.BoolType:
		return &sqlspec.Column{TypeName: "boolean"}, nil
	case *schema.FloatType:
		precision := specutil.LitAttr("precision", strconv.Itoa(t.Precision))
		return specutil.NewCol("", "float", precision), nil
	case *schema.TimeType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.JSONType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.SpatialType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *schema.UnsupportedType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *ArrayType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *BitType:
		return bitSpec(t)
	case *CurrencyType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *NetworkType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *SerialType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *UUIDType:
		return &sqlspec.Column{TypeName: t.T}, nil
	case *XMLType:
		return &sqlspec.Column{TypeName: t.T}, nil
	default:
		return nil, fmt.Errorf("failed to convert column type %T to spec", t)
	}
}

func stringSpec(t *schema.StringType) (*sqlspec.Column, error) {
	switch t.T {
	case tVarChar, tText, tChar, tCharacter, tCharVar:
		s := strconv.Itoa(t.Size)
		return specutil.NewCol("", "string", specutil.LitAttr("size", s)), nil
	default:
		return nil, fmt.Errorf("schema string failed to convert type %T", t)
	}
}

func integerSpec(t *schema.IntegerType) (*sqlspec.Column, error) {
	switch t.T {
	case tInt, tInteger:
		if t.Unsigned {
			return specutil.NewCol("", "uint"), nil
		}
		return &sqlspec.Column{TypeName: "int"}, nil
	case tBigInt:
		if t.Unsigned {
			return specutil.NewCol("", "uint64"), nil
		}
		return &sqlspec.Column{TypeName: "int64"}, nil
	default:
		return &sqlspec.Column{TypeName: t.T}, nil
	}
}

func enumSpec(t *schema.EnumType) (*sqlspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("schema enum fields to have values")
	}
	quoted := make([]string, 0, len(t.Values))
	for _, v := range t.Values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return specutil.NewCol("", "enum", specutil.ListAttr("values", quoted...)), nil
}

func bitSpec(t *BitType) (*sqlspec.Column, error) {
	var c *sqlspec.Column
	switch t.T {
	case tBit:
		if t.Len == 1 {
			c = &sqlspec.Column{TypeName: tBit}
		} else {
			c = specutil.NewCol("", fmt.Sprintf("%s(%d)", tBit, t.Len))
		}
		return c, nil
	case tBitVar:
		if t.Len == 0 {
			c = &sqlspec.Column{TypeName: tBitVar}
		} else {
			c = specutil.NewCol("", fmt.Sprintf("%s(%d)", tBitVar, t.Len))
		}
		return c, nil
	default:
		return nil, errors.New("schema bit failed to convert")
	}
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
