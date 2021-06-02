// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"time"

	"ariga.io/atlas/integration/entinteg/ent/group"
	"ariga.io/atlas/integration/entinteg/ent/predicate"
	"ariga.io/atlas/integration/entinteg/ent/user"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UserUpdate is the builder for updating User entities.
type UserUpdate struct {
	config
	hooks    []Hook
	mutation *UserMutation
}

// Where adds a new predicate for the UserUpdate builder.
func (uu *UserUpdate) Where(ps ...predicate.User) *UserUpdate {
	uu.mutation.predicates = append(uu.mutation.predicates, ps...)
	return uu
}

// SetName sets the "name" field.
func (uu *UserUpdate) SetName(s string) *UserUpdate {
	uu.mutation.SetName(s)
	return uu
}

// SetOptional sets the "optional" field.
func (uu *UserUpdate) SetOptional(s string) *UserUpdate {
	uu.mutation.SetOptional(s)
	return uu
}

// SetNillableOptional sets the "optional" field if the given value is not nil.
func (uu *UserUpdate) SetNillableOptional(s *string) *UserUpdate {
	if s != nil {
		uu.SetOptional(*s)
	}
	return uu
}

// ClearOptional clears the value of the "optional" field.
func (uu *UserUpdate) ClearOptional() *UserUpdate {
	uu.mutation.ClearOptional()
	return uu
}

// SetInt sets the "int" field.
func (uu *UserUpdate) SetInt(i int) *UserUpdate {
	uu.mutation.ResetInt()
	uu.mutation.SetInt(i)
	return uu
}

// AddInt adds i to the "int" field.
func (uu *UserUpdate) AddInt(i int) *UserUpdate {
	uu.mutation.AddInt(i)
	return uu
}

// SetUint sets the "uint" field.
func (uu *UserUpdate) SetUint(u uint) *UserUpdate {
	uu.mutation.ResetUint()
	uu.mutation.SetUint(u)
	return uu
}

// AddUint adds u to the "uint" field.
func (uu *UserUpdate) AddUint(u uint) *UserUpdate {
	uu.mutation.AddUint(u)
	return uu
}

// SetTime sets the "time" field.
func (uu *UserUpdate) SetTime(t time.Time) *UserUpdate {
	uu.mutation.SetTime(t)
	return uu
}

// SetBool sets the "bool" field.
func (uu *UserUpdate) SetBool(b bool) *UserUpdate {
	uu.mutation.SetBool(b)
	return uu
}

// SetEnum sets the "enum" field.
func (uu *UserUpdate) SetEnum(u user.Enum) *UserUpdate {
	uu.mutation.SetEnum(u)
	return uu
}

// SetEnum2 sets the "enum_2" field.
func (uu *UserUpdate) SetEnum2(u user.Enum2) *UserUpdate {
	uu.mutation.SetEnum2(u)
	return uu
}

// SetUUID sets the "uuid" field.
func (uu *UserUpdate) SetUUID(u uuid.UUID) *UserUpdate {
	uu.mutation.SetUUID(u)
	return uu
}

// SetBytes sets the "bytes" field.
func (uu *UserUpdate) SetBytes(b []byte) *UserUpdate {
	uu.mutation.SetBytes(b)
	return uu
}

// SetGroupID sets the "group_id" field.
func (uu *UserUpdate) SetGroupID(i int) *UserUpdate {
	uu.mutation.ResetGroupID()
	uu.mutation.SetGroupID(i)
	return uu
}

// SetNillableGroupID sets the "group_id" field if the given value is not nil.
func (uu *UserUpdate) SetNillableGroupID(i *int) *UserUpdate {
	if i != nil {
		uu.SetGroupID(*i)
	}
	return uu
}

// ClearGroupID clears the value of the "group_id" field.
func (uu *UserUpdate) ClearGroupID() *UserUpdate {
	uu.mutation.ClearGroupID()
	return uu
}

// SetGroup sets the "group" edge to the Group entity.
func (uu *UserUpdate) SetGroup(g *Group) *UserUpdate {
	return uu.SetGroupID(g.ID)
}

// Mutation returns the UserMutation object of the builder.
func (uu *UserUpdate) Mutation() *UserMutation {
	return uu.mutation
}

