// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysql

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"ariga.io/atlas/sql/schema"
)

// NormalizeRealm returns the normal representation of the given database.
func (d *Driver) NormalizeRealm(ctx context.Context, r *schema.Realm) (nr *schema.Realm, err error) {
	for _, s := range r.Schemas {
		switch s.Name {
		case "mysql", "information_schema", "performance_schema", "sys":
			return nil, fmt.Errorf("sql/mysql: normalizing internal schema %q is not supported", s.Name)
		}
	}
	var (
		twins   = make(map[string]string)
		changes = make([]schema.Change, 0, len(r.Schemas))
		reverse = make([]schema.Change, 0, len(r.Schemas))
		opts    = &schema.InspectRealmOption{
			Schemas: make([]string, 0, len(r.Schemas)),
		}
	)
	for _, s := range r.Schemas {
		twin := twinName(s.Name)
		twins[twin] = s.Name
		s.Name = twin
		opts.Schemas = append(opts.Schemas, s.Name)
		// Skip adding the schema.IfNotExists clause
		// to fail if the schema exists.
		st := schema.New(twin).AddAttrs(s.Attrs...)
		changes = append(changes, &schema.AddSchema{S: st})
		reverse = append(reverse, &schema.DropSchema{S: st, Extra: []schema.Clause{&schema.IfExists{}}})
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
		uerr := d.ApplyChanges(ctx, reverse)
		if err != nil {
			err = fmt.Errorf("%w: %v", err, uerr)
		}
		err = uerr
	}()
	if err := d.ApplyChanges(ctx, changes); err != nil {
		return nil, err
	}
	if nr, err = d.InspectRealm(ctx, opts); err != nil {
		return nil, err
	}
	patch(nr)
	return nr, nil
}

// NormalizeSchema returns the normal representation of the given database.
func (d *Driver) NormalizeSchema(ctx context.Context, s *schema.Schema) (*schema.Schema, error) {
	r := &schema.Realm{}
	if s.Realm != nil {
		r.Attrs = s.Realm.Attrs
	}
	r.Schemas = append(r.Schemas, s)
	nr, err := d.NormalizeRealm(ctx, r)
	if err != nil {
		return nil, err
	}
	ns, ok := nr.Schema(s.Name)
	if !ok {
		return nil, fmt.Errorf("sql/mysql: missing normalized schema %q", s.Name)
	}
	return ns, nil
}

const maxLen = 64

func twinName(name string) string {
	twin := fmt.Sprintf("atlas_twin_%s_%d", name, time.Now().Unix())
	if len(twin) <= maxLen {
		return twin
	}
	return fmt.Sprintf("%s_%x", twin[:maxLen-33], md5.Sum([]byte(twin)))
}
