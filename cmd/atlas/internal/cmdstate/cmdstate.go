// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/mitchellh/go-homedir"
)

// DefaultDir is the directory where CLI state is stored.
const DefaultDir = "~/.atlas"

// File is a state file for the given type.
type File[T any] struct {
	// Dir where the file is stored. If empty, DefaultDir is used.
	Dir string
	// Name of the file. Suffixed with .json.
	Name string
}

// Read reads the value from the file system.
func (f File[T]) Read() (v T, err error) {
	path, err := f.Path()
	if err != nil {
		return v, err
	}
	switch buf, err := os.ReadFile(path); {
	case os.IsNotExist(err):
		return newT(v), nil
	case err != nil:
		return v, err
	default:
		err = json.Unmarshal(buf, &v)
		return v, err
	}
}

// Write writes the value to the file system.
func (f File[T]) Write(t T) error {
	buf, err := json.Marshal(t)
	if err != nil {
		return err
	}
	path, err := f.Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0666)
}

// Path returns the path to the file.
func (f File[T]) Path() (string, error) {
	name := f.Name
	if filepath.Ext(name) == "" {
		name += ".json"
	}
	if f.Dir != "" {
		return filepath.Join(f.Dir, name), nil
	}
	path, err := homedir.Expand(filepath.Join(DefaultDir, name))
	if err != nil {
		return "", err
	}
	return path, nil
}

// newT ensures the type is initialized.
func newT[T any](t T) T {
	if rt := reflect.TypeOf(t); rt.Kind() == reflect.Ptr {
		return reflect.New(rt.Elem()).Interface().(T)
	}
	return t
}

// muDisableCache ensures homedir.DisableCache is not changed concurrently on tests.
var muDisableCache sync.Mutex

// TestingHome is a helper function for testing that
// sets the HOME directory to a temporary directory.
func TestingHome(t *testing.T) string {
	muDisableCache.Lock()
	homedir.DisableCache = true
	t.Cleanup(func() {
		homedir.DisableCache = false
		muDisableCache.Unlock()
	})
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}
