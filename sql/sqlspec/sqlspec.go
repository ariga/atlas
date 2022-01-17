package sqlspec

import (
	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/schema"
)

type (

	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:",name"`
		schemaspec.DefaultExtension
	}

	// Table holds a specification for an SQL table.
	Table struct {
		Name        string          `spec:",name"`
		Schema      *schemaspec.Ref `spec:"schema"`
		Columns     []*Column       `spec:"column"`
		PrimaryKey  *PrimaryKey     `spec:"primary_key"`
		ForeignKeys []*ForeignKey   `spec:"foreign_key"`
		Indexes     []*Index        `spec:"index"`
		Checks      []*Check        `spec:"check"`
		schemaspec.DefaultExtension
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name    string           `spec:",name"`
		Null    bool             `spec:"null"`
		Type    *schemaspec.Type `spec:"type"`
		Default schemaspec.Value `spec:"default"`
		schemaspec.DefaultExtension
	}

	// PrimaryKey holds a specification for the primary key of a table.
	PrimaryKey struct {
		Columns []*schemaspec.Ref `spec:"columns"`
		schemaspec.DefaultExtension
	}

	// Index holds a specification for the index key of a table.
	Index struct {
		Name    string            `spec:",name"`
		Unique  bool              `spec:"unique"`
		Columns []*schemaspec.Ref `spec:"columns"`
		schemaspec.DefaultExtension
	}

	// Check holds a specification for a check constraint on a table.
	Check struct {
		Name string `spec:",name"`
		Expr string `spec:"expr"`
		schemaspec.DefaultExtension
	}

	// ForeignKey holds a specification for the Foreign key of a table.
	ForeignKey struct {
		Symbol     string                 `spec:",name"`
		Columns    []*schemaspec.Ref      `spec:"columns"`
		RefColumns []*schemaspec.Ref      `spec:"ref_columns"`
		OnUpdate   schema.ReferenceOption `spec:"on_update"`
		OnDelete   schema.ReferenceOption `spec:"on_delete"`
		schemaspec.DefaultExtension
	}

	// Type represents a database agnostic column type.
	Type string
)

// Type<X> are Types that represent database agnostic column types to be used
// in Atlas DDL documents.
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

func init() {
	schemaspec.Register("table", &Table{})
	schemaspec.Register("schema", &Schema{})
}
