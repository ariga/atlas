// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"ariga.io/atlas/sql/schema"
)

type (
	// ExecQueryCloser is the interface that groups
	// Close with the schema.ExecQuerier methods.
	ExecQueryCloser interface {
		schema.ExecQuerier
		io.Closer
	}
	nopCloser struct {
		schema.ExecQuerier
	}
)

// Close implements the io.Closer interface.
func (nopCloser) Close() error { return nil }

// SingleConn returns a closable single connection from the given ExecQuerier.
// If the ExecQuerier is already bound to a single connection (e.g. Tx, Conn),
// the connection will return as-is with a NopCloser.
func SingleConn(ctx context.Context, conn schema.ExecQuerier) (ExecQueryCloser, error) {
	// A standard sql.DB or a wrapper of it.
	if opener, ok := conn.(interface {
		Conn(context.Context) (*sql.Conn, error)
	}); ok {
		return opener.Conn(ctx)
	}
	// Tx and Conn are bounded to a single connection.
	// We use sql/driver.Tx to cover also custom Tx structs.
	_, ok1 := conn.(driver.Tx)
	_, ok2 := conn.(*sql.Conn)
	if ok1 || ok2 {
		return nopCloser{ExecQuerier: conn}, nil
	}
	return nil, fmt.Errorf("cannot obtain a single connection from %T", conn)
}

// ValidString reports if the given string is not null and valid.
func ValidString(s sql.NullString) bool {
	return s.Valid && s.String != "" && strings.ToLower(s.String) != "null"
}

// ScanOne scans one record and closes the rows at the end.
func ScanOne(rows *sql.Rows, dest ...any) error {
	defer rows.Close()
	if !rows.Next() {
		return sql.ErrNoRows
	}
	if err := rows.Scan(dest...); err != nil {
		return err
	}
	return rows.Close()
}

// ScanNullBool scans one sql.NullBool record and closes the rows at the end.
func ScanNullBool(rows *sql.Rows) (sql.NullBool, error) {
	var b sql.NullBool
	return b, ScanOne(rows, &b)
}

// ScanStrings scans sql.Rows into a slice of strings and closes it at the end.
func ScanStrings(rows *sql.Rows) ([]string, error) {
	defer rows.Close()
	var vs []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

type (
	// ScanStringer groups the fmt.Stringer and sql.Scanner interfaces.
	ScanStringer interface {
		fmt.Stringer
		sql.Scanner
	}
	// nullString is a sql.NullString that implements the ScanStringer interface.
	nullString struct{ sql.NullString }
)

func (s nullString) String() string    { return s.NullString.String }
func (s *nullString) Scan(v any) error { return s.NullString.Scan(v) }

// SchemaFKs scans the rows and adds the foreign-key to the schema table.
// Reference elements are added as stubs and should be linked manually by the
// caller.
func SchemaFKs(s *schema.Schema, rows *sql.Rows) error {
	return TypedSchemaFKs[*nullString](s, rows)
}

// TypedSchemaFKs is a version of SchemaFKs that allows to specify the type of
// used to scan update and delete actions from the database.
func TypedSchemaFKs[T ScanStringer](s *schema.Schema, rows *sql.Rows) error {
	for rows.Next() {
		var (
			updateAction, deleteAction                                   = V(new(T)), V(new(T))
			name, table, column, tSchema, refTable, refColumn, refSchema string
		)
		if err := rows.Scan(&name, &table, &column, &tSchema, &refTable, &refColumn, &refSchema, &updateAction, &deleteAction); err != nil {
			return err
		}
		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", table)
		}
		fk, ok := t.ForeignKey(name)
		if !ok {
			fk = &schema.ForeignKey{
				Symbol:   name,
				Table:    t,
				RefTable: t,
				OnUpdate: schema.ReferenceOption(updateAction.String()),
				OnDelete: schema.ReferenceOption(deleteAction.String()),
			}
			switch {
			// Self reference.
			case tSchema == refSchema && refTable == table:
			// Reference to the same schema.
			case tSchema == refSchema && refTable != table:
				if fk.RefTable, ok = s.Table(refTable); !ok {
					fk.RefTable = &schema.Table{Name: refTable, Schema: s}
				}
			// Reference to an external schema.
			case tSchema != refSchema:
				fk.RefTable = &schema.Table{Name: refTable, Schema: &schema.Schema{Name: refSchema}}
			}
			t.ForeignKeys = append(t.ForeignKeys, fk)
		}
		c, ok := t.Column(column)
		if !ok {
			return fmt.Errorf("column %q was not found for fk %q", column, fk.Symbol)
		}
		// Rows are ordered by ORDINAL_POSITION that specifies
		// the position of the column in the FK definition.
		if _, ok := fk.Column(c.Name); !ok {
			fk.Columns = append(fk.Columns, c)
			c.ForeignKeys = append(c.ForeignKeys, fk)
		}
		// Stub referenced columns or link if it's a self-reference.
		var rc *schema.Column
		if fk.Table != fk.RefTable {
			rc = &schema.Column{Name: refColumn}
		} else if c, ok := t.Column(refColumn); ok {
			rc = c
		} else {
			return fmt.Errorf("referenced column %q was not found for fk %q", refColumn, fk.Symbol)
		}
		if _, ok := fk.RefColumn(rc.Name); !ok {
			fk.RefColumns = append(fk.RefColumns, rc)
		}
	}
	return nil
}

