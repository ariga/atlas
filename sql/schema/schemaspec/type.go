package schemaspec

// Type represents a database agnostic column type.
type Type string

const (
	TypeInt     Type = "int"
	TypeInt8    Type = "int8"
	TypeInt16   Type = "int16"
	TypeInt64   Type = "int64"
	TypeUint    Type = "uint"
	TypeUint8   Type = "uint8"
	TypeUint16  Type = "uint16"
	TypeUint64  Type = "uint64"
	TypeString  Type = "string"
	TypeBinary  Type = "binary"
	TypeEnum    Type = "enum"
	TypeBoolean Type = "boolean"
	TypeDecimal Type = "decimal"
	TypeFloat   Type = "float"
	TypeTime    Type = "time"
)
