// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package postgrescheck

import (
	"github.com/s-sokolko/atlas/sql/postgres"
	"github.com/s-sokolko/atlas/sql/sqlcheck"
)

func init() {
	sqlcheck.Register(postgres.DriverName, analyzers)
}
