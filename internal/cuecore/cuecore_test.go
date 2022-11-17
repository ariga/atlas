package cuecore

import (
	"github.com/rogpeppe/go-internal/testscript"
	"testing"
)

func TestLoad(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"load": func(ts *testscript.TestScript, neg bool, args []string) {
				cwd := ts.Getenv("WORK")
				var mod struct {
					Name string `json:"name"`
				}

				_, err := Load(cwd, []string{"module.cue"},
					WithValidation(),
					WithBasicDecoder(&mod),
				)
				ts.Check(err)

				if mod.Name == "" {
					ts.Fatalf("expected module name to be set")
				}
			},
		},
	})
}
