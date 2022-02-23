// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package action

import (
	"github.com/spf13/cobra"
)

var (
	// ServeFlags are the flags used in Serve command.
	ServeFlags struct {
		Addr    string
		Storage string
	}

	// ServeCMD represents the serve command.
	ServeCMD = &cobra.Command{
		Use:   "serve",
		Short: "Runs Atlas web UI in standalone mode with persistent storage.",
		Example: `
atlas serve --addr "localhost:8081" --storage "mysql://root:pass@tcp(localhost:3306)/"
"`,
	}
)
