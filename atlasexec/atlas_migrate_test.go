// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/atlasexec"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlcheck"
	"github.com/stretchr/testify/require"
)

func TestMigrate_Status(t *testing.T) {
	type args struct {
		ctx  context.Context
		data *atlasexec.MigrateStatusParams
	}
	tests := []struct {
		name        string
		args        args
		wantCurrent string
		wantNext    string
		wantErr     bool
	}{
		{
			args: args{
				ctx: context.Background(),
				data: &atlasexec.MigrateStatusParams{
					DirURL: "file://testdata/migrations",
				},
			},
			wantCurrent: "No migration applied yet",
			wantNext:    "20230727105553",
		},
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(wd, "atlas")
	require.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.data.URL = sqlitedb(t, "")
			got, err := c.MigrateStatus(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.wantCurrent, got.Current)
			require.Equal(t, tt.wantNext, got.Next)
		})
	}
}

func TestMigrate_Apply(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.MigrateApplyParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.MigrateApplyParams{},
			args:   "migrate apply --format {{ json . }}",
			stdout: `{"Driver":"mysql"}`,
		},
		{
			name: "with env",
			params: &atlasexec.MigrateApplyParams{
				Env: "test",
			},
			args:   "migrate apply --format {{ json . }} --env test",
			stdout: `{"Driver":"mysql"}`,
		},
		{
			name: "with url",
			params: &atlasexec.MigrateApplyParams{
				URL: "sqlite://file?_fk=1&cache=shared&mode=memory",
			},
			args:   "migrate apply --format {{ json . }} --url sqlite://file?_fk=1&cache=shared&mode=memory",
			stdout: `{"Driver":"mysql"}`,
		},
		{
			name: "with exec order",
			params: &atlasexec.MigrateApplyParams{
				ExecOrder: atlasexec.ExecOrderLinear,
			},
			args:   "migrate apply --format {{ json . }} --exec-order linear",
			stdout: `{"Driver":"mysql"}`,
		},
		{
			name: "with lock name",
			params: &atlasexec.MigrateApplyParams{
				LockName: "custom_lock",
			},
			args:   "migrate apply --format {{ json . }} --lock-name custom_lock",
			stdout: `{"Driver":"mysql"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.MigrateApply(context.Background(), tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "mysql", result.Driver)
		})
	}
}

func TestMigrate_Down(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.MigrateDownParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.MigrateDownParams{},
			args:   "migrate down --format {{ json . }}",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "with env",
			params: &atlasexec.MigrateDownParams{
				Env: "test",
			},
			args:   "migrate down --format {{ json . }} --env test",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "with url",
			params: &atlasexec.MigrateDownParams{
				URL: "sqlite://file?_fk=1&cache=shared&mode=memory",
			},
			args:   "migrate down --format {{ json . }} --url sqlite://file?_fk=1&cache=shared&mode=memory",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "with target version",
			params: &atlasexec.MigrateDownParams{
				ToVersion: "12345",
			},
			args:   "migrate down --format {{ json . }} --to-version 12345",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "with tag version",
			params: &atlasexec.MigrateDownParams{
				ToTag: "12345",
			},
			args:   "migrate down --format {{ json . }} --to-tag 12345",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "with amount",
			params: &atlasexec.MigrateDownParams{
				Amount: 10,
			},
			args:   "migrate down --format {{ json . }} 10",
			stdout: `{"Status":"Pending"}`,
		},
		{
			name: "dev-url",
			params: &atlasexec.MigrateDownParams{
				DevURL: "url",
			},
			args:   "migrate down --format {{ json . }} --dev-url url",
			stdout: `{"Status":"Pending"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.MigrateDown(context.Background(), tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "Pending", result.Status)
		})
	}
}

