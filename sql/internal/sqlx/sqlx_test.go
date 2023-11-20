// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"errors"
	"strconv"
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestModeInspectRealm(t *testing.T) {
	m := ModeInspectRealm(nil)
	require.True(t, m.Is(schema.InspectSchemas))
	require.True(t, m.Is(schema.InspectTables))

	m = ModeInspectRealm(&schema.InspectRealmOption{})
	require.True(t, m.Is(schema.InspectSchemas))
	require.True(t, m.Is(schema.InspectTables))

	m = ModeInspectRealm(&schema.InspectRealmOption{Mode: schema.InspectSchemas})
	require.True(t, m.Is(schema.InspectSchemas))
	require.False(t, m.Is(schema.InspectTables))
}

func TestModeInspectSchema(t *testing.T) {
	m := ModeInspectSchema(nil)
	require.True(t, m.Is(schema.InspectSchemas))
	require.True(t, m.Is(schema.InspectTables))

	m = ModeInspectSchema(&schema.InspectOptions{})
	require.True(t, m.Is(schema.InspectSchemas))
	require.True(t, m.Is(schema.InspectTables))

	m = ModeInspectSchema(&schema.InspectOptions{Mode: schema.InspectSchemas})
	require.True(t, m.Is(schema.InspectSchemas))
	require.False(t, m.Is(schema.InspectTables))
}

func TestBuilder(t *testing.T) {
	var (
		b       = &Builder{QuoteOpening: '"', QuoteClosing: '"'}
		columns = []string{"a", "b", "c"}
	)
	b.P("CREATE TABLE").
		Table(&schema.Table{Name: "users"}).
		Wrap(func(b *Builder) {
			b.MapComma(columns, func(i int, b *Builder) {
				b.Ident(columns[i]).P("int").P("NOT NULL")
			})
			b.Comma().P("PRIMARY KEY").Wrap(func(b *Builder) {
				b.MapComma(columns, func(i int, b *Builder) {
					b.Ident(columns[i])
				})
			})
		})
	require.Equal(t, `CREATE TABLE "users" ("a" int NOT NULL, "b" int NOT NULL, "c" int NOT NULL, PRIMARY KEY ("a", "b", "c"))`, b.String())

	// WrapErr.
	require.EqualError(t, b.WrapErr(func(*Builder) error { return errors.New("oops") }), "oops")
}

func TestBuilder_Qualifier(t *testing.T) {
	var (
		s = "other"
		b = &Builder{QuoteOpening: '"', QuoteClosing: '"', Schema: &s}
	)
	b.P("CREATE TABLE").Table(schema.NewTable("users"))
	require.Equal(t, `CREATE TABLE "other"."users"`, b.String())

	// Bypass table schema.
	b.Reset()
	b.P("CREATE TABLE").Table(schema.NewTable("users").SetSchema(schema.New("test")))
	require.Equal(t, `CREATE TABLE "other"."users"`, b.String())

	// Empty qualifier, means skip.
	s = ""
	b.Reset()
	b.P("CREATE TABLE").Table(schema.NewTable("users").SetSchema(schema.New("test")))
	require.Equal(t, `CREATE TABLE "users"`, b.String())
}

func TestQuote(t *testing.T) {
	var (
		s = "s1"
		b = &Builder{QuoteOpening: '[', QuoteClosing: ']', Schema: &s}
	)
	b.P("EXECUTE sp_rename").
		P("@newname = N'c2'").Comma().
		P("@objtype = N'COLUMN'").Comma()
	b.P("@objname = ").Quote("N", func(b *Builder) {
		b.TableResource(schema.NewTable("t1"), &schema.Column{Name: "c1"})
	})
	require.Equal(t, `EXECUTE sp_rename @newname = N'c2', @objtype = N'COLUMN', @objname = N'[s1].[t1].[c1]'`, b.String())
}
func TestMayWrap(t *testing.T) {
	tests := []struct {
		input   string
		wrapped bool
	}{
		{"", true},
		{"()", false},
		{"('text')", false},
		{"('(')", false},
		{`('(\\')`, false},
		{`('\')(')`, false},
		{`(a) in (b)`, true},
		{`a in (b)`, true},
		{`("\\\\(((('")`, false},
		{`('(')||(')')`, true},
		// Test examples from SQLite.
		{"b || 'x'", true},
		{"a+1", true},
		{"substr(x, 2)", true},
		{"(json_extract(x, '$.a'))", false},
		{"(substr(a, 2) COLLATE NOCASE)", false},
		{"(b+random())", false},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			expect := tt.input
			if tt.wrapped {
				expect = "(" + expect + ")"
			}
			require.Equal(t, expect, MayWrap(tt.input))
		})
	}
}

