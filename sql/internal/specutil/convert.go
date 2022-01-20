// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package specutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

// List of convert function types.
type (
	ConvertTableFunc      func(*sqlspec.Table, *schema.Schema) (*schema.Table, error)
	ConvertColumnFunc     func(*sqlspec.Column, *schema.Table) (*schema.Column, error)
	ConvertTypeFunc       func(*sqlspec.Column) (schema.Type, error)
	ConvertPrimaryKeyFunc func(*sqlspec.PrimaryKey, *schema.Table) (*schema.Index, error)
	ConvertIndexFunc      func(*sqlspec.Index, *schema.Table) (*schema.Index, error)
	ConvertCheckFunc      func(*sqlspec.Check) (*schema.Check, error)
	ColumnSpecFunc        func(*schema.Column, *schema.Table) (*sqlspec.Column, error)
	ColumnTypeSpecFunc    func(schema.Type) (*sqlspec.Column, error)
	TableSpecFunc         func(*schema.Table) (*sqlspec.Table, error)
	PrimaryKeySpecFunc    func(*schema.Index) (*sqlspec.PrimaryKey, error)
	IndexSpecFunc         func(*schema.Index) (*sqlspec.Index, error)
	ForeignKeySpecFunc    func(*schema.ForeignKey) (*sqlspec.ForeignKey, error)
	CheckSpecFunc         func(*schema.Check) *sqlspec.Check
)

// Realm converts the schemas and tables into a schema.Realm.
func Realm(schemas []*sqlspec.Schema, tables []*sqlspec.Table, convertTable ConvertTableFunc) (*schema.Realm, error) {
	r := &schema.Realm{}
	for _, schemaSpec := range schemas {
		var schemaTables []*sqlspec.Table
		for _, tableSpec := range tables {
			name, err := SchemaName(tableSpec.Schema)
			if err != nil {
				return nil, fmt.Errorf("specutil: cannot extract schema name for table %q: %w", tableSpec.Name, err)
			}
			if name == schemaSpec.Name {
				schemaTables = append(schemaTables, tableSpec)
			}
		}
		sch, err := Schema(schemaSpec, schemaTables, convertTable)
		if err != nil {
			return nil, err
		}
		r.Schemas = append(r.Schemas, sch)
	}
	return r, nil
}

// Schema converts a sqlspec.Schema with its relevant []sqlspec.Tables
// into a schema.Schema.
func Schema(spec *sqlspec.Schema, tables []*sqlspec.Table, convertTable ConvertTableFunc) (*schema.Schema, error) {
	sch := &schema.Schema{
		Name: spec.Name,
	}
	m := make(map[*schema.Table]*sqlspec.Table)
	for _, ts := range tables {
		table, err := convertTable(ts, sch)
		if err != nil {
			return nil, err
		}
		sch.Tables = append(sch.Tables, table)
		m[table] = ts
	}
	for _, tbl := range sch.Tables {
		if err := LinkForeignKeys(tbl, sch, m[tbl]); err != nil {
			return nil, err
		}
	}
	return sch, nil
}

