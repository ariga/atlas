package sqlite

import (
	"fmt"
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
	if v, ok := v.(*schema.Schema); ok {
		if len(d.Schemas) != 1 {
			return fmt.Errorf("mysql: expecting document to contain a single schema, got %d", len(d.Schemas))
		}
		conv, err := specutil.Schema(d.Schemas[0], d.Tables, convertTable)
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

func nconvertInteger(spec *sqlspec.Column) (schema.Type, error) {
	if strings.HasPrefix(spec.TypeName, "u") {
		// todo(rotemtam): support his once we can express CHECK(col >= 0)
		return nil, fmt.Errorf("sqlite: unsigned integers currently not supported")
	}
	typ := &schema.IntegerType{
		T: "integer",
	}
	return typ, nil
}

func nconvertBinary(spec *sqlspec.Column) (schema.Type, error) {
	bt := &schema.BinaryType{
		T: "blob",
	}
	return bt, nil
}

func nconvertString(spec *sqlspec.Column) (schema.Type, error) {
	st := &schema.StringType{
		T: "text",
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

func nconvertEnum(spec *sqlspec.Column) (schema.Type, error) {
	// sqlite does not have a enum column type
	return &schema.StringType{T: "text"}, nil
}

func nconvertBoolean(spec *sqlspec.Column) (schema.Type, error) {
	return &schema.BoolType{T: "boolean"}, nil
}

func nconvertTime(spec *sqlspec.Column) (schema.Type, error) {
	return &schema.TimeType{T: "datetime"}, nil
}

func nconvertDecimal(spec *sqlspec.Column) (schema.Type, error) {
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

func nconvertFloat(spec *sqlspec.Column) (schema.Type, error) {
	ft := &schema.FloatType{
		T: "real",
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
