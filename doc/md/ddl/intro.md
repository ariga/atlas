---
id: ddl-intro
slug: /ddl/intro
title: Data Definition Language
---

## Introduction

In the core of the Atlas project resides the Atlas Data Definition Language (DDL). The DDL is designed to capture an
organization's data topologies and other aspects of its data infrastructure.

In the design of the DDL we took two important considerations:

* Extensible - As data topologies can contain a set of diverse data technologies, the language is designed to be modular
  with different extensions extending the types of resources and relationships that can be described using it.
* Decoupled from Syntax - The initial syntax for the language is based on HCL (v2). Additional 
syntaxes are (such as YAML, TypeScript or Go) in the future.

## HCL

The Atlas DDL currently supports an HCL syntax we call Atlas HCL. It is similar to other HCL based languages such as
Terraform.

### Resources

Documents written in the Atlas DDL usually describe _resources_. Resources are described as HCL blocks and have a type,
and optionally a name. Consider this block:

```hcl
user "rotemtam" {
   ...
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
    "hcl"
  ]
}
```

### Children

Resources can have child resources. For example:

```hcl
user "rotemtam" {
   ..
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
     will equal 1
    show.friends.id,
     will equal 2 
  ]
}
```

### Reading with Go

To read an Atlas HCL document with Go use the `Unmarshal` ([doc](https://pkg.go.dev/ariga.io/atlas/schema/schemaspec/schemahcl#Unmarshal)) function
from the `schemahcl` package:

```go
func ExampleUnmarshal() {
    f := `
show "seinfeld" {
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
        Writers []*Writer `spec:"writer"`
      }
    )
  var test struct {
    Shows []*Show `spec:"show"`
  }
  err := Unmarshal([]byte(f), &test)
  if err != nil {
    panic(err)
  }
  seinfeld := test.Shows[0]
  fmt.Printf("the show %q has %d writers.", seinfeld.Name, len(seinfeld.Writers))
  // Output: the show "seinfeld" has 2 writers.
}

```

Observe that similar to the standard-library's `json.Unmarshal` function, this function
takes as arguments a byte-slice and an empty interface.  The empty interface should be a
pointer to a struct into which the `Unmarshal` function will read the values. The struct
fields must be annotated with `spec` tags that define the mapping from HCL to the Go type.
This mapping is discussed in the section about [Extensions](#Extensions).

### Writing with Go

To encode a Go struct back into HCL, use the `schemahcl.Marshal`
([doc](https://pkg.go.dev/ariga.io/atlas/schema/schemaspec/schemahcl#Marshal)) function:

```go
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
    panic(err)
  }
  fmt.Println(string(b))
  // Output: point "start" {
  //   x = 0
  //   y = 0
  // }
  // point "end" {
  //   x = 1
  //   y = 1
  // }
}
```

## Extensions

Applications working with the Atlas DDL are expected to extend the Atlas language by
defining their own type structs that objects can be handled in a type-safe way. 

The mapping between the extension struct fields and the configuration syntax is done by placing tags on the
extension struct field using the `spec` key in the tag. To specify that a field should be mapped to
the corresponding resource's name specify ",name" to the tag value. For example,
```go
type Point struct {
    ID string `spec:",name"`
    X  int    `spec:"x"`
    Y  int    `spec:"y"`
}
```

Would be able to capture a Resource defined in Atlas HCL as:

```hcl
point "origin" {
  x = 100
  y = 200
}
```

To operate correctly, struct extensions should be registered using the `schemaspec.Register`
function:

```go
schemaspec.Register("point", &Point{})
```

Extension structs may implement the [Remainer](https://pkg.go.dev/ariga.io/atlas/schema/schemaspec#Remainer)
interface if they wish to store any attributes and children that are not matched by their
tagged fields. As a convenience the `schemaspec` package exports a `DefaultExtension` type that
can be embedded to support this behavior.
