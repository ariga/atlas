---
id: ddl 
title: Data Definition Language
---

## Introduction

In the core of the Atlas project lies the Atlas Data Definition Language (DDL). The DDL is designed to capture an
organization's topologies and other aspects of its data infrastructure.

In the design of the DDL we took two important considerations:

* Extensible - As data topologies can contain a set of diverse data technologies, the language is designed to be modular
  with different extensions extending the types of resources and relationships that can be described using it.
* Decoupled from Syntax - The initial syntax for the language is based on HCL (v2).  
  It is planned that we will offer different syntaxes (Such as YAML, TypeScript or Go) in the future.

## HCL

The Atlas DDL currently supports an HCL syntax we call Atlas HCL. It is similar to other HCL based languages such as
TerraForm.

### Resources

Documents written in the Atlas DDL usually describe _resources_. Resources are described as HCL blocks and have a type,
and optionally a name. Consider this block:

```hcl
user "rotemtam" {
  // ...
}
```

This block describes a resource of type `user` with a name of `rotemtam`.

### Attributes

Resources can have named attributes with primitive types (string, boolean, integer or float)
or lists of primitive values.

For example:

```hcl
user "rotemtam" {
  email = "rotem@atlasgo.io"
  active = true
  credits = 42
  tags = [
    "data",
    "infrastructure",
    "hcl"]
}
```

### Children

Resources can have child resources. For example:

```hcl
user "rotemtam" {
  // ..
  project "atlas" {
    started = 2021
    metadata {
      category = "data"
    }
  }
}
```

### References

Attributes can hold references to other resources. The address of any resource is
`<type>.<name>`, recursively. Suppose we have this block describing some http service:

```hcl
service "todolist" {
  port "http" {
    number = 8080
  }
}
```

If we want to reference the child "port" resource of the service we can use
`service.todolist.port.http`:

```hcl
server "production" {
  endpoint "todo" {
    path = "/todo"
    service_port = service.todolist.port.http
  }
}
```

Attributes can hold references to other attributes. When a document is parsed the reference is replaced with the
referenced value. The address of any attribute is `<type>.<name>.<attr name>`.

```hcl
group "seinfeld" {
  id = 1
}
show "friends" {
  id = 2
}
playlist "comedy" {
  show_ids = [
    show.seinfeld.id
    // will equal 1
    show.friends.id,
    // will equal 2 
  ]
}
```

### Atlas HCL and Go

To read an Atlas HCL document with Go use the `Decode` function
from the `schemahcl` package:

```go
package hcl

import (
  "testing"
  "github.com/stretchr/testify/require"
  "ariga.io/atlas/schema/schemaspec"
  "ariga.io/atlas/schema/schemaspec/schemahcl"
)

func TestSeinfeld(t *testing.T) {
	f := `
  show "seinfeld" {
      writers = [
          "Jerry Seinfeld",
          "Larry David",
      ]
  }`

	s, err := schemahcl.Decode([]byte(f))
	require.NoError(t, err)
	seinfeld := s.Children[0]
	require.EqualValues(t, &schemaspec.Resource{
		Type: "show",
		Name: "seinfeld",
		Attrs: []*schemaspec.Attr{
			{
				K: "writers",
				V: &schemaspec.ListValue{
					V: []schemaspec.Value{
						&schemaspec.LiteralValue{V: `"Jerry Seinfeld"`},
						&schemaspec.LiteralValue{V: `"Larry David"`},
					},
				},
			},
		},
	}, seinfeld)
}
```