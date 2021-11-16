package schemahcl

import (
	"fmt"
	"log"
	"testing"

	"ariga.io/atlas/schema/schemaspec"
	"github.com/stretchr/testify/require"
)

func TestAttributes(t *testing.T) {
	f := `i  = 1
b  = true
s  = "hello, world"
sl = ["hello", "world", ]
bl = [true, false, ]
`
	var test struct {
		Int        int      `spec:"i"`
		Bool       bool     `spec:"b"`
		Str        string   `spec:"s"`
		StringList []string `spec:"sl"`
		BoolList   []bool   `spec:"bl"`
	}
	err := Unmarshal([]byte(f), &test)
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
  }
}
`
	type (
		Handler struct {
			Active bool   `spec:"active"`
			Addr   string `spec:"addr"`
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
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.Len(t, test.Endpoints, 1)
	expected := &Endpoint{
		Name:        "/hello",
		Description: "the hello handler",
		TimeoutMs:   100,
		Handler: &Handler{
			Active: true,
			Addr:   ":8080",
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
	type (
		Animal interface {
			animal()
		}
		Parrot struct {
			Animal
			Boss string `spec:"boss"`
		}
		Lion struct {
			Animal
			Friend string `spec:"friend"`
		}
		Zoo struct {
			Animals []Animal `spec:",interface"`
		}
	)
	schemaspec.Register("lion", &Lion{})
	schemaspec.Register("parrot", &Parrot{})
	var test Zoo
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, Zoo{
		Animals: []Animal{
			&Lion{
				Friend: "rafiki",
			},
			&Parrot{
				Boss: "jafar",
			},
		},
	}, test)
}
