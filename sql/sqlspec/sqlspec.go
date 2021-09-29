package sqlspec

import "ariga.io/atlas/schema/schemaspec"

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
		Columns *schemaspec.Ref `spec:"columns"`
		schemaspec.DefaultExtension
	}
)
