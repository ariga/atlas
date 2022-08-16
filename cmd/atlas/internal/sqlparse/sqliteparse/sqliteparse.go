// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqliteparse

import (
	"errors"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type (
	// Stmt provides extended functionality
	// to ANTLR parsed statements.
	Stmt struct {
		p     *Parser
		stmt  antlr.ParseTree
		input string
		err   error
	}

	// listenError catches parse errors.
	listenError struct {
		antlr.DefaultErrorListener
		err  error
		text string
	}
)

// SyntaxError implements ErrorListener.SyntaxError.
func (l *listenError) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	if idx := strings.Index(msg, " expecting "); idx != -1 {
		msg = msg[:idx]
	}
	l.err = fmt.Errorf("line %d:%d: %s", line, column+1, msg)
}

// ParseStmt parses a statement.
func ParseStmt(text string) (stmt *Stmt, err error) {
	l := &listenError{text: text}
	defer func() {
		if l.err != nil {
			err = l.err
			stmt = nil
		} else if perr := recover(); perr != nil {
			m := fmt.Sprint(perr)
			if v, ok := err.(antlr.RecognitionException); ok {
				m = v.GetMessage()
			}
			err = errors.New(m)
			stmt = nil
		}
	}()
	lex := NewLexer(antlr.NewInputStream(text))
	lex.RemoveErrorListeners()
	lex.AddErrorListener(l)
	p := NewParser(
		antlr.NewCommonTokenStream(lex, 0),
	)
	p.RemoveErrorListeners()
	p.AddErrorListener(l)
	p.BuildParseTrees = true
	stmt = &Stmt{
		p:    p,
		stmt: p.Sql_stmt(),
	}
	return
}

// IsAlterTable reports if the statement is type ALTER TABLE.
func (s *Stmt) IsAlterTable() bool {
	if s.stmt.GetChildCount() != 1 {
		return false
	}
	_, ok := s.stmt.GetChild(0).(*Alter_table_stmtContext)
	return ok
}

// RenameColumn returns the renamed column information from the statement, if exists.
func (s *Stmt) RenameColumn() (*struct{ From, To string }, bool) {
	if !s.IsAlterTable() {
		return nil, false
	}
	alter := s.stmt.GetChild(0).(*Alter_table_stmtContext)
	if alter.old_column_name == nil || alter.new_column_name == nil {
		return nil, false
	}
	return &struct{ From, To string }{
		From: alter.old_column_name.GetText(),
		To:   alter.new_column_name.GetText(),
	}, true
}

// FixChange fixes the changes according to the given statement.
func FixChange(s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := ParseStmt(s)
	if err != nil {
		return nil, err
	}
	if !stmt.IsAlterTable() {
		return changes, nil
	}
	if len(changes) != 1 {
		return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
	}
	modify, ok := changes[0].(*schema.ModifyTable)
	if !ok {
		return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
	}
	if rename, ok := stmt.RenameColumn(); ok {
		changes := schema.Changes(modify.Changes)
		// ALTER COLUMN cannot be combined with additional commands.
		if len(changes) > 2 {
			return nil, fmt.Errorf("unexpected number of changes found: %d", len(changes))
		}
		i1 := changes.IndexDropColumn(rename.From)
		i2 := changes.IndexAddColumn(rename.To)
		if i1 != -1 && i2 != -1 {
			modify.Changes = schema.Changes{
				&schema.RenameColumn{
					From: changes[i1].(*schema.DropColumn).C,
					To:   changes[i2].(*schema.AddColumn).C,
				},
			}
		}
	}
	return changes, nil
}
