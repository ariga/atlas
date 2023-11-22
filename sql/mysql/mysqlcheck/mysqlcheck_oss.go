// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package mysqlcheck

import (
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/sqlcheck"
)

func init() {
	sqlcheck.Register(mysql.DriverName, analyzers)
}
