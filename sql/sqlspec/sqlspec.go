package sqlspec

import (
	"fmt"

	"ariga.io/atlas/schema/schemaspec"
)

type (

	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:",name"`
		schemaspec.DefaultExtension
	}

	// Table holds a specification for an SQL table.
	Table struct {
		Name       string      `spec:",name"`
		SchemaName string      `spec:"schema"`
		Columns    []*Column   `spec:"column"`
		PrimaryKey *PrimaryKey `spec:"primary_key"`
		//ForeignKeys []*ForeignKey
		//Indexes     []*Index
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name     string                   `spec:",name"`
		Null     bool                     `spec:"null" override:"null"`
		TypeName string                   `spec:"type" override:"type"`
		Default  *schemaspec.LiteralValue `spec:"default" override:"default"`
		//Overrides []*Override
	}

	// PrimaryKey holds a specification for the primary key of a table.
	PrimaryKey struct {
		Columns []*ColumnRef `spec:"columns"`
		schemaspec.DefaultExtension
	}

	// ColumnRef is a reference to a Column described in another spec.
	ColumnRef struct {
		Name  string
		Table string
	}
)

func (c *ColumnRef) Scan(v schemaspec.Value) error {
	if ref, ok := v.(*schemaspec.Ref); ok {
		c.Table = ref.V
		c.Name = ref.V
	}
	return fmt.Errorf("sqlspec: ColumnRef expected value to be Ref")
}

func (c *ColumnRef) Value() schemaspec.Value {
	return &schemaspec.Ref{V: fmt.Sprintf("$table.%s.column.%s", c.Table, c.Name)}
}
