// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ariga.io/atlas/sql/internal/migrate/revision"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// RevisionCreate is the builder for creating a Revision entity.
type RevisionCreate struct {
	config
	mutation *RevisionMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetDescription sets the "description" field.
func (rc *RevisionCreate) SetDescription(s string) *RevisionCreate {
	rc.mutation.SetDescription(s)
	return rc
}

// SetExecutionState sets the "execution_state" field.
func (rc *RevisionCreate) SetExecutionState(rs revision.ExecutionState) *RevisionCreate {
	rc.mutation.SetExecutionState(rs)
	return rc
}

// SetExecutedAt sets the "executed_at" field.
func (rc *RevisionCreate) SetExecutedAt(t time.Time) *RevisionCreate {
	rc.mutation.SetExecutedAt(t)
	return rc
}

// SetExecutionTime sets the "execution_time" field.
func (rc *RevisionCreate) SetExecutionTime(t time.Duration) *RevisionCreate {
	rc.mutation.SetExecutionTime(t)
	return rc
}

// SetHash sets the "hash" field.
func (rc *RevisionCreate) SetHash(s string) *RevisionCreate {
	rc.mutation.SetHash(s)
	return rc
}

// SetOperatorVersion sets the "operator_version" field.
func (rc *RevisionCreate) SetOperatorVersion(s string) *RevisionCreate {
	rc.mutation.SetOperatorVersion(s)
	return rc
}

// SetMeta sets the "meta" field.
func (rc *RevisionCreate) SetMeta(m map[string]string) *RevisionCreate {
	rc.mutation.SetMeta(m)
	return rc
}

// SetID sets the "id" field.
func (rc *RevisionCreate) SetID(s string) *RevisionCreate {
	rc.mutation.SetID(s)
	return rc
}

// Mutation returns the RevisionMutation object of the builder.
func (rc *RevisionCreate) Mutation() *RevisionMutation {
	return rc.mutation
}

// Save creates the Revision in the database.
func (rc *RevisionCreate) Save(ctx context.Context) (*Revision, error) {
	var (
		err  error
		node *Revision
	)
	if len(rc.hooks) == 0 {
		if err = rc.check(); err != nil {
			return nil, err
		}
		node, err = rc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*RevisionMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = rc.check(); err != nil {
				return nil, err
			}
			rc.mutation = mutation
			if node, err = rc.sqlSave(ctx); err != nil {
				return nil, err
			}
			mutation.id = &node.ID
			mutation.done = true
			return node, err
		})
		for i := len(rc.hooks) - 1; i >= 0; i-- {
			if rc.hooks[i] == nil {
				return nil, fmt.Errorf("migrate: uninitialized hook (forgotten import migrate/runtime?)")
			}
			mut = rc.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, rc.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (rc *RevisionCreate) SaveX(ctx context.Context) *Revision {
	v, err := rc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rc *RevisionCreate) Exec(ctx context.Context) error {
	_, err := rc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rc *RevisionCreate) ExecX(ctx context.Context) {
	if err := rc.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rc *RevisionCreate) check() error {
	if _, ok := rc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`migrate: missing required field "Revision.description"`)}
	}
	if _, ok := rc.mutation.ExecutionState(); !ok {
		return &ValidationError{Name: "execution_state", err: errors.New(`migrate: missing required field "Revision.execution_state"`)}
	}
	if v, ok := rc.mutation.ExecutionState(); ok {
		if err := revision.ExecutionStateValidator(v); err != nil {
			return &ValidationError{Name: "execution_state", err: fmt.Errorf(`migrate: validator failed for field "Revision.execution_state": %w`, err)}
		}
	}
	if _, ok := rc.mutation.ExecutedAt(); !ok {
		return &ValidationError{Name: "executed_at", err: errors.New(`migrate: missing required field "Revision.executed_at"`)}
	}
	if _, ok := rc.mutation.ExecutionTime(); !ok {
		return &ValidationError{Name: "execution_time", err: errors.New(`migrate: missing required field "Revision.execution_time"`)}
	}
	if _, ok := rc.mutation.Hash(); !ok {
		return &ValidationError{Name: "hash", err: errors.New(`migrate: missing required field "Revision.hash"`)}
	}
	if _, ok := rc.mutation.OperatorVersion(); !ok {
		return &ValidationError{Name: "operator_version", err: errors.New(`migrate: missing required field "Revision.operator_version"`)}
	}
	if _, ok := rc.mutation.Meta(); !ok {
		return &ValidationError{Name: "meta", err: errors.New(`migrate: missing required field "Revision.meta"`)}
	}
	return nil
}

