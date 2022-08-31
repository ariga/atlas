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

// Stmt represents a scanned statement text along with its
// position in the file and associated comments group.
type Stmt struct {
	Text     string   // statement text
	Comments []string // associated comments
}

// stmts provides a generic implementation for extracting
// SQL statements from the given file contents.
func stmts(input string) ([]*Stmt, error) {
	var stmts []*Stmt
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
	input    string
	pos      int      // current position
	width    int      // size of latest rune
	depth    int      // depth of parentheses
	delim    string   // configured delimiter
	comments []string // collected comments
}

const (
	eos       = -1
	delimiter = ";"
)

func newLex(input string) (*lex, error) {
	delim := delimiter
	if d, ok := directive(input, directiveDelimiter, directivePrefixSQL); ok {
		if d == "" {
			return nil, errors.New("empty delimiter")
		}
		parts := strings.SplitN(input, "\n", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("not input found after delimiter %q", d)
		}
		// Unescape delimiters. e.g. "\\n" => "\n".
		delim = strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t").Replace(d)
		input = parts[1]
	}
	l := &lex{input: input, delim: delim}
	return l, nil
}

func (l *lex) stmt() (*Stmt, error) {
	var text string
	// Trim trailing whitespace.
	l.skipSpaces()
Scan:
	for {
		switch r := l.next(); {
		case r == eos:
			if l.depth > 0 {
				return nil, errors.New("unclosed parentheses")
			}
			if l.pos > 0 {
				text = l.input
				break Scan
			}
			return nil, io.EOF
		case r == '(':
			l.depth++
		case r == ')':
			if l.depth == 0 {
				return nil, fmt.Errorf("unexpected ')' at position %d", l.pos)
			}
			l.depth--
		case r == '\'', r == '"', r == '`':
			if err := l.skipQuote(r); err != nil {
				return nil, err
			}
		// Delimiters take precedence over comments.
		case strings.HasPrefix(l.input[l.pos-l.width:], l.delim) && l.depth == 0:
			l.pos += len(l.delim) - l.width
			text = l.input[:l.pos]
			break Scan
		case r == '#':
			l.comment("#", "\n")
		case r == '-' && l.next() == '-':
			l.comment("--", "\n")
		case r == '/' && l.next() == '*':
			l.comment("/*", "*/")
		}
	}
	return l.emit(text), nil
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

func (l *lex) pick() rune {
	p, w := l.pos, l.width
	r := l.next()
	l.pos, l.width = p, w
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

func (l *lex) comment(left, right string) {
	i := strings.Index(l.input[l.pos:], right)
	// Not a comment.
	if i == -1 {
		return
	}
	// If the comment reside inside a statement, collect it.
	if l.pos != len(left) {
		l.pos += i + len(right)
		return
	}
	// If we did not scan any statement characters, it
	// can be skipped and stored in the comments group.
	l.comments = append(l.comments, l.input[:l.pos+i+len(right)])
	l.input = l.input[l.pos+i+len(right):]
	l.pos = 0
	// Double \n separate the comments group from the statement.
	if strings.HasPrefix(l.input, "\n\n") || right == "\n" && strings.HasPrefix(l.input, "\n") {
		l.comments = nil
	}
	l.skipSpaces()
}

func (l *lex) skipSpaces() {
	l.input = strings.TrimLeftFunc(l.input, unicode.IsSpace)
}

func (l *lex) emit(text string) *Stmt {
	s := &Stmt{Text: text, Comments: l.comments}
	l.input = l.input[l.pos:]
	l.pos = 0
	l.comments = nil
	// Trim custom delimiter.
	if l.delim != delimiter {
		s.Text = strings.TrimSuffix(s.Text, l.delim)
	}
	s.Text = strings.TrimSpace(s.Text)
	return s
}
