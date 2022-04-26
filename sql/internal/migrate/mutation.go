// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package migrate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"ariga.io/atlas/sql/internal/migrate/predicate"
	"ariga.io/atlas/sql/internal/migrate/revision"

	"entgo.io/ent"
)

const (
	// Operation types.
	OpCreate    = ent.OpCreate
	OpDelete    = ent.OpDelete
	OpDeleteOne = ent.OpDeleteOne
	OpUpdate    = ent.OpUpdate
	OpUpdateOne = ent.OpUpdateOne

	// Node types.
	TypeRevision = "Revision"
)

// RevisionMutation represents an operation that mutates the Revision nodes in the graph.
type RevisionMutation struct {
	config
	op                Op
	typ               string
	id                *string
	description       *string
	execution_state   *revision.ExecutionState
	executed_at       *time.Time
	execution_time    *time.Duration
	addexecution_time *time.Duration
	hash              *string
	operator_version  *string
	meta              *map[string]string
	clearedFields     map[string]struct{}
	done              bool
	oldValue          func(context.Context) (*Revision, error)
	predicates        []predicate.Revision
}

var _ ent.Mutation = (*RevisionMutation)(nil)

// revisionOption allows management of the mutation configuration using functional options.
type revisionOption func(*RevisionMutation)

// newRevisionMutation creates new mutation for the Revision entity.
func newRevisionMutation(c config, op Op, opts ...revisionOption) *RevisionMutation {
	m := &RevisionMutation{
		config:        c,
		op:            op,
		typ:           TypeRevision,
		clearedFields: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// withRevisionID sets the ID field of the mutation.
func withRevisionID(id string) revisionOption {
	return func(m *RevisionMutation) {
		var (
			err   error
			once  sync.Once
			value *Revision
		)
		m.oldValue = func(ctx context.Context) (*Revision, error) {
			once.Do(func() {
				if m.done {
					err = errors.New("querying old values post mutation is not allowed")
				} else {
					value, err = m.Client().Revision.Get(ctx, id)
				}
			})
			return value, err
		}
		m.id = &id
	}
}

// withRevision sets the old Revision of the mutation.
func withRevision(node *Revision) revisionOption {
	return func(m *RevisionMutation) {
		m.oldValue = func(context.Context) (*Revision, error) {
			return node, nil
		}
		m.id = &node.ID
	}
}

// Client returns a new `ent.Client` from the mutation. If the mutation was
// executed in a transaction (ent.Tx), a transactional client is returned.
func (m RevisionMutation) Client() *Client {
	client := &Client{config: m.config}
	client.init()
	return client
}

// Tx returns an `ent.Tx` for mutations that were executed in transactions;
// it returns an error otherwise.
func (m RevisionMutation) Tx() (*Tx, error) {
	if _, ok := m.driver.(*txDriver); !ok {
		return nil, errors.New("migrate: mutation is not running in a transaction")
	}
	tx := &Tx{config: m.config}
	tx.init()
	return tx, nil
}

// SetID sets the value of the id field. Note that this
// operation is only accepted on creation of Revision entities.
func (m *RevisionMutation) SetID(id string) {
	m.id = &id
}

// ID returns the ID value in the mutation. Note that the ID is only available
// if it was provided to the builder or after it was returned from the database.
func (m *RevisionMutation) ID() (id string, exists bool) {
	if m.id == nil {
		return
	}
	return *m.id, true
}

// IDs queries the database and returns the entity ids that match the mutation's predicate.
// That means, if the mutation is applied within a transaction with an isolation level such
// as sql.LevelSerializable, the returned ids match the ids of the rows that will be updated
// or updated by the mutation.
func (m *RevisionMutation) IDs(ctx context.Context) ([]string, error) {
	switch {
	case m.op.Is(OpUpdateOne | OpDeleteOne):
		id, exists := m.ID()
		if exists {
			return []string{id}, nil
		}
		fallthrough
	case m.op.Is(OpUpdate | OpDelete):
		return m.Client().Revision.Query().Where(m.predicates...).IDs(ctx)
	default:
		return nil, fmt.Errorf("IDs is not allowed on %s operations", m.op)
	}
}

// SetDescription sets the "description" field.
func (m *RevisionMutation) SetDescription(s string) {
	m.description = &s
}

// Description returns the value of the "description" field in the mutation.
func (m *RevisionMutation) Description() (r string, exists bool) {
	v := m.description
	if v == nil {
		return
	}
	return *v, true
}

// OldDescription returns the old "description" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldDescription(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldDescription is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldDescription requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldDescription: %w", err)
	}
	return oldValue.Description, nil
}

