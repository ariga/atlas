// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stmts provides a generic implementation for extracting
// SQL statements from the given file contents.
func stmts(input string) ([]string, error) {
	var stmts []string
	l, err := newLex(input)
	if err != nil {
		return nil, err
	}
	for {
		s, err := l.stmt()
		if err == io.EOF {
			return stmts, nil
		}
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
	}
}

type lex struct {
	input string
	pos   int    // current position.
	width int    // size of latest rune.
	depth int    // depth of parentheses.
	delim string // configured delimiter.
}

const (
	eos          = -1
	delimiter    = ";"
	delimComment = "-- atlas:delimiter"
)

func newLex(input string) (*lex, error) {
	delim := delimiter
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, delimComment) {
		input = input[len(delimComment):]
		if !strings.HasPrefix(input, " ") {
			return nil, fmt.Errorf("expect space after %q, got: %s", delimComment, input[:1])
		}
		input = strings.TrimSpace(input)
		i := strings.Index(input, "\n")
		if i == -1 {
			return nil, errors.New("empty delimiter")
		}
		// Unescape delimiters. e.g. "\\n" => "\n".
		delim = strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t").
			Replace(input[:i])
		input = strings.TrimSpace(input[i+1:])
	}
	l := &lex{input: input, delim: delim}
	return l, nil
}

func (l *lex) stmt() (stmt string, err error) {
	defer func() {
		l.input = l.input[l.pos:]
		l.pos = 0
		// Trim custom delimiter.
		if l.delim != delimiter {
			stmt = strings.TrimSuffix(stmt, l.delim)
		}
	}()
	// Trim trailing whitespace.
	l.skipSpaces()
	for {
		switch r := l.next(); {
		case r == eos:
			if l.depth > 0 {
				return "", errors.New("unclosed parentheses")
			}
			if l.pos > 0 {
				return l.input, nil
			}
			return "", io.EOF
		case r == '(':
			l.depth++
		case r == ')':
			if l.depth == 0 {
				return "", fmt.Errorf("unexpected ')' at position %d", l.pos)
			}
			l.depth--
		case r == '\'', r == '"', r == '`':
			if err := l.skipQuote(r); err != nil {
				return "", err
			}
		case r == '-' && l.next() == '-':
			l.skipComment("--", "\n")
		case r == '/' && l.next() == '*':
			l.skipComment("/*", "*/")
		case strings.HasPrefix(l.input[l.pos-l.width:], l.delim) && l.depth == 0:
			l.pos += len(l.delim) - l.width
			return l.input[:l.pos], nil
		}
	}
}

func (l *lex) next() rune {
	if l.pos >= len(l.input) {
		return eos
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += w
	l.width = w
	return r
}

func (l *lex) skipQuote(quote rune) error {
	for {
		switch r := l.next(); {
		case r == eos:
			return fmt.Errorf("unclosed quote %q", quote)
		case r == '\\':
			l.next()
		case r == quote:
			return nil
		}
	}
}

func (l *lex) skipComment(left, right string) {
	i := strings.Index(l.input[l.pos:], right)
	// Not a comment.
	if i == -1 {
		return
	}
	// If we did not scan any statement
	// characters, it can be skipped.
	if l.pos == len(left) {
		l.input = l.input[l.pos+i+len(right):]
		l.pos = 0
		l.skipSpaces()
	} else {
		l.pos += i + len(right)
	}
}

func (l *lex) skipSpaces() {
	l.input = strings.TrimLeftFunc(l.input, unicode.IsSpace)
}
