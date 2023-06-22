// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

//go:build !ent

package postgres

import (
	"context"

	"ariga.io/atlas/sql/schema"
)

func (*inspect) inspectViews(context.Context, *schema.Realm, *schema.InspectOptions) error {
	return nil // unimplemented.
}

func (*state) addView(context.Context, *schema.AddView) error {
	return nil // unimplemented.
}

func (*state) dropView(context.Context, *schema.DropView) error {
	return nil // unimplemented.
}

func (*state) modifyView(context.Context, *schema.ModifyView) error {
	return nil // unimplemented.
}

func (*state) renameView(context.Context, *schema.RenameView) {
	// unimplemented.
}

// testing stubs.
var queryViews string
