package main

import (
	"log"

	"ariga.io/atlas/cmd/internal/base"
	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTree(base.RootCmd, "./")
	if err != nil {
		log.Fatal(err)
	}
}
