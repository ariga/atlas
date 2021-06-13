package main

import (
	"io/ioutil"
	"log"

	"ariga.io/atlas/integration/entinteg/entschema"
	"ariga.io/atlas/sql/schema/schemahcl"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	graph, err := entc.LoadGraph("./ent/schema", &gen.Config{})
	if err != nil {
		log.Fatalf("entinteg: failed loading graph: %v", err)
	}
	spec, err := entschema.Convert(graph)
	if err != nil {
		log.Fatalf("entinteg: failed converting graph to schema: %v", err)
	}
	encode, err := schemahcl.Encode(spec)
	if err != nil {
		log.Fatalf("entinteg: failed encoding hcl document: %v", err)
	}
	err = ioutil.WriteFile("ent.hcl", encode, 0600)
	if err != nil {
		log.Fatalf("entinteg: writing hcl file: %v", err)
	}
}
