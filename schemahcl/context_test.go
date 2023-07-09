// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
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
	err := New().EvalBytes([]byte(f), &test, nil)
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
    metadata {
        phone_prefix = "123"
    }
    metadata "geo" {
		continent = "asia"
    }
}

metadata  = country.israel.metadata.0
phone_prefix = country.israel.metadata.0.phone_prefix
phone_prefix_2 = country.israel.metadata.1.phone_prefix
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
		Test struct {
			Countries    []*Country `spec:"country"`
			MetadataRef  *Ref       `spec:"metadata"`
			PhonePrefix  string     `spec:"phone_prefix"`
			PhonePrefix2 string     `spec:"phone_prefix_2"`
			Continent    string     `spec:"continent"`
		}
	)
	var test Test
	err := New().EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, test, Test{
		Countries: []*Country{
			{
				Metadata: []*Metadata{
					{PhonePrefix: "972"},
					{PhonePrefix: "123"},
					{Continent: "asia"},
				},
			},
		},
		MetadataRef:  &Ref{V: "$country.israel.$metadata.0"},
		PhonePrefix:  "972",
		PhonePrefix2: "123",
		Continent:    "asia",
	})
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
	err := New().EvalBytes([]byte(f), &test, nil)
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
			Name  string `spec:",name"`
			Type  string `spec:"type"`
			Owner *Ref   `spec:"owner"`
		}
	)
	var test struct {
		People []*Person `spec:"person"`
		Pets   []*Pet    `spec:"pet"`
	}
	err := New().EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, &Pet{
		Name:  "garfield",
		Type:  "cat",
		Owner: &Ref{V: "$person.jon"},
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
			Name    string `spec:",name"`
			Members []*Ref `spec:"members"`
		}
	)
	var test struct {
		Users  []*User  `spec:"user"`
		Groups []*Group `spec:"group"`
	}
	err := New().EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, &Group{
		Name: "lion_kings",
		Members: []*Ref{
			{V: "$user.simba"},
			{V: "$user.mufasa"},
		},
	}, test.Groups[0])
	_, err = Marshal(&test)
	require.NoError(t, err)
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
	err := New().EvalBytes([]byte(f), &test, nil)
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

func TestSchemaRefParse(t *testing.T) {
	type Point struct {
		Z []*Ref `spec:"z"`
	}
	var test = struct {
		Points []*Point `spec:"point"`
	}{
		Points: []*Point{
			{Z: []*Ref{{V: "$point.1"}}},
			{Z: []*Ref{{V: "$point.0"}}},
		},
	}
	b, err := Marshal(&test)
	require.NoError(t, err)
	expected :=
		`point {
  z = [point.1]
}
point {
  z = [point.0]
}
`
	require.Equal(t, expected, string(b))
}

func TestWithTypes(t *testing.T) {
	f := `parent "name" {
  child "name" {
    first    = int
    second   = bool
    third    = int(10)
    sized    = varchar(255)
    variadic = enum("a","b","c")
  }
}
`
	s := New(
		WithTypes(
			"parent.child",
			[]*TypeSpec{
				{Name: "bool", T: "bool"},
				{
					Name: "int",
					T:    "int",
					Attributes: []*TypeAttr{
						{Name: "size", Kind: reflect.Int, Required: false},
						{Name: "unsigned", Kind: reflect.Bool, Required: false},
					},
				},
				{
					Name: "varchar",
					T:    "varchar",
					Attributes: []*TypeAttr{
						{Name: "size", Kind: reflect.Int, Required: false},
					},
				},
				{
					Name: "enum",
					T:    "enum",
					Attributes: []*TypeAttr{
						{Name: "values", Kind: reflect.Slice, Required: false},
					},
				},
			},
		),
	)
	var test struct {
		Parent struct {
			Name  string `spec:",name"`
			Child struct {
				Name     string `spec:",name"`
				First    *Type  `spec:"first"`
				Second   *Type  `spec:"second"`
				Third    *Type  `spec:"third"`
				Varchar  *Type  `spec:"sized"`
				Variadic *Type  `spec:"variadic"`
			} `spec:"child"`
		} `spec:"parent"`
	}
	err := s.EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, "int", test.Parent.Child.First.T)
	require.EqualValues(t, "bool", test.Parent.Child.Second.T)

	require.EqualValues(t, "varchar", test.Parent.Child.Varchar.T)
	require.Len(t, test.Parent.Child.Varchar.Attrs, 1)
	i, err := test.Parent.Child.Varchar.Attrs[0].Int()
	require.NoError(t, err)
	require.EqualValues(t, 255, i)

	require.EqualValues(t, "enum", test.Parent.Child.Variadic.T)
	require.Len(t, test.Parent.Child.Variadic.Attrs, 1)
	vs, err := test.Parent.Child.Variadic.Attrs[0].Strings()
	require.NoError(t, err)
	require.EqualValues(t, []string{"a", "b", "c"}, vs)

	require.EqualValues(t, "int", test.Parent.Child.Third.T)
	require.Len(t, test.Parent.Child.Third.Attrs, 1)
	i, err = test.Parent.Child.Third.Attrs[0].Int()
	require.NoError(t, err)
	require.EqualValues(t, 10, i)

	after, err := s.MarshalSpec(&test)
	require.NoError(t, err)
	require.EqualValues(t, f, string(after))
}

