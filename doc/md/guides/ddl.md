---
id: ddl
slug: /guides/ddl
title: Data Definition Language
---

## Introduction

In the core of the Atlas project resides the Atlas Data Definition Language (DDL). The DDL is designed to capture an
organization's data topologies and other aspects of its data infrastructure.

In the design of the DDL, we put an emphasis on extensibility: As data topologies can contain a set of diverse
data technologies, the language is designed to be modular with different extensions extending the types of
resources and relationships that can be described using it.

## HCL

The Atlas DDL currently supports an HCL syntax we call Atlas HCL. It is similar to other HCL based languages such as
Terraform.

### Resources

Documents written in the Atlas DDL usually describe _resources_. Resources are described as HCL blocks and have a type,
and optionally a name. Consider this block:

```hcl
user "rotemtam" {
   // ...
}
```

This block describes a resource of type `user` with a name of `rotemtam`.

### Qualifiers

In some cases, a document may contain multiple resources with the same name. To differentiate between
the different resources, the Atlas DDL supports _qualifiers_, an additional label preceding the resource
name:

```hcl
person "dr" "jekyll" {

}
```

This block describes a resource of type `person`, with a name of "jekyll" which is qualified by
"dr".

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
`<type>.<name>`, recursively. Suppose we have this block describing some HTTP service:

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

### Reading with Go

To read an Atlas HCL document with Go use the `EvalBytes` ([doc](https://pkg.go.dev/ariga.io/atlas/schemahcl#EvalBytes)) function
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
  err := EvalBytes([]byte(f), &test, nil)
  if err != nil {
    panic(err)
  }
  seinfeld := test.Shows[0]
  fmt.Printf("the show %q has %d writers.", seinfeld.Name, len(seinfeld.Writers))
  // Output: the show "seinfeld" has 2 writers.
}
```

This function takes a byte-slice, an empty interface as arguments and a map of strings as input.
The empty interface should be a  pointer to a struct into which the `EvalBytes` function will
read the values. The struct fields must be annotated with `spec` tags that define the mapping from HCL to the Go type.
This mapping is discussed in the section about [Extensions](#extensions). The final map argument may contain
[Input Values](../atlas-schema/hcl-input.md) to be passed as parameters of the evaluation.

### Writing with Go

To encode a Go struct back into HCL, use the `schemahcl.Marshal`
([doc](https://pkg.go.dev/ariga.io/atlas/schemahcl#Marshal)) function:

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

To operate correctly, struct extensions should be registered using the `schemahcl.Register`
function:

```go
schemahcl.Register("point", &Point{})
```

Extension structs may implement the [Remainer](https://pkg.go.dev/ariga.io/atlas/schemahcl#Remainer)
interface if they wish to store any attributes and children that are not matched by their
tagged fields. As a convenience the `schemahcl` package exports a `DefaultExtension` type that
can be embedded to support this behavior.

### Qualifiers

In cases where resources may need to be qualified, a field of the target struct can be annotated
with the `,qualifier` tag. For instance this struct:

```go
type Person struct {
  Name  string `spec:",name"`
  Title string `spec:",qualifier"`
}
```

Can capture a qualified HCL resource such as:

```hcl
person "dr" "jekyll" {

}
```
