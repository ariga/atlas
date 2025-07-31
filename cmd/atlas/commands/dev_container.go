// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package commands

import (
	"errors"
	"fmt"
	"strings"

	"ariga.io/atlas/cmd/atlas/internal/cmdapi"
	"github.com/spf13/cobra"
)

// devContainerCmd represents the base container command
var devContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Work with development database containers",
	Long: `The 'container' command group provides functionality to interact with
development database containers created by Atlas. This includes executing
commands, creating directories, and copying files to containers.`,
}

// devContainerExecCmd represents the command to execute commands in containers
var devContainerExecCmd = &cobra.Command{
	Use:   "exec [container_id] [command]",
	Short: "Execute a command in a development database container",
	Long: `Execute a command in a development database container identified by its container ID or name.

Example:
  atlas dev container exec my-postgres-container ls -la /var/lib/postgresql/data`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		cmdArgs := args[1:]
		
		output, err := cmdapi.ContainerExec(cmd.Context(), containerID, cmdArgs...)
		if err != nil {
			return err
		}
		
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(output))
		return nil
	},
}

// devContainerMkdirCmd represents the command to create directories in containers
var devContainerMkdirCmd = &cobra.Command{
	Use:   "mkdir [container_id] [path]",
	Short: "Create a directory in a development database container",
	Long: `Create a directory in a development database container identified by its container ID or name.

Example:
  atlas dev container mkdir my-postgres-container /var/lib/postgresql/app/hot`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		path := args[1]
		
		if err := cmdapi.ContainerMkdir(cmd.Context(), containerID, path); err != nil {
			return err
		}
		
		fmt.Fprintf(cmd.OutOrStdout(), "Directory %s created successfully in container %s\n", path, containerID)
		return nil
	},
}

// devContainerCopyCmd represents the command to copy files to containers
var devContainerCopyCmd = &cobra.Command{
	Use:   "cp [source] [container_id]:[destination]",
	Short: "Copy files or directories to a development database container",
	Long: `Copy files or directories to a development database container identified by its container ID or name.

Example:
  atlas dev container cp ./extension.sql my-postgres-container:/tmp/extension.sql`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		
		// Parse containerID:dst format
		parts := strings.SplitN(args[1], ":", 2)
		if len(parts) != 2 {
			return errors.New("destination must be in format 'container_id:path'")
		}
		
		containerID := parts[0]
		dst := parts[1]
		
		if err := cmdapi.ContainerCopy(cmd.Context(), src, containerID, dst); err != nil {
			return err
		}
		
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully copied %s to %s:%s\n", src, containerID, dst)
		return nil
	},
}

func init() {
	devCmd.AddCommand(devContainerCmd)
	devContainerCmd.AddCommand(devContainerExecCmd)
	devContainerCmd.AddCommand(devContainerMkdirCmd)
	devContainerCmd.AddCommand(devContainerCopyCmd)
}
