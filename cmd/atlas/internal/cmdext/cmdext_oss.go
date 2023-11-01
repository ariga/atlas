// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package cmdext

import (
	"context"
	"fmt"
)

// StateReaderAtlas returns a migrate.StateReader from an Atlas Cloud schema.
func StateReaderAtlas(context.Context, *StateReaderConfig) (*StateReadCloser, error) {
	return nil, fmt.Errorf("atlas remote state is not supported by this release. See: https://atlasgo.io/getting-started")
}