func TestMigrate_Test(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	for _, tt := range []struct {
		name   string
		params *atlasexec.MigrateTestParams
		args   string
		stdout string
	}{
		{
			name:   "no params",
			params: &atlasexec.MigrateTestParams{},
			args:   "migrate test",
			stdout: "test result",
		},
		{
			name: "with env",
			params: &atlasexec.MigrateTestParams{
				Env: "test",
			},
			args:   "migrate test --env test",
			stdout: "test result",
		},
		{
			name: "with config",
			params: &atlasexec.MigrateTestParams{
				ConfigURL: "file://config.hcl",
			},
			args:   "migrate test --config file://config.hcl",
			stdout: "test result",
		},
		{
			name: "with dev-url",
			params: &atlasexec.MigrateTestParams{
				DevURL: "sqlite://file?_fk=1&cache=shared&mode=memory",
			},
			args:   "migrate test --dev-url sqlite://file?_fk=1&cache=shared&mode=memory",
			stdout: "test result",
		},
		{
			name: "with run",
			params: &atlasexec.MigrateTestParams{
				Run: "example",
			},
			args:   "migrate test --run example",
			stdout: "test result",
		},
		{
			name: "with run and paths",
			params: &atlasexec.MigrateTestParams{
				Run:   "example",
				Paths: []string{"./foo", "./bar"},
			},
			args:   "migrate test --run example ./foo ./bar",
			stdout: "test result",
		},
		{
			name: "with revisions-schema",
			params: &atlasexec.MigrateTestParams{
				RevisionsSchema: "schema",
			},
			args:   "migrate test --revisions-schema schema",
			stdout: "test result",
		},
		{
			name: "with run context",
			params: &atlasexec.MigrateTestParams{
				Context: &atlasexec.RunContext{
					Repo: "testing-repo",
				},
			},
			args:   "migrate test --context {\"repo\":\"testing-repo\"}",
			stdout: "test result",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ARGS", tt.args)
			t.Setenv("TEST_STDOUT", tt.stdout)
			result, err := c.MigrateTest(context.Background(), tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.stdout, result)
		})
	}
}

func TestAtlasMigrate_ApplyBroken(t *testing.T) {
	c, err := atlasexec.NewClient(".", "atlas")
	require.NoError(t, err)
	got, err := c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		URL:    "sqlite://?mode=memory",
		DirURL: "file://testdata/broken",
	})
	require.ErrorContains(t, err, `sql/migrate: executing statement "broken;" from version "20231029112426": near "broken": syntax error`)
	require.Nil(t, got)
	report, ok := err.(*atlasexec.MigrateApplyError)
	require.True(t, ok)
	require.Equal(t, "20231029112426", report.Result[0].Target)
	require.Equal(t, "sql/migrate: executing statement \"broken;\" from version \"20231029112426\": near \"broken\": syntax error", report.Error())
	require.Len(t, report.Result[0].Applied, 1)
	require.Equal(t, &struct {
		Stmt, Text string
	}{
		Stmt: "broken;",
		Text: "near \"broken\": syntax error",
	}, report.Result[0].Applied[0].Error)
}

func TestAtlasMigrate_Apply(t *testing.T) {
	ec, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(os.DirFS(filepath.Join("testdata", "migrations"))),
		atlasexec.WithAtlasHCL(func(w io.Writer) error {
			_, err := w.Write([]byte(`
			variable "url" {
				type    = string
				default = getenv("DB_URL")
			}
			env {
				name = atlas.env
				url  = var.url
				migration {
					dir = "file://migrations"
				}
			}`))
			return err
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ec.Close())
	})
	c, err := atlasexec.NewClient(ec.Path(), "atlas")
	require.NoError(t, err)
	got, err := c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		Env: "test",
	})
	require.ErrorContains(t, err, `required flag "url" not set`)
	require.Nil(t, got)
	var exerr *exec.ExitError
	require.ErrorAs(t, err, &exerr)
	// Set the env var and try again
	os.Setenv("DB_URL", "sqlite://file?_fk=1&cache=shared&mode=memory")
	got, err = c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		Env: "test",
	})
	require.NoError(t, err)
	require.Equal(t, "sqlite", got.Env.Driver)
	require.Equal(t, "file://migrations", got.Env.Dir)
	require.Equal(t, "sqlite://file?_fk=1&cache=shared&mode=memory", got.Env.URL.String())
	require.Equal(t, "20230926085734", got.Target)
	// Add dirty changes and try again
	os.Setenv("DB_URL", "sqlite://test.db?_fk=1&cache=shared&mode=memory")
	drv, err := sql.Open("sqlite3", "test.db")
	require.NoError(t, err)
	defer os.Remove("test.db")
	_, err = drv.ExecContext(context.Background(), "create table atlas_schema_revisions(version varchar(255) not null primary key);")
	require.NoError(t, err)
	got, err = c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		Env:        "test",
		AllowDirty: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, "20230926085734", got.Target)
}