// LinkSchemaTables links foreign-key stub tables/columns to actual elements.
func LinkSchemaTables(schemas []*schema.Schema) {
	byName := make(map[string]map[string]*schema.Table)
	for _, s := range schemas {
		byName[s.Name] = make(map[string]*schema.Table)
		for _, t := range s.Tables {
			t.Schema = s
			byName[s.Name][t.Name] = t
		}
	}
	for _, s := range schemas {
		for _, t := range s.Tables {
			for _, fk := range t.ForeignKeys {
				rs, ok := byName[fk.RefTable.Schema.Name]
				if !ok {
					continue
				}
				ref, ok := rs[fk.RefTable.Name]
				if !ok {
					continue
				}
				fk.RefTable = ref
				for i, c := range fk.RefColumns {
					rc, ok := ref.Column(c.Name)
					if ok {
						fk.RefColumns[i] = rc
					}
				}
			}
		}
	}
}

// ValuesEqual checks if the 2 string slices are equal (including their order).
func ValuesEqual(v1, v2 []string) bool {
	if len(v1) != len(v2) {
		return false
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			return false
		}
	}
	return true
}

// ModeInspectSchema returns the InspectMode or its default.
func ModeInspectSchema(o *schema.InspectOptions) schema.InspectMode {
	if o == nil || o.Mode == 0 {
		return schema.InspectSchemas | schema.InspectTables | schema.InspectViews | schema.InspectFuncs |
			schema.InspectTypes | schema.InspectObjects | schema.InspectTriggers
	}
	return o.Mode
}

// ModeInspectRealm returns the InspectMode or its default.
func ModeInspectRealm(o *schema.InspectRealmOption) schema.InspectMode {
	if o == nil || o.Mode == 0 {
		return schema.InspectSchemas | schema.InspectTables | schema.InspectViews | schema.InspectFuncs |
			schema.InspectTypes | schema.InspectObjects | schema.InspectTriggers
	}
	return o.Mode
}

// A Builder provides a syntactic sugar API for writing SQL statements.
type Builder struct {
	bytes.Buffer
	QuoteOpening byte    // quoting identifiers
	QuoteClosing byte    // quoting identifiers
	Schema       *string // schema qualifier
	Indent       string  // indentation string
	level        int     // current indentation level
}

// P writes a list of phrases to the builder separated and
// suffixed with whitespace.
func (b *Builder) P(phrases ...string) *Builder {
	for _, p := range phrases {
		if p == "" {
			continue
		}
		if b.Len() > 0 && b.lastByte() != ' ' && b.lastByte() != '(' {
			b.WriteByte(' ')
		}
		b.WriteString(p)
		if p[len(p)-1] != ' ' {
			b.WriteByte(' ')
		}
	}
	return b
}

// Int64 writes the given value to the builder in base 10.
func (b *Builder) Int64(v int64) *Builder {
	return b.P(strconv.FormatInt(v, 10))
}

// Ident writes the given string quoted as an SQL identifier.
func (b *Builder) Ident(s string) *Builder {
	if s != "" {
		b.WriteByte(b.QuoteOpening)
		b.WriteString(s)
		b.WriteByte(b.QuoteClosing)
		b.WriteByte(' ')
	}
	return b
}

// View writes the view identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) View(v *schema.View) *Builder {
	return b.mayQualify(v.Schema, v.Name)
}

// Table writes the table identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) Table(t *schema.Table) *Builder {
	return b.mayQualify(t.Schema, t.Name)
}

// TableColumn writes the table's resource identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) TableColumn(t *schema.Table, c *schema.Column) *Builder {
	return b.mayQualify(t.Schema, t.Name, c.Name)
}

// Func writes the function identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) Func(f *schema.Func) *Builder {
	return b.mayQualify(f.Schema, f.Name)
}

// FuncCall writes the function identifier to the builder as a function call,
func (b *Builder) FuncCall(f *schema.Func, args ...string) *Builder {
	b.Func(f).rewriteLastByte('(')
	b.MapComma(args, func(i int, b *Builder) {
		b.WriteString(args[i])
	})
	b.WriteByte(')')
	return b
}

