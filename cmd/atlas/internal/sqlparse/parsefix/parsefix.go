// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package parsefix exposes shared functions to fix changes.
package parsefix

import (
	"ariga.io/atlas/sql/schema"
)

// RenameColumn patches DROP/ADD column commands to RENAME.
func RenameColumn(modify *schema.ModifyTable, from, to string) {
	changes := schema.Changes(modify.Changes)
	i := changes.IndexDropColumn(from)
	j := changes.IndexAddColumn(to)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameColumn{
			From: changes[i].(*schema.DropColumn).C,
			To:   changes[j].(*schema.AddColumn).C,
		}
		changes.RemoveIndex(min(i, j))
		modify.Changes = changes
	}
}

// RenameTable patches DROP/ADD table commands to RENAME.
func RenameTable(changes schema.Changes, from, to string) schema.Changes {
	i := changes.IndexDropTable(from)
	j := changes.IndexAddTable(to)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameTable{
			From: changes[i].(*schema.DropTable).T,
			To:   changes[j].(*schema.AddTable).T,
		}
		changes.RemoveIndex(min(i, j))
	}
	return changes
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