func TestAtlasMigrate_ApplyWithRemote(t *testing.T) {
	type (
		ContextInput struct {
			TriggerType    string `json:"triggerType,omitempty"`
			TriggerVersion string `json:"triggerVersion,omitempty"`
		}
		graphQLQuery struct {
			Query              string          `json:"query"`
			Variables          json.RawMessage `json:"variables"`
			MigrateApplyReport struct {
				Input struct {
					Context *ContextInput `json:"context,omitempty"`
				} `json:"input"`
			}
		}
	)
	token := "123456789"
	handler := func(payloads *[]graphQLQuery) http.HandlerFunc {
		return func(_ http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
			var query graphQLQuery
			require.NoError(t, json.NewDecoder(r.Body).Decode(&query))
			*payloads = append(*payloads, query)
		}
	}
	var payloads []graphQLQuery
	srv := httptest.NewServer(handler(&payloads))
	t.Cleanup(srv.Close)
	ec, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(os.DirFS(filepath.Join("testdata", "migrations"))),
		atlasexec.WithAtlasHCL(func(w io.Writer) error {
			_, err := fmt.Fprintf(w, `
			env {
				name = atlas.env
				url  = "sqlite://file?_fk=1&cache=shared&mode=memory"
				migration {
					dir = "atlas://test_dir"
				}
			}
			atlas {
				cloud {
					token = %q
					url = %q
				}
			}`, token, srv.URL)
			return err
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ec.Close())
	})
	c, err := atlasexec.NewClient(ec.Path(), "atlas")
	require.NoError(t, err)
	got, err := c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		Env: "test",
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, payloads, 3)
	reportPayload := payloads[2]
	require.Regexp(t, "mutation ReportMigration", reportPayload.Query)
	err = json.Unmarshal(reportPayload.Variables, &reportPayload.MigrateApplyReport)
	require.NoError(t, err)
	require.Nil(t, reportPayload.MigrateApplyReport.Input.Context)
	got, err = c.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
		Env:     "test",
		Context: &atlasexec.DeployRunContext{TriggerVersion: "1.2.3", TriggerType: atlasexec.TriggerTypeGithubAction},
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, payloads, 6)
	reportPayload = payloads[5]
	require.Regexp(t, "mutation ReportMigration", reportPayload.Query)
	err = json.Unmarshal(reportPayload.Variables, &reportPayload.MigrateApplyReport)
	require.NoError(t, err)
	require.NotNil(t, reportPayload.MigrateApplyReport.Input.Context)
	require.Equal(t, "GITHUB_ACTION", reportPayload.MigrateApplyReport.Input.Context.TriggerType)
	require.Equal(t, "1.2.3", reportPayload.MigrateApplyReport.Input.Context.TriggerVersion)
}