// ClearGroup clears the "group" edge to the Group entity.
func (uu *UserUpdate) ClearGroup() *UserUpdate {
	uu.mutation.ClearGroup()
	return uu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (uu *UserUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(uu.hooks) == 0 {
		if err = uu.check(); err != nil {
			return 0, err
		}
		affected, err = uu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*UserMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = uu.check(); err != nil {
				return 0, err
			}
			uu.mutation = mutation
			affected, err = uu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(uu.hooks) - 1; i >= 0; i-- {
			mut = uu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, uu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (uu *UserUpdate) SaveX(ctx context.Context) int {
	affected, err := uu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (uu *UserUpdate) Exec(ctx context.Context) error {
	_, err := uu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (uu *UserUpdate) ExecX(ctx context.Context) {
	if err := uu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (uu *UserUpdate) check() error {
	if v, ok := uu.mutation.Enum(); ok {
		if err := user.EnumValidator(v); err != nil {
			return &ValidationError{Name: "enum", err: fmt.Errorf("ent: validator failed for field \"enum\": %w", err)}
		}
	}
	if v, ok := uu.mutation.Enum2(); ok {
		if err := user.Enum2Validator(v); err != nil {
			return &ValidationError{Name: "enum_2", err: fmt.Errorf("ent: validator failed for field \"enum_2\": %w", err)}
		}
	}
	return nil
}

func (uu *UserUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   user.Table,
			Columns: user.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: user.FieldID,
			},
		},
	}
	if ps := uu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := uu.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: user.FieldName,
		})
	}
	if value, ok := uu.mutation.Optional(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: user.FieldOptional,
		})
	}
	if uu.mutation.OptionalCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: user.FieldOptional,
		})
	}
	if value, ok := uu.mutation.Int(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: user.FieldInt,
		})
	}
	if value, ok := uu.mutation.AddedInt(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: user.FieldInt,
		})
	}
	if value, ok := uu.mutation.Uint(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUint,
			Value:  value,
			Column: user.FieldUint,
		})
	}
	if value, ok := uu.mutation.AddedUint(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeUint,
			Value:  value,
			Column: user.FieldUint,
		})
	}
	if value, ok := uu.mutation.Time(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: user.FieldTime,
		})
	}
	if value, ok := uu.mutation.Bool(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: user.FieldBool,
		})
	}
	if value, ok := uu.mutation.Enum(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: user.FieldEnum,
		})
	}
	if value, ok := uu.mutation.Enum2(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: user.FieldEnum2,
		})
	}
	if value, ok := uu.mutation.UUID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: user.FieldUUID,
		})
	}
	if value, ok := uu.mutation.Bytes(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: user.FieldBytes,
		})
	}
	if uu.mutation.GroupCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.GroupTable,
			Columns: []string{user.GroupColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: group.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := uu.mutation.GroupIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.GroupTable,
			Columns: []string{user.GroupColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: group.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, uu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{user.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return 0, err
	}
	return n, nil
}

// UserUpdateOne is the builder for updating a single User entity.
type UserUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *UserMutation
}

// SetName sets the "name" field.
func (uuo *UserUpdateOne) SetName(s string) *UserUpdateOne {
	uuo.mutation.SetName(s)
	return uuo
}

// SetOptional sets the "optional" field.
func (uuo *UserUpdateOne) SetOptional(s string) *UserUpdateOne {
	uuo.mutation.SetOptional(s)
	return uuo
}

// SetNillableOptional sets the "optional" field if the given value is not nil.
func (uuo *UserUpdateOne) SetNillableOptional(s *string) *UserUpdateOne {
	if s != nil {
		uuo.SetOptional(*s)
	}
	return uuo
}

// ClearOptional clears the value of the "optional" field.
func (uuo *UserUpdateOne) ClearOptional() *UserUpdateOne {
	uuo.mutation.ClearOptional()
	return uuo
}

// SetInt sets the "int" field.
func (uuo *UserUpdateOne) SetInt(i int) *UserUpdateOne {
	uuo.mutation.ResetInt()
	uuo.mutation.SetInt(i)
	return uuo
}

// AddInt adds i to the "int" field.
func (uuo *UserUpdateOne) AddInt(i int) *UserUpdateOne {
	uuo.mutation.AddInt(i)
	return uuo
}

// SetUint sets the "uint" field.
func (uuo *UserUpdateOne) SetUint(u uint) *UserUpdateOne {
	uuo.mutation.ResetUint()
	uuo.mutation.SetUint(u)
	return uuo
}

// AddUint adds u to the "uint" field.
func (uuo *UserUpdateOne) AddUint(u uint) *UserUpdateOne {
	uuo.mutation.AddUint(u)
	return uuo
}

// SetTime sets the "time" field.
func (uuo *UserUpdateOne) SetTime(t time.Time) *UserUpdateOne {
	uuo.mutation.SetTime(t)
	return uuo
}

// SetBool sets the "bool" field.
func (uuo *UserUpdateOne) SetBool(b bool) *UserUpdateOne {
	uuo.mutation.SetBool(b)
	return uuo
}

// SetEnum sets the "enum" field.
func (uuo *UserUpdateOne) SetEnum(u user.Enum) *UserUpdateOne {
	uuo.mutation.SetEnum(u)
	return uuo
}

// SetEnum2 sets the "enum_2" field.
func (uuo *UserUpdateOne) SetEnum2(u user.Enum2) *UserUpdateOne {
	uuo.mutation.SetEnum2(u)
	return uuo
}

// SetUUID sets the "uuid" field.
func (uuo *UserUpdateOne) SetUUID(u uuid.UUID) *UserUpdateOne {
	uuo.mutation.SetUUID(u)
	return uuo
}

