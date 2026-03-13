// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecInContainer executes a command in the given container and returns the combined output.
func ExecInContainer(ctx context.Context, containerID string, cmd ...string) (string, error) {
	args := append([]string{"exec", containerID}, cmd...)
	c := exec.CommandContext(ctx, "docker", args...)
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("docker exec: %w: %s", err, buf.String())
	}
	return buf.String(), nil
}

// CopyToContainer copies a file or directory from the host to the container.
func CopyToContainer(ctx context.Context, src, containerID, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %q: %w", src, err)
	}

	// Create destination directory if it doesn't exist
	if info.IsDir() {
		// For directories, make sure the target directory exists
		if _, err := ExecInContainer(ctx, containerID, "mkdir", "-p", dst); err != nil {
			return fmt.Errorf("creating directory %q in container: %w", dst, err)
		}
	} else {
		// For files, make sure the parent directory exists
		dstDir := filepath.Dir(dst)
		if _, err := ExecInContainer(ctx, containerID, "mkdir", "-p", dstDir); err != nil {
			return fmt.Errorf("creating directory %q in container: %w", dstDir, err)
		}
	}

	// Use docker cp to copy the file or directory
	cmd := exec.CommandContext(ctx, "docker", "cp", src, containerID+":"+dst)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker cp: %w: %s", err, buf.String())
	}
	return nil
}

// MkdirInContainer creates a directory in the container.
func MkdirInContainer(ctx context.Context, containerID, path string) error {
	_, err := ExecInContainer(ctx, containerID, "mkdir", "-p", path)
	return err
}

// ContainerExists checks if a container with the given ID or name exists.
func ContainerExists(ctx context.Context, containerID string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("docker inspect: %w", err)
	}
	return true, nil
}

// ExtractContainerIDFromURL extracts container ID from a Docker URL.
// Docker URLs are typically in the format "docker://image/tag/database"
func ExtractContainerIDFromURL(url string) (string, error) {
	// Check if it's a docker URL
	if !strings.HasPrefix(url, "docker://") {
		return "", fmt.Errorf("not a docker URL: %s", url)
	}

	// Find running container for this URL
	parts := strings.Split(strings.TrimPrefix(url, "docker://"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid docker URL format: %s", url)
	}

	// The container ID should be available in the driver's connection
	// But for simplicity in this implementation, we'll return an error
	// suggesting to use the explicit container ID instead
	return "", fmt.Errorf("container ID cannot be extracted from URL, use ContainerExec/ContainerCopy with explicit ID")
}
