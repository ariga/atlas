// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	"io/ioutil"
	"testing"

	"ariga.io/atlas/integration/entinteg/entschema"
	"ariga.io/atlas/sql/schema/schemahcl"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	graph, err := entc.LoadGraph("./ent/schema", &gen.Config{})
	if err != nil {
		t.Fatalf("entinteg: failed loading graph: %v", err)
	}
	spec, err := entschema.Convert(graph)
	if err != nil {
		t.Fatalf("entinteg: failed converting graph to schema: %v", err)
	}
	encode, err := schemahcl.Encode(spec)
	if err != nil {
		t.Fatalf("entinteg: failed encoding hcl document: %v", err)
	}
	expected, err := ioutil.ReadFile("testdata/ent.hcl")
	require.NoError(t, err)
	require.EqualValues(t, string(expected), string(encode))
}