func TestExprLastIndex(t *testing.T) {
	tests := []struct {
		input   string
		wantIdx int
	}{
		{"", -1},
		{"()", 1},
		{"'('", 2},
		{"('(')", 4},
		{"('text')", 7},
		{"floor(x), y", 7},
		{"f(floor(x), y)", 13},
		{"f(floor(x), y, (z))", 18},
		{"f(x, (x*2)), y, (z)", 10},
		{"(a || ' ' || b)", 14},
		{"(a || ', ' || b)", 15},
		{"a || ', ' || b, x", 13},
		{"(a || ', ' || b), x", 15},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			idx := ExprLastIndex(tt.input)
			require.Equal(t, tt.wantIdx, idx)
		})
	}
}

func TestIsQuoted(t *testing.T) {
	tests := []struct {
		input  string
		quotes []byte
		want   bool
	}{
		{"''", []byte{'"', '\''}, true},
		{`""`, []byte{'"', '\''}, true},
		{"' '' \"\"'' '", []byte{'\''}, true},
		{"''''''''", []byte{'\''}, true},
		{"'foo'''", []byte{'\''}, true},
		{"'foo'''''", []byte{'\''}, true},
		{"'foo'', '''", []byte{'\''}, true},
		{"'foo bar'", []byte{'\''}, true},
		{`"never say \"never\""`, []byte{'"'}, true},
		{`"never say \"never\'"`, []byte{'"'}, true},
		{`'never say \"never\''`, []byte{'\''}, true},

		{"'", []byte{'"', '\''}, false},
		{`"`, []byte{'"', '\''}, false},
		{"'foo' ''", []byte{'\''}, false},
		{"'foo' ()  ''", []byte{'\''}, false},
		{"'foo', ''", []byte{'\''}, false},
		{"'foo', 'bar'", []byte{'\''}, false},
		{"'foo',\" 'bar'", []byte{'\''}, false},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			quoted := IsQuoted(tt.input, tt.quotes...)
			require.Equal(t, tt.want, quoted)
		})
	}
}

func TestReverseChanges(t *testing.T) {
	tests := []struct {
		input  []schema.Change
		expect []schema.Change
	}{
		{
			input: []schema.Change{
				(*schema.AddColumn)(nil),
			},
			expect: []schema.Change{
				(*schema.AddColumn)(nil),
			},
		},
		{
			input: []schema.Change{
				(*schema.AddColumn)(nil),
				(*schema.DropColumn)(nil),
			},
			expect: []schema.Change{
				(*schema.DropColumn)(nil),
				(*schema.AddColumn)(nil),
			},
		},
		{
			input: []schema.Change{
				(*schema.AddColumn)(nil),
				(*schema.ModifyColumn)(nil),
				(*schema.DropColumn)(nil),
			},
			expect: []schema.Change{
				(*schema.DropColumn)(nil),
				(*schema.ModifyColumn)(nil),
				(*schema.AddColumn)(nil),
			},
		},
		{
			input: []schema.Change{
				(*schema.AddColumn)(nil),
				(*schema.ModifyColumn)(nil),
				(*schema.DropColumn)(nil),
				(*schema.ModifyColumn)(nil),
			},
			expect: []schema.Change{
				(*schema.ModifyColumn)(nil),
				(*schema.DropColumn)(nil),
				(*schema.ModifyColumn)(nil),
				(*schema.AddColumn)(nil),
			},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ReverseChanges(tt.input)
			require.Equal(t, tt.expect, tt.input)
		})
	}
}

func TestIsUint(t *testing.T) {
	require.True(t, IsUint("1"))
	require.False(t, IsUint("-1"))
	require.False(t, IsUint("1.2"))
	require.False(t, IsUint("1.2.3"))
}
