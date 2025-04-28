// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestURLSetPathFunc(t *testing.T) {
	tests := []struct {
		URL  cty.Value
		Path cty.Value
		Want cty.Value
	}{
		{
			cty.StringVal("mysql://root:pass@mysql:3306"),
			cty.StringVal(""),
			cty.StringVal("mysql://root:pass@mysql:3306"),
		},
		{
			cty.StringVal("mysql://root:pass@mysql:3306?parseTime=true"),
			cty.StringVal("my-tenant"),
			cty.StringVal("mysql://root:pass@mysql:3306/my-tenant?parseTime=true"),
		},
		{
			cty.StringVal("mysql://root:pass@mysql:3306/admin?parseTime=true"),
			cty.StringVal("my-tenant"),
			cty.StringVal("mysql://root:pass@mysql:3306/my-tenant?parseTime=true"),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("(%#v,%#v)", test.URL, test.Path), func(t *testing.T) {
			got, err := urlSetPathFunc.Call([]cty.Value{test.URL, test.Path})

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestURLQuerySetFunc(t *testing.T) {
	tests := []struct {
		URL   cty.Value
		Key   cty.Value
		Value cty.Value
		Want  cty.Value
	}{
		{
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?sslmode=disable&sslmode=disable"),
			cty.StringVal("search_path"),
			cty.StringVal("schema"),
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?search_path=schema&sslmode=disable&sslmode=disable"),
		},
		{
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?sslmode=disable&search_path=admin&sslmode=disable"),
			cty.StringVal("search_path"),
			cty.StringVal("schema"),
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?search_path=schema&sslmode=disable&sslmode=disable"),
		},
		{
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database"),
			cty.StringVal("search_path"),
			cty.StringVal("schema"),
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?search_path=schema"),
		},
		{
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?sslmode=disable&search_path=admin&sslmode=disable"),
			cty.StringVal("search_path"),
			cty.StringVal(""),
			cty.StringVal("postgres://postgres:pass@0.0.0.0:5432/database?search_path=&sslmode=disable&sslmode=disable"),
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("(%#v,%#v,%#v)", test.URL, test.Key, test.Value), func(t *testing.T) {
			got, err := urlQuerySetFunc.Call([]cty.Value{test.URL, test.Key, test.Value})

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
			var (
				f = fmt.Sprintf(
					`v = urlqueryset(%q, %q, %q)`,
					test.URL.AsString(), test.Key.AsString(), test.Value.AsString(),
				)
				d struct {
					V cty.Value `spec:"v"`
				}
			)
			require.NoError(t, New().EvalBytes([]byte(f), &d, nil))
			require.True(t, d.V.RawEquals(got))
		})
	}
}

func TestURLEscapeFunc(t *testing.T) {
	for _, tt := range []string{"foo", "foo?", "foo&"} {
		t.Run(tt, func(t *testing.T) {
			got, err := urlEscapeFunc.Call([]cty.Value{cty.StringVal(tt)})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if want := url.QueryEscape(tt); got.AsString() != want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

func TestURLUserinfoFunc(t *testing.T) {
	u := cty.StringVal("mysql://localhost:3306")
	// Only user is provided
	got, err := urlUserinfoFunc.Call([]cty.Value{u, cty.StringVal("user")})
	require.NoError(t, err)
	require.Equal(t, cty.StringVal("mysql://user@localhost:3306"), got)
	// The password is null
	got, err = urlUserinfoFunc.Call([]cty.Value{u, cty.StringVal("user"), cty.NullVal(cty.String)})
	require.NoError(t, err)
	require.Equal(t, cty.StringVal("mysql://user@localhost:3306"), got)
	// Both user and password are provided
	got, err = urlUserinfoFunc.Call([]cty.Value{u, cty.StringVal("user"), cty.StringVal("pass")})
	require.NoError(t, err)
	require.Equal(t, cty.StringVal("mysql://user:pass@localhost:3306"), got)
}

func TestStartWithFunc(t *testing.T) {
	got, err := startsWithFunc.Call([]cty.Value{cty.StringVal("abc"), cty.StringVal("ab")})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
	got, err = startsWithFunc.Call([]cty.Value{cty.StringVal("abc"), cty.StringVal("bc")})
	require.NoError(t, err)
	require.Equal(t, cty.False, got)
}

func TestEndsWithFunc(t *testing.T) {
	got, err := endsWithFunc.Call([]cty.Value{cty.StringVal("abc"), cty.StringVal("ab")})
	require.NoError(t, err)
	require.Equal(t, cty.False, got)
	got, err = endsWithFunc.Call([]cty.Value{cty.StringVal("abc"), cty.StringVal("bc")})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
}

func TestEmptyFunc(t *testing.T) {
	got, err := emptyFunc.Call([]cty.Value{cty.ListValEmpty(cty.String)})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
	got, err = emptyFunc.Call([]cty.Value{cty.SetValEmpty(cty.String)})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
	got, err = emptyFunc.Call([]cty.Value{cty.MapValEmpty(cty.String)})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
	got, err = emptyFunc.Call([]cty.Value{cty.EmptyTupleVal})
	require.NoError(t, err)
	require.Equal(t, cty.True, got)
	got, err = emptyFunc.Call([]cty.Value{cty.ListVal([]cty.Value{cty.StringVal("a")})})
	require.NoError(t, err)
	require.Equal(t, cty.False, got)
	// Invalid value.
	got, err = emptyFunc.Call([]cty.Value{cty.StringVal("a")})
	require.EqualError(t, err, "collection must be a list, a map or a tuple")
}

func TestRegexpEscapeFunc(t *testing.T) {
	got, err := regexpEscape.Call([]cty.Value{cty.StringVal("a|b|c")})
	require.NoError(t, err)
	require.Equal(t, "a\\|b\\|c", got.AsString())
	got, err = regexpEscape.Call([]cty.Value{cty.StringVal("abc")})
	require.NoError(t, err)
	require.Equal(t, "abc", got.AsString())
}

func TestMakeFileFunc(t *testing.T) {
	fn := MakeFileFunc("testdata")
	_, err := fn.Call([]cty.Value{cty.StringVal("foo")})
	require.EqualError(t, err, "base directory must be an absolute path. got: testdata")
	base, err := filepath.Abs("testdata")
	require.NoError(t, err)
	fn = MakeFileFunc(base)
	v, err := fn.Call([]cty.Value{cty.StringVal("a.hcl")})
	require.NoError(t, err)
	require.Equal(t, "person \"rotemtam\" {\n  hobby = var.hobby\n}", v.AsString())
}

func TestMakeGlobFunc(t *testing.T) {
	fn := MakeGlobFunc("testdata")
	_, err := fn.Call([]cty.Value{cty.StringVal("foo")})
	require.EqualError(t, err, "base directory must be an absolute path. got: testdata")

	base, err := filepath.Abs("testdata")
	require.NoError(t, err)
	fn = MakeGlobFunc(base)
	v, err := fn.Call([]cty.Value{cty.StringVal("*.hcl")})
	require.NoError(t, err)

	var result []string
	for _, f := range v.AsValueSlice() {
		p, err := filepath.Rel(base, f.AsString())
		require.NoError(t, err)
		result = append(result, p)
	}
	require.Equal(t, []string{"a.hcl", "b.hcl", "variables.hcl"}, result)
}

func TestMakeFilesetFunc(t *testing.T) {
	base, err := filepath.Abs("testdata")
	require.NoError(t, err)

	fn := MakeFileSetFunc(base)

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "Simple HCL files",
			pattern:  "*.hcl",
			expected: []string{"a.hcl", "b.hcl", "variables.hcl"},
		},
		{
			name:     "Non-existent files",
			pattern:  "*.txt",
			expected: []string{},
		},
		{
			name:     "Nested directories",
			pattern:  "**/*.hcl",
			expected: []string{"a.hcl", "b.hcl", "nested/c.hcl", "variables.hcl"},
		},
		{
			name:     "Single file",
			pattern:  "a.hcl",
			expected: []string{"a.hcl"},
		},
		{
			name:     "Files with specific prefix",
			pattern:  "a*.hcl",
			expected: []string{"a.hcl"},
		},
		{
			name:     "Files in specific directory",
			pattern:  "nested/*.hcl",
			expected: []string{"nested/c.hcl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := fn.Call([]cty.Value{cty.StringVal(tt.pattern)})
			require.NoError(t, err)
			var result []string
			for _, f := range v.AsValueSlice() {
				p, err := filepath.Rel(base, f.AsString())
				require.NoError(t, err)
				result = append(result, filepath.ToSlash(p))
			}
			require.ElementsMatch(t, tt.expected, result)
		})
	}

	// Test with relative base path
	relativeFn := MakeFileSetFunc("testdata")
	_, err = relativeFn.Call([]cty.Value{cty.StringVal("*.hcl")})
	require.EqualError(t, err, "base directory must be an absolute path. got: testdata")
}

func Example_regexpEscapeFunc() {
	for _, f := range []string{
		`v  = regexpescape("a|b|c")`,
		`v  = regexpescape("abc")`,
		`v = regexpescape(<<TAB
id | name
---+-----
1  | foo
TAB
)`,
	} {
		var d struct {
			V string `spec:"v"`
		}
		if err := New().EvalBytes([]byte(f), &d, nil); err != nil {
			fmt.Println("failed to evaluate:", err)
			return
		}
		fmt.Printf("%s\n\n", d.V)
	}
	// Output:
	// a\|b\|c
	//
	// abc
	//
	// id \| name
	// ---\+-----
	// 1  \| foo
}

func Example_printFunc() {
	for _, f := range []string{
		`v  = print("a")`,
		`v  = print("a&b")`,
		`v  = print(1)`,
		`v  = print(true)`,
		`v  = print("hello, world")`,
		`v  = print({"hello": "world"})`,
		`v  = print(["hello", "world"])`,
	} {
		var d struct {
			V cty.Value `spec:"v"`
		}
		if err := New().EvalBytes([]byte(f), &d, nil); err != nil {
			fmt.Println("failed to evaluate:", err)
			return
		}
		fmt.Printf("%#v\n\n", d.V)
	}
	// Output:
	// a
	// cty.StringVal("a")
	//
	// a&b
	// cty.StringVal("a&b")
	//
	// 1
	// cty.NumberIntVal(1)
	//
	// true
	// cty.True
	//
	// hello, world
	// cty.StringVal("hello, world")
	//
	// {"hello":"world"}
	// cty.ObjectVal(map[string]cty.Value{"hello":cty.StringVal("world")})
	//
	// ["hello","world"]
	// cty.ListVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world")})
	//
}

func TestYAMLMerge(t *testing.T) {
	got, err := yamlMergeFunc.Call([]cty.Value{
		cty.StringVal("map:\n  zzz: true\n  foo:\n    - bar"),
		cty.StringVal("map:\n  bar:\n    - baz\n  foo:\n    - bar"),
		cty.StringVal("map:\n  map:\n    baz: true\n    foo: false"),
	})
	require.NoError(t, err)
	require.Equal(t, cty.StringVal(`map:
    bar:
        - baz
    foo:
        - bar
        - bar
    map:
        baz: true
        foo: false
    zzz: true
`), got)

	got, err = yamlMergeFunc.Call([]cty.Value{
		cty.StringVal("map:\n  zzz: true\n  foo:\n    - bar"),
	})
	require.NoError(t, err)
	require.Equal(t, cty.StringVal(`map:
  zzz: true
  foo:
    - bar`), got)

	_, err = yamlMergeFunc.Call([]cty.Value{
		cty.StringVal("map:\n  zzz: true\n  foo:\n    - bar"),
		cty.NullVal(cty.String),
		cty.StringVal("map:\n  zzz:\n    - baz"),
	})
	require.EqualError(t, err, "yamlmerge: failed to merge yaml: key zzz is defined in both src and dst, but has a different type")

	_, err = yamlMergeFunc.Call([]cty.Value{
		cty.StringVal("map:\n  zzz: true\n  foo:\n    - bar"),
		cty.StringVal("map:\n  zzz:\n    foo: true"),
	})
	require.EqualError(t, err, "yamlmerge: failed to merge yaml: key zzz is defined in both src and dst, but has a different type")
}
