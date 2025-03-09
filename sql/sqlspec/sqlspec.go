// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"bytes"
	"strings"

	"ariga.io/atlas/schemahcl"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type (
	// Schema holds a specification for a Schema.
	Schema struct {
		Name string `spec:"name,name"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
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
		Range *hcl.Range `spec:",range"`
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
		Range *hcl.Range `spec:",range"`
	}

	// Column holds a specification for a column in an SQL table.
	Column struct {
		Name string          `spec:",name"`
		Null bool            `spec:"null"`
		Type *schemahcl.Type `spec:"type"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// PrimaryKey holds a specification for the primary key of a table.
	PrimaryKey struct {
		Name    string           `spec:",name"` // Optional name.
		Parts   []*IndexPart     `spec:"on"`
		Columns []*schemahcl.Ref `spec:"columns"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// Index holds a specification for the index key of a table.
	Index struct {
		Name    string           `spec:",name"`
		Unique  bool             `spec:"unique,omitempty"`
		Parts   []*IndexPart     `spec:"on"`
		Columns []*schemahcl.Ref `spec:"columns"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// IndexPart holds a specification for the index key part.
	IndexPart struct {
		Desc   bool           `spec:"desc,omitempty"`
		Column *schemahcl.Ref `spec:"column"`
		Expr   string         `spec:"expr,omitempty"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// Check holds a specification for a check constraint on a table.
	Check struct {
		Name string `spec:",name"`
		Expr string `spec:"expr"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// ForeignKey holds a specification for the Foreign key of a table.
	ForeignKey struct {
		Symbol     string           `spec:",name"`
		Columns    []*schemahcl.Ref `spec:"columns"`
		RefColumns []*schemahcl.Ref `spec:"ref_columns"`
		OnUpdate   *schemahcl.Ref   `spec:"on_update"`
		OnDelete   *schemahcl.Ref   `spec:"on_delete"`
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// Func holds the specification for a function.
	Func struct {
		Name      string         `spec:",name"`
		Qualifier string         `spec:",qualifier"`
		Schema    *schemahcl.Ref `spec:"schema"`
		Args      []*FuncArg     `spec:"arg"`
		Lang      cty.Value      `spec:"lang"`
		// The definition and the return type are appended as additional
		// attribute by the spec creator to marshal it after the arguments.
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// FuncArg holds the specification for a function argument.
	FuncArg struct {
		Name    string          `spec:",name"`
		Type    *schemahcl.Type `spec:"type"`
		Default cty.Value       `spec:"default"`
		// Optional attributes such as mode are added by the driver,
		// as their definition can be either a string or an enum (ref).
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// Trigger holds the specification for a trigger.
	Trigger struct {
		Name string         `spec:",name"`
		On   *schemahcl.Ref `spec:"on"` // A table or a view.
		// Attributes and blocks are different for each driver.
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}

	// Sequence holds a specification for a Sequence.
	Sequence struct {
		Name      string         `spec:",name"`
		Qualifier string         `spec:",qualifier"`
		Schema    *schemahcl.Ref `spec:"schema"`
		// Type, Start, Increment, Min, Max, Cache, Cycle
		// are optionally added to the sequence definition.
		schemahcl.DefaultExtension
		Range *hcl.Range `spec:",range"`
	}
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

// QualifierLabel returns the qualifier label used for the view resource, if any.
func (v *View) QualifierLabel() string { return v.Qualifier }

// SetQualifier sets the qualifier label used for the view resource.
func (v *View) SetQualifier(q string) { v.Qualifier = q }

// SchemaRef returns the schema reference for the view.
func (v *View) SchemaRef() *schemahcl.Ref { return v.Schema }

// Label returns the defaults label used for the function resource.
func (f *Func) Label() string { return f.Name }

// QualifierLabel returns the qualifier label used for the function resource, if any.
func (f *Func) QualifierLabel() string { return f.Qualifier }

// SetQualifier sets the qualifier label used for the function resource.
func (f *Func) SetQualifier(q string) { f.Qualifier = q }

// SchemaRef returns the schema reference for the function.
func (f *Func) SchemaRef() *schemahcl.Ref { return f.Schema }

// Label returns the defaults label used for the sequence resource.
func (s *Sequence) Label() string { return s.Name }

// QualifierLabel returns the qualifier label used for the sequence resource, if any.
func (s *Sequence) QualifierLabel() string { return s.Qualifier }

// SetQualifier sets the qualifier label used for the sequence resource.
func (s *Sequence) SetQualifier(q string) { s.Qualifier = q }

// SchemaRef returns the schema reference for the sequence.
func (s *Sequence) SchemaRef() *schemahcl.Ref { return s.Schema }

func init() {
	schemahcl.Register("view", &View{})
	schemahcl.Register("materialized", &View{})
	schemahcl.Register("table", &Table{})
	schemahcl.Register("function", &Func{})
	schemahcl.Register("procedure", &Func{})
	schemahcl.Register("trigger", &Trigger{})
	schemahcl.Register("sequence", &Sequence{})
	schemahcl.Register("schema", &Schema{})
}

// normalizeCRLF for heredoc strings that inspected and printed in the HCL as-is to
// avoid having mixed endings in the printed file - Unix-like (default) and Windows-like.
func normalizeCRLF(s string) string {
	if strings.Contains(s, "\r\n") {
		return strings.ReplaceAll(s, "\r\n", "\n")
	}
	return s
}

// MightHeredoc returns the string as an indented heredoc if it has multiple lines.
func MightHeredoc(s string) string {
	s = normalizeCRLF(strings.TrimSpace(s))
	// In case the given definition is multi-line,
	// format it as indented heredoc with two spaces.
	if lines := strings.Split(s, "\n"); len(lines) > 1 {
		var b bytes.Buffer
		b.Grow(len(s))
		b.WriteString("<<-SQL\n")
		for _, l := range lines {
			// Skip spaces-only lines, as editors stripped these spaces off,
			// and HCL parser results a different string for them.
			if strings.TrimSpace(l) != "" {
				b.WriteString("  ")
				b.WriteString(l)
			}
			b.WriteByte('\n')
		}
		b.WriteString("  SQL")
		s = b.String()
	}
	return s
}
