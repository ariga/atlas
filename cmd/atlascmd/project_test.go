package atlascmd

import (
	"testing"

	"ariga.io/atlas/schema/schemaspec/schemahcl"
	"github.com/stretchr/testify/require"
)

func TestProject(t *testing.T) {
	h := `
env "local" {
	url = "mysql://root:pass@localhost:3306/"
	dev = "docker://mysql/8"
	schema = "./testdata/project"
}
`
	project := doc{}
	err := schemahcl.New().Eval([]byte(h), &project, nil)
	require.NoError(t, err)
	require.EqualValues(t, &Env{
		Name:   "local",
		URL:    "mysql://root:pass@localhost:3306/",
		DevURL: "docker://mysql/8",
		Schema: "./testdata/project",
	}, project.Envs[0])
}
