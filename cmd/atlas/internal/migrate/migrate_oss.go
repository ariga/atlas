// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package migrate

import (
	"context"
	"fmt"
	"net/url"

	"ariga.io/atlas/sql/migrate"
)

func openAtlasDir(ctx context.Context, u *url.URL) (migrate.Dir, error) {
	return nil, fmt.Errorf("atlas remote directory is not supported by this release. See: https://atlasgo.io/getting-started")
}