// ResetDescription resets all changes to the "description" field.
func (m *RevisionMutation) ResetDescription() {
	m.description = nil
}

// SetExecutionState sets the "execution_state" field.
func (m *RevisionMutation) SetExecutionState(rs revision.ExecutionState) {
	m.execution_state = &rs
}

// ExecutionState returns the value of the "execution_state" field in the mutation.
func (m *RevisionMutation) ExecutionState() (r revision.ExecutionState, exists bool) {
	v := m.execution_state
	if v == nil {
		return
	}
	return *v, true
}

// OldExecutionState returns the old "execution_state" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldExecutionState(ctx context.Context) (v revision.ExecutionState, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldExecutionState is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldExecutionState requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldExecutionState: %w", err)
	}
	return oldValue.ExecutionState, nil
}

// ResetExecutionState resets all changes to the "execution_state" field.
func (m *RevisionMutation) ResetExecutionState() {
	m.execution_state = nil
}

// SetExecutedAt sets the "executed_at" field.
func (m *RevisionMutation) SetExecutedAt(t time.Time) {
	m.executed_at = &t
}

// ExecutedAt returns the value of the "executed_at" field in the mutation.
func (m *RevisionMutation) ExecutedAt() (r time.Time, exists bool) {
	v := m.executed_at
	if v == nil {
		return
	}
	return *v, true
}

// OldExecutedAt returns the old "executed_at" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldExecutedAt(ctx context.Context) (v time.Time, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldExecutedAt is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldExecutedAt requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldExecutedAt: %w", err)
	}
	return oldValue.ExecutedAt, nil
}

// ResetExecutedAt resets all changes to the "executed_at" field.
func (m *RevisionMutation) ResetExecutedAt() {
	m.executed_at = nil
}

// SetExecutionTime sets the "execution_time" field.
func (m *RevisionMutation) SetExecutionTime(t time.Duration) {
	m.execution_time = &t
	m.addexecution_time = nil
}

// ExecutionTime returns the value of the "execution_time" field in the mutation.
func (m *RevisionMutation) ExecutionTime() (r time.Duration, exists bool) {
	v := m.execution_time
	if v == nil {
		return
	}
	return *v, true
}

// OldExecutionTime returns the old "execution_time" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldExecutionTime(ctx context.Context) (v time.Duration, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldExecutionTime is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldExecutionTime requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldExecutionTime: %w", err)
	}
	return oldValue.ExecutionTime, nil
}

// AddExecutionTime adds t to the "execution_time" field.
func (m *RevisionMutation) AddExecutionTime(t time.Duration) {
	if m.addexecution_time != nil {
		*m.addexecution_time += t
	} else {
		m.addexecution_time = &t
	}
}

// AddedExecutionTime returns the value that was added to the "execution_time" field in this mutation.
func (m *RevisionMutation) AddedExecutionTime() (r time.Duration, exists bool) {
	v := m.addexecution_time
	if v == nil {
		return
	}
	return *v, true
}

// ResetExecutionTime resets all changes to the "execution_time" field.
func (m *RevisionMutation) ResetExecutionTime() {
	m.execution_time = nil
	m.addexecution_time = nil
}

// SetHash sets the "hash" field.
func (m *RevisionMutation) SetHash(s string) {
	m.hash = &s
}

// Hash returns the value of the "hash" field in the mutation.
func (m *RevisionMutation) Hash() (r string, exists bool) {
	v := m.hash
	if v == nil {
		return
	}
	return *v, true
}

// OldHash returns the old "hash" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldHash(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldHash is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldHash requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldHash: %w", err)
	}
	return oldValue.Hash, nil
}

// ResetHash resets all changes to the "hash" field.
func (m *RevisionMutation) ResetHash() {
	m.hash = nil
}

// SetOperatorVersion sets the "operator_version" field.
func (m *RevisionMutation) SetOperatorVersion(s string) {
	m.operator_version = &s
}

