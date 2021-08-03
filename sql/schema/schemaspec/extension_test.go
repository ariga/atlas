// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schemaspec_test

import (
	"testing"

	"ariga.io/atlas/sql/internal/schemautil"
	"ariga.io/atlas/sql/schema/schemaspec"

	"github.com/stretchr/testify/require"
)

type OwnerBlock struct {
	FirstName string `spec:"first_name"`
	Born      int    `spec:"born"`
	Active    bool   `spec:"active"`
}

func (*OwnerBlock) Type() string {
	return "owner"
}

func (*OwnerBlock) Name() string {
	return ""
}

func TestExtension(t *testing.T) {
	original := &schemaspec.Resource{
		Type: "owner",
		Attrs: []*schemaspec.Attr{
			schemautil.StrLitAttr("first_name", "tzuri"),
			schemautil.LitAttr("born", "2019"),
			schemautil.LitAttr("active", "true"),
		},
	}
	owner := OwnerBlock{}
	err := original.As(&owner)
	require.NoError(t, err)
	require.EqualValues(t, "tzuri", owner.FirstName)
	require.EqualValues(t, 2019, owner.Born)
	require.EqualValues(t, true, owner.Active)

	scan := &schemaspec.Resource{}
	err = scan.Scan(&owner)
	require.NoError(t, err)
	require.EqualValues(t, original, scan)
}
