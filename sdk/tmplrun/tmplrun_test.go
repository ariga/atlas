package tmplrun

import (
	_ "embed"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/app.tmpl
	testTmpl   string
	loaderTmpl = template.Must(template.New("loader").Parse(testTmpl))
)

func TestRunner(t *testing.T) {
	runner := New("test", loaderTmpl, WithBuildTags("testdata"))
	out, err := runner.Run(struct {
		Message string
	}{
		Message: "Hello, World!",
	})
	require.NoError(t, err)
	require.Contains(t, out, "Hello, World! foo")
}
