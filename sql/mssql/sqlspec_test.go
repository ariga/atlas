// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"testing"

	"ariga.io/atlas/sql/internal/spectest"
)

func TestRegistrySanity(t *testing.T) {
	spectest.RegistrySanityTest(t, TypeRegistry, []string{
		// skip the following types as they are have different sizes in input and output
		// nchar(50) and nvarchar(50) have Size attribute as 100
		"nchar", "nvarchar",
	})
}
