package schemahcl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributes(t *testing.T) {
	f := `
i = 1
b = true
s = "hello, world"
`
	var test struct {
		Int  int    `spec:"i"`
		Bool bool   `spec:"b"`
		Str  string `spec:"s"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, 1, test.Int)
	require.EqualValues(t, true, test.Bool)
	require.EqualValues(t, "hello, world", test.Str)
}

func TestResource(t *testing.T) {
	f := `
endpoint "/hello" {
	handler {
		active = true
		addr = ":8080"
	}
	description = "the hello handler"
	timeout_ms = 100
}
`
	type Handler struct {
		Active bool   `spec:"active"`
		Addr   string `spec:"addr"`
	}
	type Endpoint struct {
		Name        string   `spec:",name"`
		Description string   `spec:"description"`
		TimeoutMs   int      `spec:"timeout_ms"`
		Handler     *Handler `spec:"handler"`
	}
	type File struct {
		Endpoints []*Endpoint `spec:"endpoint"`
	}
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
}

func TestReEncode(t *testing.T) {
	testCases := []struct {
		Name, Body string
	}{
		{
			Name: "attr",
			Body: `year = 2021`,
		},
		{
			Name: "simple resource",
			Body: `project "atlas" {
	started_year = 2021
	useful = true
}`,
		},
		{
			Name: "nested resource",
			Body: `author "rotemtam" {
	package "schemahcl" {
		lang = "go"
	}
}`,
		},
		{
			Name: "block reference",
			Body: `user "rotemtam" {
}
task "code" {
	owner = user.rotemtam
}
`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			resource, err := decode([]byte(tt.Body))
			require.NoError(t, err)
			bytes, err := Encode(resource)
			require.NoError(t, err)
			again, err := decode(bytes)
			require.NoError(t, err)
			require.EqualValues(t, resource, again, "expected resource to be the same after encoding")
		})
	}
}
