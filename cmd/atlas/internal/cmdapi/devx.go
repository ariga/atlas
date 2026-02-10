// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdapi

import (
	"context"
	"fmt"
	
	"ariga.io/atlas/cmd/atlas/internal/docker"
)

// ContainerExec executes a command in a development database container.
// containerID is the Docker container ID or name.
func ContainerExec(ctx context.Context, containerID string, cmd ...string) (string, error) {
	if containerID == "" {
		return "", fmt.Errorf("container ID cannot be empty")
	}
	
	exists, err := docker.ContainerExists(ctx, containerID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("container %q not found", containerID)
	}
	
	return docker.ExecInContainer(ctx, containerID, cmd...)
}

// ContainerCopy copies a file or directory from the host to a development database container.
// containerID is the Docker container ID or name.
func ContainerCopy(ctx context.Context, src, containerID, dst string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}
	if src == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if dst == "" {
		return fmt.Errorf("destination path cannot be empty")
	}
	
	exists, err := docker.ContainerExists(ctx, containerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container %q not found", containerID)
	}
	
	return docker.CopyToContainer(ctx, src, containerID, dst)
}

// ContainerMkdir creates a directory in a development database container.
// containerID is the Docker container ID or name.
func ContainerMkdir(ctx context.Context, containerID, path string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	exists, err := docker.ContainerExists(ctx, containerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container %q not found", containerID)
	}
	
	return docker.MkdirInContainer(ctx, containerID, path)
}
