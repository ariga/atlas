package schemahcl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiFile(t *testing.T) {
	var test struct {
		People []*struct {
			Name string `spec:",name"`
		} `spec:"person"`
	}
	paths := make([]string, 0)
	testDir := "testdata/"
	dir, err := os.ReadDir(testDir)
	require.NoError(t, err)
	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(testDir, file.Name()))
	}
	err = New().EvalFiles(paths, &test, map[string]string{
		"hobby": "coding",
	})
	require.NoError(t, err)
	require.Len(t, test.People, 2)
}
