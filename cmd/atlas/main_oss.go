// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package main

import (
	"context"
)

func extendContext(ctx context.Context) context.Context { return ctx }

func vercheckEndpoint(context.Context) string {
	return vercheckURL
}

// initialize is a no-op for the OSS version.
func initialize(context.Context) func() {
	return func() {}
}