func TestAtlasMigrate_Push(t *testing.T) {
	type (
		graphQLQuery struct {
			Query     string          `json:"query"`
			Variables json.RawMessage `json:"variables"`
			PushDir   *struct {
				Input struct {
					Slug   string `json:"slug"`
					Tag    string `json:"tag"`
					Driver string `json:"driver"`
					Dir    string `json:"dir"`
				} `json:"input"`
			}
			DiffSyncDir *struct {
				Input struct {
					Slug    string                `json:"slug"`
					Driver  string                `json:"driver"`
					Dir     string                `json:"dir"`
					Add     string                `json:"add"`
					Delete  []string              `json:"delete"`
					Context *atlasexec.RunContext `json:"context"`
				} `json:"input"`
			}
		}
		httpTest struct {
			payloads []graphQLQuery
			srv      *httptest.Server
		}
	)
	token := "123456789"
	newHTTPTest := func() (*httpTest, string) {
		tt := &httpTest{}
		handler := func() http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
				var query graphQLQuery
				require.NoError(t, json.NewDecoder(r.Body).Decode(&query))
				if strings.Contains(query.Query, "pushDir") {
					err := json.Unmarshal(query.Variables, &query.PushDir)
					require.NoError(t, err)
					fmt.Fprint(w, `{"data":{"pushDir":{"url":"https://some-org.atlasgo.cloud/dirs/314159/tags/12345"}}}`)
				}
				if strings.Contains(query.Query, "diffSyncDir") {
					err := json.Unmarshal(query.Variables, &query.DiffSyncDir)
					require.NoError(t, err)
					fmt.Fprint(w, `{"data":{"diffSyncDir":{"url":"https://some-org.atlasgo.cloud/dirs/314159/tags/12345"}}}`)
				}
				tt.payloads = append(tt.payloads, query)
			}
		}
		tt.srv = httptest.NewServer(handler())
		t.Cleanup(tt.srv.Close)
		return tt, generateHCL(t, token, tt.srv)
	}
	c, err := atlasexec.NewClient(".", "atlas")
	require.NoError(t, err)
	inputContext := &atlasexec.RunContext{
		Repo:     "testing-repo",
		Path:     "path/to/dir",
		Branch:   "testing-branch",
		Commit:   "sha123",
		URL:      "this://is/a/url",
		UserID:   "test-user-id",
		Username: "test-user",
		SCMType:  atlasexec.SCMTypeGithub,
	}
	t.Run("sync", func(t *testing.T) {
		params := &atlasexec.MigratePushParams{
			DevURL: "sqlite://file?mode=memory",
			DirURL: "file://testdata/migrations",
			Name:   "test-dir-slug",
			Env:    "test",
		}
		t.Run("with context", func(t *testing.T) {
			tt, atlasConfigURL := newHTTPTest()
			params.ConfigURL = atlasConfigURL
			got, err := c.MigratePush(context.Background(), params)
			require.NoError(t, err)
			require.Len(t, tt.payloads, 3)
			require.Equal(t, `https://some-org.atlasgo.cloud/dirs/314159/tags/12345`, got)
			p := &tt.payloads[2]
			require.Contains(t, p.Query, "diffSyncDir")
			require.Equal(t, "test-dir-slug", p.DiffSyncDir.Input.Slug)
			require.Equal(t, "SQLITE", p.DiffSyncDir.Input.Driver)
			require.NotEmpty(t, p.DiffSyncDir.Input.Dir)
		})
		t.Run("without context", func(t *testing.T) {
			tt, atlasConfigURL := newHTTPTest()
			params.ConfigURL = atlasConfigURL
			params.Context = inputContext
			got, err := c.MigratePush(context.Background(), params)
			require.NoError(t, err)
			require.Equal(t, `https://some-org.atlasgo.cloud/dirs/314159/tags/12345`, got)
			require.Len(t, tt.payloads, 3)
			p := &tt.payloads[2]
			require.Contains(t, p.Query, "diffSyncDir")
			err = json.Unmarshal(p.Variables, &p.DiffSyncDir)
			require.NoError(t, err)
			require.Equal(t, inputContext, p.DiffSyncDir.Input.Context)
		})

	})
	t.Run("push", func(t *testing.T) {
		tt, atlasConfigURL := newHTTPTest()
		params := &atlasexec.MigratePushParams{
			ConfigURL: atlasConfigURL,
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			Name:      "test-dir-slug",
			Context:   inputContext,
			Env:       "test",
			Tag:       "this-is-my-tag",
		}
		got, err := c.MigratePush(context.Background(), params)
		require.NoError(t, err)
		require.Equal(t, `https://some-org.atlasgo.cloud/dirs/314159/tags/12345`, got)
		require.Len(t, tt.payloads, 2)
		p := &tt.payloads[1]
		require.Contains(t, p.Query, "pushDir")
		require.Equal(t, "test-dir-slug", p.PushDir.Input.Slug)
		require.Equal(t, "SQLITE", p.PushDir.Input.Driver)
		require.Equal(t, "this-is-my-tag", p.PushDir.Input.Tag)
		require.NotEmpty(t, p.PushDir.Input.Dir)
	})
}

func TestMigrateHash(t *testing.T) {
	td := t.TempDir()
	require.NoError(t, os.Mkdir(fmt.Sprintf("%s/migrations", td), 0777))
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s/migrations/1.sql", td), []byte(`create table t (c int not null)`), 0666))
	c, err := atlasexec.NewClient(td, "atlas")
	require.NoError(t, err)
	inspect := func() error {
		_, err = c.SchemaInspect(context.Background(), &atlasexec.SchemaInspectParams{
			DevURL: "sqlite://file?mode=memory",
			URL:    fmt.Sprintf("file://%s/migrations", td),
		})
		return err
	}
	require.ErrorContains(t, inspect(), "checksum file not found")
	require.NoError(t, c.MigrateHash(context.Background(), &atlasexec.MigrateHashParams{}))
	require.FileExists(t, fmt.Sprintf("%s/migrations/atlas.sum", td))
	require.NoError(t, inspect())
}