func (rc *RevisionCreate) sqlSave(ctx context.Context) (*Revision, error) {
	_node, _spec := rc.createSpec()
	if err := sqlgraph.CreateNode(ctx, rc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(string); ok {
			_node.ID = id
		} else {
			return nil, fmt.Errorf("unexpected Revision.ID type: %T", _spec.ID.Value)
		}
	}
	return _node, nil
}

func (rc *RevisionCreate) createSpec() (*Revision, *sqlgraph.CreateSpec) {
	var (
		_node = &Revision{config: rc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: revision.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeString,
				Column: revision.FieldID,
			},
		}
	)
	_spec.OnConflict = rc.conflict
	if id, ok := rc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := rc.mutation.Description(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldDescription,
		})
		_node.Description = value
	}
	if value, ok := rc.mutation.ExecutionState(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: revision.FieldExecutionState,
		})
		_node.ExecutionState = value
	}
	if value, ok := rc.mutation.ExecutedAt(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: revision.FieldExecutedAt,
		})
		_node.ExecutedAt = value
	}
	if value, ok := rc.mutation.ExecutionTime(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt64,
			Value:  value,
			Column: revision.FieldExecutionTime,
		})
		_node.ExecutionTime = value
	}
	if value, ok := rc.mutation.Hash(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldHash,
		})
		_node.Hash = value
	}
	if value, ok := rc.mutation.OperatorVersion(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldOperatorVersion,
		})
		_node.OperatorVersion = value
	}
	if value, ok := rc.mutation.Meta(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: revision.FieldMeta,
		})
		_node.Meta = value
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Revision.Create().
//		SetDescription(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.RevisionUpsert) {
//			SetDescription(v+v).
//		}).
//		Exec(ctx)
//
func (rc *RevisionCreate) OnConflict(opts ...sql.ConflictOption) *RevisionUpsertOne {
	rc.conflict = opts
	return &RevisionUpsertOne{
		create: rc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Revision.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
//
func (rc *RevisionCreate) OnConflictColumns(columns ...string) *RevisionUpsertOne {
	rc.conflict = append(rc.conflict, sql.ConflictColumns(columns...))
	return &RevisionUpsertOne{
		create: rc,
	}
}

type (
	// RevisionUpsertOne is the builder for "upsert"-ing
	//  one Revision node.
	RevisionUpsertOne struct {
		create *RevisionCreate
	}

	// RevisionUpsert is the "OnConflict" setter.
	RevisionUpsert struct {
		*sql.UpdateSet
	}
)

// SetDescription sets the "description" field.
func (u *RevisionUpsert) SetDescription(v string) *RevisionUpsert {
	u.Set(revision.FieldDescription, v)
	return u
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateDescription() *RevisionUpsert {
	u.SetExcluded(revision.FieldDescription)
	return u
}

// SetExecutionState sets the "execution_state" field.
func (u *RevisionUpsert) SetExecutionState(v revision.ExecutionState) *RevisionUpsert {
	u.Set(revision.FieldExecutionState, v)
	return u
}

// UpdateExecutionState sets the "execution_state" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateExecutionState() *RevisionUpsert {
	u.SetExcluded(revision.FieldExecutionState)
	return u
}

// SetExecutedAt sets the "executed_at" field.
func (u *RevisionUpsert) SetExecutedAt(v time.Time) *RevisionUpsert {
	u.Set(revision.FieldExecutedAt, v)
	return u
}

// UpdateExecutedAt sets the "executed_at" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateExecutedAt() *RevisionUpsert {
	u.SetExcluded(revision.FieldExecutedAt)
	return u
}

// SetExecutionTime sets the "execution_time" field.
func (u *RevisionUpsert) SetExecutionTime(v time.Duration) *RevisionUpsert {
	u.Set(revision.FieldExecutionTime, v)
	return u
}

// UpdateExecutionTime sets the "execution_time" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateExecutionTime() *RevisionUpsert {
	u.SetExcluded(revision.FieldExecutionTime)
	return u
}

// AddExecutionTime adds v to the "execution_time" field.
func (u *RevisionUpsert) AddExecutionTime(v time.Duration) *RevisionUpsert {
	u.Add(revision.FieldExecutionTime, v)
	return u
}

