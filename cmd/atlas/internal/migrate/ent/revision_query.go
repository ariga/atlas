// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/internal"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/predicate"
	"ariga.io/atlas/cmd/atlas/internal/migrate/ent/revision"
	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// RevisionQuery is the builder for querying Revision entities.
type RevisionQuery struct {
	config
	ctx        *QueryContext
	order      []revision.OrderOption
	inters     []Interceptor
	predicates []predicate.Revision
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the RevisionQuery builder.
func (_q *RevisionQuery) Where(ps ...predicate.Revision) *RevisionQuery {
	_q.predicates = append(_q.predicates, ps...)
	return _q
}

// Limit the number of records to be returned by this query.
func (_q *RevisionQuery) Limit(limit int) *RevisionQuery {
	_q.ctx.Limit = &limit
	return _q
}

// Offset to start from.
func (_q *RevisionQuery) Offset(offset int) *RevisionQuery {
	_q.ctx.Offset = &offset
	return _q
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (_q *RevisionQuery) Unique(unique bool) *RevisionQuery {
	_q.ctx.Unique = &unique
	return _q
}

// Order specifies how the records should be ordered.
func (_q *RevisionQuery) Order(o ...revision.OrderOption) *RevisionQuery {
	_q.order = append(_q.order, o...)
	return _q
}

// First returns the first Revision entity from the query.
// Returns a *NotFoundError when no Revision was found.
func (_q *RevisionQuery) First(ctx context.Context) (*Revision, error) {
	nodes, err := _q.Limit(1).All(setContextOp(ctx, _q.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{revision.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (_q *RevisionQuery) FirstX(ctx context.Context) *Revision {
	node, err := _q.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Revision ID from the query.
// Returns a *NotFoundError when no Revision ID was found.
func (_q *RevisionQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(1).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{revision.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (_q *RevisionQuery) FirstIDX(ctx context.Context) string {
	id, err := _q.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Revision entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Revision entity is found.
// Returns a *NotFoundError when no Revision entities are found.
func (_q *RevisionQuery) Only(ctx context.Context) (*Revision, error) {
	nodes, err := _q.Limit(2).All(setContextOp(ctx, _q.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{revision.Label}
	default:
		return nil, &NotSingularError{revision.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (_q *RevisionQuery) OnlyX(ctx context.Context) *Revision {
	node, err := _q.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Revision ID in the query.
// Returns a *NotSingularError when more than one Revision ID is found.
// Returns a *NotFoundError when no entities are found.
func (_q *RevisionQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(2).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{revision.Label}
	default:
		err = &NotSingularError{revision.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (_q *RevisionQuery) OnlyIDX(ctx context.Context) string {
	id, err := _q.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Revisions.
func (_q *RevisionQuery) All(ctx context.Context) ([]*Revision, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryAll)
	if err := _q.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Revision, *RevisionQuery]()
	return withInterceptors[[]*Revision](ctx, _q, qr, _q.inters)
}

// AllX is like All, but panics if an error occurs.
func (_q *RevisionQuery) AllX(ctx context.Context) []*Revision {
	nodes, err := _q.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Revision IDs.
func (_q *RevisionQuery) IDs(ctx context.Context) (ids []string, err error) {
	if _q.ctx.Unique == nil && _q.path != nil {
		_q.Unique(true)
	}
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryIDs)
	if err = _q.Select(revision.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (_q *RevisionQuery) IDsX(ctx context.Context) []string {
	ids, err := _q.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (_q *RevisionQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryCount)
	if err := _q.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, _q, querierCount[*RevisionQuery](), _q.inters)
}

// CountX is like Count, but panics if an error occurs.
func (_q *RevisionQuery) CountX(ctx context.Context) int {
	count, err := _q.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (_q *RevisionQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryExist)
	switch _, err := _q.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (_q *RevisionQuery) ExistX(ctx context.Context) bool {
	exist, err := _q.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the RevisionQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (_q *RevisionQuery) Clone() *RevisionQuery {
	if _q == nil {
		return nil
	}
	return &RevisionQuery{
		config:     _q.config,
		ctx:        _q.ctx.Clone(),
		order:      append([]revision.OrderOption{}, _q.order...),
		inters:     append([]Interceptor{}, _q.inters...),
		predicates: append([]predicate.Revision{}, _q.predicates...),
		// clone intermediate query.
		sql:  _q.sql.Clone(),
		path: _q.path,
	}
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Description string `json:"description,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Revision.Query().
//		GroupBy(revision.FieldDescription).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (_q *RevisionQuery) GroupBy(field string, fields ...string) *RevisionGroupBy {
	_q.ctx.Fields = append([]string{field}, fields...)
	grbuild := &RevisionGroupBy{build: _q}
	grbuild.flds = &_q.ctx.Fields
	grbuild.label = revision.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Description string `json:"description,omitempty"`
//	}
//
//	client.Revision.Query().
//		Select(revision.FieldDescription).
//		Scan(ctx, &v)
func (_q *RevisionQuery) Select(fields ...string) *RevisionSelect {
	_q.ctx.Fields = append(_q.ctx.Fields, fields...)
	sbuild := &RevisionSelect{RevisionQuery: _q}
	sbuild.label = revision.Label
	sbuild.flds, sbuild.scan = &_q.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a RevisionSelect configured with the given aggregations.
func (_q *RevisionQuery) Aggregate(fns ...AggregateFunc) *RevisionSelect {
	return _q.Select().Aggregate(fns...)
}

func (_q *RevisionQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range _q.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, _q); err != nil {
				return err
			}
		}
	}
	for _, f := range _q.ctx.Fields {
		if !revision.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if _q.path != nil {
		prev, err := _q.path(ctx)
		if err != nil {
			return err
		}
		_q.sql = prev
	}
	return nil
}

func (_q *RevisionQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Revision, error) {
	var (
		nodes = []*Revision{}
		_spec = _q.querySpec()
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Revision).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Revision{config: _q.config}
		nodes = append(nodes, node)
		return node.assignValues(columns, values)
	}
	_spec.Node.Schema = _q.schemaConfig.Revision
	ctx = internal.NewSchemaConfigContext(ctx, _q.schemaConfig)
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, _q.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	return nodes, nil
}

func (_q *RevisionQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := _q.querySpec()
	_spec.Node.Schema = _q.schemaConfig.Revision
	ctx = internal.NewSchemaConfigContext(ctx, _q.schemaConfig)
	_spec.Node.Columns = _q.ctx.Fields
	if len(_q.ctx.Fields) > 0 {
		_spec.Unique = _q.ctx.Unique != nil && *_q.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, _q.driver, _spec)
}

func (_q *RevisionQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(revision.Table, revision.Columns, sqlgraph.NewFieldSpec(revision.FieldID, field.TypeString))
	_spec.From = _q.sql
	if unique := _q.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if _q.path != nil {
		_spec.Unique = true
	}
	if fields := _q.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, revision.FieldID)
		for i := range fields {
			if fields[i] != revision.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := _q.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := _q.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := _q.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := _q.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (_q *RevisionQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(_q.driver.Dialect())
	t1 := builder.Table(revision.Table)
	columns := _q.ctx.Fields
	if len(columns) == 0 {
		columns = revision.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if _q.sql != nil {
		selector = _q.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if _q.ctx.Unique != nil && *_q.ctx.Unique {
		selector.Distinct()
	}
	t1.Schema(_q.schemaConfig.Revision)
	ctx = internal.NewSchemaConfigContext(ctx, _q.schemaConfig)
	selector.WithContext(ctx)
	for _, p := range _q.predicates {
		p(selector)
	}
	for _, p := range _q.order {
		p(selector)
	}
	if offset := _q.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := _q.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// RevisionGroupBy is the group-by builder for Revision entities.
type RevisionGroupBy struct {
	selector
	build *RevisionQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (_g *RevisionGroupBy) Aggregate(fns ...AggregateFunc) *RevisionGroupBy {
	_g.fns = append(_g.fns, fns...)
	return _g
}

// Scan applies the selector query and scans the result into the given value.
func (_g *RevisionGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, _g.build.ctx, ent.OpQueryGroupBy)
	if err := _g.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RevisionQuery, *RevisionGroupBy](ctx, _g.build, _g, _g.build.inters, v)
}

func (_g *RevisionGroupBy) sqlScan(ctx context.Context, root *RevisionQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(_g.fns))
	for _, fn := range _g.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*_g.flds)+len(_g.fns))
		for _, f := range *_g.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*_g.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := _g.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// RevisionSelect is the builder for selecting fields of Revision entities.
type RevisionSelect struct {
	*RevisionQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (_s *RevisionSelect) Aggregate(fns ...AggregateFunc) *RevisionSelect {
	_s.fns = append(_s.fns, fns...)
	return _s
}

// Scan applies the selector query and scans the result into the given value.
func (_s *RevisionSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, _s.ctx, ent.OpQuerySelect)
	if err := _s.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RevisionQuery, *RevisionSelect](ctx, _s.RevisionQuery, _s, _s.inters, v)
}

func (_s *RevisionSelect) sqlScan(ctx context.Context, root *RevisionQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(_s.fns))
	for _, fn := range _s.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*_s.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := _s.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
