// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mssql

import (
	"context"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"
)

// DefaultPlan provides basic planning capabilities for MS-SQL dialects.
// Note, it is recommended to call Open, create a new Driver and use its
// migrate.PlanApplier when a database connection is available.
var DefaultPlan migrate.PlanApplier = &planApply{conn: conn{ExecQuerier: sqlx.NoRows}}

// A planApply provides migration capabilities for schema elements.
type planApply struct{ conn }

// ApplyChanges applies the changes on the database. An error is returned
// if the driver is unable to produce a plan to do so, or one of the statements
// is failed or unsupported.
func (p *planApply) ApplyChanges(ctx context.Context, changes []schema.Change, opts ...migrate.PlanOption) error {
	return sqlx.ApplyChanges(ctx, changes, p, opts...)
}

// PlanChanges returns a migration plan for the given schema changes.
func (p *planApply) PlanChanges(_ context.Context, _ string, _ []schema.Change, _ ...migrate.PlanOption) (*migrate.Plan, error) {
	// Not implemented yet.
	return nil, nil
}
