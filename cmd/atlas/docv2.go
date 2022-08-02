//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"os"

	"ariga.io/atlas/cmd/atlas/internal/cmdapi"
	"github.com/spf13/cobra"
)

func main() {
	docs, err := docs(cmdapi.Root, "")
	if err != nil {
		panic(err)
	}
	for k, v := range docs {
		md := "```\n" +
			string(v) + "\n" +
			"```"
		if err := os.WriteFile("../../doc/md/generated/_"+k+".mdx", []byte(md), os.ModePerm); err != nil {
			panic(err)
		}
	}
	cmdapi.Root.ResetFlags()

}

func docs(c *cobra.Command, par string) (map[string]string, error) {
	cmdhelp := make(map[string]string)
	var b bytes.Buffer
	c.SetOut(&b)
	if err := c.Help(); err != nil {
		return nil, err
	}
	n := c.Name()
	if len(par) > 0 {
		n = par + "_" + n
	}
	cmdhelp[n] = b.String()
	for _, ch := range c.Commands() {
		cd, err := docs(ch, n)
		if err != nil {
			return nil, err
		}
		for k, v := range cd {
			cmdhelp[k] = v
		}
	}
	return cmdhelp, nil
}
