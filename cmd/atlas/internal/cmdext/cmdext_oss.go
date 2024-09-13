// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package cmdext

import (
	"context"
	"fmt"

	"github.com/s-sokolko/atlas/schemahcl"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var specOptions []schemahcl.Option

// RemoteSchema is a data source that for reading remote schemas.
func RemoteSchema(context.Context, *hcl.EvalContext, *hclsyntax.Block) (cty.Value, error) {
	return cty.Zero, fmt.Errorf("data.remote_schema is not supported by this release. See: https://atlasgo.io/getting-started")
}

// RemoteDir is a data source that reads a remote migration directory.
func RemoteDir(context.Context, *hcl.EvalContext, *hclsyntax.Block) (cty.Value, error) {
	return cty.Zero, fmt.Errorf("data.remote_dir is not supported by this release. See: https://atlasgo.io/getting-started")
}

// StateReaderAtlas returns a migrate.StateReader from an Atlas Cloud schema.
func StateReaderAtlas(context.Context, *StateReaderConfig) (*StateReadCloser, error) {
	return nil, fmt.Errorf("atlas remote state is not supported by this release. See: https://atlasgo.io/getting-started")
}

// EntLoader is a StateLoader for loading ent.Schema's as StateReader's.
type EntLoader struct{}

// LoadState returns a migrate.StateReader that reads the schema from an ent.Schema.
func (l EntLoader) LoadState(context.Context, *StateReaderConfig) (*StateReadCloser, error) {
	return nil, fmt.Errorf("ent:// scheme is no longer supported by this release. See: https://atlasgo.io/getting-started")
}

// MigrateDiff returns the diff between ent.Schema and a directory.
func (l EntLoader) MigrateDiff(context.Context, *MigrateDiffOptions) error {
	return fmt.Errorf("ent:// scheme is no longer supported by this release. See: https://atlasgo.io/getting-started")
}
