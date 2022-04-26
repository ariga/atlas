// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/internal/migrate/predicate"
	"ariga.io/atlas/sql/internal/migrate/revision"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// RevisionDelete is the builder for deleting a Revision entity.
type RevisionDelete struct {
	config
	hooks    []Hook
	mutation *RevisionMutation
}

// Where appends a list predicates to the RevisionDelete builder.
func (rd *RevisionDelete) Where(ps ...predicate.Revision) *RevisionDelete {
	rd.mutation.Where(ps...)
	return rd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (rd *RevisionDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(rd.hooks) == 0 {
		affected, err = rd.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*RevisionMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			rd.mutation = mutation
			affected, err = rd.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(rd.hooks) - 1; i >= 0; i-- {
			if rd.hooks[i] == nil {
				return 0, fmt.Errorf("migrate: uninitialized hook (forgotten import migrate/runtime?)")
			}
			mut = rd.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, rd.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (rd *RevisionDelete) ExecX(ctx context.Context) int {
	n, err := rd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (rd *RevisionDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: revision.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeString,
				Column: revision.FieldID,
			},
		},
	}
	if ps := rd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return sqlgraph.DeleteNodes(ctx, rd.driver, _spec)
}

// RevisionDeleteOne is the builder for deleting a single Revision entity.
type RevisionDeleteOne struct {
	rd *RevisionDelete
}

// Exec executes the deletion query.
func (rdo *RevisionDeleteOne) Exec(ctx context.Context) error {
	n, err := rdo.rd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{revision.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (rdo *RevisionDeleteOne) ExecX(ctx context.Context) {
	rdo.rd.ExecX(ctx)
}
