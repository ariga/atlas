// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Stmt represents a scanned statement text along with its
// position in the file and associated comments group.
type Stmt struct {
	Pos      int      // statement position
	Text     string   // statement text
	Comments []string // associated comments
}

// Directive returns all directive comments with the given name.
// See: pkg.go.dev/cmd/compile#hdr-Compiler_Directives.
func (s *Stmt) Directive(name string) (ds []string) {
	for _, c := range s.Comments {
		switch {
		case strings.HasPrefix(c, "/*") && !strings.Contains(c, "\n"):
			if d, ok := directive(strings.TrimSuffix(c, "*/"), name, "/*"); ok {
				ds = append(ds, d)
			}
		default:
			for _, p := range []string{"#", "--", "-- "} {
				if d, ok := directive(c, name, p); ok {
					ds = append(ds, d)
				}
			}
		}
	}
	return
}

// Stmts provides a generic implementation for extracting SQL statements from the given file contents.
func Stmts(input string) ([]*Stmt, error) {
	return (&Scanner{
		ScannerOptions: ScannerOptions{
			// Default options for backward compatibility.
			MatchBegin:       false,
			MatchBeginAtomic: true,
			MatchDollarQuote: true,
		},
	}).Scan(input)
}

// FileStmtDecls scans atlas-format file statements using
// the Driver implementation, if implemented.
func FileStmtDecls(drv Driver, f File) ([]*Stmt, error) {
	s, ok1 := drv.(StmtScanner)
	_, ok2 := f.(*LocalFile)
	if !ok1 || !ok2 {
		return f.StmtDecls()
	}
	return s.ScanStmts(string(f.Bytes()))
}

// FileStmts is like FileStmtDecls but returns only the
// statement text without the extra info.
func FileStmts(drv Driver, f File) ([]string, error) {
	s, err := FileStmtDecls(drv, f)
	if err != nil {
		return nil, err
	}
	stmts := make([]string, len(s))
	for i := range s {
		stmts[i] = s[i].Text
	}
	return stmts, nil
}

type (
	// StmtScanner interface for scanning SQL statements from migration
	// and schema files and can be optionally implemented by drivers.
	StmtScanner interface {
		ScanStmts(input string) ([]*Stmt, error)
	}

	// Scanner scanning SQL statements from migration and schema files.
	Scanner struct {
		ScannerOptions
		// scanner state.
		src, input string   // src and current input text
		pos        int      // current phase position
		total      int      // total bytes scanned so far
		width      int      // size of latest rune
		delim      string   // configured delimiter
		comments   []string // collected comments
		// internal option to indicate if the
		// END word is parsed as a terminator.
		endterm *regexp.Regexp
	}

	// ScannerOptions controls the behavior of the scanner.
	ScannerOptions struct {
		// MatchBegin enables matching for BEGIN ... END statements block.
		MatchBegin bool
		// MatchBeginAtomic enables matching for BEGIN ATOMIC ... END statements block.
		MatchBeginAtomic bool
		// MatchBeginCatch enables matching for BEGIN TRY/CATCH ... END TRY/CATCH statements block.
		MatchBeginTryCatch bool
		// MatchDollarQuote enables the PostgreSQL dollar-quoted string syntax.
		MatchDollarQuote bool
		// BackslashEscapes enables backslash-escaped strings. By default, only MySQL/MariaDB uses backslash as
		// an escape character.  https://dev.mysql.com/doc/refman/8.4/en/sql-mode.html#sqlmode_no_backslash_escapes
		BackslashEscapes bool
		// EscapedStringExt enables the supported for PG extension for escaped strings and adopted by its flavors.
		// See: https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS-ESCAPE.
		EscapedStringExt bool
		// HashComments enables MySQL/MariaDB hash-like (#) comments.
		HashComments bool
		// Enable the "GO" command as a delimiter.
		GoCommand bool
		// BeginEndTerminator is a T-SQL specific option that allows
		// the scanner to terminate BEGIN/END blocks with a semicolon.
		BeginEndTerminator bool
	}
)

// Scan scans the statement in the given input.
func (s *Scanner) Scan(input string) ([]*Stmt, error) {
	var stmts []*Stmt
	if err := s.init(input); err != nil {
		return nil, err
	}
	for {
		s, err := s.stmt()
		if err == io.EOF {
			return stmts, nil
		}
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
	}
}

