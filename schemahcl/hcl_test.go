// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestAttributes(t *testing.T) {
	f := `i  = 1
b  = true
s  = "hello, world"
sl = ["hello", "world"]
bl = [true, false]
`
	var test struct {
		Int        int      `spec:"i"`
		Bool       bool     `spec:"b"`
		Str        string   `spec:"s"`
		StringList []string `spec:"sl"`
		BoolList   []bool   `spec:"bl"`
	}
	err := New().EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, test.Int)
	require.EqualValues(t, true, test.Bool)
	require.EqualValues(t, "hello, world", test.Str)
	require.EqualValues(t, []string{"hello", "world"}, test.StringList)
	require.EqualValues(t, []bool{true, false}, test.BoolList)
	marshal, err := Marshal(&test)
	require.NoError(t, err)
	require.EqualValues(t, f, string(marshal))
}

func TestResource(t *testing.T) {
	f := `endpoint "/hello" {
  description = "the hello handler"
  timeout_ms  = 100
  handler {
    active = true
    addr   = ":8080"
    tag {
      name  = "name"
      value = "value"
    }
  }
}
`
	type (
		Handler struct {
			Active bool   `spec:"active"`
			Addr   string `spec:"addr"`
			Tag    struct {
				Name  string `spec:"name"`
				Value string `spec:"value"`
			} `spec:"tag"`
		}

		Endpoint struct {
			Name        string   `spec:",name"`
			Description string   `spec:"description"`
			TimeoutMs   int      `spec:"timeout_ms"`
			Handler     *Handler `spec:"handler"`
		}
		File struct {
			Endpoints []*Endpoint `spec:"endpoint"`
		}
	)
	var test File
	err := New().EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.Len(t, test.Endpoints, 1)
	expected := &Endpoint{
		Name:        "/hello",
		Description: "the hello handler",
		TimeoutMs:   100,
		Handler: &Handler{
			Active: true,
			Addr:   ":8080",
			Tag: struct {
				Name  string `spec:"name"`
				Value string `spec:"value"`
			}{Name: "name", Value: "value"},
		},
	}
	require.EqualValues(t, expected, test.Endpoints[0])
	buf, err := Marshal(&test)
	require.NoError(t, err)
	require.EqualValues(t, f, string(buf))
}

func ExampleUnmarshal() {
	f := `
show "seinfeld" {
	day = SUN
	writer "jerry" {
		full_name = "Jerry Seinfeld"	
	}
	writer "larry" {
		full_name = "Larry David"	
	}
}`

	type (
		Writer struct {
			ID       string `spec:",name"`
			FullName string `spec:"full_name"`
		}
		Show struct {
			Name    string    `spec:",name"`
			Day     string    `spec:"day"`
			Writers []*Writer `spec:"writer"`
		}
	)
	var (
		test struct {
			Shows []*Show `spec:"show"`
		}
		opts = []Option{
			WithScopedEnums("show.day", "SUN", "MON", "TUE"),
		}
	)
	err := New(opts...).EvalBytes([]byte(f), &test, nil)
	if err != nil {
		panic(err)
	}
	seinfeld := test.Shows[0]
	fmt.Printf("the show %q at day %s has %d writers.", seinfeld.Name, seinfeld.Day, len(seinfeld.Writers))
	// Output: the show "seinfeld" at day SUN has 2 writers.
}

func ExampleMarshal() {
	type (
		Point struct {
			ID string `spec:",name"`
			X  int    `spec:"x"`
			Y  int    `spec:"y"`
		}
	)
	var test = struct {
		Points []*Point `spec:"point"`
	}{
		Points: []*Point{
			{ID: "start", X: 0, Y: 0},
			{ID: "end", X: 1, Y: 1},
		},
	}
	b, err := Marshal(&test)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(string(b))
	// Output:
	// point "start" {
	//   x = 0
	//   y = 0
	// }
	// point "end" {
	//   x = 1
	//   y = 1
	// }
}

func TestInterface(t *testing.T) {
	type (
		Animal interface {
			animal()
		}
		Parrot struct {
			Animal
			Name string `spec:",name"`
			Boss string `spec:"boss"`
		}
		Lion struct {
			Animal
			Name   string `spec:",name"`
			Friend string `spec:"friend"`
		}
		Zoo struct {
			Animals []Animal `spec:""`
		}
		Cast struct {
			Animal Animal `spec:""`
		}
	)
	Register("lion", &Lion{})
	Register("parrot", &Parrot{})
	t.Run("single", func(t *testing.T) {
		f := `
cast "lion_king" {
	lion "simba" {
		friend = "rafiki"
	}
}
`
		var test struct {
			Cast *Cast `spec:"cast"`
		}
		err := New().EvalBytes([]byte(f), &test, nil)
		require.NoError(t, err)
		require.EqualValues(t, &Cast{
			Animal: &Lion{
				Name:   "simba",
				Friend: "rafiki",
			},
		}, test.Cast)
	})
	t.Run("slice", func(t *testing.T) {
		f := `
zoo "ramat_gan" {
	lion "simba" {
		friend = "rafiki"
	}
	parrot "iago" {
		boss = "jafar"
	}
}
`
		var test struct {
			Zoo *Zoo `spec:"zoo"`
		}
		err := New().EvalBytes([]byte(f), &test, nil)
		require.NoError(t, err)
		require.EqualValues(t, &Zoo{
			Animals: []Animal{
				&Lion{
					Name:   "simba",
					Friend: "rafiki",
				},
				&Parrot{
					Name: "iago",
					Boss: "jafar",
				},
			},
		}, test.Zoo)
	})
}

