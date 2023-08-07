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
	"testing"

	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestClient_Dir(t *testing.T) {
	var dir migrate.MemDir
	require.NoError(t, dir.WriteFile("1.sql", []byte("create table foo (id int)")))
	ad, err := migrate.ArchiveDir(&dir)
	require.NoError(t, err)
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
		require.Equal(t, "atlas-cli/v1", r.Header.Get("User-Agent"))
		fmt.Fprintf(w, `{"data":{"dir":{"content":%q}}}`, base64.StdEncoding.EncodeToString(ad))
	}))
	client := New(srv.URL, "atlas", "v1")
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
	}))
	client := New(srv.URL, "atlas", "v1")
	defer srv.Close()
	err := client.ReportMigration(context.Background(), ReportMigrationInput{
		EnvName:     env,
		ProjectName: project,
	})
	require.NoError(t, err)
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
	}))
	client := New(srv.URL, "atlas", "v1")
	defer srv.Close()
	err := client.ReportMigrationSet(context.Background(), ReportMigrationSetInput{
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
}