// init initializes the scanner state.
func (s *Scanner) init(input string) error {
	s.comments = nil
	s.pos, s.total, s.width = 0, 0, 0
	s.src, s.input, s.delim = input, input, delimiter
	if d, ok := directive(input, directiveDelimiter, directivePrefixSQL); ok {
		if err := s.setDelim(d); err != nil {
			return err
		}
		parts := strings.SplitN(input, "\n", 2)
		if len(parts) == 1 {
			return s.error(s.pos, "no input found after delimiter %q", d)
		}
		s.input = parts[1]
	}
	return nil
}

const (
	eos          = -1
	delimiter    = ";"
	delimiterCmd = "delimiter"
)

var (
	// Dollar-quoted string as defined by the PostgreSQL scanner.
	reDollarQuote = regexp.MustCompile(`^\$([A-Za-zÈ-ÿ_][\wÈ-ÿ]*)*\$`)
	// The 'BEGIN ATOMIC' syntax as specified in the SQL 2003 standard.
	reBeginAtomic = regexp.MustCompile(`(?i)^\s*BEGIN\s+ATOMIC\s+`)
	reBeginTry    = regexp.MustCompile(`(?i)^\s*BEGIN\s+TRY\s+`)
	reBegin       = regexp.MustCompile(`(?i)^\s*BEGIN\s+`)
	reEnd         = regexp.MustCompile(`(?i)^\s*END\s*`)
	reEndCatch    = regexp.MustCompile(`(?i)^\s*END\s*CATCH\s*`)
	reGoCmd       = regexp.MustCompile(`(?i)^GO(?:\s+|$)`)
)

func (s *Scanner) stmt() (*Stmt, error) {
	var (
		depth, openingPos int
		text              string
	)
	s.skipSpaces()
Scan:
	for {
		switch r := s.next(); {
		case r == eos:
			switch {
			case depth > 0:
				return nil, s.error(openingPos, "unclosed '('")
			case s.pos > 0:
				text = s.input
				break Scan
			default:
				return nil, io.EOF
			}
		case r == '(':
			if depth == 0 {
				openingPos = s.pos
			}
			depth++
		case r == ')':
			if depth == 0 {
				return nil, s.error(s.pos, "unexpected ')'")
			}
			depth--
		case r == '\'', r == '"', r == '`':
			if err := s.skipQuote(r); err != nil {
				return nil, err
			}
		// Check if the start of the statement is the MySQL DELIMITER command.
		// See https://dev.mysql.com/doc/refman/8.0/en/mysql-commands.html.
		case s.pos == 1 && len(s.input) > len(delimiterCmd) && strings.EqualFold(s.input[:len(delimiterCmd)], delimiterCmd):
			s.addPos(len(delimiterCmd) - 1)
			if err := s.delimCmd(); err != nil {
				return nil, err
			}
			s.skipSpaces()
		// GO command takes over the delimiter '\nGO'
		// in cases it can't parse the statements correctly.
		case s.GoCommand && r == '\n' && reGoCmd.MatchString(s.input[s.pos:]):
			s.next() // skip '\n'
			fallthrough
		case s.GoCommand && (s.pos == 1 || s.pos > 1 && s.input[s.pos-2] == '\n') && reGoCmd.MatchString(s.input[s.pos-1:]):
			text = s.input[:s.pos-1]
			s.next() // skip 'O'
			if err := s.skipGoCount(); err != nil {
				return nil, err
			}
			s.skipSpaces()
			break Scan
		// Delimiters take precedence over comments.
		case depth == 0 && strings.HasPrefix(s.input[s.pos-s.width:], s.delim):
			s.addPos(len(s.delim) - s.width)
			text = s.input[:s.pos]
			break Scan
		case s.MatchDollarQuote && r == '$' && reDollarQuote.MatchString(s.input[s.pos-1:]):
			if err := s.skipDollarQuote(); err != nil {
				return nil, err
			}
		case r == '#' && s.HashComments:
			s.comment("#", "\n")
		case r == '-' && s.pick() == '-':
			s.next()
			s.comment("--", "\n")
		case r == '/' && s.pick() == '*':
			s.next()
			s.comment("/*", "*/")
		case s.endterm != nil && s.endterm.MatchString(s.input[:s.pos]):
			text = s.input[:s.pos]
			break Scan
		case s.delim == delimiter && s.MatchBeginAtomic && reBeginAtomic.MatchString(s.input[s.pos-1:]):
			if err := s.skipBeginAtomic(); err == nil {
				text = s.input[:s.pos]
				break Scan
			}
			// Not a "BEGIN ATOMIC" block.
		case s.delim == delimiter && s.MatchBeginTryCatch && reBeginTry.MatchString(s.input[s.pos-1:]):
			if err := s.skipBeginTryCatch(); err == nil {
				text = s.input[:s.pos]
				break Scan
			}
			// Not a "BEGIN TRY ... END CATCH" block.
		case s.delim == delimiter && s.MatchBegin &&
			// Either the current scanned statement starts with BEGIN, or we inside a statement and expects at least one ~space before).
			(s.pos == 1 && reBegin.MatchString(s.input[s.pos-1:]) || s.pos > 1 && reBegin.MatchString(s.input[s.pos-2:])):
			if err := s.skipBegin(); err == nil {
				text = s.input[:s.pos]
				break Scan
			}
			// Not a "BEGIN" block.
		}
	}
	return s.emit(text), nil
}