// OperatorVersion returns the value of the "operator_version" field in the mutation.
func (m *RevisionMutation) OperatorVersion() (r string, exists bool) {
	v := m.operator_version
	if v == nil {
		return
	}
	return *v, true
}

// OldOperatorVersion returns the old "operator_version" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldOperatorVersion(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldOperatorVersion is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldOperatorVersion requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldOperatorVersion: %w", err)
	}
	return oldValue.OperatorVersion, nil
}

// ResetOperatorVersion resets all changes to the "operator_version" field.
func (m *RevisionMutation) ResetOperatorVersion() {
	m.operator_version = nil
}

// SetMeta sets the "meta" field.
func (m *RevisionMutation) SetMeta(value map[string]string) {
	m.meta = &value
}

// Meta returns the value of the "meta" field in the mutation.
func (m *RevisionMutation) Meta() (r map[string]string, exists bool) {
	v := m.meta
	if v == nil {
		return
	}
	return *v, true
}

// OldMeta returns the old "meta" field's value of the Revision entity.
// If the Revision object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *RevisionMutation) OldMeta(ctx context.Context) (v map[string]string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldMeta is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldMeta requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldMeta: %w", err)
	}
	return oldValue.Meta, nil
}

// ResetMeta resets all changes to the "meta" field.
func (m *RevisionMutation) ResetMeta() {
	m.meta = nil
}

// Where appends a list predicates to the RevisionMutation builder.
func (m *RevisionMutation) Where(ps ...predicate.Revision) {
	m.predicates = append(m.predicates, ps...)
}

// Op returns the operation name.
func (m *RevisionMutation) Op() Op {
	return m.op
}

// Type returns the node type of this mutation (Revision).
func (m *RevisionMutation) Type() string {
	return m.typ
}

// Fields returns all fields that were changed during this mutation. Note that in
// order to get all numeric fields that were incremented/decremented, call
// AddedFields().
func (m *RevisionMutation) Fields() []string {
	fields := make([]string, 0, 7)
	if m.description != nil {
		fields = append(fields, revision.FieldDescription)
	}
	if m.execution_state != nil {
		fields = append(fields, revision.FieldExecutionState)
	}
	if m.executed_at != nil {
		fields = append(fields, revision.FieldExecutedAt)
	}
	if m.execution_time != nil {
		fields = append(fields, revision.FieldExecutionTime)
	}
	if m.hash != nil {
		fields = append(fields, revision.FieldHash)
	}
	if m.operator_version != nil {
		fields = append(fields, revision.FieldOperatorVersion)
	}
	if m.meta != nil {
		fields = append(fields, revision.FieldMeta)
	}
	return fields
}

// Field returns the value of a field with the given name. The second boolean
// return value indicates that this field was not set, or was not defined in the
// schema.
func (m *RevisionMutation) Field(name string) (ent.Value, bool) {
	switch name {
	case revision.FieldDescription:
		return m.Description()
	case revision.FieldExecutionState:
		return m.ExecutionState()
	case revision.FieldExecutedAt:
		return m.ExecutedAt()
	case revision.FieldExecutionTime:
		return m.ExecutionTime()
	case revision.FieldHash:
		return m.Hash()
	case revision.FieldOperatorVersion:
		return m.OperatorVersion()
	case revision.FieldMeta:
		return m.Meta()
	}
	return nil, false
}

// OldField returns the old value of the field from the database. An error is
// returned if the mutation operation is not UpdateOne, or the query to the
// database failed.
func (m *RevisionMutation) OldField(ctx context.Context, name string) (ent.Value, error) {
	switch name {
	case revision.FieldDescription:
		return m.OldDescription(ctx)
	case revision.FieldExecutionState:
		return m.OldExecutionState(ctx)
	case revision.FieldExecutedAt:
		return m.OldExecutedAt(ctx)
	case revision.FieldExecutionTime:
		return m.OldExecutionTime(ctx)
	case revision.FieldHash:
		return m.OldHash(ctx)
	case revision.FieldOperatorVersion:
		return m.OldOperatorVersion(ctx)
	case revision.FieldMeta:
		return m.OldMeta(ctx)
	}
	return nil, fmt.Errorf("unknown Revision field %s", name)
}

