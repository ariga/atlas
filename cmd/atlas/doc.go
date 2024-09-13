// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/s-sokolko/atlas/cmd/atlas/internal/cmdapi"

	"github.com/spf13/cobra"
)

func main() {
	f, err := os.Create("../../doc/md/reference.md")
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
	blocks := prepare(cmdapi.Root, make([]*block, 0), 0)
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
		if !child.Hidden {
			existing = prepare(child, existing, depth+1)
		}
	}
	return existing
}
