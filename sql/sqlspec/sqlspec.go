package sqlspec

import "ariga.io/atlas/schema/schemaspec"

type (

	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:",name"`
		schemaspec.Resource
		schemaspec.Extension
	}

	// Table holds a specification for an SQL table.
	Table struct {
		Name       string    `spec:",name"`
		SchemaName string    `spec:"schema"`
		Columns    []*Column `spec:"column"`
		//PrimaryKey  *PrimaryKey
		//ForeignKeys []*ForeignKey
		//Indexes     []*Index
		schemaspec.Resource
		schemaspec.Extension
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name     string                   `spec:",name"`
		Null     bool                     `spec:"null" override:"null"`
		TypeName string                   `spec:"type" override:"type"`
		Default  *schemaspec.LiteralValue `spec:"default" override:"default"`
		//Overrides []*Override
		schemaspec.Resource
		schemaspec.Extension
	}
)
