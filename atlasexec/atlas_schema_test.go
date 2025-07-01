package atlasexec_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/atlasexec"
	"github.com/stretchr/testify/require"
)

func TestSchema_Test(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.SchemaTestParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaTestParams{},
			args:   "schema test",
			stdout: "test result",
		},
		{
			name: "with env",
			params: &atlasexec.SchemaTestParams{
				Env: "test",
			},
			args:   "schema test --env test",
			stdout: "test result",
		},
		{
			name: "with config",
			params: &atlasexec.SchemaTestParams{
				ConfigURL: "file://config.hcl",
			},
			args:   "schema test --config file://config.hcl",
			stdout: "test result",
		},
		{
			name: "with dev-url",
			params: &atlasexec.SchemaTestParams{
				DevURL: "sqlite://file?_fk=1&cache=shared&mode=memory",
			},
			args:   "schema test --dev-url sqlite://file?_fk=1&cache=shared&mode=memory",
			stdout: "test result",
		},
		{
			name: "with run",
			params: &atlasexec.SchemaTestParams{
				Run: "example",
			},
			args:   "schema test --run example",
			stdout: "test result",
		},
		{
			name: "with run and paths",
			params: &atlasexec.SchemaTestParams{
				Run:   "example",
				Paths: []string{"./foo", "./bar"},
			},
			args:   "schema test --run example ./foo ./bar",
			stdout: "test result",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.SchemaTest(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.stdout, result)
		})
	}
}

func TestSchema_Inspect(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.SchemaInspectParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaInspectParams{},
			args:   "schema inspect",
			stdout: `schema "public" {}`,
		},
		{
			name: "with env",
			params: &atlasexec.SchemaInspectParams{
				Env: "test",
			},
			args:   "schema inspect --env test",
			stdout: `schema "public" {}`,
		},
		{
			name: "with config",
			params: &atlasexec.SchemaInspectParams{
				ConfigURL: "file://config.hcl",
				Env:       "test",
			},
			args:   "schema inspect --env test --config file://config.hcl",
			stdout: `schema "public" {}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.SchemaInspect(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.stdout, result)
		})
	}
}

func TestAtlasSchema_Apply(t *testing.T) {
	ce, err := atlasexec.NewWorkingDir()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ce.Close())
	})
	f, err := os.CreateTemp("", "sqlite-test")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	u := fmt.Sprintf("sqlite://%s?_fk=1", f.Name())
	c, err := atlasexec.NewClient(ce.Path(), "atlas")
	require.NoError(t, err)

	s1 := `
	-- create table "users
	CREATE TABLE users(
		id int NOT NULL,
		name varchar(100) NULL,
		PRIMARY KEY(id)
	);`
	path, err := ce.WriteFile("schema.sql", []byte(s1))
	to := fmt.Sprintf("file://%s", path)
	require.NoError(t, err)
	_, err = c.SchemaApply(context.Background(), &atlasexec.SchemaApplyParams{
		URL:         u,
		To:          to,
		DevURL:      "sqlite://file?_fk=1&cache=shared&mode=memory",
		AutoApprove: true,
	})
	require.NoError(t, err)
	_, err = ce.WriteFile("schema.sql", []byte(s1+`
	-- create table "blog_posts"
	CREATE TABLE blog_posts(
		id int NOT NULL,
		title varchar(100) NULL,
		body text NULL,
		author_id int NULL,
		PRIMARY KEY(id),
		CONSTRAINT author_fk FOREIGN KEY(author_id) REFERENCES users(id)
	);`))
	require.NoError(t, err)
	_, err = c.SchemaApply(context.Background(), &atlasexec.SchemaApplyParams{
		URL:         u,
		To:          to,
		DevURL:      "sqlite://file?_fk=1&cache=shared&mode=memory",
		AutoApprove: true,
	})
	require.NoError(t, err)

	s, err := c.SchemaInspect(context.Background(), &atlasexec.SchemaInspectParams{
		URL: u,
	})
	require.NoError(t, err)
	require.Equal(t, `table "users" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  column "name" {
    null = true
    type = varchar
  }
  primary_key {
    columns = [column.id]
  }
}
table "blog_posts" {
  schema = schema.main
  column "id" {
    null = false
    type = int
  }
  column "title" {
    null = true
    type = varchar
  }
  column "body" {
    null = true
    type = text
  }
  column "author_id" {
    null = true
    type = int
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "author_fk" {
    columns     = [column.author_id]
    ref_columns = [table.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
schema "main" {
}
`, s)
}

func TestSchema_Plan(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanParams
		args   string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaPlanParams{},
			args:   "schema plan --format {{ json . }} --auto-approve",
		},
		{
			name: "with env",
			params: &atlasexec.SchemaPlanParams{
				Env: "test",
			},
			args: "schema plan --format {{ json . }} --env test --auto-approve",
		},
		{
			name: "with from to",
			params: &atlasexec.SchemaPlanParams{
				From: []string{"1", "2"},
				To:   []string{"2", "3"},
			},
			args: `schema plan --format {{ json . }} --from 1,2 --to 2,3 --auto-approve`,
		},
		{
			name: "with from to and schema",
			params: &atlasexec.SchemaPlanParams{
				From:   []string{"1", "2"},
				To:     []string{"2", "3"},
				Schema: []string{"public", "bupisu"},
			},
			args: `schema plan --format {{ json . }} --schema public,bupisu --from 1,2 --to 2,3 --auto-approve`,
		},
		{
			name: "with from to and directives",
			params: &atlasexec.SchemaPlanParams{
				From:       []string{"1", "2"},
				To:         []string{"2", "3"},
				Directives: []string{"atlas:nolint", "atlas:txmode none"},
			},
			args: `schema plan --format {{ json . }} --from 1,2 --to 2,3 --auto-approve --directive "atlas:nolint" --directive "atlas:txmode none"`,
		},
		{
			name: "with config",
			params: &atlasexec.SchemaPlanParams{
				ConfigURL: "file://config.hcl",
			},
			args: "schema plan --format {{ json . }} --config file://config.hcl --auto-approve",
		},
		{
			name: "with dev-url",
			params: &atlasexec.SchemaPlanParams{
				DevURL: "sqlite://file?_fk=1&cache=shared&mode=memory",
			},
			args: "schema plan --format {{ json . }} --dev-url sqlite://file?_fk=1&cache=shared&mode=memory --auto-approve",
		},
		{
			name: "with name",
			params: &atlasexec.SchemaPlanParams{
				Name: "example",
			},
			args: "schema plan --format {{ json . }} --name example --auto-approve",
		},
		{
			name: "with dry-run",
			params: &atlasexec.SchemaPlanParams{
				DryRun: true,
			},
			args: "schema plan --format {{ json . }} --dry-run",
		},
		{
			name: "with save",
			params: &atlasexec.SchemaPlanParams{
				Save: true,
			},
			args: "schema plan --format {{ json . }} --save --auto-approve",
		},
		{
			name: "with push",
			params: &atlasexec.SchemaPlanParams{
				Repo: "testing-repo",
				Push: true,
			},
			args: "schema plan --format {{ json . }} --repo testing-repo --push --auto-approve",
		},
		{
			name: "with include",
			params: &atlasexec.SchemaPlanParams{
				Include: []string{"public", "bupisu"},
			},
			args: "schema plan --format {{ json . }} --include public,bupisu --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Repo":"foo"}`)
			result, err := c.SchemaPlan(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "foo", result.Repo)
		})
	}
}

