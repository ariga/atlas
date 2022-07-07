// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"ariga.io/atlas/schemahcl"
	"github.com/hashicorp/hcl/v2/hclparse"
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

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name    string          `spec:",name"`
		Null    bool            `spec:"null"`
		Type    *schemahcl.Type `spec:"type"`
		Default schemahcl.Value `spec:"default"`
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

	// Evaluator is the interface that wraps the Eval function.
	Evaluator interface {
		// Eval evaluates parsed HCL files using input variables into a schema.Realm.
		Eval(*hclparse.Parser, interface{}, map[string]string) error
	}

	// EvalFunc is an adapter that allows the use of an ordinary function as an Evaluator.
	EvalFunc func(*hclparse.Parser, interface{}, map[string]string) error
)

// Eval implements the Evaluator interface.
func (f EvalFunc) Eval(p *hclparse.Parser, i interface{}, input map[string]string) error {
	return f(p, i, input)
}

func init() {
	schemahcl.Register("table", &Table{})
	schemahcl.Register("schema", &Schema{})
}