func TestMigrateRebase(t *testing.T) {
	td := t.TempDir()
	require.NoError(t, os.Mkdir(fmt.Sprintf("%s/migrations", td), 0777))
	// create initial migrations dir state
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s/migrations/2024030709.sql", td), []byte(`create table t (c int not null)`), 0666))
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s/migrations/2024030711.sql", td), []byte("alter table `t` add column `c3` text not null;"), 0666))
	c, err := atlasexec.NewClient(td, "atlas")
	require.NoError(t, err)
	require.NoError(t, c.MigrateHash(context.Background(), &atlasexec.MigrateHashParams{}))
	require.FileExists(t, fmt.Sprintf("%s/migrations/atlas.sum", td))
	// Print atlas.sum before adding a new migration
	before, err := os.ReadFile(fmt.Sprintf("%s/migrations/atlas.sum", td))
	require.NoError(t, err)

	// add a new migration
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s/migrations/2024030710.sql", td), []byte("alter table `t` add column `c2` text not null;"), 0666))
	require.NoError(t, c.MigrateHash(context.Background(), &atlasexec.MigrateHashParams{}))
	require.FileExists(t, fmt.Sprintf("%s/migrations/atlas.sum", td))
	require.NoError(t, c.MigrateRebase(context.Background(), &atlasexec.MigrateRebaseParams{
		Files: []string{
			"2024030711.sql",
		},
		DirURL: fmt.Sprintf("file://%s/migrations", td),
	}))
	inspect := func() error {
		_, err = c.SchemaInspect(context.Background(), &atlasexec.SchemaInspectParams{
			DevURL: "sqlite://file?mode=memory",
			URL:    fmt.Sprintf("file://%s/migrations", td),
		})
		return err
	}
	// ensure sum file changes after rebase
	after, err := os.ReadFile(fmt.Sprintf("%s/migrations/atlas.sum", td))
	require.NotEqual(t, before, after)
	require.NoError(t, inspect())
}

func TestAtlasMigrate_Lint(t *testing.T) {
	t.Run("with broken config", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		got, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			ConfigURL: "file://config-broken.hcl",
		})
		require.ErrorContains(t, err, `file "config-broken.hcl" was not found`)
		require.Nil(t, got)
	})
	t.Run("with broken dev-url", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		got, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			DirURL: "file://atlasexec/testdata/migrations",
		})
		require.ErrorContains(t, err, `required flag(s) "dev-url" not set`)
		require.Nil(t, got)
	})
	t.Run("broken dir", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		got, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			DevURL: "sqlite://file?mode=memory",
			DirURL: "file://atlasexec/testdata/doesnotexist",
		})
		require.ErrorContains(t, err, `stat atlasexec/testdata/doesnotexist: no such file or directory`)
		require.Nil(t, got)
	})
	t.Run("lint error parsing", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		got, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			DevURL: "sqlite://file?mode=memory",
			DirURL: "file://testdata/migrations",
			Latest: 1,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, 4, len(got.Steps))
		require.Equal(t, "sqlite", got.Env.Driver)
		require.Equal(t, "testdata/migrations", got.Env.Dir)
		require.Equal(t, "sqlite://file?mode=memory", got.Env.URL.String())
		require.Equal(t, 1, len(got.Files))
		expectedReport := &atlasexec.FileReport{
			Name: "20230926085734_destructive-change.sql",
			Text: "DROP TABLE t2;\n",
			Reports: []sqlcheck.Report{{
				Text: "destructive changes detected",
				Diagnostics: []sqlcheck.Diagnostic{{
					Pos:  0,
					Text: `Dropping table "t2"`,
					Code: "DS102",
					SuggestedFixes: []sqlcheck.SuggestedFix{{
						Message: "Add a pre-migration check to ensure table \"t2\" is empty before dropping it",
						TextEdit: &sqlcheck.TextEdit{
							Line:    1,
							End:     1,
							NewText: "-- atlas:txtar\n\n-- checks/destructive.sql --\n-- atlas:assert DS102\nSELECT NOT EXISTS (SELECT 1 FROM `t2`) AS `is_empty`;\n\n-- migration.sql --\nDROP TABLE t2;",
						},
					}},
				}},
			}},
			Error: "destructive changes detected",
		}
		require.EqualValues(t, expectedReport, got.Files[0])
	})
	t.Run("lint with manually parsing output", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		var buf bytes.Buffer
		err = c.MigrateLintError(context.Background(), &atlasexec.MigrateLintParams{
			DevURL: "sqlite://file?mode=memory",
			DirURL: "file://testdata/migrations",
			Latest: 1,
			Writer: &buf,
		})
		require.Equal(t, atlasexec.ErrLint, err)
		var raw json.RawMessage
		require.NoError(t, json.NewDecoder(&buf).Decode(&raw))
		require.Contains(t, string(raw), "destructive changes detected")
	})
	t.Run("lint uses --base and --latest", func(t *testing.T) {
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		summary, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			DevURL: "sqlite://file?mode=memory",
			DirURL: "file://testdata/migrations",
			Latest: 1,
			Base:   "atlas://test-dir",
		})
		require.ErrorContains(t, err, "--latest, --git-base, and --base are mutually exclusive")
		require.Nil(t, summary)
	})
}