// SetHash sets the "hash" field.
func (u *RevisionUpsert) SetHash(v string) *RevisionUpsert {
	u.Set(revision.FieldHash, v)
	return u
}

// UpdateHash sets the "hash" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateHash() *RevisionUpsert {
	u.SetExcluded(revision.FieldHash)
	return u
}

// SetOperatorVersion sets the "operator_version" field.
func (u *RevisionUpsert) SetOperatorVersion(v string) *RevisionUpsert {
	u.Set(revision.FieldOperatorVersion, v)
	return u
}

// UpdateOperatorVersion sets the "operator_version" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateOperatorVersion() *RevisionUpsert {
	u.SetExcluded(revision.FieldOperatorVersion)
	return u
}

// SetMeta sets the "meta" field.
func (u *RevisionUpsert) SetMeta(v map[string]string) *RevisionUpsert {
	u.Set(revision.FieldMeta, v)
	return u
}

// UpdateMeta sets the "meta" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateMeta() *RevisionUpsert {
	u.SetExcluded(revision.FieldMeta)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Revision.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(revision.FieldID)
//			}),
//		).
//		Exec(ctx)
//
func (u *RevisionUpsertOne) UpdateNewValues() *RevisionUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(revision.FieldID)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//  client.Revision.Create().
//      OnConflict(sql.ResolveWithIgnore()).
//      Exec(ctx)
//
func (u *RevisionUpsertOne) Ignore() *RevisionUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *RevisionUpsertOne) DoNothing() *RevisionUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the RevisionCreate.OnConflict
// documentation for more info.
func (u *RevisionUpsertOne) Update(set func(*RevisionUpsert)) *RevisionUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&RevisionUpsert{UpdateSet: update})
	}))
	return u
}

// SetDescription sets the "description" field.
func (u *RevisionUpsertOne) SetDescription(v string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetDescription(v)
	})
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateDescription() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateDescription()
	})
}

// SetExecutionState sets the "execution_state" field.
func (u *RevisionUpsertOne) SetExecutionState(v revision.ExecutionState) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutionState(v)
	})
}

// UpdateExecutionState sets the "execution_state" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateExecutionState() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutionState()
	})
}

// SetExecutedAt sets the "executed_at" field.
func (u *RevisionUpsertOne) SetExecutedAt(v time.Time) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutedAt(v)
	})
}

// UpdateExecutedAt sets the "executed_at" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateExecutedAt() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutedAt()
	})
}

// SetExecutionTime sets the "execution_time" field.
func (u *RevisionUpsertOne) SetExecutionTime(v time.Duration) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutionTime(v)
	})
}

// AddExecutionTime adds v to the "execution_time" field.
func (u *RevisionUpsertOne) AddExecutionTime(v time.Duration) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.AddExecutionTime(v)
	})
}

// UpdateExecutionTime sets the "execution_time" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateExecutionTime() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutionTime()
	})
}

// SetHash sets the "hash" field.
func (u *RevisionUpsertOne) SetHash(v string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetHash(v)
	})
}

// UpdateHash sets the "hash" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateHash() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateHash()
	})
}

// SetOperatorVersion sets the "operator_version" field.
func (u *RevisionUpsertOne) SetOperatorVersion(v string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetOperatorVersion(v)
	})
}

// UpdateOperatorVersion sets the "operator_version" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateOperatorVersion() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateOperatorVersion()
	})
}

// SetMeta sets the "meta" field.
func (u *RevisionUpsertOne) SetMeta(v map[string]string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetMeta(v)
	})
}

// UpdateMeta sets the "meta" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateMeta() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateMeta()
	})
}

