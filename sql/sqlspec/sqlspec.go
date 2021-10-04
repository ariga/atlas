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
		Name        string        `spec:",name"`
		SchemaName  string        `spec:"schema"`
		Columns     []*Column     `spec:"column"`
		PrimaryKey  *PrimaryKey   `spec:"primary_key"`
		ForeignKeys []*ForeignKey `spec:"foreign_key"`
		Indexes     []*Index      `spec:"index"`
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name      string                   `spec:",name"`
		Null      bool                     `spec:"null" override:"null"`
		TypeName  string                   `spec:"type" override:"type"`
		Default   *schemaspec.LiteralValue `spec:"default" override:"default"`
		Overrides []*Override
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

	// ForeignKey holds a specification for the Foreign key of a table.
	ForeignKey struct {
		Symbol     string                 `spec:",name"`
		Columns    []*schemaspec.Ref      `spec:"columns"`
		RefColumns []*schemaspec.Ref      `spec:"ref_columns"`
		OnUpdate   schema.ReferenceOption `spec:"on_update"`
		OnDelete   schema.ReferenceOption `spec:"on_delete"`
		schemaspec.DefaultExtension
	}

	// Override contains information about how to override some attributes of an Element
	// for a specific dialect and version. For example, to select a specific column type or add
	// special attributes when using MySQL, but not when using SQLite or Postgres.
	Override struct {
		Dialect string `spec:"dialect"`
		Version string `spec:"version"`
		schemaspec.DefaultExtension
	}
)

// Override searches the Column's Overrides for ones matching any of the versions
// passed to it. It then creates an Override by merging the overrides for all of
// the matching versions in the order they were passed.
func (c *Column) Override(versions ...string) *Override {
	var override *Override
	for _, version := range versions {
		for _, o := range c.Overrides {
			if o.version() == version {
				if override == nil {
					override = o
				}
				for _, a := range o.Extra.Attrs {
					override.Extra.SetAttr(a)
				}
			}
		}
	}
	return override
}

func (o *Override) version() string {
	v := o.Dialect
	if o.Version != "" {
		v += " " + o.Version
	}
	return v
}