func TestAtlasMigrate_LintWithLogin(t *testing.T) {
	type (
		migrateLintReport struct {
			Context *atlasexec.RunContext `json:"context"`
		}
		graphQLQuery struct {
			Query             string          `json:"query"`
			Variables         json.RawMessage `json:"variables"`
			MigrateLintReport struct {
				migrateLintReport `json:"input"`
			}
		}
		Dir struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Slug    string `json:"slug"`
		}
		dirsQueryResponse struct {
			Data struct {
				Dirs []Dir `json:"dirs"`
			} `json:"data"`
		}
	)
	token := "123456789"
	handler := func(payloads *[]graphQLQuery) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
			var query graphQLQuery
			require.NoError(t, json.NewDecoder(r.Body).Decode(&query))
			*payloads = append(*payloads, query)
			switch {
			case strings.Contains(query.Query, "mutation reportMigrationLint"):
				_, err := fmt.Fprintf(w, `{ "data": { "reportMigrationLint": { "url": "https://migration-lint-report-url" } } }`)
				require.NoError(t, err)
			case strings.Contains(query.Query, "query dirs"):
				dir, err := migrate.NewLocalDir("./testdata/migrations")
				require.NoError(t, err)
				ad, err := migrate.ArchiveDir(dir)
				require.NoError(t, err)
				var resp dirsQueryResponse
				resp.Data.Dirs = []Dir{{
					Name:    "test-dir-name",
					Slug:    "test-dir-slug",
					Content: base64.StdEncoding.EncodeToString(ad),
				}}
				st2bytes, err := json.Marshal(resp)
				require.NoError(t, err)
				_, err = fmt.Fprint(w, string(st2bytes))
				require.NoError(t, err)
			}
		}
	}
	t.Run("Web and Writer params produces an error", func(t *testing.T) {
		var payloads []graphQLQuery
		srv := httptest.NewServer(handler(&payloads))
		t.Cleanup(srv.Close)
		atlasConfigURL := generateHCL(t, token, srv)
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		params := &atlasexec.MigrateLintParams{
			ConfigURL: atlasConfigURL,
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			Latest:    1,
			Web:       true,
		}
		got, err := c.MigrateLint(context.Background(), params)
		require.ErrorContains(t, err, "Writer or Web reporting are not supported")
		require.Nil(t, got)
		params.Web = false
		params.Writer = &bytes.Buffer{}
		got, err = c.MigrateLint(context.Background(), params)
		require.ErrorContains(t, err, "Writer or Web reporting are not supported")
		require.Nil(t, got)
	})
	t.Run("lint parse web output - no error - custom format", func(t *testing.T) {
		var payloads []graphQLQuery
		srv := httptest.NewServer(handler(&payloads))
		t.Cleanup(srv.Close)
		atlasConfigURL := generateHCL(t, token, srv)
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		var buf bytes.Buffer
		err = c.MigrateLintError(context.Background(), &atlasexec.MigrateLintParams{
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			ConfigURL: atlasConfigURL,
			Latest:    1,
			Writer:    &buf,
			Format:    "{{ .URL }}",
			Web:       true,
		})
		require.Equal(t, err, atlasexec.ErrLint)
		require.Equal(t, strings.TrimSpace(buf.String()), "https://migration-lint-report-url")
	})
	t.Run("lint parse web output - no error - default format", func(t *testing.T) {
		var payloads []graphQLQuery
		srv := httptest.NewServer(handler(&payloads))
		t.Cleanup(srv.Close)
		atlasConfigURL := generateHCL(t, token, srv)
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		var buf bytes.Buffer
		err = c.MigrateLintError(context.Background(), &atlasexec.MigrateLintParams{
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			ConfigURL: atlasConfigURL,
			Latest:    1,
			Writer:    &buf,
			Web:       true,
		})
		require.Equal(t, atlasexec.ErrLint, err)
		var sr atlasexec.SummaryReport
		require.NoError(t, json.NewDecoder(&buf).Decode(&sr))
		require.Equal(t, "https://migration-lint-report-url", sr.URL)
	})
	t.Run("lint uses --base", func(t *testing.T) {
		var payloads []graphQLQuery
		srv := httptest.NewServer(handler(&payloads))
		t.Cleanup(srv.Close)
		atlasConfigURL := generateHCL(t, token, srv)
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		summary, err := c.MigrateLint(context.Background(), &atlasexec.MigrateLintParams{
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			ConfigURL: atlasConfigURL,
			Base:      "atlas://test-dir-slug",
		})
		require.NoError(t, err)
		require.NotNil(t, summary)
	})
	t.Run("lint uses --context has error", func(t *testing.T) {
		var payloads []graphQLQuery
		srv := httptest.NewServer(handler(&payloads))
		t.Cleanup(srv.Close)
		c, err := atlasexec.NewClient(".", "atlas")
		require.NoError(t, err)
		var (
			buf            bytes.Buffer
			atlasConfigURL = generateHCL(t, token, srv)
			runContext     = &atlasexec.RunContext{
				Repo:     "testing-repo",
				Path:     "path/to/dir",
				Branch:   "testing-branch",
				Commit:   "sha123",
				URL:      "this://is/a/url",
				Username: "test-user",
				UserID:   "test-user-id",
				SCMType:  atlasexec.SCMTypeGithub,
			}
		)
		err = c.MigrateLintError(context.Background(), &atlasexec.MigrateLintParams{
			DevURL:    "sqlite://file?mode=memory",
			DirURL:    "file://testdata/migrations",
			ConfigURL: atlasConfigURL,
			Base:      "atlas://test-dir-slug",
			Context:   runContext,
			Writer:    &buf,
			Web:       true,
		})
		require.Equal(t, atlasexec.ErrLint, err)
		var sr atlasexec.SummaryReport
		require.NoError(t, json.NewDecoder(&buf).Decode(&sr))
		require.Equal(t, "https://migration-lint-report-url", sr.URL)
		found := false
		for _, query := range payloads {
			if !strings.Contains(query.Query, "mutation reportMigrationLint") {
				continue
			}
			found = true
			require.NoError(t, json.Unmarshal(query.Variables, &query.MigrateLintReport))
			require.Equal(t, runContext, query.MigrateLintReport.Context)
		}
		require.True(t, found)
	})
}