func TestEmptyStrSQL(t *testing.T) {
	s := New(WithTypes("", nil))
	h := `x = sql("")`
	err := s.EvalBytes([]byte(h), &struct{}{}, nil)
	require.ErrorContains(t, err, "empty expression")
}

func TestOptionalArgs(t *testing.T) {
	s := New(
		WithTypes("block", []*TypeSpec{
			{
				T:    "float",
				Name: "float",
				Attributes: []*TypeAttr{
					PrecisionTypeAttr(),
					ScaleTypeAttr(),
				},
			},
		}),
	)
	f := `
block "name" {
  arg_0 = float
  arg_1 = float(10)
  arg_2 = float(10,2)
}
`
	var test struct {
		Block struct {
			Name string `spec:",name"`
			Arg0 *Type  `spec:"arg_0"`
			Arg1 *Type  `spec:"arg_1"`
			Arg2 *Type  `spec:"arg_2"`
		} `spec:"block"`
	}
	err := s.EvalBytes([]byte(f), &test, nil)
	require.NoError(t, err)
	require.Nil(t, test.Block.Arg0.Attrs)

	require.Len(t, test.Block.Arg1.Attrs, 1)
	require.Equal(t, "precision", test.Block.Arg1.Attrs[0].K)
	i, err := test.Block.Arg1.Attrs[0].Int()
	require.NoError(t, err)
	require.EqualValues(t, 10, i)

	require.Len(t, test.Block.Arg2.Attrs, 2)
	require.Equal(t, "precision", test.Block.Arg2.Attrs[0].K)
	i, err = test.Block.Arg2.Attrs[0].Int()
	require.NoError(t, err)
	require.EqualValues(t, 10, i)
	require.Equal(t, "scale", test.Block.Arg2.Attrs[1].K)
	i, err = test.Block.Arg2.Attrs[1].Int()
	require.NoError(t, err)
	require.EqualValues(t, 2, i)
}

func TestQualifiedRefs(t *testing.T) {
	h := `user "atlas" "cli" {
	version = "v0.3.9"
}
v = user.atlas.cli.version
r = user.atlas.cli
`
	var test struct {
		V string `spec:"v"`
		R *Ref   `spec:"r"`
	}
	err := New().EvalBytes([]byte(h), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, "v0.3.9", test.V)
	require.EqualValues(t, "$user.atlas.cli", test.R.V)
}

