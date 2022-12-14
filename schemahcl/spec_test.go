// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemahcl

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRef_Path(t *testing.T) {
	tests := []struct {
		ref      string
		wantPath []PathIndex
		wantErr  bool
	}{
		{
			ref:     "invalid",
			wantErr: true, // invalid path
		},
		{
			ref:     "$schema",
			wantErr: true, // missing identifier
		},
		{
			ref:     "$schema.",
			wantErr: true, // empty identifier
		},
		{
			ref:     `$schema[""].foo`,
			wantErr: true, // empty identifier
		},
		{
			ref:     "$.main",
			wantErr: true, // empty type
		},
		{
			ref: "$schema.main",
			wantPath: []PathIndex{
				{T: "schema", V: []string{"main"}},
			},
		},
		{
			ref: "$schema.main.$table.users",
			wantPath: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"users"}},
			},
		},
		{
			ref: "$schema.main.$table.global.users",
			wantPath: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"global", "users"}},
			},
		},
		{
			ref: "$schema.main.$table.global.users.$column.name",
			wantPath: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"global", "users"}},
				{T: "column", V: []string{"name"}},
			},
		},
		{
			ref: `$account["ariga.cloud"]["a8m.dev"]`,
			wantPath: []PathIndex{
				{T: "account", V: []string{"ariga.cloud", "a8m.dev"}},
			},
		},
		{
			ref: `$foo.bar["baz"].qux.$bar.baz["qux"].$baz.qux`,
			wantPath: []PathIndex{
				{T: "foo", V: []string{"bar", "baz", "qux"}},
				{T: "bar", V: []string{"baz", "qux"}},
				{T: "baz", V: []string{"qux"}},
			},
		},
		{
			ref: `$foo["bar"]["baz"]["qux"].$bar["baz"]["qux"].$baz["qux"]`,
			wantPath: []PathIndex{
				{T: "foo", V: []string{"bar", "baz", "qux"}},
				{T: "bar", V: []string{"baz", "qux"}},
				{T: "baz", V: []string{"qux"}},
			},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := &Ref{V: tt.ref}
			path, err := r.Path()
			require.Equal(t, tt.wantErr, err != nil, err)
			require.Equal(t, tt.wantPath, path)
		})
	}
}

func TestBuildRef(t *testing.T) {
	tests := []struct {
		path    []PathIndex
		wantRef string
	}{
		{
			path:    []PathIndex{{T: "schema"}, {T: "table"}},
			wantRef: `$schema.$table`,
		},
		{
			path:    []PathIndex{{T: "schema", V: []string{"main"}}},
			wantRef: `$schema.main`,
		},
		{
			path: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"users"}},
			},
			wantRef: `$schema.main.$table.users`,
		},
		{
			path: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"main", "users"}},
			},
			wantRef: `$schema.main.$table.main.users`,
		},
		{
			path: []PathIndex{
				{T: "schema", V: []string{"main"}},
				{T: "table", V: []string{"main", "foo.bar"}},
			},
			wantRef: `$schema.main.$table.main["foo.bar"]`,
		},
		{
			path: []PathIndex{
				{T: "schema", V: []string{"schema-name"}},
				{T: "table", V: []string{"other.schema", "foo.bar"}},
				{T: "column", V: []string{"column-name"}},
			},
			wantRef: `$schema.schema-name.$table["other.schema"]["foo.bar"].$column.column-name`,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ref := BuildRef(tt.path)
			require.Equal(t, tt.wantRef, ref.V)
		})
	}
}
