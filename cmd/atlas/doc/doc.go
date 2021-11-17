//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"path"

	"ariga.io/atlas/cmd/internal/base"
	"github.com/spf13/cobra/doc"
)

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	d := path.Join(pwd, "../../../doc/md/CLI")
	base.RootCmd.DisableAutoGenTag = true
	err = doc.GenMarkdownTree(base.RootCmd, d)
	if err != nil {
		log.Fatal(err)
	}
}
