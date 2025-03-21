// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package sqlite

import (
	"context"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/specutil"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlspec"
)

var (
	specOptions []schemahcl.Option
	scanFuncs   = &specutil.ScanFuncs{
		Table: convertTable,
		View:  convertView,
	}
)

func triggersSpec([]*schema.Trigger, *specutil.Doc) ([]*sqlspec.Trigger, error) {
	return nil, nil // unimplemented.
}

func (*inspect) inspectViews(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*inspect) inspectTriggers(context.Context, *schema.Realm, *schema.InspectOptions) error {
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

func (*state) renameView(*schema.RenameView) error {
	return nil // unimplemented.
}

func (*state) addTrigger(*schema.AddTrigger) error {
	return nil // unimplemented.
}

func (*state) dropTrigger(*schema.DropTrigger) error {
	return nil // unimplemented.
}

func verifyChanges(context.Context, []schema.Change) error {
	return nil // unimplemented.
}

// SupportChange reports if the change is supported by the differ.
func (*diff) SupportChange(c schema.Change) bool {
	switch c.(type) {
	case *schema.RenameConstraint:
		return false
	}
	return true
}
