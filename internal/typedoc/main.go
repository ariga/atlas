// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	_ "embed"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/sqlspec"
)

//go:embed output.tmpl
var tmpl string

type (
	Driver struct {
		Name  string
		Types []Type
	}
	Type struct {
		*schemahcl.TypeSpec
		Info            string
		MarshalOverride string
	}
)

//go:generate go run main.go
func main() {
	drivers := []*Driver{
		{Name: "MySQL/MariaDB", Types: wrap(mysql.TypeRegistry.Specs())},
		{Name: "Postgres", Types: append(wrap(postgres.TypeRegistry.Specs()), Type{
			TypeSpec: &schemahcl.TypeSpec{
				Name: "enum",
				T:    "my_enum",
			},
			Info: `In Postgres an enum type is created as a custom type and can then be referenced in a column 
definition. Therefore, you have to add an enum block to your HCL schema like below:
<pre>
enum "my_enum" &#123;
	values = ["on", "off"]
&#125;
</pre>`,
			MarshalOverride: "enum = enum.my_enum",
		})},
		{Name: "SQLite", Types: wrap(sqlite.TypeRegistry.Specs())},
	}
	parse, err := template.New("tmp").Funcs(template.FuncMap{
		"col_hcl": colHcl,
	}).Parse(tmpl)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	for _, drv := range drivers {
		sort.Slice(drv.Types, func(i, j int) bool {
			return drv.Types[i].Name < drv.Types[j].Name
		})
	}
	f, err := os.Create("../../doc/md/ddl/sql_types.md")
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	defer f.Close()
	if err := parse.Execute(f, drivers); err != nil {
		log.Fatalf("error: %s", err)
	}
}

func colHcl(ts *Type, d *Driver) []string {
	td := ts.MarshalOverride
	if ts.MarshalOverride == "" {
		dt := dummyType(ts.TypeSpec)
		col := &sqlspec.Column{
			Name: "column",
			Type: dt,
		}
		spec, err := schemahcl.New(schemahcl.WithTypes(unwrap(d.Types))).MarshalSpec(col)
		if err != nil {
			log.Fatalf("failed: %s", err)
		}
		split := strings.Split(string(spec), "\n")
		td = split[1]
	}
	res := []string{td}
	for _, attr := range ts.Attributes {
		if attr.Name == "unsigned" {
			res = append(res, "unsigned = true")
		}
	}
	return res
}

func dummyType(ts *schemahcl.TypeSpec) *schemahcl.Type {
	spec := &schemahcl.Type{T: ts.T}
	for _, attr := range ts.Attributes {
		var a *schemahcl.Attr
		switch attr.Kind {
		case reflect.Int, reflect.Int64:
			n := "255"
			if attr.Name == "precision" {
				n = "6"
			}
			if attr.Name == "scale" {
				n = "2"
			}
			a = LitAttr(attr.Name, n)
		case reflect.String:
			a = LitAttr(attr.Name, `"a"`)
		case reflect.Slice:
			a = ListAttr(attr.Name, `"a"`, `"b"`)
		case reflect.Bool:
			a = LitAttr(attr.Name, "false")
		default:
			log.Fatalf("unsupported kind: %s", attr.Kind)
		}
		spec.Attrs = append(spec.Attrs, a)
	}
	return spec
}

// StrAttr is a helper method for constructing *schemaspec.Attr of type string.
func StrAttr(k, v string) *schemahcl.Attr {
	return &schemahcl.Attr{
		K: k,
		V: &schemahcl.LiteralValue{V: strconv.Quote(v)},
	}
}

// LitAttr is a helper method for constructing *schemaspec.Attr instances that contain literal values.
func LitAttr(k, v string) *schemahcl.Attr {
	return &schemahcl.Attr{
		K: k,
		V: &schemahcl.LiteralValue{V: v},
	}
}

// ListAttr is a helper method for constructing *schemaspec.Attr instances that contain list values.
func ListAttr(k string, litValues ...string) *schemahcl.Attr {
	lv := &schemahcl.ListValue{}
	for _, v := range litValues {
		lv.V = append(lv.V, &schemahcl.LiteralValue{V: v})
	}
	return &schemahcl.Attr{
		K: k,
		V: lv,
	}
}

// wrap iterates over the given slice of schemaspec.TypeSpec and wraps them with Type.
func wrap(tss []*schemahcl.TypeSpec) []Type {
	res := make([]Type, len(tss))
	for i, ts := range tss {
		res[i] = Type{TypeSpec: ts}
	}
	return res
}

// unwrap undoes wrap.
func unwrap(tss []Type) []*schemahcl.TypeSpec {
	res := make([]*schemahcl.TypeSpec, len(tss))
	for i, ts := range tss {
		res[i] = ts.TypeSpec
	}
	return res
}
