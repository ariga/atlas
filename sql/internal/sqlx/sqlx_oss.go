// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqlx

import (
	"ariga.io/atlas/sql/schema"
)

// PlanViewChanges (should) plan view changes in the current order they should be applied.
// It is unimplemented for community version as views are not supported there.
func PlanViewChanges(changes []schema.Change) ([]schema.Change, error) {
	return changes, nil // unimplemented.
}