func TestMigrate_Diff(t *testing.T) {
	c, err := atlasexec.NewClient(".", "atlas")
	require.NoError(t, err)
	td := t.TempDir()
	require.NoError(t, os.WriteFile(fmt.Sprintf("%s/schema.sql", td), []byte(`create table t (c int not null)`), 0666))
	params := &atlasexec.MigrateDiffParams{
		ToURL:  fmt.Sprintf("file://%s/schema.sql", td),
		DevURL: "sqlite://file?mode=memory",
		DirURL: "file://testdata/migrations",
		Name:   "test-diff",
	}
	output, err := c.MigrateDiff(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, output.Files, 1)
	require.Contains(t, output.Files[0].Name, "test-diff.sql")
	require.Equal(t, output.Files[0].Content, "-- Disable the enforcement of foreign-keys constraints\nPRAGMA foreign_keys = off;\n-- Drop \"t1\" table\nDROP TABLE `t1`;\n-- Create \"t\" table\nCREATE TABLE `t` (\n  `c` int NOT NULL\n);\n-- Enable back the enforcement of foreign-keys constraints\nPRAGMA foreign_keys = on;\n")
	require.Equal(t, output.Dir, "file://testdata/migrations?format=atlas")

	// No diff
	params = &atlasexec.MigrateDiffParams{
		ToURL:  "file://testdata/migrations",
		DevURL: "sqlite://file?mode=memory",
		DirURL: "file://testdata/migrations",
		Name:   "test-diff",
	}
	output, err = c.MigrateDiff(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, output.Files, 0)
}
