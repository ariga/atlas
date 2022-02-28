// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sqlx

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

var hasher = fnv.New128()

// TwinDriver is a driver that provides additional functionality
// to interact with the twin/dev database.
type TwinDriver struct {
	// A Driver connected to the twin database.
	migrate.Driver

	// MaxNameLen configures the max length of object names in
	// the connected database (e.g. 64 in MySQL). Longer names
	// are trimmed and suffixed with their hash.
	MaxNameLen int

	// DropClause holds optional clauses that
	// can be added to the DropSchema change.
	DropClause []schema.Clause
}

// NormalizeRealm implements the schema.Normalizer interface.
//
// The implementation converts schema objects in "natural form" (e.g. HCL or DSL)
// to their "normal presentation" in the database, by creating them temporarily in
// a "twin database", and then inspects them from there.
func (t *TwinDriver) NormalizeRealm(ctx context.Context, r *schema.Realm) (nr *schema.Realm, err error) {
	var (
		twins   = make(map[string]string)
		changes = make([]schema.Change, 0, len(r.Schemas))
		reverse = make([]schema.Change, 0, len(r.Schemas))
		opts    = &schema.InspectRealmOption{
			Schemas: make([]string, 0, len(r.Schemas)),
		}
	)
	for _, s := range r.Schemas {
		twin := t.formatName(s.Name)
		twins[twin] = s.Name
		s.Name = twin
		opts.Schemas = append(opts.Schemas, s.Name)
		// Skip adding the schema.IfNotExists clause
		// to fail if the schema exists.
		st := schema.New(twin).AddAttrs(s.Attrs...)
		changes = append(changes, &schema.AddSchema{S: st})
		reverse = append(reverse, &schema.DropSchema{S: st, Extra: append(t.DropClause, &schema.IfExists{})})
		for _, t := range s.Tables {
			// If objects are not strongly connected.
			if t.Schema != s {
				t.Schema = s
			}
			changes = append(changes, &schema.AddTable{T: t})
		}
	}
	patch := func(r *schema.Realm) {
		for _, s := range r.Schemas {
			s.Name = twins[s.Name]
		}
	}
	// Delete the twin resources, and return
	// the source realm to its initial state.
	defer func() {
		patch(r)
		uerr := t.ApplyChanges(ctx, reverse)
		if err != nil {
			err = fmt.Errorf("%w: %v", err, uerr)
		}
		err = uerr
	}()
	if err := t.ApplyChanges(ctx, changes); err != nil {
		return nil, err
	}
	if nr, err = t.InspectRealm(ctx, opts); err != nil {
		return nil, err
	}
	patch(nr)
	return nr, nil
}

// NormalizeSchema returns the normal representation of the given database. See NormalizeRealm for more info.
func (t *TwinDriver) NormalizeSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	r := &schema.Realm{}
	if s.Realm != nil {
		r.Attrs = s.Realm.Attrs
	}
	r.Schemas = append(r.Schemas, s)
	nr, err := t.NormalizeRealm(ctx, r)
	if err != nil {
		return nil, err
	}
	ns, ok := nr.Schema(s.Name)
	if !ok {
		return nil, fmt.Errorf("missing normalized schema %q", s.Name)
	}
	return ns, nil
}

func (t *TwinDriver) formatName(name string) string {
	twin := fmt.Sprintf("atlas_twin_%s_%d", name, time.Now().Unix())
	if t.MaxNameLen == 0 || len(twin) <= t.MaxNameLen {
		return twin
	}
	return fmt.Sprintf("%s_%x", twin[:t.MaxNameLen-1-hasher.Size()*2], hasher.Sum([]byte(twin)))
}
