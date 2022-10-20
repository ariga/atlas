// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"fmt"
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