func TestSchema_PlanPush(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanPushParams
		args   string
	}{
		{
			name: "with auto-approve",
			params: &atlasexec.SchemaPlanPushParams{
				Repo: "testing-repo",
				File: "file://plan.hcl",
			},
			args: "schema plan push --format {{ json . }} --file file://plan.hcl --repo testing-repo --auto-approve",
		},
		{
			name: "with auto-approve and schema",
			params: &atlasexec.SchemaPlanPushParams{
				Repo:   "testing-repo",
				File:   "file://plan.hcl",
				Schema: []string{"public", "bupisu"},
			},
			args: "schema plan push --format {{ json . }} --schema public,bupisu --file file://plan.hcl --repo testing-repo --auto-approve",
		},
		{
			name: "with pending status",
			params: &atlasexec.SchemaPlanPushParams{
				Pending: true,
				File:    "file://plan.hcl",
			},
			args: "schema plan push --format {{ json . }} --file file://plan.hcl --pending",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Repo":"foo"}`)
			result, err := c.SchemaPlanPush(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, `{"Repo":"foo"}`, result)
		})
	}
}

func TestSchema_PlanLint(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanLintParams
		args   string
	}{
		{
			name: "with repo",
			params: &atlasexec.SchemaPlanLintParams{
				Repo: "testing-repo",
				File: "file://plan.hcl",
			},
			args: "schema plan lint --format {{ json . }} --file file://plan.hcl --repo testing-repo --auto-approve",
		},
		{
			name: "with file only",
			params: &atlasexec.SchemaPlanLintParams{
				File: "file://plan.hcl",
			},
			args: "schema plan lint --format {{ json . }} --file file://plan.hcl --auto-approve",
		},
		{
			name: "with file and schema",
			params: &atlasexec.SchemaPlanLintParams{
				File:   "file://plan.hcl",
				Schema: []string{"public", "bupisu"},
			},
			args: "schema plan lint --format {{ json . }} --schema public,bupisu --file file://plan.hcl --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Repo":"foo"}`)
			result, err := c.SchemaPlanLint(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "foo", result.Repo)
		})
	}
}

func TestSchema_PlanValidate(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanValidateParams
		args   string
	}{
		{
			name: "with repo",
			params: &atlasexec.SchemaPlanValidateParams{
				Repo: "testing-repo",
				File: "file://plan.hcl",
			},
			args: "schema plan validate --file file://plan.hcl --repo testing-repo --auto-approve",
		},
		{
			name: "with file only",
			params: &atlasexec.SchemaPlanValidateParams{
				File: "file://plan.hcl",
			},
			args: "schema plan validate --file file://plan.hcl --auto-approve",
		},
		{
			name: "with file and schema",
			params: &atlasexec.SchemaPlanValidateParams{
				File:   "file://plan.hcl",
				Schema: []string{"public", "bupisu"},
			},
			args: "schema plan validate --schema public,bupisu --file file://plan.hcl --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Repo":"foo"}`)
			err := c.SchemaPlanValidate(context.Background(), tt.params)
			require.NoError(t, err)
		})
	}
}

