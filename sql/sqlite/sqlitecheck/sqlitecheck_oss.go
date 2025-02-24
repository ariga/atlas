// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqlitecheck

import (
	"ariga.io/atlas/sql/sqlcheck"
	"ariga.io/atlas/sql/sqlite"
)

func init() {
	sqlcheck.Register(sqlite.DriverName, analyzers)
}
