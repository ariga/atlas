// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cmdext_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/cloudapi"
	"ariga.io/atlas/cmd/atlas/internal/cmdext"
	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestRuntimeVarSrc(t *testing.T) {
	var (
		v struct {
			V string `spec:"v"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world&decoder=binary"
}

v = data.runtimevar.pass
`), &v, nil)
	require.EqualError(t, err, `data.runtimevar.pass: unsupported decoder: "binary"`)

	err = state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world"
}

v = data.runtimevar.pass
`), &v, nil)
	require.NoError(t, err)
	require.Equal(t, v.V, "hello world")

	err = state.EvalBytes([]byte(`
data "runtimevar" "pass" {
  url = "constant://?val=hello+world&decoder=string"
}

v = data.runtimevar.pass
`), &v, nil)
	require.NoError(t, err, "nop decoder")
	require.Equal(t, v.V, "hello world")
}

func TestRDSToken(t *testing.T) {
	t.Cleanup(
		backupEnv("AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"),
	)
	// Mock AWS env vars.
	require.NoError(t, os.Setenv("AWS_ACCESS_KEY_ID", "EXAMPLE_KEY_ID"))
	require.NoError(t, os.Setenv("AWS_SECRET_ACCESS_KEY", "EXAMPLE_SECRET_KEY"))
	var (
		v struct {
			V string `spec:"v"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(`
data "aws_rds_token" "token" {
	endpoint = "localhost:3306"
	region = "us-east-1"
	username = "root"
}
v = data.aws_rds_token.token
`), &v, nil)
	require.NoError(t, err)
	parse, err := url.Parse(v.V)
	require.NoError(t, err)
	q := parse.Query()
	require.Equal(t, "connect", q.Get("Action"))
	require.Contains(t, q.Get("X-Amz-Credential"), "EXAMPLE_KEY_ID")
}

