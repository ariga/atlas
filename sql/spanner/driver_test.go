// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package spanner

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDriver_LockAcquired(t *testing.T) {
	drv := &Driver{}

	// Acquiring a lock does work.
	unlock, err := drv.Lock(context.Background(), "lock", time.Second)
	require.NoError(t, err)
	require.NotNil(t, unlock)

	// Acquiring a lock on the same value will fail.
	_, err = drv.Lock(context.Background(), "lock", time.Second)
	require.Error(t, err)

	// After unlock it will succeed again.
	require.NoError(t, unlock())
	_, err = drv.Lock(context.Background(), "lock", time.Second)
	require.NoError(t, err)
	require.NotNil(t, unlock)

	// Acquiring a lock on a value that has been expired works.
	dir, err := os.UserCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "lock.lock"),
		[]byte(strconv.FormatInt(time.Now().Add(-time.Second).UnixNano(), 10)),
		0666,
	))
	_, err = drv.Lock(context.Background(), "lock", time.Second)

	// Acquiring a lock on another value works as well.
	_, err = drv.Lock(context.Background(), "another", time.Second)
}
