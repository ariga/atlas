package schemahcl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferences(t *testing.T) {
	f := `
backend "app" {
	image = "ariga/app:1.2.3"
	addr = "127.0.0.1:8081"
}
backend "admin" {
	image = "ariga/admin:1.2.3"
	addr = "127.0.0.1:8082"
}
endpoint "home" {
	path = "/"
	addr = backend.app.addr
	timeout_ms = config.defaults.timeout_ms
	retry = config.defaults.retry
	description = "default: ${config.defaults.description}"
}
endpoint "admin" {
	path = "/admin"
	addr = backend.admin.addr
}
config "defaults" {
	timeout_ms = 10
	retry = false
	description = "generic"
}
`
	res, err := Decode([]byte(f))
	require.NoError(t, err)
	home := res.Children[2]
	attr, ok := home.Attr("addr")
	require.True(t, ok)
	s, err := attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "127.0.0.1:8081", s)

	attr, ok = home.Attr("timeout_ms")
	require.True(t, ok)
	timeoutMs, err := attr.Int()
	require.NoError(t, err)
	require.EqualValues(t, 10, timeoutMs)

	attr, ok = home.Attr("retry")
	require.True(t, ok)
	retry, err := attr.Bool()
	require.NoError(t, err)
	require.EqualValues(t, false, retry)

	attr, ok = home.Attr("description")
	require.True(t, ok)
	interpolated, err := attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "default: generic", interpolated)
}

func TestNestedReferences(t *testing.T) {
	f := `
country "israel" {
	city "tel_aviv" {
		phone_area_code = "03"
	}
	city "jerusalem" {
		phone_area_code = "02"
	}
	city "givatayim" {
		phone_area_code = country.israel.city.tel_aviv.phone_area_code
	}
}
`
	res, err := Decode([]byte(f))
	require.NoError(t, err)
	israel := res.Children[0]
	givatyaim := israel.Children[2]
	attr, ok := givatyaim.Attr("phone_area_code")
	require.True(t, ok)
	s, err := attr.String()
	require.NoError(t, err)
	require.EqualValues(t, "03", s)
}

func TestBlockReference(t *testing.T) {
	f := `
person "jon" {
	
}
pet "garfield" {
	type = "cat"
	owner = person.jon
}
`
	res, err := Decode([]byte(f))
	require.NoError(t, err)
	garfield := res.Children[1]
	attr, ok := garfield.Attr("owner")
	require.True(t, ok)
	ref, err := attr.Ref()
	require.NoError(t, err)
	require.EqualValues(t, "/person/jon", ref)
}
