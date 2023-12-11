// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package parseutil exposes shared functions used by the different parsers.
package parseutil

import (
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/exp/slices"
)

// Rename describes rename of a resource.
type Rename struct {
	From, To string
}

// RenameColumn patches DROP/ADD column commands to RENAME.
func RenameColumn(modify *schema.ModifyTable, r *Rename) {
	changes := schema.Changes(modify.Changes)
	switch i, j := changes.IndexDropColumn(r.From), changes.IndexAddColumn(r.To); {
	case j == -1:
	// Rename column.
	case i != -1:
		changes[max(i, j)] = &schema.RenameColumn{
			From: changes[i].(*schema.DropColumn).C,
			To:   changes[j].(*schema.AddColumn).C,
		}
		changes.RemoveIndex(min(i, j))
	// Rename column and add the previous name back.
	default:
		if i = changes.IndexModifyColumn(r.From); i == -1 {
			// ADD COLUMN must come after the RENAME.
			return
		}
		modify := changes[i].(*schema.ModifyColumn)
		changes[min(i, j)] = &schema.RenameColumn{
			From: modify.From,
			To:   changes[j].(*schema.AddColumn).C,
		}
		changes[max(i, j)] = &schema.AddColumn{
			C: modify.To,
		}
	}
	modify.Changes = changes
}

// RenameIndex patches DROP/ADD index commands to RENAME.
func RenameIndex(modify *schema.ModifyTable, r *Rename) {
	changes := schema.Changes(modify.Changes)
	i := changes.IndexDropIndex(r.From)
	j := changes.IndexAddIndex(r.To)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameIndex{
			From: changes[i].(*schema.DropIndex).I,
			To:   changes[j].(*schema.AddIndex).I,
		}
		changes.RemoveIndex(min(i, j))
		modify.Changes = changes
	}
}

// RenameTable patches DROP/ADD table commands to RENAME.
func RenameTable(changes schema.Changes, r *Rename) schema.Changes {
	i := changes.LastIndexDropTable(r.From)
	j := changes.LastIndexAddTable(r.To)
	if i != -1 && j != -1 {
		changes[max(i, j)] = &schema.RenameTable{
			From: changes[i].(*schema.DropTable).T,
			To:   changes[j].(*schema.AddTable).T,
		}
		changes.RemoveIndex(min(i, j))
	}
	return changes
}

// MatchStmtBefore reports if the file contains any statement that matches the predicate before the given position.
func MatchStmtBefore(stmts []*migrate.Stmt, pos int, p func(*migrate.Stmt) (bool, error)) (bool, error) {
	i := slices.IndexFunc(stmts, func(s *migrate.Stmt) bool {
		return s.Pos >= pos
	})
	if i != -1 {
		stmts = stmts[:i]
	}
	for _, s := range stmts {
		m, err := p(s)
		if err != nil {
			return false, err
		}
		if m {
			return true, nil
		}
	}
	return false, nil
}

// MatchStmtAfter reports if the file contains any statement that matches the predicate after the given position.
func MatchStmtAfter(stmts []*migrate.Stmt, pos int, p func(*migrate.Stmt) (bool, error)) (bool, error) {
	i := slices.IndexFunc(stmts, func(s *migrate.Stmt) bool {
		return s.Pos > pos
	})
	if i == -1 {
		return false, nil
	}
	stmts = stmts[i:]
	for _, s := range stmts {
		m, err := p(s)
		if err != nil {
			return false, err
		}
		if m {
			return true, nil
		}
	}
	return false, nil
}
