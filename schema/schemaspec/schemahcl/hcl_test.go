package schemahcl

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"

	"github.com/stretchr/testify/require"
)

func TestAttributes(t *testing.T) {
	f := `
i = 1
b = true
s = "hello, world"
arr = ["yada", "yada", "yada"]
`
	resource, err := Decode([]byte(f))
	require.NoError(t, err)
	require.Len(t, resource.Attrs, 4)

	attr, ok := resource.Attr("i")
	require.True(t, ok)
	i, err := attr.Int()
	require.NoError(t, err)
	require.EqualValues(t, 1, i)

	attr, ok = resource.Attr("b")
	require.True(t, ok)
	b, err := attr.Bool()
	require.NoError(t, err)
	require.EqualValues(t, true, b)

	attr, ok = resource.Attr("s")
	require.True(t, ok)
	s, err := attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "hello, world", s)

	attr, ok = resource.Attr("arr")
	require.True(t, ok)
	arr, err := attr.Strings()
	require.NoError(t, err)
	require.EqualValues(t, []string{"yada", "yada", "yada"}, arr)
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
	resource, err := Decode([]byte(f))
	require.NoError(t, err)
	require.Len(t, resource.Children, 1)
	expected := &schemaspec.Resource{
		Name: "/hello",
		Type: "endpoint",
		Children: []*schemaspec.Resource{
			{
				Type: "handler",
				Attrs: []*schemaspec.Attr{
					{K: "active", V: &schemaspec.LiteralValue{V: `true`}},
					{K: "addr", V: &schemaspec.LiteralValue{V: `":8080"`}},
				},
			},
		},
		Attrs: []*schemaspec.Attr{
			{K: "description", V: &schemaspec.LiteralValue{V: `"the hello handler"`}},
			{K: "timeout_ms", V: &schemaspec.LiteralValue{V: `100`}},
		},
	}
	require.EqualValues(t, expected, resource.Children[0])
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
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			resource, err := Decode([]byte(tt.Body))
			require.NoError(t, err)
			bytes, err := encode(resource)
			require.NoError(t, err)
			again, err := Decode(bytes)
			require.NoError(t, err)
			require.EqualValues(t, resource, again, "expected resource to be the same after encoding")
		})
	}
}
