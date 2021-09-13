package sqlspec

import "ariga.io/atlas/schema/schemaspec"

type (
	File struct {
		Schemas []*Schema `spec:"schema"`
		Tables  []*Table  `spec:"table"`
		schemaspec.Resource
	}

	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:",name"`
		schemaspec.Resource
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
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name     string                   `spec:",name"`
		Null     bool                     `spec:"null" override:"null"`
		TypeName string                   `spec:"type" override:"type"`
		Default  *schemaspec.LiteralValue `spec:"default" override:"default"`
		//Overrides []*Override
		schemaspec.Resource
	}
)

func (*File) Type() string {
	return "file"
}

func (*Schema) Type() string {
	return "schema"
}

func (*Table) Type() string {
	return "table"
}

func (*Column) Type() string {
	return "column"
}