// Proc writes the procedure identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) Proc(p *schema.Proc) *Builder {
	return b.mayQualify(p.Schema, p.Name)
}

// ProcCall writes the procedure identifier to the builder as a procedure call,
func (b *Builder) ProcCall(p *schema.Proc, args ...string) *Builder {
	b.Proc(p).rewriteLastByte('(')
	b.MapComma(args, func(i int, b *Builder) {
		b.WriteString(args[i])
	})
	b.WriteByte(')')
	return b
}

// TableResource writes the table resource identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) TableResource(t *schema.Table, r any) *Builder {
	switch c := r.(type) {
	case *schema.Column:
		return b.TableColumn(t, c)
	case *schema.Index:
		return b.mayQualify(t.Schema, t.Name, c.Name)
	default:
		panic(fmt.Sprintf("unexpected table resource: %T", r))
	}
}

// SchemaResource writes the schema resource identifier to the builder, prefixed
// with the schema name if exists.
func (b *Builder) SchemaResource(s *schema.Schema, name string) *Builder {
	return b.mayQualify(s, name)
}

func (b *Builder) mayQualify(s *schema.Schema, top string, children ...string) *Builder {
	switch {
	// Custom qualifier.
	case b.Schema != nil:
		// Empty means skip prefix.
		if *b.Schema != "" {
			b.Ident(*b.Schema)
			b.rewriteLastByte('.')
		}
	// Default schema qualifier.
	case s != nil && s.Name != "":
		b.Ident(s.Name)
		b.rewriteLastByte('.')
	}
	b.Ident(top)
	for _, ident := range children {
		b.rewriteLastByte('.')
		b.Ident(ident)
	}
	return b
}

// IndentIn adds one indentation in.
func (b *Builder) IndentIn() *Builder {
	b.level++
	return b
}

// IndentOut removed one indentation level.
func (b *Builder) IndentOut() *Builder {
	b.level--
	return b
}

// NL adds line break and prefix the new line with
// indentation in case indentation is enabled.
func (b *Builder) NL() *Builder {
	if b.Indent != "" {
		if b.lastByte() == ' ' {
			b.rewriteLastByte('\n')
		} else {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(b.Indent, b.level))
	}
	return b
}

// Comma writes a comma in case the buffer is not empty, or
// replaces the last char if it is a whitespace.
func (b *Builder) Comma() *Builder {
	switch {
	case b.Len() == 0:
	case b.lastByte() == ' ':
		b.rewriteLastByte(',')
		b.WriteByte(' ')
	default:
		b.WriteString(", ")
	}
	return b
}

// MapComma maps the slice x using the function f and joins the result with
// a comma separating between the written elements.
func (b *Builder) MapComma(x any, f func(i int, b *Builder)) *Builder {
	s := reflect.ValueOf(x)
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			b.Comma()
		}
		f(i, b)
	}
	return b
}

// Quote wraps the given function with a single quote and a prefix
func (b *Builder) Quote(prefix string, fn func(b *Builder)) *Builder {
	b.WriteString(prefix)
	b.WriteByte('\'')
	fn(b)
	if b.lastByte() != ' ' {
		b.WriteByte('\'')
	} else {
		b.rewriteLastByte('\'')
	}
	return b
}

// MapIndent is like MapComma, but writes a new line before each element.
func (b *Builder) MapIndent(x any, f func(i int, b *Builder)) *Builder {
	return b.MapComma(x, func(i int, b *Builder) {
		f(i, b.NL())
	})
}

// MapCommaErr is like MapComma, but returns an error if f returns an error.
func (b *Builder) MapCommaErr(x any, f func(i int, b *Builder) error) error {
	s := reflect.ValueOf(x)
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			b.Comma()
		}
		if err := f(i, b); err != nil {
			return err
		}
	}
	return nil
}

// MapIndentErr is like MapCommaErr, but writes a new line before each element.
func (b *Builder) MapIndentErr(x any, f func(i int, b *Builder) error) error {
	return b.MapCommaErr(x, func(i int, b *Builder) error {
		return f(i, b.NL())
	})
}

// Wrap wraps the written string with parentheses.
func (b *Builder) Wrap(f func(b *Builder)) *Builder {
	b.WriteByte('(')
	f(b)
	if b.lastByte() != ' ' {
		b.WriteByte(')')
	} else {
		b.rewriteLastByte(')')
	}
	return b
}

// WrapErr wraps the written string with parentheses
func (b *Builder) WrapErr(f func(b *Builder) error) error {
	var err error
	b.Wrap(func(b *Builder) { err = f(b) })
	return err
}