// SetBytes sets the "bytes" field.
func (uuo *UserUpdateOne) SetBytes(b []byte) *UserUpdateOne {
	uuo.mutation.SetBytes(b)
	return uuo
}

// SetGroupID sets the "group_id" field.
func (uuo *UserUpdateOne) SetGroupID(i int) *UserUpdateOne {
	uuo.mutation.ResetGroupID()
	uuo.mutation.SetGroupID(i)
	return uuo
}

// SetNillableGroupID sets the "group_id" field if the given value is not nil.
func (uuo *UserUpdateOne) SetNillableGroupID(i *int) *UserUpdateOne {
	if i != nil {
		uuo.SetGroupID(*i)
	}
	return uuo
}

// ClearGroupID clears the value of the "group_id" field.
func (uuo *UserUpdateOne) ClearGroupID() *UserUpdateOne {
	uuo.mutation.ClearGroupID()
	return uuo
}

// SetGroup sets the "group" edge to the Group entity.
func (uuo *UserUpdateOne) SetGroup(g *Group) *UserUpdateOne {
	return uuo.SetGroupID(g.ID)
}

// Mutation returns the UserMutation object of the builder.
func (uuo *UserUpdateOne) Mutation() *UserMutation {
	return uuo.mutation
}

// ClearGroup clears the "group" edge to the Group entity.
func (uuo *UserUpdateOne) ClearGroup() *UserUpdateOne {
	uuo.mutation.ClearGroup()
	return uuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (uuo *UserUpdateOne) Select(field string, fields ...string) *UserUpdateOne {
	uuo.fields = append([]string{field}, fields...)
	return uuo
}

// Save executes the query and returns the updated User entity.
func (uuo *UserUpdateOne) Save(ctx context.Context) (*User, error) {
	var (
		err  error
		node *User
	)
	if len(uuo.hooks) == 0 {
		if err = uuo.check(); err != nil {
			return nil, err
		}
		node, err = uuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*UserMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = uuo.check(); err != nil {
				return nil, err
			}
			uuo.mutation = mutation
			node, err = uuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(uuo.hooks) - 1; i >= 0; i-- {
			mut = uuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, uuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (uuo *UserUpdateOne) SaveX(ctx context.Context) *User {
	node, err := uuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (uuo *UserUpdateOne) Exec(ctx context.Context) error {
	_, err := uuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (uuo *UserUpdateOne) ExecX(ctx context.Context) {
	if err := uuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (uuo *UserUpdateOne) check() error {
	if v, ok := uuo.mutation.Enum(); ok {
		if err := user.EnumValidator(v); err != nil {
			return &ValidationError{Name: "enum", err: fmt.Errorf("ent: validator failed for field \"enum\": %w", err)}
		}
	}
	if v, ok := uuo.mutation.Enum2(); ok {
		if err := user.Enum2Validator(v); err != nil {
			return &ValidationError{Name: "enum_2", err: fmt.Errorf("ent: validator failed for field \"enum_2\": %w", err)}
		}
	}
	return nil
}

func (uuo *UserUpdateOne) sqlSave(ctx context.Context) (_node *User, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   user.Table,
			Columns: user.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: user.FieldID,
			},
		},
	}
	id, ok := uuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing User.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := uuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, user.FieldID)
		for _, f := range fields {
			if !user.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != user.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := uuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := uuo.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: user.FieldName,
		})
	}
	if value, ok := uuo.mutation.Optional(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: user.FieldOptional,
		})
	}
	if uuo.mutation.OptionalCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: user.FieldOptional,
		})
	}
	if value, ok := uuo.mutation.Int(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: user.FieldInt,
		})
	}
	if value, ok := uuo.mutation.AddedInt(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: user.FieldInt,
		})
	}
	if value, ok := uuo.mutation.Uint(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUint,
			Value:  value,
			Column: user.FieldUint,
		})
	}
	if value, ok := uuo.mutation.AddedUint(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeUint,
			Value:  value,
			Column: user.FieldUint,
		})
	}
	if value, ok := uuo.mutation.Time(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: user.FieldTime,
		})
	}
	if value, ok := uuo.mutation.Bool(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: user.FieldBool,
		})
	}
	if value, ok := uuo.mutation.Enum(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: user.FieldEnum,
		})
	}
	if value, ok := uuo.mutation.Enum2(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: user.FieldEnum2,
		})
	}
	if value, ok := uuo.mutation.UUID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: user.FieldUUID,
		})
	}
	if value, ok := uuo.mutation.Bytes(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBytes,
			Value:  value,
			Column: user.FieldBytes,
		})
	}
	if uuo.mutation.GroupCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.GroupTable,
			Columns: []string{user.GroupColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: group.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := uuo.mutation.GroupIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   user.GroupTable,
			Columns: []string{user.GroupColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: group.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &User{config: uuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, uuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{user.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	return _node, nil
}
