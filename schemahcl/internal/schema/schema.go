// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package schema is a temporary hack to avoid having circular
// dependency with ariga.io/atlas/sql/schema.
package schema

import "ariga.io/atlas/sql/schema"

// A list of aliased types.
type (
	Type            = schema.Type
	UnsupportedType = schema.UnsupportedType
)
