// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"ariga.io/atlas/sql/migrate"
)

type (
	// WorkingDir is a temporary directory with a copy of the files from dir.
	// It can be used to run commands in the temporary directory.
	// The temporary directory is removed when Close is called.
	WorkingDir struct {
		dir string
	}
	// Option is a function that modifies a ContextExecer.
	Option func(ce *WorkingDir) error
)

// WithAtlasHCLString creates the atlas.hcl file with the given string.
func WithAtlasHCLString(s string) Option {
	return WithAtlasHCL(func(w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	})
}

// WithAtlasHCLPath creates the atlas.hcl file by copying the file at the given path.
func WithAtlasHCLPath(path string) Option {
	return WithAtlasHCL(func(w io.Writer) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}

// WithAtlasHCL accept a function to create the atlas.hcl file.
func WithAtlasHCL(fn func(w io.Writer) error) Option {
	return func(ce *WorkingDir) error {
		return ce.CreateFile("atlas.hcl", fn)
	}
}

// WithMigrations copies all files from dir to the migrations directory.
// If dir is nil, no files are copied.
func WithMigrations(dir fs.FS) Option {
	return func(ce *WorkingDir) error {
		if dir == nil {
			return nil
		}
		return ce.CopyFS("migrations", dir)
	}
}

// NewWorkingDir creates a new ContextExecer.
// It creates a temporary directory and copies all files from dir to the temporary directory.
// The atlasHCL function is called with a writer to create the atlas.hcl file.
// If atlasHCL is nil, no atlas.hcl file is created.
func NewWorkingDir(opts ...Option) (*WorkingDir, error) {
	tmpDir, err := os.MkdirTemp("", "atlasexec-*")
	if err != nil {
		if err2 := os.RemoveAll(tmpDir); err2 != nil {
			err = errors.Join(err, err2)
		}
		return nil, err
	}
	c := &WorkingDir{dir: tmpDir}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Close removes the temporary directory.
func (ce *WorkingDir) Close() error {
	return os.RemoveAll(ce.dir)
}

// DirFS returns a fs.FS for the temporary directory.
func (ce *WorkingDir) DirFS() fs.FS {
	return os.DirFS(ce.dir)
}

// Dir returns the path to the temporary directory.
func (ce *WorkingDir) Path(elem ...string) string {
	if len(elem) == 0 {
		return ce.dir
	}
	return filepath.Join(append([]string{ce.dir}, elem...)...)
}

// RunCommand runs the command in the temporary directory.
func (ce *WorkingDir) RunCommand(cmd *exec.Cmd) error {
	// Restore the current directory after the command is run.
	defer func(d string) { cmd.Dir = d }(cmd.Dir)
	cmd.Dir = ce.dir
	return cmd.Run()
}

// WriteFile writes the file to the temporary directory.
func (ce *WorkingDir) WriteFile(name string, data []byte) (string, error) {
	err := ce.CreateFile(name, func(w io.Writer) (err error) {
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return "", err
	}
	return ce.Path(name), err
}

// CreateFile creates the file in the temporary directory.
func (ce *WorkingDir) CreateFile(name string, fn func(w io.Writer) error) error {
	f, err := os.Create(ce.Path(name))
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		if err2 := f.Close(); err2 != nil {
			err = errors.Join(err, err2)
		}
		return err
	}
	return f.Close()
}

// CopyFS copies all files from source FileSystem to the destination directory
// in the temporary directory.
// If source is nil, an error is returned.
func (ce *WorkingDir) CopyFS(name string, src fs.FS) error {
	dst := ce.Path(name)
	// Ensure destination directory exists.
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}
	switch dir := src.(type) {
	case nil:
		return errors.New("atlasexec: source is nil")
	case migrate.Dir:
		// The migrate.MemDir doesn't 100% compatible with fs.FS.
		// It returns fs.ErrNotExist error when open "." directory.
		// So, we need to handle it separately using the Files method.
		files, err := dir.Files()
		if err != nil {
			return err
		}
		for _, f := range files {
			name := filepath.Join(dst, f.Name())
			if err := os.WriteFile(name, f.Bytes(), 0644); err != nil { //nolint:gosec
				return err
			}
		}
		// If the atlas.sum file exists, copy it to the destination directory.
		if hf, err := dir.Open(migrate.HashFileName); err == nil {
			data, err := io.ReadAll(hf)
			if err != nil {
				return err
			}
			name := filepath.Join(dst, migrate.HashFileName)
			if err := os.WriteFile(name, data, 0644); err != nil { //nolint:gosec
				return err
			}
		}
		return nil
	default:
		return fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || path == "." {
				return err
			}
			name := filepath.Join(dst, path)
			if d.IsDir() {
				return os.Mkdir(name, 0700)
			}
			data, err := fs.ReadFile(dir, path)
			if err != nil {
				return err
			}
			return os.WriteFile(name, data, 0644) //nolint:gosec
		})
	}
}
