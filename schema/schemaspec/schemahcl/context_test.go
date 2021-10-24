package schemahcl

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec"
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
	type (
		Backend struct {
			Name  string `spec:",name"`
			Image string `spec:"image"`
			Addr  string `spec:"addr"`
		}
		Endpoint struct {
			Name      string `spec:",name"`
			Path      string `spec:"path"`
			Addr      string `spec:"addr"`
			TimeoutMs int    `spec:"timeout_ms"`
			Retry     bool   `spec:"retry"`
			Desc      string `spec:"description"`
		}
	)
	var test struct {
		Backends  []*Backend  `spec:"backend"`
		Endpoints []*Endpoint `spec:"endpoint"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, []*Endpoint{
		{
			Name:      "home",
			Path:      "/",
			Addr:      "127.0.0.1:8081",
			Retry:     false,
			TimeoutMs: 10,
			Desc:      "default: generic",
		},
		{
			Name: "admin",
			Path: "/admin",
			Addr: "127.0.0.1:8082",
		},
	}, test.Endpoints)
}

func TestUnlabeledBlockReferences(t *testing.T) {
	f := `
country "israel" {
    metadata {
        phone_prefix = "972"
    }
    metadata "geo" {
		continent = "asia"
    }
}

metadata  = country.israel.metadata.0
phone_prefix = country.israel.metadata.0.phone_prefix
continent = country.israel.metadata.geo.continent
`
	type (
		Metadata struct {
			PhonePrefix string `spec:"phone_prefix"`
			Continent   string `spec:"continent"`
		}
		Country struct {
			Metadata []*Metadata `spec:"metadata"`
		}
	)
	var test struct {
		Countries   []*Country      `spec:"country"`
		MetadataRef *schemaspec.Ref `spec:"metadata"`
		PhonePrefix string          `spec:"phone_prefix"`
		Continent   string          `spec:"continent"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, "972", test.PhonePrefix)
	require.EqualValues(t, "asia", test.Continent)
	require.EqualValues(t, "$country.israel.$metadata.0", test.MetadataRef.V)
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
	type (
		City struct {
			Name          string `spec:",name"`
			PhoneAreaCode string `spec:"phone_area_code"`
		}
		Country struct {
			Name   string  `spec:",name"`
			Cities []*City `spec:"city"`
		}
	)
	var test struct {
		Countries []*Country `spec:"country"`
	}
	err := Unmarshal([]byte(f), &test)
	israel := &Country{
		Name: "israel",
		Cities: []*City{
			{Name: "tel_aviv", PhoneAreaCode: "03"},
			{Name: "jerusalem", PhoneAreaCode: "02"},
			{Name: "givatayim", PhoneAreaCode: "03"},
		},
	}
	require.NoError(t, err)
	require.EqualValues(t, israel, test.Countries[0])
}

func TestBlockReference(t *testing.T) {
	f := `person "jon" {
}
pet "garfield" {
  type  = "cat"
  owner = person.jon
}
`
	type (
		Person struct {
			Name string `spec:",name"`
		}
		Pet struct {
			Name  string          `spec:",name"`
			Type  string          `spec:"type"`
			Owner *schemaspec.Ref `spec:"owner"`
		}
	)
	var test struct {
		People []*Person `spec:"person"`
		Pets   []*Pet    `spec:"pet"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, &Pet{
		Name:  "garfield",
		Type:  "cat",
		Owner: &schemaspec.Ref{V: "$person.jon"},
	}, test.Pets[0])
	marshal, err := Marshal(&test)
	require.NoError(t, err)
	require.EqualValues(t, f, string(marshal))
}

func TestListRefs(t *testing.T) {
	f := `
user "simba" {
	
}
user "mufasa" {

}
group "lion_kings" {
	members = [
		user.simba,
		user.mufasa,
	]
}
`
	type (
		User struct {
			Name string `spec:",name"`
		}
		Group struct {
			Name    string            `spec:",name"`
			Members []*schemaspec.Ref `spec:"members"`
		}
	)
	var test struct {
		Users  []*User  `spec:"user"`
		Groups []*Group `spec:"group"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	require.EqualValues(t, &Group{
		Name: "lion_kings",
		Members: []*schemaspec.Ref{
			{V: "$user.simba"},
			{V: "$user.mufasa"},
		},
	}, test.Groups[0])
}

func TestNestedDifference(t *testing.T) {
	f := `
person "john" {
	nickname = "jonnie"
	hobby "hockey" {
		active = true
	}
}
person "jane" {
	nickname = "janie"
	hobby "football" {
		budget = 1000
	}
	car "ferrari" {
		year = 1960
	}
}
`
	type (
		Hobby struct {
			Name   string `spec:",name"`
			Active bool   `spec:"active"`
			Budget int    `spec:"budget"`
		}
		Car struct {
			Name string `spec:",name"`
			Year int    `spec:"year"`
		}
		Person struct {
			Name     string   `spec:",name"`
			Nickname string   `spec:"nickname"`
			Hobbies  []*Hobby `spec:"hobby"`
			Car      *Car     `spec:"car"`
		}
	)
	var test struct {
		People []*Person `spec:"person"`
	}
	err := Unmarshal([]byte(f), &test)
	require.NoError(t, err)
	john := &Person{
		Name:     "john",
		Nickname: "jonnie",
		Hobbies: []*Hobby{
			{Name: "hockey", Active: true},
		},
	}
	require.EqualValues(t, john, test.People[0])
	jane := &Person{
		Name:     "jane",
		Nickname: "janie",
		Hobbies: []*Hobby{
			{Name: "football", Budget: 1000},
		},
		Car: &Car{
			Name: "ferrari",
			Year: 1960,
		},
	}
	require.EqualValues(t, jane, test.People[1])
}