// SetField sets the value of a field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *RevisionMutation) SetField(name string, value ent.Value) error {
	switch name {
	case revision.FieldDescription:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetDescription(v)
		return nil
	case revision.FieldExecutionState:
		v, ok := value.(revision.ExecutionState)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetExecutionState(v)
		return nil
	case revision.FieldExecutedAt:
		v, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetExecutedAt(v)
		return nil
	case revision.FieldExecutionTime:
		v, ok := value.(time.Duration)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetExecutionTime(v)
		return nil
	case revision.FieldHash:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetHash(v)
		return nil
	case revision.FieldOperatorVersion:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetOperatorVersion(v)
		return nil
	case revision.FieldMeta:
		v, ok := value.(map[string]string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetMeta(v)
		return nil
	}
	return fmt.Errorf("unknown Revision field %s", name)
}

// AddedFields returns all numeric fields that were incremented/decremented during
// this mutation.
func (m *RevisionMutation) AddedFields() []string {
	var fields []string
	if m.addexecution_time != nil {
		fields = append(fields, revision.FieldExecutionTime)
	}
	return fields
}

// AddedField returns the numeric value that was incremented/decremented on a field
// with the given name. The second boolean return value indicates that this field
// was not set, or was not defined in the schema.
func (m *RevisionMutation) AddedField(name string) (ent.Value, bool) {
	switch name {
	case revision.FieldExecutionTime:
		return m.AddedExecutionTime()
	}
	return nil, false
}

// AddField adds the value to the field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *RevisionMutation) AddField(name string, value ent.Value) error {
	switch name {
	case revision.FieldExecutionTime:
		v, ok := value.(time.Duration)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.AddExecutionTime(v)
		return nil
	}
	return fmt.Errorf("unknown Revision numeric field %s", name)
}

// ClearedFields returns all nullable fields that were cleared during this
// mutation.
func (m *RevisionMutation) ClearedFields() []string {
	return nil
}

// FieldCleared returns a boolean indicating if a field with the given name was
// cleared in this mutation.
func (m *RevisionMutation) FieldCleared(name string) bool {
	_, ok := m.clearedFields[name]
	return ok
}

// ClearField clears the value of the field with the given name. It returns an
// error if the field is not defined in the schema.
func (m *RevisionMutation) ClearField(name string) error {
	return fmt.Errorf("unknown Revision nullable field %s", name)
}

// ResetField resets all changes in the mutation for the field with the given name.
// It returns an error if the field is not defined in the schema.
func (m *RevisionMutation) ResetField(name string) error {
	switch name {
	case revision.FieldDescription:
		m.ResetDescription()
		return nil
	case revision.FieldExecutionState:
		m.ResetExecutionState()
		return nil
	case revision.FieldExecutedAt:
		m.ResetExecutedAt()
		return nil
	case revision.FieldExecutionTime:
		m.ResetExecutionTime()
		return nil
	case revision.FieldHash:
		m.ResetHash()
		return nil
	case revision.FieldOperatorVersion:
		m.ResetOperatorVersion()
		return nil
	case revision.FieldMeta:
		m.ResetMeta()
		return nil
	}
	return fmt.Errorf("unknown Revision field %s", name)
}

// AddedEdges returns all edge names that were set/added in this mutation.
func (m *RevisionMutation) AddedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// AddedIDs returns all IDs (to other nodes) that were added for the given edge
// name in this mutation.
func (m *RevisionMutation) AddedIDs(name string) []ent.Value {
	return nil
}

// RemovedEdges returns all edge names that were removed in this mutation.
func (m *RevisionMutation) RemovedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// RemovedIDs returns all IDs (to other nodes) that were removed for the edge with
// the given name in this mutation.
func (m *RevisionMutation) RemovedIDs(name string) []ent.Value {
	return nil
}

// ClearedEdges returns all edge names that were cleared in this mutation.
func (m *RevisionMutation) ClearedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// EdgeCleared returns a boolean which indicates if the edge with the given name
// was cleared in this mutation.
func (m *RevisionMutation) EdgeCleared(name string) bool {
	return false
}

// ClearEdge clears the value of the edge with the given name. It returns an error
// if that edge is not defined in the schema.
func (m *RevisionMutation) ClearEdge(name string) error {
	return fmt.Errorf("unknown Revision unique edge %s", name)
}

// ResetEdge resets all changes to the edge with the given name in this mutation.
// It returns an error if the edge is not defined in the schema.
func (m *RevisionMutation) ResetEdge(name string) error {
	return fmt.Errorf("unknown Revision edge %s", name)
}
