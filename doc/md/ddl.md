---
id: ddl 
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

Observe that `Decode` returns a `schemaspec.Resource`. This Go type is
a generic container for resources described in the Atlas DDL. Applications
are not expected to work with it directly as it is not structured and 
not type-safe. In the section about [Extensions](#Extensions), we discuss the way applications can
work with data coming from Atlas HCL documents. 

### Writing with Go

To encode `schemaspec.Resource` instances back into HCL, use the `schemahcl.Encode` 
([doc](https://pkg.go.dev/ariga.io/atlas@v0.0.0-20211004124157-99ae6aaad16b/schema/schemaspec/schemahcl#Encode)) function:

```go
func TestSeinfeldEncode(t *testing.T) {
	r := &schemaspec.Resource{
		Children: []*schemaspec.Resource{
			{
				Name: "seinfeld",
				Type: "show",
			},
		},
	}
	encode, err := Encode(r)
	require.NoError(t, err)
	expected := `show "seinfeld" {
}
`
	require.EqualValues(t, expected, string(encode))
}
```

## Extensions

Applications working with `schemaspec` objects are expected to extend the Atlas language by
defining their own type structs that objects can be handled in a type-safe way. Resource
objects provide the `As` method to read a resource into an extension struct, as well as a
`Scan` method to read an extension struct back into a Resource.

The mapping between the extension struct fields and a Resource is done by placing tags on the
extension struct field using the `spec` key in the tag. To specify that a field should be mapped to
the corresponding Resource's `Name` specify ",name" to the tag value. For example,
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

### Reading from Resource
Reading the `schemaspec.Resource` into the extension struct is done using the `As` method.
For example:
```go
func TestPoint(t *testing.T) {
	f := `
point "A" {
	x = 100
	y = 100
}`
	decode, err := Decode([]byte(f))
	require.NoError(t, err)
	p := Point{}
	err = decode.Children[0].As(&p)
	require.NoError(t, err)
	expected := Point{
		ID: "A",
		X:  100,
		Y:  100,
	}
	require.EqualValues(t, expected, p)
}
```
### Writing to Resource

Going from the extension struct back into Resource form is possible using the `Scan` method:

```go

func TestPointScan(t *testing.T) {
  point := &Point{
    ID: "A",
    X:  100,
    Y:  100,
  }
  r := &schemaspec.Resource{}
  err := r.Scan(point)
  require.NoError(t, err)
  require.EqualValues(t, &schemaspec.Resource{
    Type: "point",
    Name: "A",
    Attrs: []*schemaspec.Attr{
      {K: "x", V: &schemaspec.LiteralValue{V: "100"}},
      {K: "y", V: &schemaspec.LiteralValue{V: "100"}},
    },
  }, r)
}
```