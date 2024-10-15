// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cloudapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestClient_Dir(t *testing.T) {
	var dir migrate.MemDir
	require.NoError(t, dir.WriteFile("1.sql", []byte("create table foo (id int)")))
	ad, err := migrate.ArchiveDir(&dir)
	require.NoError(t, err)
	SetVersion("v0.13.0", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Variables struct {
				DirInput DirInput `json:"input"`
			} `json:"variables"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		require.Equal(t, "foo", input.Variables.DirInput.Name)
		require.Equal(t, "x", input.Variables.DirInput.Tag)
		require.Equal(t, "Bearer atlas", r.Header.Get("Authorization"))
		expUA := fmt.Sprintf("Atlas/v0.13.0 (%s/%s)", runtime.GOOS, runtime.GOARCH)
		require.Equal(t, expUA, r.Header.Get("User-Agent"))
		fmt.Fprintf(w, `{"data":{"dirState":{"content":%q}}}`, base64.StdEncoding.EncodeToString(ad))
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	gd, err := client.Dir(context.Background(), DirInput{
		Name: "foo",
		Tag:  "x",
	})
	require.NoError(t, err)
	gcheck, err := gd.Checksum()
	require.NoError(t, err)
	dcheck, err := dir.Checksum()
	require.NoError(t, err)
	require.Equal(t, dcheck.Sum(), gcheck.Sum())
}

func TestClient_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, err := w.Write([]byte(`{"errors":[{"message":"error\n","path":["variable","input","driver"],"extensions":{}}],"data":null}`))
		require.NoError(t, err)
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	link, err := client.ReportMigration(context.Background(), ReportMigrationInput{
		EnvName:     "foo",
		ProjectName: "bar",
	})
	require.EqualError(t, err, "variable.input.driver error", "error is trimmed")
	require.Empty(t, link)
}

func TestClient_ReportMigration(t *testing.T) {
	const project, env = "atlas", "dev"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Variables struct {
				Input ReportMigrationInput `json:"input"`
			} `json:"variables"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		require.Equal(t, env, input.Variables.Input.EnvName)
		require.Equal(t, project, input.Variables.Input.ProjectName)
		fmt.Fprintf(w, `{"data":{"reportMigration":{"url":"https://atlas.com"}}}`)
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	link, err := client.ReportMigration(context.Background(), ReportMigrationInput{
		EnvName:     env,
		ProjectName: project,
	})
	require.NoError(t, err)
	require.NotEmpty(t, link)
}

func TestClient_ReportMigrationSet(t *testing.T) {
	const (
		planned               = 2
		id, log, project, env = "deployment-set-1", "started deployment", "atlas", "dev"
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Variables struct {
				Input ReportMigrationSetInput `json:"input"`
			} `json:"variables"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		require.Equal(t, id, input.Variables.Input.ID)
		require.Equal(t, []ReportStep{{Text: log}}, input.Variables.Input.Log)
		require.Equal(t, planned, input.Variables.Input.Planned)
		require.Equal(t, env, input.Variables.Input.Completed[0].EnvName)
		require.Equal(t, project, input.Variables.Input.Completed[0].ProjectName)
		require.Equal(t, "dir-1", input.Variables.Input.Completed[0].DirName)
		require.Equal(t, env, input.Variables.Input.Completed[1].EnvName)
		require.Equal(t, project, input.Variables.Input.Completed[1].ProjectName)
		require.Equal(t, "dir-2", input.Variables.Input.Completed[1].DirName)
		fmt.Fprintf(w, `{"data":{"reportMigrationSet":{"url":"https://atlas.com"}}}`)
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	link, err := client.ReportMigrationSet(context.Background(), ReportMigrationSetInput{
		ID:      id,
		Planned: planned,
		Log:     []ReportStep{{Text: log}},
		Completed: []ReportMigrationInput{
			{
				EnvName:     env,
				ProjectName: project,
				DirName:     "dir-1",
			},
			{
				EnvName:     env,
				ProjectName: project,
				DirName:     "dir-2",
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, link)
}

func TestRedactedURL(t *testing.T) {
	u, err := RedactedURL("mysql://user:pass@:3306/db")
	require.NoError(t, err)
	require.Equal(t, "mysql://user:xxxxx@:3306/db", u)
	u, err = RedactedURL("\\n mysql://user:pass@:3306/db")
	require.EqualError(t, err, `first path segment in URL cannot contain colon`)
	require.Empty(t, u)
}

func TestUserAgent(t *testing.T) {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	require.Equal(t, fmt.Sprintf("Atlas/%s (%s)", version, platform), UserAgent())
	require.Equal(t, fmt.Sprintf("Atlas/%s (%s; foo/bar; bar/baz)", version, platform), UserAgent("foo/bar", "bar/baz"))
	require.Equal(t, fmt.Sprintf("Atlas/%s (%s; bar/baz)", version, platform), UserAgent("  ", "", "bar/baz"))
}

func TestClient_AddHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "val", r.Header.Get("key"))
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	client.AddHeader("key", "val")
	_, err := client.ReportMigration(context.Background(), ReportMigrationInput{
		EnvName:     "foo",
		ProjectName: "bar",
	})
	require.NoError(t, err)
}

func TestClient_Retry(t *testing.T) {
	var (
		calls = []int{http.StatusInternalServerError, http.StatusInternalServerError, http.StatusOK}
		srv   = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "val", r.Header.Get("key"))
			require.Equal(t, "Bearer atlas", r.Header.Get("Authorization"))
			w.WriteHeader(calls[0])
			calls = calls[1:]
		}))
		client = New(srv.URL, "atlas")
	)
	defer srv.Close()
	client.AddHeader("key", "val")
	_, err := client.ReportMigration(context.Background(), ReportMigrationInput{
		EnvName: "foo",
	})
	require.NoError(t, err)
	require.Empty(t, calls)
}