func TestQualified(t *testing.T) {
	type Person struct {
		Name  string `spec:",name"`
		Title string `spec:",qualifier"`
	}
	var test struct {
		Person *Person `spec:"person"`
	}
	h := `person "dr" "jekyll" {
}
`
	err := New().EvalBytes([]byte(h), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, test.Person, &Person{
		Title: "dr",
		Name:  "jekyll",
	})
	out, err := Marshal(&test)
	require.NoError(t, err)
	require.EqualValues(t, h, string(out))
}

func TestNameAttr(t *testing.T) {
	h := `
named "block_id" {
  name = "atlas"
}
ref = named.block_id.name
`
	type Named struct {
		Name string `spec:"name,name"`
	}
	var test struct {
		Named *Named `spec:"named"`
		Ref   string `spec:"ref"`
	}
	err := New().EvalBytes([]byte(h), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, &Named{
		Name: "atlas",
	}, test.Named)
	require.EqualValues(t, "atlas", test.Ref)
}

func TestRefPatch(t *testing.T) {
	type (
		Family struct {
			Name string `spec:"name,name"`
		}
		Person struct {
			Name   string `spec:",name"`
			Family *Ref   `spec:"family"`
		}
	)
	Register("family", &Family{})
	Register("person", &Person{})
	var test struct {
		Families []*Family `spec:"family"`
		People   []*Person `spec:"person"`
	}
	h := `
variable "family_name" {
  type = string
}

family "default" {
	name = var.family_name
}

person "rotem" {
	family = family.default
}
`
	err := New().EvalBytes([]byte(h), &test, map[string]cty.Value{
		"family_name": cty.StringVal("tam"),
	})
	require.NoError(t, err)
	require.EqualValues(t, "$family.tam", test.People[0].Family.V)
}

func TestMultiFile(t *testing.T) {
	type Person struct {
		Name   string `spec:",name"`
		Hobby  string `spec:"hobby"`
		Parent *Ref   `spec:"parent"`
	}
	var test struct {
		People []*Person `spec:"person"`
	}
	var (
		paths   []string
		testDir = "testdata/"
	)
	dir, err := os.ReadDir(testDir)
	require.NoError(t, err)
	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(testDir, file.Name()))
	}
	err = New().EvalFiles(paths, &test, map[string]cty.Value{
		"hobby": cty.StringVal("coding"),
	})
	require.NoError(t, err)
	require.Len(t, test.People, 2)
	require.EqualValues(t, &Person{Name: "rotemtam", Hobby: "coding"}, test.People[0])
	require.EqualValues(t, &Person{
		Name:   "tzuri",
		Hobby:  "ice-cream",
		Parent: &Ref{V: "$person.rotemtam"},
	}, test.People[1])
}

func TestForEachResources(t *testing.T) {
	type (
		Env struct {
			Name string `spec:",name"`
			URL  string `spec:"url"`
		}
	)
	var (
		doc struct {
			Envs []*Env `spec:"env"`
		}
		b = []byte(`
variable "tenants" {
  type    = list(string)
  default = ["atlas", "ent"]
}

variable "domains" {
  type = list(object({
    name = string
    port = number
  }))
  default = [
    {
      name = "atlasgo.io"
      port = 443
    },
    {
      name = "entgo.io"
      port = 443
    },
  ]
}

env "prod" {
  for_each = toset(var.tenants)
  url = "mysql://root:pass@:3306/${each.value}"
}

env "staging" {
  for_each = toset(var.domains)
  url = "${each.value.name}:${each.value.port}"
}

env "dev" {
  for_each = {
    atlas = "atlasgo.io"
    ent   = "entgo.io"
  }
  url = "${each.value}/${each.key}"
}
`)
	)
	require.NoError(t, New(
		WithDataSource("sql", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
			attrs, diags := b.Body.JustAttributes()
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			v, diags := attrs["query"].Expr.Value(ctx)
			if diags.HasErrors() {
				return cty.NilVal, diags
			}
			return cty.ObjectVal(map[string]cty.Value{"query": v}), nil
		}),
	).EvalBytes(b, &doc, nil))
	require.Len(t, doc.Envs, 6)
	require.Equal(t, "prod", doc.Envs[0].Name)
	require.EqualValues(t, doc.Envs[0].URL, "mysql://root:pass@:3306/atlas")
	require.Equal(t, "prod", doc.Envs[1].Name)
	require.EqualValues(t, doc.Envs[1].URL, "mysql://root:pass@:3306/ent")
	require.Equal(t, "staging", doc.Envs[2].Name)
	require.EqualValues(t, doc.Envs[2].URL, "atlasgo.io:443")
	require.Equal(t, "staging", doc.Envs[3].Name)
	require.EqualValues(t, doc.Envs[3].URL, "entgo.io:443")
	require.Equal(t, "dev", doc.Envs[4].Name)
	require.EqualValues(t, doc.Envs[4].URL, "atlasgo.io/atlas")
	require.Equal(t, "dev", doc.Envs[5].Name)
	require.EqualValues(t, doc.Envs[5].URL, "entgo.io/ent")

	// Mismatched element types.
	err := New().EvalBytes(b, &doc, map[string]cty.Value{
		"domains": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("a"),
				"port": cty.StringVal("b"),
			}),
		}),
	})
	require.EqualError(t, err, `variable "domains": a number is required`)
}

