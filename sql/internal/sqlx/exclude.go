// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"encoding/csv"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// ExcludeRealm filters resources in the realm based on the given patterns.
func ExcludeRealm(r *schema.Realm, patterns []string) (*schema.Realm, error) {
	if len(patterns) == 0 {
		return r, nil
	}
	var schemas []*schema.Schema
	globs, err := split(patterns)
	if err != nil {
		return nil, err
	}
Filter:
	for _, s := range r.Schemas {
		for i, g := range globs {
			if len(g) > 3 {
				return nil, fmt.Errorf("too many parts in pattern: %q", patterns[i])
			}
			match, err := filepath.Match(g[0], s.Name)
			if err != nil {
				return nil, err
			}
			if match {
				// In case there is a match, and it is
				// a single glob we exclude this schema.
				if len(g) == 1 {
					continue Filter
				}
				if err := excludeS(s, g[1:]); err != nil {
					return nil, err
				}
			}
		}
		schemas = append(schemas, s)
	}
	r.Schemas = schemas
	return r, nil
}

// ExcludeSchema filters resources in the schema based on the given patterns.
func ExcludeSchema(s *schema.Schema, patterns []string) (*schema.Schema, error) {
	if len(patterns) == 0 {
		return s, nil
	}
	if s.Realm == nil {
		return nil, fmt.Errorf("missing realm for schema %q", s.Name)
	}
	for i, p := range patterns {
		patterns[i] = fmt.Sprintf("%s.%s", s.Name, p)
	}
	if _, err := ExcludeRealm(s.Realm, patterns); err != nil {
		return nil, err
	}
	return s, nil
}

// split parses the list of patterns into chain of resource-globs.
// For example, 's*.t.*' is split to ['s*', 't', *].
func split(patterns []string) ([][]string, error) {
	globs := make([][]string, len(patterns))
	for i, p := range patterns {
		r := csv.NewReader(strings.NewReader(p))
		r.Comma = '.'
		switch parts, err := r.ReadAll(); {
		case err != nil:
			return nil, err
		case len(parts) != 1:
			return nil, fmt.Errorf("unexpected pattern: %q", p)
		case len(parts[0]) == 0:
			return nil, fmt.Errorf("empty pattern: %q", p)
		default:
			globs[i] = parts[0]
		}
	}
	return globs, nil
}

func excludeS(s *schema.Schema, glob []string) error {
	if globT, exclude := excludeType(typeT, glob[0]); exclude {
		var tables []*schema.Table
		for _, t := range s.Tables {
			match, err := filepath.Match(globT, t.Name)
			if err != nil {
				return err
			}
			if match {
				// In case there is a match, and it is
				// a single glob we exclude this table.
				if len(glob) == 1 {
					continue
				}
				if err := excludeT(t, glob[1]); err != nil {
					return err
				}
			}
			// No match or glob has more than one pattern.
			tables = append(tables, t)
		}
		s.Tables = tables
	}
	if globV, exclude := excludeType(typeV, glob[0]); exclude {
		var views []*schema.View
		for _, v := range s.Views {
			match, err := filepath.Match(globV, v.Name)
			if err != nil {
				return err
			}
			if match {
				if len(glob) == 1 {
					continue
				}
				if err := excludeV(v, glob[1]); err != nil {
					return err
				}
			}
			views = append(views, v)
		}
		s.Views = views
	}
	return nil
}

func excludeT(t *schema.Table, pattern string) (err error) {
	ex := make(map[*schema.Index]struct{})
	ef := make(map[*schema.ForeignKey]struct{})
	if p, exclude := excludeType(typeC, pattern); exclude {
		t.Columns, err = filter(t.Columns, func(c *schema.Column) (bool, error) {
			match, err := filepath.Match(p, c.Name)
			if !match || err != nil {
				return false, err
			}
			for _, idx := range c.Indexes {
				ex[idx] = struct{}{}
			}
			for _, fk := range c.ForeignKeys {
				ef[fk] = struct{}{}
			}
			return true, nil
		})
	}
	if p, exclude := excludeType(typeI, pattern); exclude {
		t.Indexes, err = filter(t.Indexes, func(idx *schema.Index) (bool, error) {
			if _, ok := ex[idx]; ok {
				return true, nil
			}
			return filepath.Match(p, idx.Name)
		})
	}
	if p, exclude := excludeType(typeF, pattern); exclude {
		t.ForeignKeys, err = filter(t.ForeignKeys, func(fk *schema.ForeignKey) (bool, error) {
			if _, ok := ef[fk]; ok {
				return true, nil
			}
			return filepath.Match(p, fk.Symbol)
		})
	}
	if p, exclude := excludeType(typeK, pattern); exclude {
		t.Attrs, err = filter(t.Attrs, func(a schema.Attr) (bool, error) {
			c, ok := a.(*schema.Check)
			if !ok {
				return false, nil
			}
			match, err := filepath.Match(p, c.Name)
			if !match || err != nil {
				return false, err
			}
			return true, nil
		})
	}
	return
}

func excludeV(t *schema.View, pattern string) (err error) {
	if p, exclude := excludeType(typeC, pattern); exclude {
		t.Columns, err = filter(t.Columns, func(c *schema.Column) (bool, error) {
			match, err := filepath.Match(p, c.Name)
			if !match || err != nil {
				return false, err
			}
			return true, nil
		})
	}
	return
}

const (
	typeV = "view"
	typeT = "table"
	typeC = "column"
	typeI = "index"
	typeF = "fk"
	typeK = "check"
)

var reTypeVT = regexp.MustCompile(`\[type=([a-z|]+)+\]$`)

func excludeType(t, v string) (string, bool) {
	matches := reTypeVT.FindStringSubmatch(v)
	if len(matches) != 2 {
		return v, true
	}
	v = strings.TrimSuffix(v, matches[0])
	for _, m := range strings.Split(matches[1], "|") {
		if m == t {
			// Selector matches.
			return v, true
		}
	}
	// There is a selector with no match.
	return v, false
}

func filter[T any](s []T, f func(T) (bool, error)) ([]T, error) {
	r := make([]T, 0, len(s))
	for i := range s {
		match, err := f(s[i])
		if err != nil {
			return nil, err
		}
		if !match {
			r = append(r, s[i])
		}
	}
	return r, nil
}
