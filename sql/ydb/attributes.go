// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package ydb

import "ariga.io/atlas/sql/schema"

// [IndexAttributes] represents YDB-specific index attributes.
type IndexAttributes struct {
	schema.Attr
	Async        bool
	CoverColumns []*schema.Column
}
