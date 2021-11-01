package postgres

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		return nconvertInteger(spec)
	case sqlspec.TypeString:
		return nconvertString(spec)
	case sqlspec.TypeEnum:
		return nconvertEnum(spec)
	case sqlspec.TypeDecimal:
		return nconvertDecimal(spec)
	case sqlspec.TypeFloat:
		return nconvertFloat(spec)
	case sqlspec.TypeTime:
		return &schema.TimeType{T: tTimestamp}, nil
	case sqlspec.TypeBinary:
		return &schema.BinaryType{T: tBytea}, nil
	case sqlspec.TypeBoolean:
		return &schema.BoolType{T: tBoolean}, nil
	default:
		return nparseRawType(spec)
	}
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertInteger(spec *sqlspec.Column) (schema.Type, error) {
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
	case st.Size < maxCharSize:
		st.T = tVarChar
	default:
		st.T = tText
	}
	return st, nil
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nconvertEnum(spec *sqlspec.Column) (schema.Type, error) {
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

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nparseRawType(spec *sqlspec.Column) (schema.Type, error) {
	cm, err := parseColumn(spec.TypeName)
	if err != nil {
		return nil, err
	}
	return columnType(cm), nil
}

// schemaSpec converts from a concrete MySQL schema to Atlas specification.
func schemaSpec(schem *schema.Schema) (*sqlspec.Schema, []*sqlspec.Table, error) {
	return specutil.FromSchema(schem, tableSpec)
}

// tableSpec converts from a concrete MySQL sqlspec.Table to a schema.Table.
func tableSpec(tab *schema.Table) (*sqlspec.Table, error) {
	return specutil.FromTable(tab, columnSpec, specutil.FromPrimaryKey, specutil.FromIndex, specutil.FromForeignKey)
}

// columnSpec converts from a concrete MySQL schema.Column into a sqlspec.Column.
func columnSpec(col *schema.Column) (*sqlspec.Column, error) {
	ct, err := ncolumnTypeSpec(col.Type.Type)
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

// columnTypeSpec converts from a concrete MySQL schema.Type into sqlspec.Column Type.
func ncolumnTypeSpec(t schema.Type) (*sqlspec.Column, error) {
	switch t := t.(type) {
	case *schema.EnumType:
		return nenumSpec(t)
	case *schema.IntegerType:
		return nintegerSpec(t)
	case *schema.StringType:
		return nstringSpec(t)
	case *schema.DecimalType:
		precision := specutil.LitAttr("precision", strconv.Itoa(t.Precision))
		scale := specutil.LitAttr("scale", strconv.Itoa(t.Scale))
		return specutil.NewCol("", "decimal", precision, scale), nil
	case *schema.BinaryType:
		return nbinarySpec(t)
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
	default:
		return nil, fmt.Errorf("mysql: failed to convert column type %T to spec", t)
	}
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nbinarySpec(t *schema.BinaryType) (*sqlspec.Column, error) {
	//switch t.T {
	//case tBlob:
	//	return &sqlspec.Column{TypeName: "binary"}, nil
	//case tTinyBlob, tMediumBlob, tLongBlob:
	//	size := specutil.LitAttr("size", strconv.Itoa(t.Size))
	//	return specutil.NewCol("", "binary", size), nil
	//}
	return nil, errors.New("mysql: schema binary failed to convert")
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nstringSpec(t *schema.StringType) (*sqlspec.Column, error) {
	//switch t.T {
	//case tVarchar, tMediumText, tLongText:
	//	s := strconv.Itoa(t.Size)
	//	return specutil.NewCol("", "string", specutil.LitAttr("size", s)), nil
	//}
	return nil, errors.New("mysql: schema string failed to convert")
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nintegerSpec(t *schema.IntegerType) (*sqlspec.Column, error) {
	switch t.T {
	case tInt:
		if t.Unsigned {
			return specutil.NewCol("", "uint"), nil
		}
		return &sqlspec.Column{TypeName: "int"}, nil
	//case tTinyInt:
	//	return &sqlspec.Column{TypeName: "int8"}, nil
	case tBigInt:
		if t.Unsigned {
			return specutil.NewCol("", "uint64"), nil
		}
		return &sqlspec.Column{TypeName: "int64"}, nil
	}
	return nil, errors.New("mysql: schema integer failed to convert")
}

// temporarily prefixed with "n" until we complete the refactor of replacing sql/schemaspec with sqlspec.
func nenumSpec(t *schema.EnumType) (*sqlspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("mysql: schema enum fields to have values")
	}
	quoted := make([]string, 0, len(t.Values))
	for _, v := range t.Values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return specutil.NewCol("", "enum", specutil.ListAttr("values", quoted...)), nil
}
