//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"path"

	"ariga.io/atlas/cmd/action"
	"github.com/spf13/cobra/doc"
)

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	d := path.Join(pwd, "../../doc/md/CLI")
	action.RootCmd.DisableAutoGenTag = true
	err = doc.GenMarkdownTree(action.RootCmd, d)
	if err != nil {
		log.Fatal(err)
	}
}
