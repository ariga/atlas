// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package vercheck

import (
	"context"
	"net/http"

	"github.com/s-sokolko/atlas/cmd/atlas/internal/cloudapi"
)

func addHeaders(_ context.Context, req *http.Request) {
	req.Header.Set("User-Agent", cloudapi.UserAgent())
}