func TestQuotedRefs(t *testing.T) {
	h := `
user "ariel.mashraki" {
  username = "a8m"
}
account "ariga.cloud" "a8m.dev" {
  org = "dev"
}
env "dev" "a8m.dev" {}
env "prod.1" "a8m" {}

v = user["ariel.mashraki"].username
r = user["ariel.mashraki"]
vs = [account["ariga.cloud"]["a8m.dev"].org]
rs = [
	account["ariga.cloud"]["a8m.dev"],
	env["dev"]["a8m.dev"],
	env["prod.1"]["a8m"],
]
`
	var test struct {
		V  string   `spec:"v"`
		R  *Ref     `spec:"r"`
		Vs []string `spec:"vs"`
		Rs []*Ref   `spec:"rs"`
	}
	err := New().EvalBytes([]byte(h), &test, nil)
	require.NoError(t, err)
	require.EqualValues(t, "a8m", test.V)
	require.EqualValues(t, `$user["ariel.mashraki"]`, test.R.V)
	require.EqualValues(t, []string{"dev"}, test.Vs)
	require.EqualValues(t, []*Ref{
		{`$account["ariga.cloud"]["a8m.dev"]`},
		{`$env.dev["a8m.dev"]`},
		{`$env["prod.1"].a8m`},
	}, test.Rs)
}

func TestInputValues(t *testing.T) {
	h := `
variable "name" {
  type = string
}

variable "default" {
  type = string
  default = "hello"
}

variable "int" {
  type = int
}

variable "bool" {
  type = bool
}

variable "convert_int" {
  type = int
}

variable "convert_bool" {
  type = bool
}

variable "strings" {
  type = list(string)
  default = ["a", "b"]
  description = "description is a valid attribute"
}

name = var.name
default = var.default
int = var.int
bool = var.bool
convert_int = var.convert_int
convert_bool = var.convert_bool
strings = var.strings
`
	var test struct {
		Name        string   `spec:"name"`
		Default     string   `spec:"default"`
		Int         int      `spec:"int"`
		Bool        bool     `spec:"bool"`
		ConvertInt  int      `spec:"convert_int"`
		ConvertBool bool     `spec:"convert_bool"`
		Strings     []string `spec:"strings"`
	}
	err := New().EvalBytes([]byte(h), &test, map[string]cty.Value{
		"name":         cty.StringVal("rotemtam"),
		"int":          cty.NumberIntVal(42),
		"bool":         cty.BoolVal(true),
		"convert_int":  cty.StringVal("1"),
		"convert_bool": cty.StringVal("true"),
		"strings":      cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}),
	})
	require.NoError(t, err)
	require.EqualValues(t, "rotemtam", test.Name)
	require.EqualValues(t, "hello", test.Default)
	require.EqualValues(t, 42, test.Int)
	require.EqualValues(t, true, test.Bool)
	require.EqualValues(t, 1, test.ConvertInt)
	require.EqualValues(t, true, test.ConvertBool)
	require.EqualValues(t, []string{"a", "b"}, test.Strings)
}

func TestVariable_InvalidType(t *testing.T) {
	h := `
variable "name" {
  type = "int"
  default = "boring"
}`
	err := New().EvalBytes([]byte(h), &struct{}{}, nil)
	require.EqualError(t, err, `invalid type "int" for variable "name". Valid types are: string, number, bool, list, map, or set`)

	h = `
variable "name" {
  type = "boring"
  default = "boring"
}`
	err = New().EvalBytes([]byte(h), &struct{}{}, nil)
	require.EqualError(t, err, `invalid type "boring" for variable "name". Valid types are: string, number, bool, list, map, or set`)
}

func TestTemplateReferences(t *testing.T) {
	var (
		d struct {
			Stmt1 string `spec:"stmt1"`
			Stmt2 string `spec:"stmt2"`
		}
		b = []byte(`
table "foo" {}

table "bar" {
  name = "baz"
  column "id" {
    type = int
  }
}

stmt1 = <<-SQL
   SELECT * FROM ${table.foo.name}
  SQL
stmt2 = <<-SQL
   SELECT ${table.bar.column.id.name} FROM ${table.bar.name}
  SQL
`)
	)
	require.NoError(t, New().EvalBytes(b, &d, nil))
	require.Equal(t, "SELECT * FROM foo", strings.TrimSpace(d.Stmt1))
	require.Equal(t, "SELECT id FROM baz", strings.TrimSpace(d.Stmt2))

	err := New().EvalBytes([]byte(`v = "${unknown}"`), &d, nil)
	require.EqualError(t, err, `:1,8-15: Unknown variable; There is no variable named "unknown".`)
}
