//go:build ignore
// +build ignore

package main

import (
	"io"
	"log"
	"os"
	"strings"
	"text/template"

	"ariga.io/atlas/cmd/action"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	f, err := os.Create("../../doc/md/cli/reference.md")
	if err != nil {
		log.Fatal(err)
	}
	t, err := template.New("").
		Funcs(template.FuncMap{
			"header": func(depth int) string {
				return strings.Repeat("#", depth+1)
			},
			"subheader": func(depth int) string {
				return strings.Repeat("#", depth+2)
			},
		}).
		ParseFiles("doc.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.ExecuteTemplate(f, "header", nil); err != nil {
		log.Fatal(err)
	}
	blocks := prepare(action.RootCmd, make([]*block, 0), 0)
	if err := t.ExecuteTemplate(f, "body", struct {
		Blocks []*block
	}{Blocks: blocks}); err != nil {
		log.Fatal(err)
	}
}

type block struct {
	Depth int
	*cobra.Command
}

func prepare(cmd *cobra.Command, existing []*block, depth int) []*block {
	if depth > 0 {
		existing = append(existing, &block{
			Depth:   depth,
			Command: cmd,
		})
	}
	for _, child := range cmd.Commands() {
		existing = prepare(child, existing, depth+1)
	}
	return existing
}

func markdown(cmd *cobra.Command, w io.Writer, i int) {
	if i > 0 {
		doc.GenMarkdown(cmd, w)
	}
	for _, child := range cmd.Commands() {
		markdown(child, w, i+1)
	}
}
