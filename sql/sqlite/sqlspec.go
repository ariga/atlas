package sqlite

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
	switch sqlspec.Type(spec.TypeName) {
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
	default:
		return parseRawType(spec.TypeName)
	}
}

func convertInteger(spec *sqlspec.Column) (schema.Type, error) {
	if strings.HasPrefix(spec.TypeName, "u") {
		// todo(rotemtam): support his once we can express CHECK(col >= 0)
		return nil, fmt.Errorf("sqlite: unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{
		T: tInteger,
	}
	return typ, nil
}

func convertBinary(spec *sqlspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{
		T: tBlob,
	}
	return bt, nil
}

func convertString(spec *sqlspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		T: tText,
	}
	if attr, ok := spec.Attr("size"); ok {
		s, err := attr.Int()
		if err != nil {
			return nil, err
		}
		st.Size = s
	}
	return st, nil
}

func convertEnum(spec *sqlspec.Column) (schema.Type, error) {
	// sqlite does not have a enum column type
	return &schema.StringType{T: "text"}, nil
}

func convertBoolean(spec *sqlspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func convertTime(spec *sqlspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "datetime"}, nil
}

func convertDecimal(spec *sqlspec.Column) (schema.Type, error) {
	dt := &schema.DecimalType{
		T: "decimal",
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
	return ft, nil
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
		return nil, fmt.Errorf("sqlite: failed to convert column type %T to spec", t)
	}
}

func binarySpec(t *schema.BinaryType) (*sqlspec.Column, error) {
	if t.T != tBlob {
		return nil, fmt.Errorf("sqlite/spec/binary: failed to convert type %q", t.T)
	}
	return &sqlspec.Column{TypeName: "binary"}, nil
}

func stringSpec(t *schema.StringType) (*sqlspec.Column, error) {
	if t.T != tText {
		return nil, fmt.Errorf("sqlite/spec/string: failed to convert type %q", t.T)
	}
	s := strconv.Itoa(t.Size)
	return specutil.NewCol("", "string", specutil.LitAttr("size", s)), nil
}

func integerSpec(t *schema.IntegerType) (*sqlspec.Column, error) {
	if t.Unsigned {
		return nil, errors.New("sqlite/spec/integer: unsigned integers currently not supported")
	}
	if t.T != tInteger {
		return nil, fmt.Errorf("sqlite/spec/integer: failed to convert type %q", t.T)
	}
	return &sqlspec.Column{TypeName: "int"}, nil
}

func enumSpec(t *schema.EnumType) (*sqlspec.Column, error) {
	if len(t.Values) == 0 {
		return nil, errors.New("sqlite/spec/enum: schema enum fields to have values")
	}
	quoted := make([]string, 0, len(t.Values))
	for _, v := range t.Values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return specutil.NewCol("", "enum", specutil.ListAttr("values", quoted...)), nil
}