// Exec executes the query.
func (u *RevisionUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("migrate: missing options for RevisionCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *RevisionUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *RevisionUpsertOne) ID(ctx context.Context) (id string, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("migrate: RevisionUpsertOne.ID is not supported by MySQL driver. Use RevisionUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *RevisionUpsertOne) IDX(ctx context.Context) string {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// RevisionCreateBulk is the builder for creating many Revision entities in bulk.
type RevisionCreateBulk struct {
	config
	builders []*RevisionCreate
	conflict []sql.ConflictOption
}

// Save creates the Revision entities in the database.
func (rcb *RevisionCreateBulk) Save(ctx context.Context) ([]*Revision, error) {
	specs := make([]*sqlgraph.CreateSpec, len(rcb.builders))
	nodes := make([]*Revision, len(rcb.builders))
	mutators := make([]Mutator, len(rcb.builders))
	for i := range rcb.builders {
		func(i int, root context.Context) {
			builder := rcb.builders[i]
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*RevisionMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				nodes[i], specs[i] = builder.createSpec()
				var err error
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, rcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = rcb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, rcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{err.Error(), err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, rcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (rcb *RevisionCreateBulk) SaveX(ctx context.Context) []*Revision {
	v, err := rcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (rcb *RevisionCreateBulk) Exec(ctx context.Context) error {
	_, err := rcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (rcb *RevisionCreateBulk) ExecX(ctx context.Context) {
	if err := rcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Revision.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.RevisionUpsert) {
//			SetDescription(v+v).
//		}).
//		Exec(ctx)
//
func (rcb *RevisionCreateBulk) OnConflict(opts ...sql.ConflictOption) *RevisionUpsertBulk {
	rcb.conflict = opts
	return &RevisionUpsertBulk{
		create: rcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Revision.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
//
func (rcb *RevisionCreateBulk) OnConflictColumns(columns ...string) *RevisionUpsertBulk {
	rcb.conflict = append(rcb.conflict, sql.ConflictColumns(columns...))
	return &RevisionUpsertBulk{
		create: rcb,
	}
}

// RevisionUpsertBulk is the builder for "upsert"-ing
// a bulk of Revision nodes.
type RevisionUpsertBulk struct {
	create *RevisionCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Revision.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(revision.FieldID)
//			}),
//		).
//		Exec(ctx)
//
func (u *RevisionUpsertBulk) UpdateNewValues() *RevisionUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(revision.FieldID)
				return
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Revision.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
//
func (u *RevisionUpsertBulk) Ignore() *RevisionUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *RevisionUpsertBulk) DoNothing() *RevisionUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the RevisionCreateBulk.OnConflict
// documentation for more info.
func (u *RevisionUpsertBulk) Update(set func(*RevisionUpsert)) *RevisionUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&RevisionUpsert{UpdateSet: update})
	}))
	return u
}

// SetDescription sets the "description" field.
func (u *RevisionUpsertBulk) SetDescription(v string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetDescription(v)
	})
}

// UpdateDescription sets the "description" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateDescription() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateDescription()
	})
}

// SetExecutionState sets the "execution_state" field.
func (u *RevisionUpsertBulk) SetExecutionState(v revision.ExecutionState) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutionState(v)
	})
}

// UpdateExecutionState sets the "execution_state" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateExecutionState() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutionState()
	})
}

// SetExecutedAt sets the "executed_at" field.
func (u *RevisionUpsertBulk) SetExecutedAt(v time.Time) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutedAt(v)
	})
}

// UpdateExecutedAt sets the "executed_at" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateExecutedAt() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutedAt()
	})
}

// SetExecutionTime sets the "execution_time" field.
func (u *RevisionUpsertBulk) SetExecutionTime(v time.Duration) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetExecutionTime(v)
	})
}

// AddExecutionTime adds v to the "execution_time" field.
func (u *RevisionUpsertBulk) AddExecutionTime(v time.Duration) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.AddExecutionTime(v)
	})
}

// UpdateExecutionTime sets the "execution_time" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateExecutionTime() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateExecutionTime()
	})
}

// SetHash sets the "hash" field.
func (u *RevisionUpsertBulk) SetHash(v string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetHash(v)
	})
}

// UpdateHash sets the "hash" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateHash() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateHash()
	})
}

// SetOperatorVersion sets the "operator_version" field.
func (u *RevisionUpsertBulk) SetOperatorVersion(v string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetOperatorVersion(v)
	})
}

// UpdateOperatorVersion sets the "operator_version" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateOperatorVersion() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateOperatorVersion()
	})
}

// SetMeta sets the "meta" field.
func (u *RevisionUpsertBulk) SetMeta(v map[string]string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetMeta(v)
	})
}

// UpdateMeta sets the "meta" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateMeta() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateMeta()
	})
}

// Exec executes the query.
func (u *RevisionUpsertBulk) Exec(ctx context.Context) error {
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("migrate: OnConflict was set for builder %d. Set it on the RevisionCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("migrate: missing options for RevisionCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *RevisionUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
