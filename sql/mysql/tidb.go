// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

type tinspect struct {
	inspect
}

func (i *tinspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	s, err := i.inspect.InspectSchema(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return i.patchSchema(ctx, s)
}

func (i *tinspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	r, err := i.inspect.InspectRealm(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, s := range r.Schemas {
		if _, err := i.patchSchema(ctx, s); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (i *tinspect) patchSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	for _, t := range s.Tables {
		var createStmt CreateStmt
		if ok := sqlx.Has(t.Attrs, &createStmt); !ok {
			if err := i.createStmt(ctx, t); err != nil {
				return nil, err
			}
		}
		if err := i.setCollate(t); err != nil {
			return nil, err
		}
		if err := i.setFKs(s, t); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// e.g CONSTRAINT "" FOREIGN KEY ("foo_id") REFERENCES "foo" ("id")
var reFK = regexp.MustCompile("(?i)CONSTRAINT\\s+[\"`]*(\\w+)[\"`]*\\s+FOREIGN\\s+KEY\\s*\\(([,\"` \\w]+)\\)\\s+REFERENCES\\s+[\"`]*(\\w+)[\"`]*\\s*\\(([,\"` \\w]+)\\)")

func (i *tinspect) setFKs(s *schema.Schema, t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("missing CREATE TABLE statment in attribuets for %q", t.Name)
	}
	for _, m := range reFK.FindAllStringSubmatch(c.S, -1) {
		if len(m) != 5 {
			return fmt.Errorf("unexpected number of matches for a table constraint: %q", m)
		}
		ctName, clmns, refTableName, refClmns := m[1], m[2], m[3], m[4]
		fk := &schema.ForeignKey{
			Symbol: ctName,
			Table:  t,
			// There is no support in TiDB for FKs so inherently there are no actions on update/delete.
			OnUpdate: schema.NoAction,
			OnDelete: schema.NoAction,
		}
		refTable, ok := s.Table(refTableName)
		if !ok {
			return fmt.Errorf("couldn't resolve ref table %s on ", m[3])
		}
		fk.RefTable = refTable
		for _, c := range columns(s, clmns) {
			column, ok := t.Column(c)
			if !ok {
				return fmt.Errorf("column %q was not found for fk %q", c, ctName)
			}
			fk.Columns = append(fk.Columns, column)
		}
		for _, c := range columns(s, refClmns) {
			column, ok := t.Column(c)
			if !ok {
				return fmt.Errorf("ref column %q was not found for fk %q", c, ctName)
			}
			fk.RefColumns = append(fk.RefColumns, column)
		}
		t.ForeignKeys = append(t.ForeignKeys, fk)
	}
	return nil
}

// columns from the matched regex above.
func columns(schema *schema.Schema, s string) []string {
	names := strings.Split(s, ",")
	for i := range names {
		names[i] = strings.Trim(strings.TrimSpace(names[i]), "`\"")
	}
	return names
}

// e.g CHARSET=utf8mb4 COLLATE=utf8mb4_bin
var reColl = regexp.MustCompile(`(?i)CHARSET\s*=\s*(\w+)\s*COLLATE\s*=\s*(\w+)`)

// setCollate extracts the updated Collation from CREATE TABLE statement.
func (i *tinspect) setCollate(t *schema.Table) error {
	var c CreateStmt
	if !sqlx.Has(t.Attrs, &c) {
		return fmt.Errorf("missing CREATE TABLE statment in attribuets for %q", t.Name)
	}
	matches := reColl.FindStringSubmatch(c.S)
	if len(matches) != 3 {
		return fmt.Errorf("missing COLLATE and/or CHARSET information on CREATE TABLE statment for %q", t.Name)
	}
	t.SetCharset(matches[1])
	t.SetCollation(matches[2])
	return nil
}
