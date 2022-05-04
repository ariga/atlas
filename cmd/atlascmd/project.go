package atlascmd

import (
	"fmt"
	"os"

	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/schema/schemaspec/schemahcl"
)

// doc represents an atlas.hcl file.
type doc struct {
	Envs []*Env `spec:"env"`
}

// Env represents an Atlas environment.
type Env struct {
	Name   string `spec:"name,name"`
	URL    string `spec:"url"`
	DevURL string `spec:"dev"`
	Schema string `spec:"schema"`
	schemaspec.DefaultExtension
}

func loadProject(path string) (map[string]*Env, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open project file: %w", err)
	}
	projEnvs := make(map[string]*Env)
	var d doc
	if err := schemahcl.New().Eval(b, &d, nil); err != nil {
		return nil, fmt.Errorf("reading project file: %w", err)
	}
	for _, env := range d.Envs {
		if _, ok := projEnvs[env.Name]; ok {
			return nil, fmt.Errorf("duplicate environment name %q", env.Name)
		}
		projEnvs[env.Name] = env
	}
	return projEnvs, nil
}

func init() {
	schemaspec.Register("env", &Env{})
}
