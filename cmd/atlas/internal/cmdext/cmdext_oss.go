// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package cmdext

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// RemoteSchema is a data source that for reading remote schemas.
func RemoteSchema(*hcl.EvalContext, *hclsyntax.Block) (cty.Value, error) {
	return cty.Zero, fmt.Errorf("data.remote_schema is not supported by this release. See: https://atlasgo.io/getting-started")
}

// StateReaderAtlas returns a migrate.StateReader from an Atlas Cloud schema.
func StateReaderAtlas(context.Context, *StateReaderConfig) (*StateReadCloser, error) {
	return nil, fmt.Errorf("atlas remote state is not supported by this release. See: https://atlasgo.io/getting-started")
}
