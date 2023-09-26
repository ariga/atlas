// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"ariga.io/atlas/schemahcl"

	"github.com/zclconf/go-cty/cty"
)

type (
	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:"name,name"`
		schemahcl.DefaultExtension
	}

	// Table holds a specification for an SQL table.
	Table struct {
		Name        string         `spec:",name"`
		Qualifier   string         `spec:",qualifier"`
		Schema      *schemahcl.Ref `spec:"schema"`
		Columns     []*Column      `spec:"column"`
		PrimaryKey  *PrimaryKey    `spec:"primary_key"`
		ForeignKeys []*ForeignKey  `spec:"foreign_key"`
		Indexes     []*Index       `spec:"index"`
		Checks      []*Check       `spec:"check"`
		schemahcl.DefaultExtension
	}

	// View holds a specification for an SQL view.
	View struct {
		Name      string         `spec:",name"`
		Qualifier string         `spec:",qualifier"`
		Schema    *schemahcl.Ref `spec:"schema"`
		Columns   []*Column      `spec:"column"`
		// Indexes on (materialized) views are supported
		// by some databases, like PostgreSQL.
		Indexes []*Index `spec:"index"`
		// The definition is appended as additional attribute
		// by the spec creator to marshal it after the columns.
		schemahcl.DefaultExtension
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name    string          `spec:",name"`
		Null    bool            `spec:"null"`
		Type    *schemahcl.Type `spec:"type"`
		Default cty.Value       `spec:"default"`
		schemahcl.DefaultExtension
	}

	// PrimaryKey holds a specification for the primary key of a table.
	PrimaryKey struct {
		Columns []*schemahcl.Ref `spec:"columns"`
		schemahcl.DefaultExtension
	}

	// Index holds a specification for the index key of a table.
	Index struct {
		Name    string           `spec:",name"`
		Unique  bool             `spec:"unique,omitempty"`
		Parts   []*IndexPart     `spec:"on"`
		Columns []*schemahcl.Ref `spec:"columns"`
		schemahcl.DefaultExtension
	}

	// IndexPart holds a specification for the index key part.
	IndexPart struct {
		Desc   bool           `spec:"desc,omitempty"`
		Column *schemahcl.Ref `spec:"column"`
		Expr   string         `spec:"expr,omitempty"`
		schemahcl.DefaultExtension
	}

	// Check holds a specification for a check constraint on a table.
	Check struct {
		Name string `spec:",name"`
		Expr string `spec:"expr"`
		schemahcl.DefaultExtension
	}

	// ForeignKey holds a specification for the Foreign key of a table.
	ForeignKey struct {
		Symbol     string           `spec:",name"`
		Columns    []*schemahcl.Ref `spec:"columns"`
		RefColumns []*schemahcl.Ref `spec:"ref_columns"`
		OnUpdate   *schemahcl.Ref   `spec:"on_update"`
		OnDelete   *schemahcl.Ref   `spec:"on_delete"`
		schemahcl.DefaultExtension
	}

	// Type represents a database agnostic column type.
	Type string
)

// Label returns the defaults label used for the table resource.
func (t *Table) Label() string { return t.Name }

// QualifierLabel returns the qualifier label used for the table resource, if any.
func (t *Table) QualifierLabel() string { return t.Qualifier }

// SetQualifier sets the qualifier label used for the table resource.
func (t *Table) SetQualifier(q string) { t.Qualifier = q }

// SchemaRef returns the schema reference for the table.
func (t *Table) SchemaRef() *schemahcl.Ref { return t.Schema }

// Label returns the defaults label used for the view resource.
func (v *View) Label() string { return v.Name }

// QualifierLabel returns the qualifier label used for the table resource, if any.
func (v *View) QualifierLabel() string { return v.Qualifier }

// SetQualifier sets the qualifier label used for the view resource.
func (v *View) SetQualifier(q string) { v.Qualifier = q }

// SchemaRef returns the schema reference for the view.
func (v *View) SchemaRef() *schemahcl.Ref { return v.Schema }

func init() {
	schemahcl.Register("view", &View{})
	schemahcl.Register("materialized", &View{})
	schemahcl.Register("table", &Table{})
	schemahcl.Register("schema", &Schema{})
}
