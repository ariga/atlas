// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

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
		drv = &mockDriver{
			realm: schema.NewRealm(schema.New("test").SetCharset("utf8mb4")),
		}
		dev = &DevDriver{
			Driver:     drv,
			MaxNameLen: 64,
		}
	)
	normal, err := dev.NormalizeRealm(context.Background(), schema.NewRealm(schema.New("test")))
	require.NoError(t, err)
	require.Equal(t, normal, drv.realm)

	require.Len(t, drv.schemas, 1)
	require.True(t, strings.HasPrefix(drv.schemas[0], "atlas_dev_test_"))

	require.Len(t, drv.changes, 2, "expect 2 calls (create and drop)")
	require.Len(t, drv.changes[0], 1)
	require.Equal(t, &schema.AddSchema{S: schema.New(drv.schemas[0])}, drv.changes[0][0])
	require.Len(t, drv.changes[1], 1)
	require.Equal(t, &schema.DropSchema{S: schema.New(drv.schemas[0]), Extra: []schema.Clause{&schema.IfExists{}}}, drv.changes[1][0])
}

type mockDriver struct {
	migrate.Driver
	// Inspect.
	schemas []string
	realm   *schema.Realm
	// Apply.
	changes [][]schema.Change
}

func (m *mockDriver) InspectRealm(_ context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	m.schemas = append(m.schemas, opts.Schemas...)
	return m.realm, nil
}

func (m *mockDriver) ApplyChanges(_ context.Context, changes []schema.Change, _ ...migrate.PlanOption) error {
	m.changes = append(m.changes, changes)
	return nil
}
