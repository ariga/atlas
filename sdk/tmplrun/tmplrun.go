// package tmplrun provides a Runner for templated go programs. It is commonly used
// by Go Atlas providers to compile ad-hoc programs that emit the desired SQL schema for
// data models defined for Go ORMs.

package tmplrun

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

type (
	// Runner is a go template runner.  It accepts a go template and data, and runs the
	// rendered template as a go program.
	Runner struct {
		name      string
		tmpl      *template.Template
		buildTags string
	}
	// Option is a function that configures the Runner.
	Option func(*Runner)
)

// WithBuildTags sets the build tags for the Runner.
func WithBuildTags(tags string) Option {
	return func(r *Runner) {
		r.buildTags = tags
	}
}

// New returns a new Runner.
func New(name string, tmpl *template.Template, opts ...Option) *Runner {
	r := &Runner{name: name, tmpl: tmpl}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run runs the template and returns the output.
func (r *Runner) Run(data any) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	s, err := format.Source(buf.Bytes())
	if err != nil {
		return "", err
	}
	return r.goRun(s)
}

func (r *Runner) goRun(src []byte) (string, error) {
	dir := fmt.Sprintf(".%s", r.name)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	target := fmt.Sprintf("%s/%s.go", dir, r.filename(r.name))
	if err := os.WriteFile(target, src, 0644); err != nil {
		return "", fmt.Errorf("%s: write file %s: %w", r.name, target, err)
	}
	return goRun(target, r.buildTags)
}

func (r *Runner) filename(pkg string) string {
	name := strings.ReplaceAll(pkg, "/", "_")
	return fmt.Sprintf("%s_%s_%d", r.name, name, time.Now().Unix())
}

// run 'go run' command and return its output.
func goRun(target, buildTags string) (string, error) {
	s, err := gocmd("run", buildTags, target)
	if err != nil {
		return "", fmt.Errorf("tmplrun: %s", err)
	}
	return s, nil
}

// goCmd runs a go command and returns its output.
func gocmd(command, buildTags, target string) (string, error) {
	args := []string{command}
	if buildTags != "" {
		args = append(args, "-tags", buildTags)
	}
	args = append(args, target)
	cmd := exec.Command("go", args...)
	stderr := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New(stderr.String())
	}
	return stdout.String(), nil
}
