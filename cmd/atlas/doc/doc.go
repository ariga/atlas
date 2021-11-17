//go:build ignore
// +build ignore

package main

import (
	"fmt"
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
	d := path.Join(pwd, "../../../doc/md/cli")
	fmt.Println(d)
	err = doc.GenMarkdownTree(base.RootCmd, d)
	if err != nil {
		log.Fatal(err)
	}
}
