// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"ariga.io/atlas/sql/migrate"
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

// SetType sets the "type" field.
func (rc *RevisionCreate) SetType(mt migrate.RevisionType) *RevisionCreate {
	rc.mutation.SetType(mt)
	return rc
}

// SetNillableType sets the "type" field if the given value is not nil.
func (rc *RevisionCreate) SetNillableType(mt *migrate.RevisionType) *RevisionCreate {
	if mt != nil {
		rc.SetType(*mt)
	}
	return rc
}

// SetApplied sets the "applied" field.
func (rc *RevisionCreate) SetApplied(i int) *RevisionCreate {
	rc.mutation.SetApplied(i)
	return rc
}

// SetTotal sets the "total" field.
func (rc *RevisionCreate) SetTotal(i int) *RevisionCreate {
	rc.mutation.SetTotal(i)
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

// SetError sets the "error" field.
func (rc *RevisionCreate) SetError(s string) *RevisionCreate {
	rc.mutation.SetError(s)
	return rc
}

// SetNillableError sets the "error" field if the given value is not nil.
func (rc *RevisionCreate) SetNillableError(s *string) *RevisionCreate {
	if s != nil {
		rc.SetError(*s)
	}
	return rc
}

// SetHash sets the "hash" field.
func (rc *RevisionCreate) SetHash(s string) *RevisionCreate {
	rc.mutation.SetHash(s)
	return rc
}

// SetPartialHashes sets the "partial_hashes" field.
func (rc *RevisionCreate) SetPartialHashes(s []string) *RevisionCreate {
	rc.mutation.SetPartialHashes(s)
	return rc
}

// SetOperatorVersion sets the "operator_version" field.
func (rc *RevisionCreate) SetOperatorVersion(s string) *RevisionCreate {
	rc.mutation.SetOperatorVersion(s)
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
	rc.defaults()
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
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = rc.hooks[i](mut)
		}
		v, err := mut.Mutate(ctx, rc.mutation)
		if err != nil {
			return nil, err
		}
		nv, ok := v.(*Revision)
		if !ok {
			return nil, fmt.Errorf("unexpected node type %T returned from RevisionMutation", v)
		}
		node = nv
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

// defaults sets the default values of the builder before save.
func (rc *RevisionCreate) defaults() {
	if _, ok := rc.mutation.GetType(); !ok {
		v := revision.DefaultType
		rc.mutation.SetType(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (rc *RevisionCreate) check() error {
	if _, ok := rc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "Revision.description"`)}
	}
	if _, ok := rc.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "Revision.type"`)}
	}
	if _, ok := rc.mutation.Applied(); !ok {
		return &ValidationError{Name: "applied", err: errors.New(`ent: missing required field "Revision.applied"`)}
	}
	if v, ok := rc.mutation.Applied(); ok {
		if err := revision.AppliedValidator(v); err != nil {
			return &ValidationError{Name: "applied", err: fmt.Errorf(`ent: validator failed for field "Revision.applied": %w`, err)}
		}
	}
	if _, ok := rc.mutation.Total(); !ok {
		return &ValidationError{Name: "total", err: errors.New(`ent: missing required field "Revision.total"`)}
	}
	if v, ok := rc.mutation.Total(); ok {
		if err := revision.TotalValidator(v); err != nil {
			return &ValidationError{Name: "total", err: fmt.Errorf(`ent: validator failed for field "Revision.total": %w`, err)}
		}
	}
	if _, ok := rc.mutation.ExecutedAt(); !ok {
		return &ValidationError{Name: "executed_at", err: errors.New(`ent: missing required field "Revision.executed_at"`)}
	}
	if _, ok := rc.mutation.ExecutionTime(); !ok {
		return &ValidationError{Name: "execution_time", err: errors.New(`ent: missing required field "Revision.execution_time"`)}
	}
	if _, ok := rc.mutation.Hash(); !ok {
		return &ValidationError{Name: "hash", err: errors.New(`ent: missing required field "Revision.hash"`)}
	}
	if _, ok := rc.mutation.OperatorVersion(); !ok {
		return &ValidationError{Name: "operator_version", err: errors.New(`ent: missing required field "Revision.operator_version"`)}
	}
	return nil
}

func (rc *RevisionCreate) sqlSave(ctx context.Context) (*Revision, error) {
	_node, _spec := rc.createSpec()
	if err := sqlgraph.CreateNode(ctx, rc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
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
	_spec.Schema = rc.schemaConfig.Revision
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
	if value, ok := rc.mutation.GetType(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeUint,
			Value:  value,
			Column: revision.FieldType,
		})
		_node.Type = value
	}
	if value, ok := rc.mutation.Applied(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: revision.FieldApplied,
		})
		_node.Applied = value
	}
	if value, ok := rc.mutation.Total(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: revision.FieldTotal,
		})
		_node.Total = value
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
	if value, ok := rc.mutation.Error(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldError,
		})
		_node.Error = value
	}
	if value, ok := rc.mutation.Hash(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldHash,
		})
		_node.Hash = value
	}
	if value, ok := rc.mutation.PartialHashes(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: revision.FieldPartialHashes,
		})
		_node.PartialHashes = value
	}
	if value, ok := rc.mutation.OperatorVersion(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: revision.FieldOperatorVersion,
		})
		_node.OperatorVersion = value
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

