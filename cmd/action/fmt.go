// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"bytes"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
)

var (
	// FmtCmd represents the fmt command.
	FmtCmd = &cobra.Command{
		Use:   "fmt [path]",
		Short: "Formats Atlas HCL files",
		Long: "`atlas schema fmt`" + ` formats all ".hcl" files under the given path using
cannonical HCL layout style as defined by the github.com/hashicorp/hcl/v2/hclwrite package. 
Unless stated otherwise, the fmt command will use the current directory.

After running, the command will print the names of the files it has formatted. If all
files in the directory are formatted, no input will be printed out.
`,
		Run:  CmdFmtRun,
		Args: cobra.MaximumNArgs(1),
	}
)

func init() {
	schemaCmd.AddCommand(FmtCmd)
}

// CmdFmtRun formats all HCL files in a given directory using canonical HCL formatting
// rules.
func CmdFmtRun(cmd *cobra.Command, args []string) {
	path := "./"
	if len(args) > 0 {
		path = args[0]
	}
	tasks, err := tasks(path)
	cobra.CheckErr(err)
	for _, task := range tasks {
		changed, err := fmtFile(task)
		cobra.CheckErr(err)
		if changed {
			cmd.Println(task.path)
		}
	}
}

func tasks(path string) ([]fmttask, error) {
	var tasks []fmttask
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		if strings.HasSuffix(path, ".hcl") {
			tasks = append(tasks, fmttask{
				path: path,
				info: stat,
			})
		}
		return tasks, nil
	}
	all, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range all {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".hcl") {
			tasks = append(tasks, fmttask{
				path: filepath.Join(path, f.Name()),
				info: f,
			})
		}
	}
	return tasks, nil
}

type fmttask struct {
	path string
	info fs.FileInfo
}

// fmtFile tries to format a file and reports if formatting occurred.
func fmtFile(task fmttask) (bool, error) {
	orig, err := os.ReadFile(task.path)
	if err != nil {
		return false, err
	}
	formatted := hclwrite.Format(orig)
	if !bytes.Equal(formatted, orig) {
		return true, os.WriteFile(task.path, formatted, task.info.Mode())
	}
	return false, nil
}