// WrapIndent is like Wrap but with extra level of indentation.
func (b *Builder) WrapIndent(f func(b *Builder)) *Builder {
	return b.Wrap(func(b *Builder) {
		b.IndentIn()
		f(b)
		b.IndentOut()
		b.NL()
	})
}

// WrapIndentErr is like WrapErr but with extra level of indentation.
func (b *Builder) WrapIndentErr(f func(b *Builder) error) error {
	var err error
	b.Wrap(func(b *Builder) {
		b.IndentIn()
		err = f(b)
		b.IndentOut()
		b.NL()
	})
	return err
}

// Clone returns a duplicate of the builder.
func (b *Builder) Clone() *Builder {
	return &Builder{
		QuoteOpening: b.QuoteOpening,
		QuoteClosing: b.QuoteClosing,
		Buffer:       *bytes.NewBufferString(b.Buffer.String()),
	}
}

// String overrides the Buffer.String method and ensure no spaces pad the returned statement.
func (b *Builder) String() string {
	return strings.TrimSpace(b.Buffer.String())
}

func (b *Builder) lastByte() byte {
	if b.Len() == 0 {
		return 0
	}
	buf := b.Buffer.Bytes()
	return buf[len(buf)-1]
}

func (b *Builder) rewriteLastByte(c byte) {
	if b.Len() == 0 {
		return
	}
	buf := b.Buffer.Bytes()
	buf[len(buf)-1] = c
}

// IsQuoted reports if the given string is quoted with one of the given quotes (e.g. ', ", `).
func IsQuoted(s string, q ...byte) bool {
	last := len(s) - 1
	if last < 1 {
		return false
	}
Top:
	for _, quote := range q {
		if s[0] != quote || s[last] != quote {
			continue
		}
		for i := 1; i < last-1; i++ {
			switch c := s[i]; {
			case c == '\\', c == quote && s[i+1] == quote:
				i++
			// Accept only escaped quotes and reject otherwise.
			case c == quote:
				continue Top
			}
		}
		return true
	}
	return false
}

// IsLiteralBool reports if the given string is a valid literal bool.
func IsLiteralBool(s string) bool {
	_, err := strconv.ParseBool(s)
	return err == nil
}

// IsLiteralNumber reports if the given string is a literal number.
func IsLiteralNumber(s string) bool {
	// Hex digits.
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// Some databases allow odd length hex string.
		_, err := strconv.ParseUint(s[2:], 16, 64)
		return err == nil
	}
	// Digits with optional exponent.
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// DefaultValue returns the string represents the DEFAULT of a column.
func DefaultValue(c *schema.Column) (string, bool) {
	switch x := schema.UnderlyingExpr(c.Default).(type) {
	case nil:
		return "", false
	case *schema.Literal:
		return x.V, true
	case *schema.RawExpr:
		return x.X, true
	default:
		panic(fmt.Sprintf("unexpected default value type: %T", x))
	}
}

// MayWrap ensures the given string is wrapped with parentheses.
// Used by the different drivers to turn strings valid expressions.
func MayWrap(s string) string {
	n := len(s) - 1
	if len(s) < 2 || s[0] != '(' || s[n] != ')' || !balanced(s[1:n]) {
		return "(" + s + ")"
	}
	return s
}

func balanced(expr string) bool {
	return ExprLastIndex(expr) == len(expr)-1
}

// ExprLastIndex scans the first expression in the given string until
// its end and returns its last index.
func ExprLastIndex(expr string) int {
	var l, r int
	for i := 0; i < len(expr); i++ {
	Top:
		switch expr[i] {
		case '(':
			l++
		case ')':
			r++
		// String or identifier.
		case '\'', '"', '`':
			for j := i + 1; j < len(expr); j++ {
				switch expr[j] {
				case '\\':
					j++
				case expr[i]:
					i = j
					break Top
				}
			}
			// Unexpected EOS.
			return -1
		}
		// Balanced parens and we reached EOS or a terminator.
		if l == r && (i == len(expr)-1 || expr[i+1] == ',') {
			return i
		} else if r > l {
			return -1
		}
	}
	return -1
}

// ReverseChanges reverses the order of the changes.
func ReverseChanges(c []schema.Change) {
	for i, n := 0, len(c); i < n/2; i++ {
		c[i], c[n-i-1] = c[n-i-1], c[i]
	}
}

// P returns a pointer to v.
func P[T any](v T) *T {
	return &v
}

// V returns the value p is pointing to.
// If p is nil, the zero value is returned.
func V[T any](p *T) (v T) {
	if p != nil {
		v = *p
	}
	return
}

// IsUint reports whether the string represents an unsigned integer.
func IsUint(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