func TestSchema_PlanApprove(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanApproveParams
		args   string
	}{
		{
			name: "with url",
			params: &atlasexec.SchemaPlanApproveParams{
				URL: "atlas://app1/plans/foo-plan",
			},
			args: "schema plan approve --format {{ json . }} --url atlas://app1/plans/foo-plan",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"URL":"atlas://app1/plans/foo-plan", "Link":"some-link", "Status":"APPROVED"}`)
			result, err := c.SchemaPlanApprove(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "atlas://app1/plans/foo-plan", result.URL)
		})
	}
}

func TestSchema_PlanPull(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanPullParams
		args   string
	}{
		{
			name: "with url",
			params: &atlasexec.SchemaPlanPullParams{
				URL: "atlas://app1/plans/foo-plan",
			},
			args: "schema plan pull --url atlas://app1/plans/foo-plan",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", "excited-plan")
			result, err := c.SchemaPlanPull(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "excited-plan", result)
		})
	}
}

func TestSchema_PlanList(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPlanListParams
		args   string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaPlanListParams{},
			args:   "schema plan list --format {{ json . }} --auto-approve",
		},
		{
			name: "with repo",
			params: &atlasexec.SchemaPlanListParams{
				Repo: "atlas://testing-repo",
				From: []string{"env://url"},
			},
			args: "schema plan list --format {{ json . }} --from env://url --repo atlas://testing-repo --auto-approve",
		},
		{
			name: "with repo and schema",
			params: &atlasexec.SchemaPlanListParams{
				Repo:   "atlas://testing-repo",
				From:   []string{"env://url"},
				Schema: []string{"public", "bupisu"},
			},
			args: "schema plan list --format {{ json . }} --schema public,bupisu --from env://url --repo atlas://testing-repo --auto-approve",
		},
		{
			name: "with repo and pending",
			params: &atlasexec.SchemaPlanListParams{
				Repo:    "atlas://testing-repo",
				Pending: true,
			},
			args: "schema plan list --format {{ json . }} --repo atlas://testing-repo --pending --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `[{"Name":"pr-2-ufnTS7Nr"}]`)
			result, err := c.SchemaPlanList(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "pr-2-ufnTS7Nr", result[0].Name)
		})
	}
}

func TestSchema_Push(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaPushParams
		args   string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaPushParams{},
			args:   "schema push --format {{ json . }}",
		},
		{
			name: "push with 1 URL",
			params: &atlasexec.SchemaPushParams{
				URL: []string{"file://foo.hcl"},
			},
			args: "schema push --format {{ json . }} --url file://foo.hcl",
		},
		{
			name: "push with 2 URLs",
			params: &atlasexec.SchemaPushParams{
				URL: []string{"file://foo.hcl", "file://bupisu.hcl"},
			},
			args: "schema push --format {{ json . }} --url file://foo.hcl --url file://bupisu.hcl",
		},
		{
			name: "with repo",
			params: &atlasexec.SchemaPushParams{
				Name: "atlas-action",
			},
			args: "schema push --format {{ json . }} atlas-action",
		},
		{
			name: "with repo and schemas",
			params: &atlasexec.SchemaPushParams{
				Name:   "atlas-action",
				Schema: []string{"public", "bupisu"},
			},
			args: "schema push --format {{ json . }} --schema public,bupisu atlas-action",
		},
		{
			name: "with repo and tag",
			params: &atlasexec.SchemaPushParams{
				Name: "atlas-action",
				Tag:  "v1.0.0",
			},
			args: "schema push --format {{ json . }} --tag v1.0.0 atlas-action",
		},
		{
			name: "with repo and tag and description",
			params: &atlasexec.SchemaPushParams{
				Name:        "atlas-action",
				Tag:         "v1.0.0",
				Description: "release-v1",
			},
			args: "schema push --format {{ json . }} --tag v1.0.0 --desc release-v1 atlas-action",
		},
		{
			name: "with repo and tag, version and description",
			params: &atlasexec.SchemaPushParams{
				Name:        "atlas-action",
				Tag:         "v1.0.0",
				Version:     "20240829100417",
				Description: "release-v1",
			},
			args: "schema push --format {{ json . }} --tag v1.0.0 --version 20240829100417 --desc release-v1 atlas-action",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Link":"https://gh.atlasgo.cloud/schemas/141733920810","Slug":"awesome-app","URL":"atlas://awesome-app?tag=latest"}`)
			result, err := c.SchemaPush(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "https://gh.atlasgo.cloud/schemas/141733920810", result.Link)
			require.Equal(t, "atlas://awesome-app?tag=latest", result.URL)
			require.Equal(t, "awesome-app", result.Slug)
		})
	}
}