// SetType sets the "type" field.
func (u *RevisionUpsert) SetType(v migrate.RevisionType) *RevisionUpsert {
	u.Set(revision.FieldType, v)
	return u
}

// UpdateType sets the "type" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateType() *RevisionUpsert {
	u.SetExcluded(revision.FieldType)
	return u
}

// AddType adds v to the "type" field.
func (u *RevisionUpsert) AddType(v migrate.RevisionType) *RevisionUpsert {
	u.Add(revision.FieldType, v)
	return u
}

// SetApplied sets the "applied" field.
func (u *RevisionUpsert) SetApplied(v int) *RevisionUpsert {
	u.Set(revision.FieldApplied, v)
	return u
}

// UpdateApplied sets the "applied" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateApplied() *RevisionUpsert {
	u.SetExcluded(revision.FieldApplied)
	return u
}

// AddApplied adds v to the "applied" field.
func (u *RevisionUpsert) AddApplied(v int) *RevisionUpsert {
	u.Add(revision.FieldApplied, v)
	return u
}

// SetTotal sets the "total" field.
func (u *RevisionUpsert) SetTotal(v int) *RevisionUpsert {
	u.Set(revision.FieldTotal, v)
	return u
}

// UpdateTotal sets the "total" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateTotal() *RevisionUpsert {
	u.SetExcluded(revision.FieldTotal)
	return u
}

// AddTotal adds v to the "total" field.
func (u *RevisionUpsert) AddTotal(v int) *RevisionUpsert {
	u.Add(revision.FieldTotal, v)
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

// SetError sets the "error" field.
func (u *RevisionUpsert) SetError(v string) *RevisionUpsert {
	u.Set(revision.FieldError, v)
	return u
}

// UpdateError sets the "error" field to the value that was provided on create.
func (u *RevisionUpsert) UpdateError() *RevisionUpsert {
	u.SetExcluded(revision.FieldError)
	return u
}

// ClearError clears the value of the "error" field.
func (u *RevisionUpsert) ClearError() *RevisionUpsert {
	u.SetNull(revision.FieldError)
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

// SetPartialHashes sets the "partial_hashes" field.
func (u *RevisionUpsert) SetPartialHashes(v []string) *RevisionUpsert {
	u.Set(revision.FieldPartialHashes, v)
	return u
}

// UpdatePartialHashes sets the "partial_hashes" field to the value that was provided on create.
func (u *RevisionUpsert) UpdatePartialHashes() *RevisionUpsert {
	u.SetExcluded(revision.FieldPartialHashes)
	return u
}

// ClearPartialHashes clears the value of the "partial_hashes" field.
func (u *RevisionUpsert) ClearPartialHashes() *RevisionUpsert {
	u.SetNull(revision.FieldPartialHashes)
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
func (u *RevisionUpsertOne) UpdateNewValues() *RevisionUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(revision.FieldID)
		}
		if _, exists := u.create.mutation.Description(); exists {
			s.SetIgnore(revision.FieldDescription)
		}
		if _, exists := u.create.mutation.ExecutedAt(); exists {
			s.SetIgnore(revision.FieldExecutedAt)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Revision.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
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

// SetType sets the "type" field.
func (u *RevisionUpsertOne) SetType(v migrate.RevisionType) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetType(v)
	})
}

// AddType adds v to the "type" field.
func (u *RevisionUpsertOne) AddType(v migrate.RevisionType) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.AddType(v)
	})
}

// UpdateType sets the "type" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateType() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateType()
	})
}

// SetApplied sets the "applied" field.
func (u *RevisionUpsertOne) SetApplied(v int) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetApplied(v)
	})
}