// TestRDSTokenProfile verifies the profile option propagates to the AWS SDK.
func TestRDSTokenProfile(t *testing.T) {
	doc := `
data "aws_rds_token" "token" {
	username = "root"
	endpoint = "localhost:3306"
	region = "us-east-1"
	profile = "errorneous"
}

v = data.aws_rds_token.token
`
	var (
		v struct {
			V string `spec:"v"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(doc), &v, nil)
	require.EqualError(t, err, "data.aws_rds_token.token: loading aws config: failed to get shared config profile, errorneous")
}

func TestGCPToken(t *testing.T) {
	t.Cleanup(
		backupEnv("GOOGLE_APPLICATION_CREDENTIALS"),
	)
	credsFile := filepath.Join(t.TempDir(), "foo.json")
	require.NoError(t, os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credsFile))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/token" {
			t.Errorf("Unexpected exchange request URL, %v is found.", r.URL)
		}
		jwt := r.FormValue("assertion")
		payload, err := base64.RawStdEncoding.DecodeString(strings.Split(jwt, ".")[1])
		require.NoError(t, err)
		// Ensure we request correct scopes
		require.Contains(t, string(payload), `"https://www.googleapis.com/auth/sqlservice.admin"`)
		w.Header().Set("Content-Type", "application/json")
		// Write a fake access token to the client
		w.Write([]byte(`{"access_token":"foo-bar-token","scope":"user","token_type":"bearer","expires_in":86400}`))
	}))
	defer ts.Close()
	require.NoError(t, os.WriteFile(credsFile, []byte(fmt.Sprintf(`{
		"type": "service_account",
		"project_id": "foo-bar",
		"private_key_id": "foo-bar",
		"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCxLnH1p8E1IiWw\nQrSv8BtXfOaFzPvYt6tcwsti9O3LhG6KtTEbXXbUe72tga5B8awQXYkRtdST2uV6\nhjvFHzmHzLOJVa/Qm1duO4iTkjz7Hj7kbfEI4dF5iLRn8+QF8YwGJCewSS8IXmbl\nu/4w64dtdC5h880p33gW73oNSLr6d6tlifc/oUAVdu4Bz8qSARpF+4nIN3uZGqr1\n8wqsHx9N5twaEO7Ky5ezNWv2TfiBk4hPtGJUWXPM++mKZpZcmpzXT9dP9gfPX0mN\np45FNXhjN8uA7aauqVzl2dWYmED32k1EGK2/m66lPq+IEo7p/90FUbvR/x+pbx0r\nYOgGKrhfAgMBAAECggEAVUWqGPVcmirOAq+H8GjZb9ivxVNrHdj/gwxJAF4ql9kr\nrlwXvzjTON444mlYKWqbSeEKV9iv71zZNoel+m/Vq1LMUVtI21f30xiZ2ZP2/1CG\nKj/zUjgELb6qPKF3a5jdsBL0evYtyZRNZ2F7q6WfLwFMVV4VroJbdIZaskv/mQzx\ny45FWU14/J/Vuk6Bqv0AtWb3ZSGnKGRWjOSlr9OI8nXEDg69LE3pGB3/XrPtfrbo\n7YdFC24DFUXRUIkNHnktQZ14U+0HmbPgs6OWUNvMfdvckP87e+7eoBiUkPrJA1wi\nrSm2ZW70Wvf1sD2h9kgpABe+cuWoqWTWBBXlfkuwUQKBgQD3MjGN8QIfhIKMEz9X\nFkL9BdFPswcawVaiTXfrhHtPmcJLmT5VGEnyh6jvigdKSpQe/s5IzLnFglqKO5Ge\nW57YiBVwfNREpzahJULaAL45NtwJtasSz1tNz3EKm00Z5o6tcCk2dZ6rzFKRY3Sz\nUfSo0lc7+rfNQzC4+GVlxTcNNwKBgQC3feTmNL917xceMwAA3g0nh+aHi4rPIN3H\nkhghDvCYMg4gYml/vZnUMkjfTsdS/TrXvIE1Pd6QDCSRx/VZFIBFA2P5c+g6l5fo\nBSS5CUm+R3j27NsGQXIfr5bANuKECjugZtbmsZ2taAtzLVjoO1yDDFBf9FWie9I8\nnbKmr9ACGQKBgQD2yt/6jEHIYa1MV/MG6SzcHDDK1zwilCAATkOJmWzbHfGDNG2s\n22EIiDQ7YpzAqRCUmWQt/mcCL5BhLfPGHEbMe6Cb+6SZHjBGVkMWD2PbD1BDSWKQ\nlwDbAF4lbsNdNnf/5FjhDDDr6EQO7zKVzR7sZYO+WCOlBI3iPexN3MWHpQKBgGYA\nxk5y5DxbPS68izPwPL/M/Io9OF0MmD1pKaC2/Wid6tx12M/6Rpl/mqMI2CV6QEvN\nrsY6Lo9FMM8ZqXpruyKiT+FMXby0qO2CbneugiAU+1nJMbi4iQi0Q8l2uVVNmvgA\nM1brRgwv2q2cd+Ahn7v6DHRLD4/T5Xts7vNaqPeBAoGAQZ/Yzp40aDvlv9D6MUKi\ngDvmjQPeI6H08MlCLTnbzJusf1nL3whVa5xbbp7+iVl0nMLzogxNC0dCNUUzdXov\n/PxhteomqwnQb9He0PSSYKQUoL+iHoTy3BY+jNPsCNsWgNm04k/vaB5le4zipc6M\npEWCIJtjmdEC1tzBtTEN1aY=\n-----END PRIVATE KEY-----\n",
		"client_email": "foo@bar.iam.gserviceaccount.com",
		"client_id": "100000000000000000000",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "%s/token",
		"auth_provider_x509_cert_url": "",
		"client_x509_cert_url": "",
		"universe_domain": "googleapis.com"
	}`, ts.URL)), 0644))
	var (
		v struct {
			V string `spec:"v"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(`
data "gcp_cloudsql_token" "helloworld" {}
v = data.gcp_cloudsql_token.helloworld
`), &v, nil)
	require.NoError(t, err)
	require.Equal(t, "foo-bar-token", v.V)
}

func TestQuerySrc(t *testing.T) {
	ctx := context.Background()
	u := fmt.Sprintf("sqlite3://file:%s?cache=shared&_fk=1", filepath.Join(t.TempDir(), "test.db"))
	drv, err := sqlclient.Open(context.Background(), u)
	require.NoError(t, err)
	_, err = drv.ExecContext(ctx, "CREATE TABLE users (name text)")
	require.NoError(t, err)
	_, err = drv.ExecContext(ctx, "INSERT INTO users (name) VALUES ('a8m')")
	require.NoError(t, err)

	var (
		v struct {
			C  int      `spec:"c"`
			V  string   `spec:"v"`
			Vs []string `spec:"vs"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err = state.EvalBytes([]byte(fmt.Sprintf(`
data "sql" "user" {
  url   = %q
  query = "SELECT name FROM users"
}

c = data.sql.user.count
v = data.sql.user.value
vs = data.sql.user.values
`, u)), &v, nil)
	require.NoError(t, err)
	require.Equal(t, 1, v.C)
	require.Equal(t, "a8m", v.V)
	require.Equal(t, []string{"a8m"}, v.Vs)
}

func TestTemplateDir(t *testing.T) {
	var (
		v struct {
			Dir string `spec:"dir"`
		}
		dir   = t.TempDir()
		ctx   = context.Background()
		state = schemahcl.New(cmdext.SpecOptions...)
		// language=hcl
		cfg = `
variable "path" {
  type = string
}

data "template_dir" "tenant" {
  path = var.path
  vars = {
    Schema = "main"
  }
}

dir = data.template_dir.tenant.url
`
	)
	err := os.WriteFile(filepath.Join(dir, "1.sql"), []byte("create table {{ .Schema }}.t(c int);"), 0644)
	require.NoError(t, err)
	err = state.EvalBytes([]byte(cfg), &v, map[string]cty.Value{
		"path": cty.StringVal(dir),
	})
	require.NoError(t, err)
	require.NotEmpty(t, v.Dir)
	d := migrate.OpenMemDir(strings.TrimPrefix(v.Dir, "mem://"))
	require.NoError(t, migrate.Validate(d))
	files, err := d.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "1.sql", files[0].Name())
	require.Equal(t, "create table main.t(c int);", string(files[0].Bytes()))

	// Directory was loaded to memory as source for readers.
	mem, ok := cmdext.States.Loader("mem")
	require.True(t, ok)
	u, err := url.Parse(v.Dir)
	require.NoError(t, err)
	dev, err := sqlclient.Open(ctx, "sqlite://test?mode=memory")
	require.NoError(t, err)
	sr, err := mem.LoadState(ctx, &cmdext.StateReaderConfig{
		URLs: []*url.URL{u},
		Dev:  dev,
	})
	require.NoError(t, err)
	r, err := sr.ReadState(ctx)
	require.NoError(t, err)
	require.Len(t, r.Schemas[0].Tables, 1)

	// Should not accept non-directories.
	err = state.EvalBytes([]byte(cfg), &v, map[string]cty.Value{
		"path": cty.StringVal(filepath.Join(dir, "1.sql")),
	})
	require.ErrorContains(t, err, "data.template_dir.tenant: path", "error prefix")
	require.ErrorContains(t, err, "1.sql is not a directory", "error suffix")
}

func TestSchemaHCL(t *testing.T) {
	var (
		v struct {
			Schema string `spec:"schema"`
		}
		dir   = t.TempDir()
		ctx   = context.Background()
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(`
variable "schema" {
  type = string
}

schema "dynamic" {
  name = var.schema
}

table "t" {
  schema = schema.dynamic
  column "c" {
    type = int
  }
}
`), 0644)
	require.NoError(t, err)
	err = state.EvalBytes([]byte(`
variable "path" {
  type = string
}

data "hcl_schema" "a8m" {
  path = var.path
  vars = {
    schema = "a8m"
  }
}

schema = data.hcl_schema.a8m.url
`), &v, map[string]cty.Value{
		"path": cty.StringVal(dir),
	})
	require.NoError(t, err)
	require.NotEmpty(t, v.Schema)
	u, err := url.Parse(v.Schema)
	require.NoError(t, err)
	loader, ok := cmdext.States.Loader(u.Scheme)
	require.True(t, ok)
	drv, err := sqlclient.Open(ctx, "sqlite://test?mode=memory&_fk=1")
	require.NoError(t, err)
	sr, err := loader.LoadState(ctx, &cmdext.StateReaderConfig{
		Dev:  drv,
		URLs: []*url.URL{u},
		// Variables are not needed at this stage,
		// as they are defined on the data source.
	})
	require.NoError(t, err)
	realm, err := sr.ReadState(ctx)
	require.NoError(t, err)
	buf, err := drv.MarshalSpec(realm)
	require.NoError(t, err)
	require.Equal(t, `table "t" {
  schema = schema.a8m
  column "c" {
    null = false
    type = int
  }
}
schema "a8m" {
}
`, string(buf))

	// An empty schema case.
	err = os.WriteFile(filepath.Join(dir, "schema.hcl"), []byte(``), 0644)
	require.NoError(t, err)
	sr, err = loader.LoadState(ctx, &cmdext.StateReaderConfig{
		Dev:  drv,
		URLs: []*url.URL{u},
		// Variables are not needed at this stage,
		// as they are defined on the data source.
	})
	require.NoError(t, err)
	realm, err = sr.ReadState(ctx)
	require.NoError(t, err)
	buf, err = drv.MarshalSpec(realm)
	require.NoError(t, err)
	require.Equal(t, ``, string(buf))
}

func TestExternalSchema(t *testing.T) {
	var (
		v struct {
			Schema string `spec:"schema"`
		}
		ctx   = context.Background()
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(`
data "external_schema" "a8m" {
  program = [
    "echo",
    "CREATE TABLE t(c int);",
  ]
}

schema = data.external_schema.a8m.url
`), &v, nil)
	require.NoError(t, err)
	require.NotEmpty(t, v.Schema)
	u, err := url.Parse(v.Schema)
	require.NoError(t, err)
	loader, ok := cmdext.States.Loader(u.Scheme)
	require.True(t, ok)
	drv, err := sqlclient.Open(ctx, "sqlite://test?mode=memory&_fk=1")
	require.NoError(t, err)
	sr, err := loader.LoadState(ctx, &cmdext.StateReaderConfig{
		Dev:  drv,
		URLs: []*url.URL{u},
	})
	require.NoError(t, err)
	realm, err := sr.ReadState(ctx)
	require.NoError(t, err)
	buf, err := drv.MarshalSpec(realm)
	require.NoError(t, err)
	require.Equal(t, `table "t" {
  schema = schema.main
  column "c" {
    null = true
    type = int
  }
}
schema "main" {
}
`, string(buf))

	// Read state error.
	err = state.EvalBytes([]byte(`
data "external_schema" "a8m" {
  program = [
    "echo",
    "CREATE TABLE t(c int);",
    "CREATE UNKNOWN",
  ]
}

schema = data.external_schema.a8m.url
`), &v, nil)
	require.NoError(t, err)
	loader, ok = cmdext.States.Loader(u.Scheme)
	require.True(t, ok)
	sr, err = loader.LoadState(ctx, &cmdext.StateReaderConfig{
		Dev:  drv,
		URLs: []*url.URL{u},
	})
	require.EqualError(t, err, `read state from "a8m/schema.sql": executing statement: "CREATE UNKNOWN": near "UNKNOWN": syntax error`)
}

func TestExternal(t *testing.T) {
	var (
		v struct {
			Output string `spec:"output"`
		}
		state = schemahcl.New(cmdext.SpecOptions...)
	)
	err := state.EvalBytes([]byte(`
data "external" "program" {
  program = [
    "echo",
    "value",
  ]
}

output = trimspace(data.external.program)
`), &v, nil)
	require.NoError(t, err)
	require.Equal(t, "value", v.Output)

	err = state.EvalBytes([]byte(`
data "external" "program" {
  program = [
    "echo",
    "{\"hello\": \"world\"}",
  ]
}

output = jsondecode(data.external.program).hello
`), &v, nil)
	require.NoError(t, err)
	require.Equal(t, "world", v.Output)

	err = state.EvalBytes([]byte(`
variable "dot_env" {
  type = string
}

data "external" "dot_env" {
  program = [
    "echo",
	"${var.dot_env}",
  ]
}

locals {
  dot_env = jsondecode(data.external.dot_env)
}

output = local.dot_env.URL
`), &v, map[string]cty.Value{
		"dot_env": cty.StringVal(`{"URL": "https://example.com"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com", v.Output)
}

func TestAtlasConfig(t *testing.T) {
	var (
		v struct {
			Env       string    `spec:"env"`
			HasClient bool      `spec:"has_client"`
			CloudKeys []string  `spec:"cloud_keys"`
			Atlas     cty.Value `spec:"atlas"`
		}
		cfg   = &cmdext.AtlasConfig{}
		state = schemahcl.New(append(cmdext.SpecOptions, cfg.InitBlock(), schemahcl.WithVariables(map[string]cty.Value{
			"atlas": cty.ObjectVal(map[string]cty.Value{
				"env": cty.StringVal("dev"),
			}),
		}))...)
	)
	require.Nil(t, cfg.Client)
	require.Empty(t, cfg.Project)
	err := state.EvalBytes([]byte(`
atlas {
  cloud {
    url = "url"
    token = "token"
    project = "atlasgo.io"
  }
}

env = atlas.env
has_client = atlas.cloud != null
cloud_keys = keys(atlas.cloud)
`), &v, map[string]cty.Value{})
	require.NoError(t, err)
	require.Equal(t, "dev", v.Env)
	require.True(t, v.HasClient)
	require.Equal(t, []string{"client", "project"}, v.CloudKeys, "token and url should not be exported")
	// Config options should be populated from the init block.
	require.NotNil(t, cfg.Client)
	require.Equal(t, "token", cfg.Token)
	require.Equal(t, "atlasgo.io", cfg.Project)

	err = state.EvalBytes([]byte(`
atlas {
  cloud {
    url = "url"
    token = "token"
  }
}
`), &v, map[string]cty.Value{})
	require.NoError(t, err)
	require.Equal(t, cloudapi.DefaultProjectName, cfg.Project)
}

func TestRemoteDir(t *testing.T) {
	var (
		v struct {
			Dir string `spec:"dir"`
		}
		token, tag string
		cfg        = &cmdext.AtlasConfig{}
		state      = schemahcl.New(append(cmdext.SpecOptions, cfg.InitBlock())...)
		srv        = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token = r.Header.Get("Authorization")
			di := struct {
				Variables struct {
					DirInput cloudapi.DirInput `json:"input"`
				} `json:"variables"`
			}{}
			err := json.NewDecoder(r.Body).Decode(&di)
			require.NoError(t, err)
			tag = di.Variables.DirInput.Tag
			d := migrate.MemDir{}
			if err := d.WriteFile("1.sql", []byte("create table t(c int);")); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			sum, err := d.Checksum()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if err := migrate.WriteSumFile(&d, sum); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			arch, err := migrate.ArchiveDir(&d)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			fmt.Fprintf(w, `{"data":{"dirState":{"content":%q}}}`, base64.StdEncoding.EncodeToString(arch))
		}))
	)
	defer srv.Close()

	err := state.EvalBytes([]byte(`
data "remote_dir" "hello" {
  name  = "atlas"
}

dir = data.remote_dir.hello.url
`), &v, map[string]cty.Value{"cloud_url": cty.StringVal(srv.URL)})
	require.EqualError(t, err, "data.remote_dir.hello: missing atlas cloud config")

	err = state.EvalBytes([]byte(`
variable "cloud_url" {
  type = string
}

variable "tag" {
  type = string
}

atlas {
  cloud {
    token = "token"
    url = var.cloud_url
  }
}

data "remote_dir" "hello" {
  name  = "atlas"
  tag = var.tag
}

dir = data.remote_dir.hello.url
`), &v, map[string]cty.Value{"cloud_url": cty.StringVal(srv.URL), "tag": cty.StringVal("xyz")})
	require.NoError(t, err)
	require.Equal(t, "Bearer token", token)
	require.Equal(t, "xyz", tag)
	md := migrate.OpenMemDir(strings.TrimPrefix(v.Dir, "mem://"))
	defer md.Close()
	files, err := md.Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "1.sql", files[0].Name())
	require.Equal(t, "create table t(c int);", string(files[0].Bytes()))
	require.NoError(t, migrate.Validate(md))
	_, err = md.Open(migrate.HashFileName)
	require.NoError(t, err)
}

// backupEnv backs up the current value of an environment variable
// and returns a function to restore it.
func backupEnv(keys ...string) (restoreFunc func()) {
	backup := make(map[string]string, len(keys))
	for _, key := range keys {
		originalValue, exists := os.LookupEnv(key)
		if exists {
			backup[key] = originalValue
		}
	}
	return func() {
		for _, key := range keys {
			if originalValue, exists := backup[key]; exists {
				os.Setenv(key, originalValue)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