func (s *Scanner) next() rune {
	if s.pos >= len(s.input) {
		return eos
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.width = w
	s.addPos(w)
	return r
}

func (s *Scanner) pick() rune {
	p, w, t := s.pos, s.width, s.total
	r := s.next()
	s.pos, s.width, s.total = p, w, t
	return r
}

func (s *Scanner) addPos(p int) {
	s.pos += p
	s.total += p
}

func (s *Scanner) skipQuote(quote rune) error {
	var (
		pos     = s.pos
		escaped = s.BackslashEscapes || s.EscapedStringExt && s.pos > 0 && (s.input[s.pos-1] == 'E' || s.input[s.pos-1] == 'e')
	)
	for {
		switch r := s.next(); {
		case r == eos:
			return s.error(pos, "unclosed quote %q", quote)
		case r == '\\' && escaped:
			s.next()
		case r == quote:
			return nil
		}
	}
}

func (s *Scanner) skipDollarQuote() error {
	m := reDollarQuote.FindString(s.input[s.pos-1:])
	if m == "" {
		return s.error(s.pos, "unexpected dollar quote")
	}
	s.addPos(len(m) - 1)
	for {
		switch r := s.next(); {
		case r == eos:
			// Fail only if a delimiter was not set.
			if s.delim == "" {
				return s.error(s.pos, "unclosed dollar-quoted string")
			}
			return nil
		case r == '$' && strings.HasPrefix(s.input[s.pos-1:], m):
			s.addPos(len(m) - 1)
			return nil
		}
	}
}

func (s *Scanner) skipBeginAtomic() error {
	m := reBeginAtomic.FindString(s.input[s.pos-1:])
	if m == "" {
		return s.error(s.pos, "unexpected missing BEGIN ATOMIC block")
	}
	s.addPos(len(m) - 1)
	body := &Scanner{ScannerOptions: s.ScannerOptions}
	if err := body.init(s.input[s.pos:]); err != nil {
		return err
	}
	for {
		stmt, err := body.stmt()
		if err == io.EOF {
			return s.error(s.pos, "unexpected eof when scanning sql body")
		}
		if err != nil {
			return s.error(s.pos, "scan sql body: %v", err)
		}
		if reEnd.MatchString(stmt.Text) {
			break
		}
	}
	s.addPos(body.total)
	return nil
}

func (s *Scanner) skipBeginTryCatch() error {
	m := reBeginTry.FindString(s.input[s.pos-1:])
	if m == "" {
		return s.error(s.pos, "unexpected missing BEGIN TRY block")
	}
	s.addPos(len(m) - 1)
	body := &Scanner{ScannerOptions: s.ScannerOptions}
	if err := body.init(s.input[s.pos:]); err != nil {
		return err
	}
	for {
		stmt, err := body.stmt()
		if err == io.EOF {
			return s.error(s.pos, "unexpected eof when scanning sql body")
		}
		if err != nil {
			return s.error(s.pos, "scan sql body: %v", err)
		}
		if end := reEndCatch.FindString(stmt.Text); end != "" {
			// In case "END CATCH" is not followed by a semicolon (\n instead),
			// backup the extra consumed statement (it might be END;) and exit.
			if !strings.HasSuffix(strings.TrimSpace(end), ";") {
				s.addPos(-(len(stmt.Text) - len(end)))
			}
			break
		}
	}
	s.addPos(body.total)
	return nil
}

var (
	reEndTerm = regexp.MustCompile(`(?i)\s*END\s*$`)
)

func (s *Scanner) skipBegin() error {
	m := reBegin.FindString(s.input[s.pos-1:])
	if m == "" {
		return s.error(s.pos, "unexpected missing BEGIN block")
	}
	s.addPos(len(m) - 1)
	group := &Scanner{ScannerOptions: s.ScannerOptions}
	if s.BeginEndTerminator {
		group.endterm = reEndTerm
	}
	if err := group.init(s.input[s.pos:]); err != nil {
		return err
	}
Loop:
	for {
		switch stmt, err := group.stmt(); {
		case err == io.EOF:
			return s.error(s.pos, "unexpected eof when scanning compound statements")
		case err != nil:
			return s.error(s.pos, "scan compound statements: %v", err)
		case reEnd.MatchString(stmt.Text):
			if m := reEnd.FindString(stmt.Text); len(m) == len(stmt.Text) || strings.TrimPrefix(stmt.Text, m) == s.delim {
				break Loop
			}
		case s.BeginEndTerminator && reEndTerm.MatchString(stmt.Text):
			break Loop
		}
	}
	s.addPos(group.total)
	return nil
}

func (s *Scanner) comment(left, right string) {
	i := strings.Index(s.input[s.pos:], right)
	// Not a comment.
	if i == -1 {
		return
	}
	// If the comment reside inside a statement, collect it.
	if s.pos != len(left) {
		s.addPos(i + len(right))
		return
	}
	s.addPos(i + len(right))
	// If we did not scan any statement characters, it
	// can be skipped and stored in the comments group.
	s.comments = append(s.comments, s.input[:s.pos])
	s.input = s.input[s.pos:]
	s.pos = 0
	// Double \n separate the comments group from the statement.
	if strings.HasPrefix(s.input, "\n\n") || right == "\n" && strings.HasPrefix(s.input, "\n") {
		s.comments = nil
	}
	s.skipSpaces()
}

func (s *Scanner) skipSpaces() {
	n := len(s.input)
	s.input = strings.TrimLeftFunc(s.input, unicode.IsSpace)
	s.total += n - len(s.input)
}

func (s *Scanner) emit(text string) *Stmt {
	stmt := &Stmt{Pos: s.total - len(text), Text: text, Comments: s.comments}
	s.input = s.input[s.pos:]
	s.pos = 0
	s.comments = nil
	// Trim custom delimiter.
	if s.delim != delimiter {
		stmt.Text = strings.TrimSuffix(stmt.Text, s.delim)
	}
	stmt.Text = strings.TrimSpace(stmt.Text)
	return stmt
}

// delimCmd checks if the scanned "DELIMITER"
// text represents an actual delimiter command.
func (s *Scanner) delimCmd() error {
	// A space must come after the delimiter.
	if s.pick() != ' ' {
		return nil
	}
	// Scan delimiter.
	for r := s.pick(); r != eos && r != '\n'; r = s.next() {
	}
	delim := strings.TrimSpace(s.input[len(delimiterCmd):s.pos])
	// MySQL client allows quoting delimiters.
	if strings.HasPrefix(delim, "'") && strings.HasSuffix(delim, "'") {
		delim = strings.ReplaceAll(delim[1:len(delim)-1], "''", "'")
	}
	if err := s.setDelim(delim); err != nil {
		return err
	}
	// Skip all we saw until now.
	s.emit(s.input[:s.pos])
	return nil
}

// skipGoCount checks if the scanned "GO"
func (s *Scanner) skipGoCount() (err error) {
	// GO [count]\n
	if s.pick() == ' ' {
		c := s.pos
		// Scan [count]\n
		for r := s.pick(); r != eos && r != '\n'; {
			r = s.next()
		}
		_, err := strconv.Atoi(strings.TrimSpace(s.input[c:s.pos]))
		if err != nil {
			return fmt.Errorf("sql/migrate: invalid GO command, expect digits got %q: %w",
				s.input[c:s.pos], err)
		}
	}
	return nil
}

func (s *Scanner) setDelim(d string) error {
	if d == "" {
		return errors.New("empty delimiter")
	}
	// Unescape delimiters. e.g. "\\n" => "\n".
	s.delim = strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t").Replace(d)
	return nil
}

func (s *Scanner) error(pos int, format string, args ...any) error {
	format = "%d:%d: " + format
	var (
		p    = len(s.src) - len(s.input) + pos
		src  = s.src[:p]
		col  = strings.LastIndex(src, "\n")
		line = 1 + strings.Count(src, "\n")
	)
	if line == 1 {
		col = p
	} else {
		col = p - col - 1
	}
	return fmt.Errorf(format, append([]any{line, col}, args...)...)
}