func TestSchema_Apply(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaApplyParams
		args   string
	}{
		{
			name:   "no params",
			params: &atlasexec.SchemaApplyParams{},
			args:   "schema apply --format {{ json . }}",
		},
		{
			name: "with plan",
			params: &atlasexec.SchemaApplyParams{
				PlanURL: "atlas://app1/plans/foo-plan",
			},
			args: "schema apply --format {{ json . }} --plan atlas://app1/plans/foo-plan",
		},
		{
			name: "with auto-approve",
			params: &atlasexec.SchemaApplyParams{
				AutoApprove: true,
			},
			args: "schema apply --format {{ json . }} --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Driver":"sqlite3"}`)
			result, err := c.SchemaApply(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "sqlite3", result.Driver)
		})
	}
}

func TestSchema_Clean(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaCleanParams
		args   string
	}{
		{
			name: "with env and dry-run",
			params: &atlasexec.SchemaCleanParams{
				Env:    "test",
				URL:    "sqlite://app1.db",
				DryRun: true,
			},
			args: "schema clean --format {{ json . }} --env test --url sqlite://app1.db --dry-run",
		},
		{
			name: "with auto-approve",
			params: &atlasexec.SchemaCleanParams{
				URL:         "sqlite://app1.db",
				AutoApprove: true,
			},
			args: "schema clean --format {{ json . }} --url sqlite://app1.db --auto-approve",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", "{\"Start\":\"2024-09-20T14:51:40.439499+07:00\",\"End\":\"2024-09-20T14:51:40.439533+07:00\",\"Applied\":{\"Name\":\"20240920075140.sql\",\"Version\":\"20240920075140\",\"Start\":\"2024-09-20T14:51:40.43952+07:00\",\"End\":\"2024-09-20T14:51:40.439533+07:00\",\"Applied\":[\"PRAGMA foreign_keys = off;\",\"DROP TABLE `t1`;\", \"PRAGMA foreign_keys = on;\"]}}")
			result, err := c.SchemaClean(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, "20240920075140.sql", result.Applied.Name)
		})
	}
}

func TestSchema_ApplyEnvs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)
	require.NoError(t, c.SetEnv(map[string]string{
		"TEST_ARGS": "schema apply --format {{ json . }} --env test",
		"TEST_STDOUT": `{"Driver":"sqlite3","URL":{"Scheme":"sqlite","Host":"local-su.db"}}
{"Driver":"sqlite3","URL":{"Scheme":"sqlite","Host":"local-pi.db"}}
{"Driver":"sqlite3","URL":{"Scheme":"sqlite","Host":"local-bu.db"}}`,
		"TEST_STDERR": `Abort: The plan "From" hash does not match the current state hash (passed with --from):

  [31m- iHZMQ1EoarAXt/KU0KQbBljbbGs8gVqX2ZBXefePSGE=[0m[90m (plan value)[0m
  [32m+ Cp8xCVYilZuwULkggsfJLqIQHaxYcg/IpU+kgjVUBA4=[0m[90m (current hash)[0m

`,
	}))
	result, err := c.SchemaApply(context.Background(), &atlasexec.SchemaApplyParams{
		Env: "test",
	})
	require.ErrorContains(t, err, `The plan "From" hash does not match the current state hash`)
	require.Nil(t, result)

	err2, ok := err.(*atlasexec.SchemaApplyError)
	require.True(t, ok, "should be a SchemaApplyError, got %T", err)
	require.Contains(t, err2.Stderr, `Abort: The plan "From" hash does not match the current state hash (passed with --from)`)
	require.Len(t, err2.Result, 3, "should returns succeed results")
	require.Equal(t, "sqlite://local-su.db", err2.Result[0].URL.String())
	require.Equal(t, "sqlite://local-pi.db", err2.Result[1].URL.String())
	require.Equal(t, "sqlite://local-bu.db", err2.Result[2].URL.String())
}

func TestAtlasSchema_Lint(t *testing.T) {
	t.Run("with broken config", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		_, err = c.SchemaLint(context.Background(), &atlasexec.SchemaLintParams{
			ConfigURL: "file://config-broken.hcl",
		})
		require.ErrorContains(t, err, `file "config-broken.hcl" was not found`)
	})

	t.Run("with missing dev-url", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		_, err = c.SchemaLint(context.Background(), &atlasexec.SchemaLintParams{
			URL: []string{"file://testdata/schema.hcl"},
		})
		require.ErrorContains(t, err, `required flag(s) "dev-url" not set`)
	})
	var (
		atlashcl = filepath.Join(t.TempDir(), "atlas.hcl")
		srv      = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"data": {"me":{ "name": "p", "org": "life"}}}`)
		}))
	)
	t.Cleanup(srv.Close)
	require.NoError(t, os.WriteFile(atlashcl, []byte(fmt.Sprintf(`
		atlas { 
			cloud {	
				token = "aci_token"
				url = %q
				org = "life"
			}
		}
		lint {
		  naming {
			  table {
			    match = "^[a-z_]+$"
			  }
		  }
		}`, srv.URL)), 0600))
	t.Run("with non-existent schema file", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		_, err = c.SchemaLint(context.Background(), &atlasexec.SchemaLintParams{
			ConfigURL: "file://" + atlashcl,
			DevURL:    "sqlite://file?mode=memory",
			URL:       []string{"file://testdata/doesnotexist.hcl"},
		})
		require.ErrorContains(t, err, "Error: stat testdata/doesnotexist.hcl: no such file or directory")
	})
	t.Run("with schema containing problems", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		report, err := c.SchemaLint(context.Background(), &atlasexec.SchemaLintParams{
			ConfigURL: "file://" + atlashcl,
			DevURL:    "sqlite://file?mode=memory",
			URL:       []string{sqlitedb(t, "create table T1(id int);")},
		})
		require.NoError(t, err)
		require.NotNil(t, report)
		require.NotEmpty(t, report.Steps)
		require.Len(t, report.Steps, 1)
		require.Len(t, report.Steps[0].Diagnostics, 1)
		require.Equal(t, "Table \"main.T1\" violates the naming policy", report.Steps[0].Diagnostics[0].Text)
		require.Equal(t, "NM102", report.Steps[0].Diagnostics[0].Code)
	})
}

func TestSchema_Lint(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	testCases := []struct {
		name   string
		params *atlasexec.SchemaLintParams
		args   string
	}{
		{
			name: "with dev-url and url",
			params: &atlasexec.SchemaLintParams{
				URL:    []string{"file://testdata/schema.hcl"},
				DevURL: "sqlite://file?mode=memory",
			},
			args: "schema lint --format {{ json . }} --dev-url sqlite://file?mode=memory --url file://testdata/schema.hcl",
		},
		{
			name: "with dev-url and url and schema",
			params: &atlasexec.SchemaLintParams{
				URL:    []string{"file://testdata/schema.hcl"},
				DevURL: "sqlite://file?mode=memory",
				Schema: []string{"main", "bupisu"},
			},
			args: "schema lint --format {{ json . }} --dev-url sqlite://file?mode=memory --url file://testdata/schema.hcl --schema main,bupisu",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", `{"Steps":[{"Diagnostics":[{"Text":"Table \"main.T1\" violates the naming policy","Code":"NM102"}]}]}`)
			result, err := c.SchemaLint(context.Background(), tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Steps)
			require.Len(t, result.Steps, 1)
			require.Len(t, result.Steps[0].Diagnostics, 1)
			require.Equal(t, "Table \"main.T1\" violates the naming policy", result.Steps[0].Diagnostics[0].Text)
		})
	}
}
