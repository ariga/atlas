// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
	"net/url"
	"testing"

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
		})
	}
}

func TestURLEscapeFunc(t *testing.T) {
	for _, tt := range []string{"foo", "foo?", "foo&"} {
		t.Run(tt, func(t *testing.T) {
			got, err := urlEscape.Call([]cty.Value{cty.StringVal(tt)})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if want := url.QueryEscape(tt); got.AsString() != want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}