func TestDataLocalsRefs(t *testing.T) {
	var (
		opts = []Option{
			WithDataSource("sql", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
				attrs, diags := b.Body.JustAttributes()
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				v, diags := attrs["result"].Expr.Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				return cty.ObjectVal(map[string]cty.Value{"output": v}), nil
			}),
			WithDataSource("text", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
				attrs, diags := b.Body.JustAttributes()
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				v, diags := attrs["value"].Expr.Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				return cty.ObjectVal(map[string]cty.Value{"output": v}), nil
			}),
			WithInitBlock("atlas", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
				org, diags := b.Body.Attributes["org"].Expr.Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				if len(b.Body.Blocks) != 1 || b.Body.Blocks[0].Type != "auth" {
					return cty.NilVal, errors.New("expected auth block")
				}
				attrs, diags := b.Body.Blocks[0].Body.JustAttributes()
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				host, diags := attrs["host"].Expr.Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				return cty.ObjectVal(map[string]cty.Value{
					"org": org,
					"auth": cty.ObjectVal(map[string]cty.Value{
						"host": host,
					}),
				}), nil
			}),
			WithDataSource("remote_dir", func(ctx *hcl.EvalContext, b *hclsyntax.Block) (cty.Value, error) {
				attrs, diags := b.Body.JustAttributes()
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				host, diags := attrs["host"].Expr.Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				org, diags := (&hclsyntax.ScopeTraversalExpr{
					Traversal: hcl.Traversal{
						hcl.TraverseRoot{Name: "atlas", SrcRange: b.Range()},
						hcl.TraverseAttr{Name: "org", SrcRange: b.Range()},
					},
				}).Value(ctx)
				if diags.HasErrors() {
					return cty.NilVal, diags
				}
				return cty.ObjectVal(map[string]cty.Value{
					"url": cty.StringVal("atlas://" + path.Join(host.AsString(), org.AsString(), b.Labels[1])),
				}), nil
			}),
		}
		doc struct {
			Values []string `spec:"vs"`
		}
		b = []byte(`
variable "url" {
  type    = string
  default = "mysql://root:pass@:3306/atlas"
}

locals {
  a = "local-a"
  // locals can reference other locals.
  b = "local-b-ref-local-a: ${local.a}"
  // locals can reference data sources.
  c = "local-c-ref-data-a: ${data.text.a.output}"
  d = "local-d"
  host = "atlasgo.io"
}

data "sql" "tenants" {
  url = var.url
  // language=mysql
  query = <<EOS
SELECT schema_name
  FROM information_schema.schemata
  WHERE schema_name LIKE 'tenant_%'
EOS
  // fake result.
  result = "data-sql-tenants"
}

data "text" "a" {
  // data sources can reference data sources.
  value = "data-text-a-ref-data-sql-tenants: ${data.sql.tenants.output}"
}

data "text" "b" {
  // data sources can reference locals.
  value = "data-text-b-ref-local-d: ${local.d}"
}

atlas {
  org = "ent"
  auth {
    host = local.host
  }
}

data "remote_dir" "migrations" {
  host = atlas.auth.host
}

vs = [
  local.a,
  local.b,
  local.c,
  data.sql.tenants.output,
  data.text.a.output,
  data.text.b.output,
  data.remote_dir.migrations.url,
]
`)
	)
	require.NoError(t, New(opts...).EvalBytes(b, &doc, nil))
	require.Equal(t, []string{
		"local-a",
		"local-b-ref-local-a: local-a",
		"local-c-ref-data-a: data-text-a-ref-data-sql-tenants: data-sql-tenants",
		"data-sql-tenants",
		"data-text-a-ref-data-sql-tenants: data-sql-tenants",
		"data-text-b-ref-local-d: local-d",
		"atlas://atlasgo.io/ent/migrations",
	}, doc.Values)

	b = []byte(`locals { a = local.a }`)
	require.EqualError(t, New(opts...).EvalBytes(b, &doc, nil), `cyclic reference to "local.a"`)

	b = []byte(`
locals {
  a = "a"
  b = local.c
  c = local.b
}
`)
	require.Error(t, New(opts...).EvalBytes(b, &doc, nil), `cyclic reference to "local.c"`)

	b = []byte(`
data "text" "a" {
  value = local.a
}

locals {
  a = data.text.a.output
}
`)
	require.Error(t, New(opts...).EvalBytes(b, &doc, nil), `cyclic reference to "data.text.a"`)
}
