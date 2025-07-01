package atlasexec

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"
	"text/template"

	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestContextExecer(t *testing.T) {
	src := fstest.MapFS{
		"bar": &fstest.MapFile{Data: []byte("bar-content")},
	}
	ce, err := NewWorkingDir()
	checkFileContent := func(t *testing.T, name, expected string) {
		t.Helper()
		full := filepath.Join(ce.dir, name)
		require.FileExists(t, full, "The file %q should exist", name)
		actual, err := os.ReadFile(full)
		require.NoError(t, err)
		require.Equal(t, expected, string(actual), "The file %q should have the expected content", name)
	}
	require.NoError(t, err)
	require.DirExists(t, ce.dir, "The temporary directory should exist")
	require.NoFileExists(t, filepath.Join(ce.dir, "atlas.hcl"), "The file atlas.hcl should not exist")
	require.NoError(t, ce.Close())

	// Test WithMigrations.
	ce, err = NewWorkingDir(WithMigrations(src))
	require.NoError(t, err)
	checkFileContent(t, filepath.Join("migrations", "bar"), "bar-content")
	require.NoError(t, ce.Close())

	// Test WithMigrations - MemDir.
	dir := &migrate.MemDir{}
	require.NoError(t, dir.WriteFile("1.sql", []byte("-- only .sql files are copied\nmem-content")))
	require.NoError(t, dir.WriteFile(migrate.HashFileName, []byte("-- And the atlas.sum")))
	ce, err = NewWorkingDir(WithMigrations(dir))
	require.NoError(t, err)
	checkFileContent(t, filepath.Join("migrations", "1.sql"), "-- only .sql files are copied\nmem-content")
	checkFileContent(t, filepath.Join("migrations", migrate.HashFileName), "-- And the atlas.sum")
	require.NoError(t, ce.Close())

	// Test WithAtlasHCL.
	ce, err = NewWorkingDir(
		WithAtlasHCL(func(w io.Writer) error {
			return template.Must(template.New("").Parse(`{{ .foo }} & {{ .bar }}`)).
				Execute(w, map[string]any{
					"foo": "foo",
					"bar": "bar",
				})
		}),
		WithMigrations(src),
	)
	require.NoError(t, err)
	require.DirExists(t, ce.dir, "tmpDir")
	checkFileContent(t, filepath.Join("migrations", "bar"), "bar-content")
	checkFileContent(t, "atlas.hcl", "foo & bar")

	// Test WriteFile.
	_, err = ce.WriteFile(filepath.Join("migrations", "foo"), []byte("foo-content"))
	require.NoError(t, err)
	checkFileContent(t, filepath.Join("migrations", "foo"), "foo-content")

	// Test RunCommand.
	buf := &bytes.Buffer{}
	cmd := exec.Command("ls")
	cmd.Dir = "fake-dir"
	cmd.Stdout = buf
	require.NoError(t, ce.RunCommand(cmd))
	require.Equal(t, "fake-dir", cmd.Dir)
	require.Equal(t, "atlas.hcl\nmigrations\n", buf.String())
	require.NoError(t, ce.Close())
}

func TestMaintainOriginalWorkingDir(t *testing.T) {
	dir := t.TempDir()
	c, err := NewClient(dir, "atlas")
	require.NoError(t, err)
	require.Equal(t, dir, c.workingDir)
	require.NoError(t, c.WithWorkDir("bar", func(c *Client) error {
		require.Equal(t, "bar", c.workingDir)
		return nil
	}))
	require.Equal(t, dir, c.workingDir, "The working directory should not be changed")
}
