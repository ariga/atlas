// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

const tpl = `// Client is the client that holds all ent builders.
type Client struct {
	config
	{{- range $n := $.Nodes }}
		// {{ $n.Name }} is the client for interacting with the {{ $n.Name }} builders.
		{{ $n.Name }} *{{ $n.Name }}Client
	{{- end }}
	{{- template "client/fields/additional" $ }}
	{{- with $tmpls := matchTemplate "client/fields/additional/*" }}
		{{- range $tmpl := $tmpls }}
			{{- xtemplate $tmpl $ }}
		{{- end }}
	{{- end }}
}

// NewClient creates a new client configured with the given options.
func NewClient(opts ...Option) *Client {
	cfg := config{log: log.Println, hooks: &hooks{}}
	cfg.options(opts...)
	client := &Client{config: cfg}
	client.init()
	return client
}

func (c *Client) init() {
	{{- range $n := $.Nodes }}
    	c.{{ $n.Name }} = New{{ $n.Name }}Client(c.config)
	{{- end }}
}`

func main() {
	err := entc.Generate("./schema", &gen.Config{
		Header: `// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.
`,
		Features: []gen.Feature{gen.FeatureUpsert, gen.FeatureExecQuery},
		Templates: []*gen.Template{
			gen.MustParse(gen.NewTemplate("client/init").Parse(tpl)),
		},
	})
	if err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}
