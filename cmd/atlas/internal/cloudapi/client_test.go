// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cloudapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	var (
		dir = migrate.MemDir{}
		err = dir.WriteFile("1.sql", []byte("create table foo (id int)"))
	)
	require.NoError(t, err)
	ad, err := migrate.ArchiveDir(&dir)
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer atlas", r.Header.Get("Authorization"))
		require.Equal(t, "atlas-cli", r.Header.Get("User-Agent"))
		fmt.Fprintf(w, `{"data":{"dir":{"content":%q}}}`, base64.StdEncoding.EncodeToString(ad))
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	gd, err := client.Dir(context.Background(), DirInput{
		Name: "foo",
	})
	gcheck, err := gd.Checksum()
	require.NoError(t, err)
	dcheck, err := dir.Checksum()
	require.NoError(t, err)
	require.Equal(t, dcheck.Sum(), gcheck.Sum())
}