// Table converts a sqlspec.Table to a schema.Table. Table conversion is done without converting
// ForeignKeySpecs into ForeignKeys, as the target tables do not necessarily exist in the schema
// at this point. Instead, the linking is done by the Schema function.
func Table(spec *sqlspec.Table, parent *schema.Schema, convertColumn ConvertColumnFunc,
	convertPk ConvertPrimaryKeyFunc, convertIndex ConvertIndexFunc, convertCheck ConvertCheckFunc) (*schema.Table, error) {
	tbl := &schema.Table{
		Name:   spec.Name,
		Schema: parent,
	}
	for _, csp := range spec.Columns {
		col, err := convertColumn(csp, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Columns = append(tbl.Columns, col)
	}
	if spec.PrimaryKey != nil {
		pk, err := convertPk(spec.PrimaryKey, tbl)
		if err != nil {
			return nil, err
		}
		tbl.PrimaryKey = pk
	}
	for _, idx := range spec.Indexes {
		i, err := convertIndex(idx, tbl)
		if err != nil {
			return nil, err
		}
		tbl.Indexes = append(tbl.Indexes, i)
	}
	for _, c := range spec.Checks {
		c, err := convertCheck(c)
		if err != nil {
			return nil, err
		}
		tbl.AddChecks(c)
	}
	if err := convertCommentFromSpec(spec, &tbl.Attrs); err != nil {
		return nil, err
	}
	return tbl, nil
}

// Column converts a sqlspec.Column into a schema.Column.
func Column(spec *sqlspec.Column, conv ConvertTypeFunc) (*schema.Column, error) {
	out := &schema.Column{
		Name: spec.Name,
		Type: &schema.ColumnType{
			Null: spec.Null,
		},
	}
	if spec.Default != nil {
		switch d := spec.Default.(type) {
		case *schemaspec.LiteralValue:
			out.Default = &schema.Literal{V: d.V}
		case *schemaspec.RawExpr:
			out.Default = &schema.RawExpr{X: d.X}
		default:
			return nil, fmt.Errorf("unsupported value type for default: %T", d)
		}
	}
	ct, err := conv(spec)
	if err != nil {
		return nil, err
	}
	out.Type.Type = ct
	if err := convertCommentFromSpec(spec, &out.Attrs); err != nil {
		return nil, err
	}
	return out, err
}

// Index converts a sqlspec.Index to a schema.Index.
func Index(spec *sqlspec.Index, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		cn, err := columnName(c)
		if err != nil {
			return nil, fmt.Errorf("specutil: failed converting column to index: %w", err)
		}
		col, ok := parent.Column(cn)
		if !ok {
			return nil, fmt.Errorf("specutil: unknown column %q in table %q", cn, parent.Name)
		}
		parts = append(parts, &schema.IndexPart{
			SeqNo: seqno,
			C:     col,
		})
	}
	i := &schema.Index{
		Name:   spec.Name,
		Unique: spec.Unique,
		Table:  parent,
		Parts:  parts,
	}
	if err := convertCommentFromSpec(spec, &i.Attrs); err != nil {
		return nil, err
	}
	return i, nil
}

// Check converts a sqlspec.Check to a schema.Check.
func Check(spec *sqlspec.Check) (*schema.Check, error) {
	return &schema.Check{
		Name: spec.Name,
		Expr: spec.Expr,
	}, nil
}

// PrimaryKey converts a sqlspec.PrimaryKey to a schema.Index.
func PrimaryKey(spec *sqlspec.PrimaryKey, parent *schema.Table) (*schema.Index, error) {
	parts := make([]*schema.IndexPart, 0, len(spec.Columns))
	for seqno, c := range spec.Columns {
		n, err := columnName(c)
		if err != nil {
			return nil, fmt.Errorf("sqlspec: cannot get column name %q as primary key for table %q", c.V, parent.Name)
		}
		pkc, ok := parent.Column(n)
		if !ok {
			return nil, fmt.Errorf("sqlspec: cannot set column %q as primary key for table %q", n, parent.Name)
		}
		parts = append(parts, &schema.IndexPart{
			SeqNo: seqno,
			C:     pkc,
		})
	}
	return &schema.Index{
		Table: parent,
		Parts: parts,
	}, nil
}

// LinkForeignKeys creates the foreign keys defined in the Table's spec by creating references
// to column in the provided Schema. It is assumed that the schema contains all of the tables
// referenced by the FK definitions in the spec.
func LinkForeignKeys(tbl *schema.Table, sch *schema.Schema, table *sqlspec.Table) error {
	for _, spec := range table.ForeignKeys {
		fk := &schema.ForeignKey{
			Symbol:   spec.Symbol,
			Table:    tbl,
			OnUpdate: spec.OnUpdate,
			OnDelete: spec.OnDelete,
		}
		for _, ref := range spec.Columns {
			col, err := resolveCol(ref, sch)
			if err != nil {
				return err
			}
			fk.Columns = append(fk.Columns, col)
		}
		if len(spec.RefColumns) == 0 {
			return fmt.Errorf("sqlspec: missing reference (parent) columns for foreign key: %q", spec.Symbol)
		}
		name, err := tableName(spec.RefColumns[0])
		if err != nil {
			return err
		}
		t, ok := sch.Table(name)
		if !ok {
			return fmt.Errorf("sqlspec: undefined table %q for foreign key: %q", name, spec.Symbol)
		}
		fk.RefTable = t
		for _, ref := range spec.RefColumns {
			col, err := resolveCol(ref, sch)
			if err != nil {
				return err
			}
			fk.RefColumns = append(fk.RefColumns, col)
		}
		tbl.ForeignKeys = append(tbl.ForeignKeys, fk)
	}
	return nil
}

func resolveCol(ref *schemaspec.Ref, sch *schema.Schema) (*schema.Column, error) {
	t, err := tableName(ref)
	if err != nil {
		return nil, fmt.Errorf("sqlspec: table %q not found", ref.V)
	}
	tbl, ok := sch.Table(t)
	if !ok {
		return nil, fmt.Errorf("sqlspec: table %q not found", t)
	}
	c, err := columnName(ref)
	if err != nil {
		return nil, fmt.Errorf("sqlspec: column %q not found", ref.V)
	}
	col, ok := tbl.Column(c)
	if !ok {
		return nil, fmt.Errorf("sqlspec: column %q not found in table %q", c, t)
	}
	return col, nil
}

