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

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/sqlite"
	"ariga.io/atlas/sql/sqlspec"
)

//go:embed output.tmpl
var tmpl string

type Driver struct {
	Name      string
	Types     []*schemaspec.TypeSpec
	Marshaler schemaspec.Marshaler
}

//go:generate go run main.go
func main() {
	drivers := []*Driver{
		{Name: "MySQL/MariaDB", Types: mysql.TypeRegistry.Specs(), Marshaler: mysql.MarshalHCL},
		{Name: "Postgres", Types: postgres.TypeRegistry.Specs(), Marshaler: postgres.MarshalHCL},
		{Name: "SQLite", Types: sqlite.TypeRegistry.Specs(), Marshaler: sqlite.MarshalHCL},
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
	f, err := os.Create("../../../doc/md/sql_types.md")
	defer f.Close()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	if err := parse.Execute(f, drivers); err != nil {
		log.Fatalf("error: %s", err)
	}
}

func colHcl(ts *schemaspec.TypeSpec, t []*schemaspec.TypeSpec) []string {
	dt := dummyType(ts)
	col := &sqlspec.Column{
		Name: "column",
		Type: dt,
	}
	spec, err := schemahcl.New(schemahcl.WithTypes(t)).MarshalSpec(col)
	if err != nil {
		log.Fatalf("failed: %s", err)
	}
	split := strings.Split(string(spec), "\n")
	td := []string{split[2]}
	for _, attr := range ts.Attributes {
		if attr.Name == "unsigned" {
			td = append(td, "unisgned = true")
		}
	}
	return td
}

func dummyType(ts *schemaspec.TypeSpec) *schemaspec.Type {
	spec := &schemaspec.Type{T: ts.T}
	for _, attr := range ts.Attributes {
		var a *schemaspec.Attr
		switch attr.Kind {
		case reflect.Int, reflect.Int64:
			n := "255"
			if attr.Name == "precision" {
				n = "10"
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
func StrAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: strconv.Quote(v)},
	}
}

// LitAttr is a helper method for constructing *schemaspec.Attr instances that contain literal values.
func LitAttr(k, v string) *schemaspec.Attr {
	return &schemaspec.Attr{
		K: k,
		V: &schemaspec.LiteralValue{V: v},
	}
}

// ListAttr is a helper method for constructing *schemaspec.Attr instances that contain list values.
func ListAttr(k string, litValues ...string) *schemaspec.Attr {
	lv := &schemaspec.ListValue{}
	for _, v := range litValues {
		lv.V = append(lv.V, &schemaspec.LiteralValue{V: v})
	}
	return &schemaspec.Attr{
		K: k,
		V: lv,
	}
}
