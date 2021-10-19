package mysql

import (
	"fmt"
	"math"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// UnmarshalSpec unmarshals an Atlas DDL document using an unmarshaler into v.
func UnmarshalSpec(data []byte, unmarshaler schemaspec.Unmarshaler, v interface{}) error {
	var doc struct {
		Tables  []*sqlspec.Table  `spec:"table"`
		Schemas []*sqlspec.Schema `spec:"schema"`
	}
	if err := unmarshaler.UnmarshalSpec(data, &doc); err != nil {
		return err
	}
	if v, ok := v.(*schema.Schema); ok {
		if len(doc.Schemas) != 1 {
			return fmt.Errorf("mysql: expecting document to contain a single schema, got %d", len(doc.Schemas))
		}
		conv, err := specutil.Schema(doc.Schemas[0], doc.Tables, convertTable)
		if err != nil {
			return fmt.Errorf("mysql: failed converting to *schema.Schema: %w", err)
		}
		*v = *conv
		return nil
	}
	return fmt.Errorf("mysql: failed unmarshaling spec. %T is not supported", v)
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

// convertColumnType converts a sqlspec.Column into a concrete MySQL schema.Type.
func convertColumnType(spec *sqlspec.Column) (schema.Type, error) {
	switch sqlspec.Type(spec.TypeName) {
	case sqlspec.TypeInt, sqlspec.TypeInt8, sqlspec.TypeInt16,
		sqlspec.TypeInt64, sqlspec.TypeUint, sqlspec.TypeUint8,
		sqlspec.TypeUint16, sqlspec.TypeUint64:
		return nconvertInteger(spec)
	case sqlspec.TypeString:
		return nconvertString(spec)
	case sqlspec.TypeBinary:
		return nconvertBinary(spec)
	case sqlspec.TypeEnum:
		return nconvertEnum(spec)
	case sqlspec.TypeBoolean:
		return nconvertBoolean(spec)
	case sqlspec.TypeDecimal:
		return nconvertDecimal(spec)
	case sqlspec.TypeFloat:
		return nconvertFloat(spec)
	case sqlspec.TypeTime:
		return nconvertTime(spec)
	}
	return parseRawType(spec.TypeName)
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertInteger(spec *sqlspec.Column) (schema.Type, error) {
	typ := &schema.IntegerType{
		Unsigned: strings.HasPrefix(spec.TypeName, "u"),
	}
	switch spec.TypeName {
	case "int8", "uint8":
		typ.T = tTinyInt
	case "int16", "uint16":
		typ.T = tSmallInt
	case "int32", "uint32", "int", "integer", "uint":
		typ.T = tInt
	case "int64", "uint64":
		typ.T = tBigInt
	default:
		return nil, fmt.Errorf("mysql: unknown integer column type %q", spec.TypeName)
	}
	return typ, nil
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertBinary(spec *sqlspec.Column) (schema.Type, error) {
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

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertString(spec *sqlspec.Column) (schema.Type, error) {
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

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertEnum(spec *sqlspec.Column) (schema.Type, error) {
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

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertBoolean(_ *sqlspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertTime(_ *sqlspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "timestamp"}, nil
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertDecimal(spec *sqlspec.Column) (schema.Type, error) {
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

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertFloat(spec *sqlspec.Column) (schema.Type, error) {
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
