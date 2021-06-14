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
	require.EqualValues(t, expected, encode)
}
