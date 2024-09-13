// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package mysqlcheck

import (
	"github.com/s-sokolko/atlas/sql/mysql"
	"github.com/s-sokolko/atlas/sql/sqlcheck"
)

func init() {
	sqlcheck.Register(mysql.DriverName, analyzers)
}
