// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

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
