package mysql

import (
	"fmt"

	"ariga.io/atlas/sql/schema"
)

// Concretizer implements schema.Concretizer.
type Concretizer struct {
	// current driver MySQL version
	version string
}

func (c *Concretizer) Concretize(column *schema.Column) error {
	switch t := column.Type.Type.(type) {
	case *schema.IntegerType:
		typeName, err := byteSizeIntType(t.Size)
		if err != nil {
			return err
		}
		t.T = typeName
		raw := typeName
		if t.Unsigned {
			raw += " unsigned"
		}
		column.Type.Raw = raw
	case *schema.StringType:
		if t.Size == 0 {
			t.Size = 255
		}
		typeName, err := byteSizeStringType(t.Size)
		if err != nil {
			return err
		}
		t.T = typeName
		raw := typeName
		if typeName == "varchar" {
			raw = fmt.Sprintf("varchar(%d)", t.Size)
		}
		column.Type.Raw = raw
	case *schema.BoolType:
		t.T = "boolean"
		column.Type.Raw = "boolean"
	// TODO(rotemtam): implement the rest
	//case *schema.BinaryType:
	//case *schema.EnumType:
	//case *schema.TimeType:
	//case *schema.FloatType:
	//case *schema.DecimalType:
	//case *schema.JSONType:
	//case *schema.SpatialType:
	default:
		return fmt.Errorf("mysql: unsupported column type %T", t)
	}
	return nil
}

func byteSizeIntType(s uint8) (string, error) {
	switch s {
	case 1:
		return tTinyInt, nil
	case 2:
		return tSmallInt, nil
	case 3:
		return tMediumInt, nil
	case 4:
		return tInt, nil
	case 8:
		return tBigInt, nil
	default:
		return "", fmt.Errorf("mysql: cannot map int of size %d bytes to a known column type", s)
	}
}

func byteSizeStringType(s int) (string, error) {
	var (
		size64KB int = 65535
		size16MB     = 16777215
		size4GB      = 4294967295
	)
	switch {
	case s <= size64KB:
		return "varchar", nil
	case s > size64KB && s <= size16MB:
		return "mediumtext", nil
	case s > size16MB && s <= size4GB:
		return "longtext", nil
	default:
		return "", fmt.Errorf("mysql: string fields can be up to 4GB long")
	}
}
