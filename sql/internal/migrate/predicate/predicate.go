// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package predicate

import (
	"entgo.io/ent/dialect/sql"
)

// Revision is the predicate function for revision builders.
type Revision func(*sql.Selector)