// FromRealm converts a schema.Realm into []sqlspec.Schema and []sqlspec.Table.
func FromRealm(r *schema.Realm, fn TableSpecFunc) ([]*sqlspec.Schema, []*sqlspec.Table, error) {
	var (
		schemas []*sqlspec.Schema
		tables  []*sqlspec.Table
	)
	for _, sch := range r.Schemas {
		schemaSpec, tableSpecs, err := FromSchema(sch, fn)
		if err != nil {
			return nil, nil, fmt.Errorf("specutil: cannot convert schema %q: %w", sch.Name, err)
		}
		schemas = append(schemas, schemaSpec)
		tables = append(tables, tableSpecs...)
	}
	return schemas, tables, nil
}

// FromSchema converts a schema.Schema into sqlspec.Schema and []sqlspec.Table.
func FromSchema(s *schema.Schema, fn TableSpecFunc) (*sqlspec.Schema, []*sqlspec.Table, error) {
	spec := &sqlspec.Schema{
		Name: s.Name,
	}
	tables := make([]*sqlspec.Table, 0, len(s.Tables))
	for _, t := range s.Tables {
		table, err := fn(t)
		if err != nil {
			return nil, nil, err
		}
		if s.Name != "" {
			table.Schema = SchemaRef(s.Name)
		}
		tables = append(tables, table)
	}
	return spec, tables, nil
}

// FromTable converts a schema.Table to a sqlspec.Table.
func FromTable(t *schema.Table, colFn ColumnSpecFunc, pkFn PrimaryKeySpecFunc, idxFn IndexSpecFunc,
	fkFn ForeignKeySpecFunc, ckFn CheckSpecFunc) (*sqlspec.Table, error) {
	spec := &sqlspec.Table{
		Name: t.Name,
	}
	for _, c := range t.Columns {
		col, err := colFn(c, t)
		if err != nil {
			return nil, err
		}
		spec.Columns = append(spec.Columns, col)
	}
	if t.PrimaryKey != nil {
		pk, err := pkFn(t.PrimaryKey)
		if err != nil {
			return nil, err
		}
		spec.PrimaryKey = pk
	}
	for _, idx := range t.Indexes {
		i, err := idxFn(idx)
		if err != nil {
			return nil, err
		}
		spec.Indexes = append(spec.Indexes, i)
	}
	for _, fk := range t.ForeignKeys {
		f, err := fkFn(fk)
		if err != nil {
			return nil, err
		}
		spec.ForeignKeys = append(spec.ForeignKeys, f)
	}
	for _, attr := range t.Attrs {
		if c, ok := attr.(*schema.Check); ok {
			spec.Checks = append(spec.Checks, ckFn(c))
		}
	}
	convertCommentFromSchema(t.Attrs, &spec.Extra.Attrs)
	return spec, nil
}

// FromPrimaryKey converts schema.Index to a sqlspec.PrimaryKey.
func FromPrimaryKey(s *schema.Index) (*sqlspec.PrimaryKey, error) {
	c := make([]*schemaspec.Ref, 0, len(s.Parts))
	for _, v := range s.Parts {
		c = append(c, colRef(v.C.Name, s.Table.Name))
	}
	return &sqlspec.PrimaryKey{
		Columns: c,
	}, nil
}

// FromColumn converts a *schema.Column into a *sqlspec.Column using the ColumnTypeSpecFunc.
func FromColumn(col *schema.Column, columnTypeSpec ColumnTypeSpecFunc) (*sqlspec.Column, error) {
	ct, err := columnTypeSpec(col.Type.Type)
	if err != nil {
		return nil, err
	}
	spec := &sqlspec.Column{
		Name: col.Name,
		Type: ct.Type,
		Null: col.Type.Null,
		DefaultExtension: schemaspec.DefaultExtension{
			Extra: schemaspec.Resource{Attrs: ct.DefaultExtension.Extra.Attrs},
		},
	}
	if col.Default != nil {
		lv, err := toValue(col.Default)
		if err != nil {
			return nil, err
		}
		spec.Default = lv
	}
	convertCommentFromSchema(col.Attrs, &spec.Extra.Attrs)
	return spec, nil
}

