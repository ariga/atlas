// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package pgparse

import (
	"ariga.io/atlas/sql/schema"
	pgquery "github.com/pganalyze/pg_query_go/v5"
)

func FixAlterTable(_ string, _ *pgquery.AlterTableStmt, changes schema.Changes) (schema.Changes, error) {
	return changes, nil // Unimplemented.
}
