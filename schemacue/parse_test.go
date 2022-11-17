package schemacue

import (
	cuemod "ariga.io/atlas/cue.mod"
	"ariga.io/atlas/internal/cuecore"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParse(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"load": func(ts *testscript.TestScript, neg bool, args []string) {
				cwd := ts.Getenv("WORK")
				err := cuemod.Copy(cwd)
				ts.Check(err)

				var expected = newTestRealm()
				var actual = schema.NewRealm()

				_, err = cuecore.Load(cwd, []string{"module.cue"},
					cuecore.WithValidation(),
					cuecore.WithBasicDecoder(&actual),
				)
				ts.Check(err)
				require.NotEmpty(t, actual.Schemas)
				require.Equal(t, expected.Schemas[0].Name, actual.Schemas[0].Name)
				require.Equal(t, expected.Schemas[0].Tables[0].Name, actual.Schemas[0].Tables[0].Name)
				require.Equal(t, expected.Schemas[0].Tables[0].Columns[0].Name, actual.Schemas[0].Tables[0].Columns[0].Name)
			},
		},
	})
}

func newTestRealm() *schema.Realm {
	var realm = schema.NewRealm()

	var auth = schema.
		NewSchema("auth")

	var id = schema.
		NewStringColumn("id", postgres.TypeText)

	var users = schema.
		NewTable("users").
		AddColumns(id)

	auth.AddTables(users)

	return realm.AddSchemas(auth)
}
