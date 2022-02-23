// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/stretchr/testify/require"
)

func TestDriver_NormalizeRealm(t *testing.T) {
	var (
		apply   = &mockApply{}
		inspect = &mockInspect{
			realm: schema.NewRealm(schema.New("test").SetCharset("utf8mb4")),
		}
		drv = &Driver{
			Inspector:   inspect,
			PlanApplier: apply,
		}
	)
	normal, err := drv.NormalizeRealm(context.Background(), schema.NewRealm(schema.New("test")))
	require.NoError(t, err)
	require.Equal(t, normal, inspect.realm)

	require.Len(t, inspect.schemas, 1)
	require.True(t, strings.HasPrefix(inspect.schemas[0], "atlas_twin_test_"))

	require.Len(t, apply.changes, 2, "expect 2 calls (create and drop)")
	require.Len(t, apply.changes[0], 1)
	require.Equal(t, &schema.AddSchema{S: schema.New(inspect.schemas[0])}, apply.changes[0][0])
	require.Len(t, apply.changes[1], 1)
	require.Equal(t, &schema.DropSchema{S: schema.New(inspect.schemas[0]), Extra: []schema.Clause{&schema.IfExists{}}}, apply.changes[1][0])
}

type mockInspect struct {
	schema.Inspector
	schemas []string
	realm   *schema.Realm
}

func (m *mockInspect) InspectRealm(_ context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	m.schemas = append(m.schemas, opts.Schemas...)
	return m.realm, nil
}

type mockApply struct {
	migrate.PlanApplier
	changes [][]schema.Change
}

func (m *mockApply) ApplyChanges(_ context.Context, changes []schema.Change) error {
	m.changes = append(m.changes, changes)
	return nil
}
