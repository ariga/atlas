// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/cmdlog"
	"ariga.io/atlas/cmd/atlas/internal/cmdstate"
)

func extendContext(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func vercheckEndpoint(context.Context) string {
	return vercheckURL
}

// initialize is a no-op for the OSS version.
func initialize(ctx context.Context) (context.Context, func(error)) {
	return ctx, func(err error) {
		if err == nil {
			return
		}
		const errorsFileName = "community_error.json"
		type prompt struct {
			LastSuggested time.Time `json:"last_suggested"`
		}
		state := &cmdstate.File[prompt]{Name: errorsFileName}
		prev, err := state.Read()
		if err != nil || time.Since(prev.LastSuggested) < 24*time.Hour {
			return
		}
		release := "curl -sSf https://atlasgo.sh | sh"
		if runtime.GOOS == "windows" {
			release = "https://release.ariga.io/atlas/atlas-windows-amd64-latest.exe"
		}
		if err := cmdlog.WarnOnce(os.Stderr, cmdlog.ColorCyan(fmt.Sprintf(`You're running the community build of Atlas, which may differ from the official version.
If this error persists, try installing the official version as a troubleshooting step:

  %s

More installation options: https://atlasgo.io/docs#installation
`, release))); err == nil {
			prev.LastSuggested = time.Now()
			_ = state.Write(prev)
		}
	}
}