// AddApplied adds v to the "applied" field.
func (u *RevisionUpsertOne) AddApplied(v int) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.AddApplied(v)
	})
}

// UpdateApplied sets the "applied" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateApplied() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateApplied()
	})
}

// SetTotal sets the "total" field.
func (u *RevisionUpsertOne) SetTotal(v int) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetTotal(v)
	})
}

// AddTotal adds v to the "total" field.
func (u *RevisionUpsertOne) AddTotal(v int) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.AddTotal(v)
	})
}

// UpdateTotal sets the "total" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateTotal() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateTotal()
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

// SetError sets the "error" field.
func (u *RevisionUpsertOne) SetError(v string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetError(v)
	})
}

// UpdateError sets the "error" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdateError() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateError()
	})
}

// ClearError clears the value of the "error" field.
func (u *RevisionUpsertOne) ClearError() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.ClearError()
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

// SetPartialHashes sets the "partial_hashes" field.
func (u *RevisionUpsertOne) SetPartialHashes(v []string) *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.SetPartialHashes(v)
	})
}

// UpdatePartialHashes sets the "partial_hashes" field to the value that was provided on create.
func (u *RevisionUpsertOne) UpdatePartialHashes() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdatePartialHashes()
	})
}

// ClearPartialHashes clears the value of the "partial_hashes" field.
func (u *RevisionUpsertOne) ClearPartialHashes() *RevisionUpsertOne {
	return u.Update(func(s *RevisionUpsert) {
		s.ClearPartialHashes()
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

// Exec executes the query.
func (u *RevisionUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for RevisionCreate.OnConflict")
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
		return id, errors.New("ent: RevisionUpsertOne.ID is not supported by MySQL driver. Use RevisionUpsertOne.Exec instead")
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
			builder.defaults()
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
							err = &ConstraintError{msg: err.Error(), wrap: err}
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
func (u *RevisionUpsertBulk) UpdateNewValues() *RevisionUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(revision.FieldID)
				return
			}
			if _, exists := b.mutation.Description(); exists {
				s.SetIgnore(revision.FieldDescription)
			}
			if _, exists := b.mutation.ExecutedAt(); exists {
				s.SetIgnore(revision.FieldExecutedAt)
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

// SetType sets the "type" field.
func (u *RevisionUpsertBulk) SetType(v migrate.RevisionType) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetType(v)
	})
}

// AddType adds v to the "type" field.
func (u *RevisionUpsertBulk) AddType(v migrate.RevisionType) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.AddType(v)
	})
}

// UpdateType sets the "type" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateType() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateType()
	})
}

// SetApplied sets the "applied" field.
func (u *RevisionUpsertBulk) SetApplied(v int) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetApplied(v)
	})
}

// AddApplied adds v to the "applied" field.
func (u *RevisionUpsertBulk) AddApplied(v int) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.AddApplied(v)
	})
}

// UpdateApplied sets the "applied" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateApplied() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateApplied()
	})
}

// SetTotal sets the "total" field.
func (u *RevisionUpsertBulk) SetTotal(v int) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetTotal(v)
	})
}

// AddTotal adds v to the "total" field.
func (u *RevisionUpsertBulk) AddTotal(v int) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.AddTotal(v)
	})
}

// UpdateTotal sets the "total" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateTotal() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateTotal()
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

// SetError sets the "error" field.
func (u *RevisionUpsertBulk) SetError(v string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetError(v)
	})
}

// UpdateError sets the "error" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdateError() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdateError()
	})
}

// ClearError clears the value of the "error" field.
func (u *RevisionUpsertBulk) ClearError() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.ClearError()
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

// SetPartialHashes sets the "partial_hashes" field.
func (u *RevisionUpsertBulk) SetPartialHashes(v []string) *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.SetPartialHashes(v)
	})
}

// UpdatePartialHashes sets the "partial_hashes" field to the value that was provided on create.
func (u *RevisionUpsertBulk) UpdatePartialHashes() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.UpdatePartialHashes()
	})
}

// ClearPartialHashes clears the value of the "partial_hashes" field.
func (u *RevisionUpsertBulk) ClearPartialHashes() *RevisionUpsertBulk {
	return u.Update(func(s *RevisionUpsert) {
		s.ClearPartialHashes()
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

// Exec executes the query.
func (u *RevisionUpsertBulk) Exec(ctx context.Context) error {
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("ent: OnConflict was set for builder %d. Set it on the RevisionCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for RevisionCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *RevisionUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