func toValue(expr schema.Expr) (schemaspec.Value, error) {
	var (
		v   string
		err error
	)
	switch expr := expr.(type) {
	case *schema.RawExpr:
		return &schemaspec.RawExpr{X: expr.X}, nil
	case *schema.Literal:
		v, err = normalizeQuotes(expr.V)
		if err != nil {
			return nil, err
		}
		return &schemaspec.LiteralValue{V: v}, nil
	default:
		return nil, fmt.Errorf("converting expr %T to literal value", expr)
	}
}

func normalizeQuotes(s string) (string, error) {
	if len(s) < 2 {
		return s, nil
	}
	// If string is quoted with single quotes:
	if strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		uq := strings.ReplaceAll(s[1:len(s)-1], "''", "'")
		return strconv.Quote(uq), nil
	}
	return s, nil
}

// FromIndex converts schema.Index to sqlspec.Index.
func FromIndex(idx *schema.Index) (*sqlspec.Index, error) {
	c := make([]*schemaspec.Ref, 0, len(idx.Parts))
	for _, p := range idx.Parts {
		if p.C == nil {
			return nil, errors.New("index expression is not supported")
		}
		c = append(c, colRef(p.C.Name, idx.Table.Name))
	}
	i := &sqlspec.Index{
		Name:    idx.Name,
		Unique:  idx.Unique,
		Columns: c,
	}
	convertCommentFromSchema(idx.Attrs, &i.Extra.Attrs)
	return i, nil
}

// FromForeignKey converts schema.ForeignKey to sqlspec.ForeignKey.
func FromForeignKey(s *schema.ForeignKey) (*sqlspec.ForeignKey, error) {
	c := make([]*schemaspec.Ref, 0, len(s.Columns))
	for _, v := range s.Columns {
		c = append(c, colRef(v.Name, s.Table.Name))
	}
	r := make([]*schemaspec.Ref, 0, len(s.RefColumns))
	for _, v := range s.RefColumns {
		r = append(r, colRef(v.Name, s.RefTable.Name))
	}
	return &sqlspec.ForeignKey{
		Symbol:     s.Symbol,
		Columns:    c,
		RefColumns: r,
		OnDelete:   s.OnDelete,
		OnUpdate:   s.OnUpdate,
	}, nil
}

// FromCheck converts schema.Check to sqlspec.Check.
func FromCheck(s *schema.Check) *sqlspec.Check {
	return &sqlspec.Check{
		Name: s.Name,
		Expr: s.Expr,
	}
}

// SchemaName returns the name from a ref to a schema.
func SchemaName(ref *schemaspec.Ref) (string, error) {
	parts := strings.Split(ref.V, ".")
	if len(parts) < 2 || parts[0] != "$schema" {
		return "", fmt.Errorf("expected ref format of $schema.name")
	}
	return parts[1], nil
}

func columnName(ref *schemaspec.Ref) (string, error) {
	s := strings.Split(ref.V, "$column.")
	if len(s) != 2 {
		return "", fmt.Errorf("sqlspec: failed to extract column name from %q", ref)
	}
	return s[1], nil
}

func tableName(ref *schemaspec.Ref) (string, error) {
	s := strings.Split(ref.V, "$column.")
	if len(s) != 2 {
		return "", fmt.Errorf("sqlspec: failed to split by column name from %q", ref)

	}
	s = strings.Split(s[0], ".")
	if len(s) != 3 {
		return "", fmt.Errorf("sqlspec: failed to extract table name from %q", s)
	}
	return s[1], nil
}

func colRef(cName string, tName string) *schemaspec.Ref {
	v := "$table." + tName + ".$column." + cName
	return &schemaspec.Ref{
		V: v,
	}
}

// SchemaRef returns the schemaspec.Ref to the schema with the given name.
func SchemaRef(n string) *schemaspec.Ref {
	return &schemaspec.Ref{V: "$schema." + n}
}

// Attrer is the interface that wraps the Attr method.
type Attrer interface {
	Attr(string) (*schemaspec.Attr, bool)
}

// convertCommentFromSpec converts a spec comment attribute to a schema element attribute.
func convertCommentFromSpec(spec Attrer, attrs *[]schema.Attr) error {
	if c, ok := spec.Attr("comment"); ok {
		s, err := c.String()
		if err != nil {
			return err
		}
		*attrs = append(*attrs, &schema.Comment{Text: s})
	}
	return nil
}

// convertCommentFromSchema converts a schema element comment attribute to a spec comment attribute.
func convertCommentFromSchema(src []schema.Attr, trgt *[]*schemaspec.Attr) {
	var c schema.Comment
	if sqlx.Has(src, &c) {
		*trgt = append(*trgt, StrAttr("comment", c.Text))
	}
}
