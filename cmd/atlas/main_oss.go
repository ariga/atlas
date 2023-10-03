// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package main

import (
	"context"
	"os"
	"os/signal"
)

func newContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return ctx
}

func vercheckEndpoint(_ context.Context) string {
	return vercheckURL
}
