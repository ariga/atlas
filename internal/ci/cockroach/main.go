// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package main

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"
)

type params struct {
	Version string
}

//go:embed Dockerfile.tmpl
var dockerTmpl string

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please supply version as argument e.g. 'v22.1.0'")
		os.Exit(1)
	}

	p := params{
		Version: os.Args[1],
	}
	t, err := template.New("docker").Parse(dockerTmpl)
	if err != nil {
		panic(err)
	}
	err = t.Execute(os.Stdout, p)
	if err != nil {
		panic(err)
	}
}
