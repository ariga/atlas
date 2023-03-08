// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqliteparse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/cmd/atlas/internal/sqlparse/parseutil"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"golang.org/x/exp/slices"
)

type (
	// Stmt provides extended functionality
	// to ANTLR parsed statements.
	Stmt struct {
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
func (l *listenError) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
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
func (s *Stmt) RenameColumn() (*parseutil.Rename, bool) {
	if !s.IsAlterTable() {
		return nil, false
	}
	alter := s.stmt.GetChild(0).(*Alter_table_stmtContext)
	if alter.old_column_name == nil || alter.new_column_name == nil {
		return nil, false
	}
	return &parseutil.Rename{
		From: unquote(alter.old_column_name.GetText()),
		To:   unquote(alter.new_column_name.GetText()),
	}, true
}

// RenameTable returns the renamed table information from the statement, if exists.
func (s *Stmt) RenameTable() (*parseutil.Rename, bool) {
	if !s.IsAlterTable() {
		return nil, false
	}
	alter := s.stmt.GetChild(0).(*Alter_table_stmtContext)
	if alter.new_table_name == nil {
		return nil, false
	}
	return &parseutil.Rename{
		From: unquote(alter.Table_name(0).GetText()),
		To:   unquote(alter.new_table_name.GetText()),
	}, true
}

// TableUpdate reports if the statement is an UPDATE command for the given table.
func (s *Stmt) TableUpdate(t *schema.Table) (*Update_stmtContext, bool) {
	if s.stmt.GetChildCount() != 1 {
		return nil, false
	}
	u, ok := s.stmt.GetChild(0).(*Update_stmtContext)
	if !ok {
		return nil, false
	}
	name, ok := u.Qualified_table_name().(*Qualified_table_nameContext)
	if !ok || unquote(name.Table_name().GetText()) != t.Name {
		return nil, false
	}
	return u, true
}

// CreateView reports if the statement is a CREATE VIEW command with the given name.
func (s *Stmt) CreateView(name string) (*Create_view_stmtContext, bool) {
	if s.stmt.GetChildCount() != 1 {
		return nil, false
	}
	v, ok := s.stmt.GetChild(0).(*Create_view_stmtContext)
	if !ok || unquote(v.View_name().GetText()) != name {
		return nil, false
	}
	return v, true
}

// FileParser implements the sqlparse.Parser
type FileParser struct{}

// ColumnFilledBefore checks if the column was filled before the given position.
func (p *FileParser) ColumnFilledBefore(f migrate.File, t *schema.Table, c *schema.Column, pos int) (bool, error) {
	return parseutil.MatchStmtBefore(f, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := ParseStmt(s.Text)
		if err != nil {
			return false, err
		}
		u, ok := stmt.TableUpdate(t)
		if !ok {
			return false, nil
		}
		// Accept UPDATE that fills all rows or those with NULL values as we cannot
		// determine if NULL values were filled in case there is a custom filtering.
		affectC := func() bool {
			x := u.GetWhere()
			if x == nil {
				return true
			}
			if x.GetChildCount() != 3 {
				return false
			}
			x1, ok := x.GetChild(0).(*ExprContext)
			if !ok || unquote(x1.GetText()) != c.Name {
				return false
			}
			x2, ok := x.GetChild(1).(*antlr.TerminalNodeImpl)
			if !ok || x2.GetSymbol().GetTokenType() != ParserIS_ {
				return false
			}
			return isnull(x.GetChild(2))
		}()
		list, ok := u.Assignment_list().(*Assignment_listContext)
		if !ok {
			return false, nil
		}
		idx := slices.IndexFunc(list.AllAssignment(), func(a IAssignmentContext) bool {
			as, ok := a.(*AssignmentContext)
			return ok && unquote(as.Column_name().GetText()) == c.Name && !isnull(as.Expr())
		})
		// Ensure the column was filled.
		return affectC && idx != -1, nil
	})
}

// CreateViewAfter checks if a view was created after the position with the given name to a table.
func (p *FileParser) CreateViewAfter(f migrate.File, old, new string, pos int) (bool, error) {
	return parseutil.MatchStmtAfter(f, pos, func(s *migrate.Stmt) (bool, error) {
		stmt, err := ParseStmt(s.Text)
		if err != nil {
			return false, err
		}
		v, ok := stmt.CreateView(old)
		if !ok {
			return false, nil
		}
		sc, ok := v.Select_stmt().(*Select_stmtContext)
		if !ok {
			return false, nil
		}
		idx := slices.IndexFunc(sc.Select_core(0).GetChildren(), func(t antlr.Tree) bool {
			ts, ok := t.(*Table_or_subqueryContext)
			return ok && unquote(ts.GetText()) == new
		})
		return idx != -1, nil
	})
}

// FixChange fixes the changes according to the given statement.
func (p *FileParser) FixChange(_ migrate.Driver, s string, changes schema.Changes) (schema.Changes, error) {
	stmt, err := ParseStmt(s)
	if err != nil {
		return nil, err
	}
	if !stmt.IsAlterTable() {
		return changes, nil
	}
	if r, ok := stmt.RenameColumn(); ok {
		if len(changes) != 1 {
			return nil, fmt.Errorf("unexected number fo changes: %d", len(changes))
		}
		modify, ok := changes[0].(*schema.ModifyTable)
		if !ok {
			return nil, fmt.Errorf("expected modify-table change for alter-table statement, but got: %T", changes[0])
		}
		// ALTER COLUMN cannot be combined with additional commands.
		if len(changes) > 2 {
			return nil, fmt.Errorf("unexpected number of changes found: %d", len(changes))
		}
		parseutil.RenameColumn(modify, r)
	}
	if r, ok := stmt.RenameTable(); ok {
		changes = parseutil.RenameTable(changes, r)
	}
	return changes, nil
}

func isnull(t antlr.Tree) bool {
	x, ok := t.(*ExprContext)
	if !ok || x.GetChildCount() != 1 {
		return false
	}
	l, ok := x.GetChild(0).(*Literal_valueContext)
	return ok && l.GetChildCount() == 1 && len(l.GetTokens(ParserNULL_)) > 0
}

func unquote(s string) string {
	switch {
	case len(s) < 2:
	case s[0] == '`' && s[len(s)-1] == '`', s[0] == '"' && s[len(s)-1] == '"':
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}
