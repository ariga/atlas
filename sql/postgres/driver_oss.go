// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package postgres

import (
	"context"
	"fmt"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

var (
	specOptions []schemahcl.Option
	specFuncs   = &specutil.SchemaFuncs{
		Table: tableSpec,
		View:  viewSpec,
	}
	scanFuncs = &specutil.ScanFuncs{
		Table: convertTable,
		View:  convertView,
	}
)

func triggersSpec([]*schema.Trigger, *doc) error {
	return nil // unimplemented.
}

func (*inspect) inspectViews(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectFuncs(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectTypes(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectSequences(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectTriggers(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectDeps(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*state) addView(*schema.AddView) error {
	return nil // unimplemented.
}

func (*state) dropView(*schema.DropView) error {
	return nil // unimplemented.
}

func (*state) modifyView(*schema.ModifyView) error {
	return nil // unimplemented.
}

func (*state) renameView(*schema.RenameView) {
	// unimplemented.
}

func (s *state) addFunc(*schema.AddFunc) error {
	return nil // unimplemented.
}

func (s *state) dropFunc(*schema.DropFunc) error {
	return nil // unimplemented.
}

func (s *state) modifyFunc(*schema.ModifyFunc) error {
	return nil // unimplemented.
}

func (s *state) renameFunc(*schema.RenameFunc) error {
	return nil // unimplemented.
}

func (s *state) addProc(*schema.AddProc) error {
	return nil // unimplemented.
}

func (s *state) dropProc(*schema.DropProc) error {
	return nil // unimplemented.
}

func (s *state) modifyProc(*schema.ModifyProc) error {
	return nil // unimplemented.
}

func (s *state) renameProc(*schema.RenameProc) error {
	return nil // unimplemented.
}

func (s *state) addObject(add *schema.AddObject) error {
	switch o := add.O.(type) {
	case *schema.EnumType:
		create, drop := s.createDropEnum(o)
		s.append(&migrate.Change{
			Source:  add,
			Cmd:     create,
			Reverse: drop,
			Comment: fmt.Sprintf("create enum type %q", o.T),
		})
	default:
		// unsupported object type.
	}
	return nil
}

func (s *state) dropObject(drop *schema.DropObject) error {
	switch o := drop.O.(type) {
	case *schema.EnumType:
		create, dropE := s.createDropEnum(o)
		s.append(&migrate.Change{
			Source:  drop,
			Cmd:     dropE,
			Reverse: create,
			Comment: fmt.Sprintf("drop enum type %q", o.T),
		})
	default:
		// unsupported object type.
	}
	return nil
}

func (s *state) modifyObject(modify *schema.ModifyObject) error {
	if _, ok := modify.From.(*schema.EnumType); ok {
		return s.alterEnum(modify)
	}
	return nil // unimplemented.
}

func (*state) addTrigger(*schema.AddTrigger) error {
	return nil // unimplemented.
}

func (*state) dropTrigger(*schema.DropTrigger) error {
	return nil // unimplemented.
}

func (*state) renameTrigger(*schema.RenameTrigger) error {
	return nil // unimplemented.
}

func (*state) modifyTrigger(*schema.ModifyTrigger) error {
	return nil // unimplemented.
}

func (d *diff) ViewAttrChanged(_, _ *schema.View) bool {
	return false // unimplemented.
}

// SchemaObjectDiff returns a changeset for migrating schema objects from
// one state to the other.
func (*diff) SchemaObjectDiff(from, to *schema.Schema) ([]schema.Change, error) {
	var changes []schema.Change
	// Drop or modify enums.
	for _, o1 := range from.Objects {
		e1, ok := o1.(*schema.EnumType)
		if !ok {
			continue // Unsupported object type.
		}
		o2, ok := to.Object(func(o schema.Object) bool {
			e2, ok := o.(*schema.EnumType)
			return ok && e1.T == e2.T
		})
		if !ok {
			changes = append(changes, &schema.DropObject{O: o1})
			continue
		}
		if e2 := o2.(*schema.EnumType); !sqlx.ValuesEqual(e1.Values, e2.Values) {
			changes = append(changes, &schema.ModifyObject{From: e1, To: e2})
		}
	}
	// Add new enums.
	for _, o1 := range to.Objects {
		e1, ok := o1.(*schema.EnumType)
		if !ok {
			continue // Unsupported object type.
		}
		if _, ok := from.Object(func(o schema.Object) bool {
			e2, ok := o.(*schema.EnumType)
			return ok && e1.T == e2.T
		}); !ok {
			changes = append(changes, &schema.AddObject{O: e1})
		}
	}
	return changes, nil
}

func verifyChanges(context.Context, []schema.Change) error {
	return nil // unimplemented.
}

func convertDomains(_ []*sqlspec.Table, domains []*domain, _ *schema.Realm) error {
	if len(domains) > 0 {
		return fmt.Errorf("postgres: domains are not supported by this version. Use: https://atlasgo.io/getting-started")
	}
	return nil
}

func convertSequences(_ []*sqlspec.Table, seqs []*sequence, _ *schema.Realm) error {
	if len(seqs) > 0 {
		return fmt.Errorf("postgres: sequences are not supported by this version. Use: https://atlasgo.io/getting-started")
	}
	return nil
}

func normalizeRealm(*schema.Realm) error {
	return nil
}

func qualifySeqRefs([]*sequence, []*sqlspec.Table, *schema.Realm) error {
	return nil // unimplemented.
}

// objectSpec converts from a concrete schema objects into specs.
func objectSpec(d *doc, spec *specutil.SchemaSpec, s *schema.Schema) error {
	for _, o := range s.Objects {
		if e, ok := o.(*schema.EnumType); ok {
			d.Enums = append(d.Enums, &enum{
				Name:   e.T,
				Values: e.Values,
				Schema: specutil.SchemaRef(spec.Schema.Name),
			})
		}
	}
	return nil
}
