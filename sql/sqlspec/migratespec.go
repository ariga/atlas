// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlspec

import (
	"ariga.io/atlas/schemahcl"
)

type (
	// Change is the interface implemented by change specifications. Change instances are supposed
	// to be mappable to schema.Change instances.
	Change interface {
		change()
	}

	// ModifyTable is a specification for a schema.ModifyTable.
	ModifyTable struct {
		Change
		Table   string   `spec:"table"`
		Changes []Change `spec:""`
	}

	// AddColumn is a specification for a schema.AddColumn.
	AddColumn struct {
		Change
		Column *Column `spec:"column"`
	}
)

func init() {
	schemahcl.Register("modify_table", &ModifyTable{})
	schemahcl.Register("add_column", &AddColumn{})
}
