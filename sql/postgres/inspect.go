// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/postgres/internal/postgresop"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a PostgreSQL implementation for schema.Inspector.
type inspect struct{ *conn }

var _ schema.Inspector = (*inspect)(nil)

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		opts = &schema.InspectRealmOption{}
	}
	var (
		r    = schema.NewRealm(schemas...)
		mode = sqlx.ModeInspectRealm(opts)
	)
	if len(schemas) > 0 {
		if mode.Is(schema.InspectTables) {
			if err := i.inspectTables(ctx, r, nil); err != nil {
				return nil, err
			}
			sqlx.LinkSchemaTables(schemas)
		}
		if mode.Is(schema.InspectViews) {
			if err := i.inspectViews(ctx, r, nil); err != nil {
				return nil, err
			}
		}
		if mode.Is(schema.InspectFuncs) {
			if err := i.inspectFuncs(ctx, r, nil); err != nil {
				return nil, err
			}
		}
		if err := i.inspectEnums(ctx, r); err != nil {
			return nil, err
		}
		if mode.Is(schema.InspectTypes) {
			if err := i.inspectTypes(ctx, r, nil); err != nil {
				return nil, err
			}
		}
		if mode.Is(schema.InspectObjects) {
			if err := i.inspectSequences(ctx, r, nil); err != nil {
				return nil, err
			}
		}
		if mode.Is(schema.InspectTriggers) {
			if err := i.inspectTriggers(ctx, r, nil); err != nil {
				return nil, err
			}
		}
		if err := i.inspectDeps(ctx, r, nil); err != nil {
			return nil, err
		}
	}
	return schema.ExcludeRealm(r, opts.Exclude)
}

// InspectSchema returns schema descriptions of the tables in the given schema.
// If the schema name is empty, the result will be the attached schema.
func (i *inspect) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (s *schema.Schema, err error) {
	schemas, err := i.schemas(ctx, &schema.InspectRealmOption{Schemas: []string{name}})
	if err != nil {
		return nil, err
	}
	switch n := len(schemas); {
	case n == 0:
		// Empty string indicates current connected schema.
		if name == "" {
			return nil, &schema.NotExistError{Err: errors.New("postgres: current_schema() defined in search_path was not found")}
		}
		return nil, &schema.NotExistError{Err: fmt.Errorf("postgres: schema %q was not found", name)}
	case n > 1:
		return nil, fmt.Errorf("postgres: %d schemas were found for %q", n, name)
	}
	if opts == nil {
		opts = &schema.InspectOptions{}
	}
	var (
		r    = schema.NewRealm(schemas...)
		mode = sqlx.ModeInspectSchema(opts)
	)
	if mode.Is(schema.InspectTables) {
		if err := i.inspectTables(ctx, r, opts); err != nil {
			return nil, err
		}
		sqlx.LinkSchemaTables(schemas)
	}
	if mode.Is(schema.InspectViews) {
		if err := i.inspectViews(ctx, r, opts); err != nil {
			return nil, err
		}
	}
	if mode.Is(schema.InspectFuncs) {
		if err := i.inspectFuncs(ctx, r, opts); err != nil {
			return nil, err
		}
	}
	if err := i.inspectEnums(ctx, r); err != nil {
		return nil, err
	}
	if mode.Is(schema.InspectTypes) {
		if err := i.inspectTypes(ctx, r, opts); err != nil {
			return nil, err
		}
	}
	if mode.Is(schema.InspectObjects) {
		if err := i.inspectSequences(ctx, r, opts); err != nil {
			return nil, err
		}
	}
	if mode.Is(schema.InspectTriggers) {
		if err := i.inspectTriggers(ctx, r, nil); err != nil {
			return nil, err
		}
	}
	if err := i.inspectDeps(ctx, r, opts); err != nil {
		return nil, err
	}
	return schema.ExcludeSchema(r.Schemas[0], opts.Exclude)
}

func (i *inspect) inspectTables(ctx context.Context, r *schema.Realm, opts *schema.InspectOptions) error {
	if err := i.tables(ctx, r, opts); err != nil {
		return err
	}
	for _, s := range r.Schemas {
		if len(s.Tables) == 0 {
			continue
		}
		if err := i.columns(ctx, s); err != nil {
			return err
		}
		if err := i.indexes(ctx, s); err != nil {
			return err
		}
		if err := i.partitions(s); err != nil {
			return err
		}
		if err := i.fks(ctx, s); err != nil {
			return err
		}
		if err := i.checks(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (i *inspect) tables(ctx context.Context, realm *schema.Realm, opts *schema.InspectOptions) error {
	var (
		args  []any
		query = fmt.Sprintf(tablesQuery, nArgs(0, len(realm.Schemas)))
	)
	for _, s := range realm.Schemas {
		args = append(args, s.Name)
	}
	if opts != nil && len(opts.Tables) > 0 {
		for _, t := range opts.Tables {
			args = append(args, t)
		}
		query = fmt.Sprintf(tablesQueryArgs, nArgs(0, len(realm.Schemas)), nArgs(len(realm.Schemas), len(opts.Tables)))
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			oid                                                     sql.NullInt64
			tSchema, name, comment, partattrs, partstart, partexprs sql.NullString
		)
		if err := rows.Scan(&oid, &tSchema, &name, &comment, &partattrs, &partstart, &partexprs); err != nil {
			return fmt.Errorf("scan table information: %w", err)
		}
		if !sqlx.ValidString(tSchema) || !sqlx.ValidString(name) {
			return fmt.Errorf("invalid schema or table name: %q.%q", tSchema.String, name.String)
		}
		s, ok := realm.Schema(tSchema.String)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", tSchema.String)
		}
		t := schema.NewTable(name.String)
		s.AddTables(t)
		if oid.Valid {
			t.AddAttrs(&OID{V: oid.Int64})
		}
		if sqlx.ValidString(comment) {
			t.SetComment(comment.String)
		}
		if sqlx.ValidString(partattrs) {
			t.AddAttrs(&Partition{
				start: partstart.String,
				attrs: partattrs.String,
				exprs: partexprs.String,
			})
		}
	}
	return rows.Close()
}

// columns queries and appends the columns of the given table.
func (i *inspect) columns(ctx context.Context, s *schema.Schema) error {
	query := columnsQuery
	if i.crdb {
		query = crdbColumnsQuery
	}
	rows, err := i.querySchema(ctx, query, s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q columns: %w", s.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := i.addColumn(s, rows); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}
	return rows.Close()
}

// addColumn scans the current row and adds a new column from it to the scope (table or view).
func (i *inspect) addColumn(s *schema.Schema, rows *sql.Rows) (err error) {
	var (
		typid, typelem, maxlen, precision, timeprecision, scale, seqstart, seqinc, seqlast                                                  sql.NullInt64
		table, name, typ, fmtype, nullable, defaults, identity, genidentity, genexpr, charset, collate, comment, typtype, elemtyp, interval sql.NullString
	)
	if err = rows.Scan(
		&table, &name, &typ, &fmtype, &nullable, &defaults, &maxlen, &precision, &timeprecision, &scale, &interval, &charset,
		&collate, &identity, &seqstart, &seqinc, &seqlast, &genidentity, &genexpr, &comment, &typtype, &typelem, &elemtyp, &typid,
	); err != nil {
		return err
	}
	t, ok := s.Table(table.String)
	if !ok {
		return fmt.Errorf("table %q was not found in schema", table.String)
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Null: nullable.String == "YES",
			Raw: func() string {
				// For domains, use the domain type instead of the base type.
				if typtype.String == "d" {
					return fmtype.String
				}
				return typ.String
			}(),
		},
	}
	c.Type.Type, err = columnType(&columnDesc{
		typ:           typ.String,
		fmtype:        fmtype.String,
		size:          maxlen.Int64,
		scale:         scale.Int64,
		typtype:       typtype.String,
		typelem:       typelem.Int64,
		elemtyp:       elemtyp.String,
		typid:         typid.Int64,
		interval:      interval.String,
		precision:     precision.Int64,
		timePrecision: &timeprecision.Int64,
	})
	if defaults.Valid {
		columnDefault(c, defaults.String)
	}
	if identity.String == "YES" {
		c.Attrs = append(c.Attrs, &Identity{
			Generation: genidentity.String,
			Sequence: &Sequence{
				Last:      seqlast.Int64,
				Start:     seqstart.Int64,
				Increment: seqinc.Int64,
			},
		})
	}
	if sqlx.ValidString(genexpr) {
		c.Attrs = append(c.Attrs, &schema.GeneratedExpr{
			Expr: genexpr.String,
		})
	}
	if sqlx.ValidString(comment) {
		c.SetComment(comment.String)
	}
	if sqlx.ValidString(charset) {
		c.SetCharset(charset.String)
	}
	if sqlx.ValidString(collate) {
		c.SetCollation(collate.String)
	}
	t.AddColumns(c)
	return nil
}

// enumValues fills enum columns with their values from the database.
func (i *inspect) inspectEnums(ctx context.Context, r *schema.Realm) error {
	var (
		ids  = make(map[int64]*schema.EnumType)
		args = make([]any, 0, len(r.Schemas))
		newE = func(e1 *enumType) *schema.EnumType {
			e2, ok := ids[e1.ID]
			if ok {
				return e2
			}
			e2 = &schema.EnumType{T: e1.T}
			ids[e1.ID] = e2
			return e2
		}
		scanC = func(cs []*schema.Column) {
			for _, c := range cs {
				switch t := c.Type.Type.(type) {
				case *enumType:
					e := newE(t)
					c.Type.Type = e
					c.Type.Raw = e.T
				case *ArrayType:
					if e, ok := t.Type.(*enumType); ok {
						t.Type = newE(e)
					}
				}
			}
		}
	)
	for _, s := range r.Schemas {
		args = append(args, s.Name)
		for _, t := range s.Tables {
			scanC(t.Columns)
		}
		for _, v := range s.Views {
			scanC(v.Columns)
		}
	}
	if len(args) == 0 {
		return nil
	}
	rows, err := i.QueryContext(ctx, fmt.Sprintf(enumsQuery, nArgs(0, len(args))), args...)
	if err != nil {
		return fmt.Errorf("postgres: querying enum values: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id       int64
			ns, n, v string
		)
		if err := rows.Scan(&ns, &id, &n, &v); err != nil {
			return fmt.Errorf("postgres: scanning enum label: %w", err)
		}
		e, ok := ids[id]
		if !ok {
			e = &schema.EnumType{T: n}
			ids[id] = e
		}
		if e.Schema == nil {
			s, ok := r.Schema(ns)
			if !ok {
				return fmt.Errorf("postgres: schema %q for enum %q was not found in inspection", ns, e.T)
			}
			e.Schema = s
			s.Objects = append(s.Objects, e)
		}
		e.Values = append(e.Values, v)
	}
	return nil
}

// indexes queries and appends the indexes of the given table.
func (i *inspect) indexes(ctx context.Context, s *schema.Schema) error {
	if i.crdb {
		return i.crdbIndexes(ctx, s)
	}
	rows, err := i.querySchema(ctx, i.indexesQuery(), s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q indexes: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.addIndexes(s, rows, queryScope{
		hasT: func(tv string) bool {
			_, ok := s.Table(tv)
			return ok
		},
		setPK: func(tv string, idx *schema.Index) error {
			if t, ok := s.Table(tv); ok {
				t.SetPrimaryKey(idx)
				return nil
			}
			return fmt.Errorf("postgres: table %q for primary key was not found in schema", tv)
		},
		addIndex: func(tv string, idx *schema.Index) error {
			if t, ok := s.Table(tv); ok {
				t.AddIndexes(idx)
				return nil
			}
			return fmt.Errorf("postgres: table %q for index was not found in schema", tv)
		},
		column: func(tv, name string) (*schema.Column, bool) {
			if t, ok := s.Table(tv); ok {
				return t.Column(name)
			}
			return nil, false
		},
	}); err != nil {
		return err
	}
	return rows.Err()
}

func (i *inspect) indexesQuery() (q string) {
	switch {
	case i.supportsIndexNullsDistinct():
		q = indexesAbove15
	case i.supportsIndexInclude():
		q = indexesAbove11
	default:
		q = indexesBelow11
	}
	return
}

type queryScope struct {
	hasT     func(tv string) bool
	setPK    func(tv string, idx *schema.Index) error
	addIndex func(tv string, idx *schema.Index) error
	column   func(tv, name string) (*schema.Column, bool)
}

// addIndexes scans the rows and adds the indexes to the table.
func (i *inspect) addIndexes(s *schema.Schema, rows *sql.Rows, scope queryScope) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			table, name, typ                                                      string
			uniq, primary, included, nullsnotdistinct                             bool
			desc, nullsfirst, nullslast, opcdefault                               sql.NullBool
			column, constraints, pred, expr, comment, options, opcname, opcparams sql.NullString
		)
		if err := rows.Scan(
			&table, &name, &typ, &column, &included, &primary, &uniq, &constraints, &pred, &expr, &desc,
			&nullsfirst, &nullslast, &comment, &options, &opcname, &opcdefault, &opcparams, &nullsnotdistinct,
		); err != nil {
			return fmt.Errorf("postgres: scanning indexes for schema %q: %w", s.Name, err)
		}
		if !scope.hasT(table) {
			return fmt.Errorf("table %q was not found in schema", table)
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: uniq,
				Attrs: []schema.Attr{
					&IndexType{T: typ},
				},
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(idx.Attrs, &schema.Comment{Text: comment.String})
			}
			if sqlx.ValidString(constraints) {
				var m map[string]string
				if err := json.Unmarshal([]byte(constraints.String), &m); err != nil {
					return fmt.Errorf("postgres: unmarshaling index constraints: %w", err)
				}
				for n, t := range m {
					idx.AddAttrs(&Constraint{N: n, T: t})
				}
			}
			if sqlx.ValidString(pred) {
				idx.AddAttrs(&IndexPredicate{P: pred.String})
			}
			if sqlx.ValidString(options) {
				p, err := newIndexStorage(options.String)
				if err != nil {
					return err
				}
				idx.AddAttrs(p)
			}
			if nullsnotdistinct {
				idx.AddAttrs(&IndexNullsDistinct{V: false})
			}
			names[name] = idx
			var err error
			if primary {
				err = scope.setPK(table, idx)
			} else {
				err = scope.addIndex(table, idx)
			}
			if err != nil {
				return err
			}
		}
		part := &schema.IndexPart{SeqNo: len(idx.Parts) + 1, Desc: desc.Bool}
		if nullsfirst.Bool || nullslast.Bool {
			part.Attrs = append(part.Attrs, &IndexColumnProperty{
				NullsFirst: nullsfirst.Bool,
				NullsLast:  nullslast.Bool,
			})
		}
		switch {
		case included:
			c, ok := scope.column(table, column.String)
			if !ok {
				return fmt.Errorf("postgres: INCLUDE column %q was not found for index %q", column.String, idx.Name)
			}
			var include IndexInclude
			sqlx.Has(idx.Attrs, &include)
			include.Columns = append(include.Columns, c)
			schema.ReplaceOrAppend(&idx.Attrs, &include)
		case sqlx.ValidString(column):
			part.C, ok = scope.column(table, column.String)
			if !ok {
				return fmt.Errorf("postgres: column %q was not found for index %q", column.String, idx.Name)
			}
			part.C.Indexes = append(part.C.Indexes, idx)
			idx.Parts = append(idx.Parts, part)
		case sqlx.ValidString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
			idx.Parts = append(idx.Parts, part)
		default:
			return fmt.Errorf("postgres: invalid part for index %q", idx.Name)
		}
		if err := mayAppendOps(part, opcname.String, opcparams.String, opcdefault.Bool); err != nil {
			return err
		}
	}
	return nil
}

// mayAppendOps appends an operator_class attribute to the part in case it is not the default.
func mayAppendOps(part *schema.IndexPart, name string, params string, defaults bool) error {
	if name == "" || defaults && params == "" {
		return nil
	}
	op := &IndexOpClass{Name: name, Default: defaults}
	if err := op.parseParams(params); err != nil {
		return err
	}
	part.Attrs = append(part.Attrs, op)
	return nil
}

// partitions builds the partition each table in the schema.
func (i *inspect) partitions(s *schema.Schema) error {
	for _, t := range s.Tables {
		var d Partition
		if !sqlx.Has(t.Attrs, &d) {
			continue
		}
		switch s := strings.ToLower(d.start); s {
		case "r":
			d.T = PartitionTypeRange
		case "l":
			d.T = PartitionTypeList
		case "h":
			d.T = PartitionTypeHash
		default:
			return fmt.Errorf("postgres: unexpected partition strategy %q", s)
		}
		idxs := strings.Split(strings.TrimSpace(d.attrs), " ")
		if len(idxs) == 0 {
			return fmt.Errorf("postgres: no columns/expressions were found in partition key for column %q", t.Name)
		}
		for i := range idxs {
			switch idx, err := strconv.Atoi(idxs[i]); {
			case err != nil:
				return fmt.Errorf("postgres: faild parsing partition key index %q", idxs[i])
			// An expression.
			case idx == 0:
				j := sqlx.ExprLastIndex(d.exprs)
				if j == -1 {
					return fmt.Errorf("postgres: no expression found in partition key: %q", d.exprs)
				}
				d.Parts = append(d.Parts, &PartitionPart{
					X: &schema.RawExpr{X: d.exprs[:j+1]},
				})
				d.exprs = strings.TrimPrefix(d.exprs[j+1:], ", ")
			// A column at index idx-1.
			default:
				if idx > len(t.Columns) {
					return fmt.Errorf("postgres: unexpected column index %d", idx)
				}
				d.Parts = append(d.Parts, &PartitionPart{
					C: t.Columns[idx-1],
				})
			}
		}
		schema.ReplaceOrAppend(&t.Attrs, &d)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (i *inspect) fks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, fksQuery, s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q foreign keys: %w", s.Name, err)
	}
	defer rows.Close()
	if err := sqlx.TypedSchemaFKs[*ReferenceOption](s, rows); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return rows.Err()
}

// checks queries and appends the check constraints of the given table.
func (i *inspect) checks(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, checksQuery, s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q check constraints: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.addChecks(s, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addChecks scans the rows and adds the checks to the table.
func (i *inspect) addChecks(s *schema.Schema, rows *sql.Rows) error {
	names := make(map[string]*schema.Check)
	for rows.Next() {
		var (
			noInherit                            bool
			table, name, column, clause, indexes string
		)
		if err := rows.Scan(&table, &name, &clause, &column, &indexes, &noInherit); err != nil {
			return fmt.Errorf("postgres: scanning check: %w", err)
		}
		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", table)
		}
		if _, ok := t.Column(column); !ok {
			return fmt.Errorf("postgres: column %q was not found for check %q", column, name)
		}
		check, ok := names[name]
		if !ok {
			check = &schema.Check{Name: name, Expr: clause, Attrs: []schema.Attr{&CheckColumns{}}}
			if noInherit {
				check.Attrs = append(check.Attrs, &NoInherit{})
			}
			names[name] = check
			t.Attrs = append(t.Attrs, check)
		}
		c := check.Attrs[0].(*CheckColumns)
		c.Columns = append(c.Columns, column)
	}
	return nil
}

// schemas returns the list of the schemas in the database.
func (i *inspect) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []any
		query = schemasQuery
	)
	if opts != nil {
		switch n := len(opts.Schemas); {
		case n == 1 && opts.Schemas[0] == "":
			query = fmt.Sprintf(schemasQueryArgs, "= CURRENT_SCHEMA()")
		case n == 1 && opts.Schemas[0] != "":
			query = fmt.Sprintf(schemasQueryArgs, "= $1")
			args = append(args, opts.Schemas[0])
		case n > 0:
			query = fmt.Sprintf(schemasQueryArgs, "IN ("+nArgs(0, len(opts.Schemas))+")")
			for _, s := range opts.Schemas {
				args = append(args, s)
			}
		}
	}
	rows, err := i.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: querying schemas: %w", err)
	}
	defer rows.Close()
	var schemas []*schema.Schema
	for rows.Next() {
		var (
			name    string
			comment sql.NullString
		)
		if err := rows.Scan(&name, &comment); err != nil {
			return nil, err
		}
		s := schema.New(name)
		if comment.Valid {
			s.SetComment(comment.String)
		}
		schemas = append(schemas, s)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return schemas, nil
}

func (i *inspect) querySchema(ctx context.Context, query string, s *schema.Schema) (*sql.Rows, error) {
	args := []any{s.Name}
	for _, t := range s.Tables {
		args = append(args, t.Name)
	}
	return i.QueryContext(ctx, fmt.Sprintf(query, nArgs(1, len(s.Tables))), args...)
}

func nArgs(start, n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(start + i))
	}
	return b.String()
}

// A regexp to extracts the sequence name from a "nextval" expression.
// nextval('<optional (quoted) schema>.<sequence name>'::regclass).
var reNextval = regexp.MustCompile(`(?i) *nextval\('(?:"?[\w$]+"?\.)?"?([\w$]+_[\w$]+_seq)"?'(?:::regclass)*\) *$`)

func columnDefault(c *schema.Column, s string) {
	switch m := reNextval.FindStringSubmatch(s); {
	// The definition of "<column> <serial type>" is equivalent to specifying:
	// "<column> <int type> NOT NULL DEFAULT nextval('<table>_<column>_seq')".
	// https://postgresql.org/docs/current/datatype-numeric.html#DATATYPE-SERIAL.
	case len(m) == 2:
		tt, ok := c.Type.Type.(*schema.IntegerType)
		if !ok {
			return
		}
		st := &SerialType{SequenceName: m[1]}
		st.SetType(tt)
		c.Type.Raw = st.T
		c.Type.Type = st
	default:
		c.Default = defaultExpr(c.Type.Type, s)
	}
}

func defaultExpr(t schema.Type, s string) schema.Expr {
	switch {
	case sqlx.IsLiteralBool(s), sqlx.IsLiteralNumber(s), sqlx.IsQuoted(s, '\''):
		return &schema.Literal{V: s}
	default:
		var x schema.Expr = &schema.RawExpr{X: s}
		// Try casting or fallback to raw expressions (e.g. column text[] has the default of '{}':text[]).
		if v, ok := canConvert(t, s); ok {
			x = &schema.Literal{V: v}
		}
		return x
	}
}

func canConvert(t schema.Type, x string) (string, bool) {
	i := strings.LastIndex(x, "::")
	if i == -1 || !sqlx.IsQuoted(x[:i], '\'') {
		return "", false
	}
	q := x[0:i]
	x = x[1 : i-1]
	switch t.(type) {
	case *enumType:
		return q, true
	case *schema.BoolType:
		if sqlx.IsLiteralBool(x) {
			return x, true
		}
	case *schema.DecimalType, *schema.IntegerType, *schema.FloatType:
		if sqlx.IsLiteralNumber(x) {
			return x, true
		}
	case *ArrayType, *schema.BinaryType, *schema.JSONType, *NetworkType, *schema.SpatialType, *schema.StringType, *schema.TimeType, *schema.UUIDType, *XMLType:
		return q, true
	}
	return "", false
}

type (
	// UserDefinedType defines a user-defined type attribute.
	UserDefinedType struct {
		schema.Type
		T string
	}

	// PseudoType defines a non-column pseudo-type, such as function arguments and return types.
	// https://www.postgresql.org/docs/current/datatype-pseudo.html
	PseudoType struct {
		schema.Type
		T string // e.g., void, any, cstring, etc.
	}

	// OID is the object identifier as defined in the Postgres catalog.
	OID struct {
		schema.Attr
		V int64
	}

	// enumType represents an enum type. It serves aa intermediate representation of a Postgres enum type,
	// to temporary save TypeID and TypeName of an enum column until the enum values can be extracted.
	enumType struct {
		schema.Type
		T      string // Type name.
		Schema string // Optional schema name.
		ID     int64  // Type id.
		Values []string
	}

	// ArrayType defines an array type.
	// https://postgresql.org/docs/current/arrays.html
	ArrayType struct {
		schema.Type        // Underlying items type (e.g. varchar(255)).
		T           string // Formatted type (e.g. int[]).
	}

	// BitType defines a bit type.
	// https://postgresql.org/docs/current/datatype-bit.html
	BitType struct {
		schema.Type
		T   string
		Len int64
	}

	// DomainType represents a domain type.
	// https://www.postgresql.org/docs/current/domains.html
	DomainType struct {
		schema.Type
		schema.Object
		T       string          // Type name.
		Schema  *schema.Schema  // Optional schema.
		Null    bool            // Nullability.
		Default schema.Expr     // Default value.
		Checks  []*schema.Check // Check constraints.
		Attrs   []schema.Attr   // Extra attributes, such as OID.
		Deps    []schema.Object // Objects this domain depends on.
	}

	// IntervalType defines an interval type.
	// https://postgresql.org/docs/current/datatype-datetime.html
	IntervalType struct {
		schema.Type
		T         string // Type name.
		F         string // Optional field. YEAR, MONTH, ..., MINUTE TO SECOND.
		Precision *int   // Optional precision.
	}

	// A NetworkType defines a network type.
	// https://postgresql.org/docs/current/datatype-net-types.html
	NetworkType struct {
		schema.Type
		T   string
		Len int64
	}

	// A CurrencyType defines a currency type.
	CurrencyType struct {
		schema.Type
		T string
	}

	// A RangeType defines a range type.
	// https://www.postgresql.org/docs/current/rangetypes.html
	RangeType struct {
		schema.Type
		T string
	}

	// A SerialType defines a serial type.
	// https://postgresql.org/docs/current/datatype-numeric.html#DATATYPE-SERIAL
	SerialType struct {
		schema.Type
		T         string
		Precision int
		// SequenceName holds the inspected sequence name attached to the column.
		// It defaults to <Table>_<Column>_seq when the column is created, but may
		// be different in case the table or the column was renamed.
		SequenceName string
	}

	// A TextSearchType defines full text search types.
	// https://www.postgresql.org/docs/current/datatype-textsearch.html
	TextSearchType struct {
		schema.Type
		T string
	}

	// UUIDType is alias to schema.UUIDType.
	// Defined here for backward compatibility reasons.
	UUIDType = schema.UUIDType

	// OIDType defines an object identifier type.
	OIDType struct {
		schema.Type
		T string
	}

	// A XMLType defines an XML type.
	XMLType struct {
		schema.Type
		T string
	}

	// Constraint describes a postgres constraint.
	// https://postgresql.org/docs/current/catalog-pg-constraint.html
	Constraint struct {
		schema.Attr
		N string // constraint name
		T string // c, f, p, u, t, x.
	}

	// Sequence defines (the supported) sequence options.
	// https://postgresql.org/docs/current/sql-createsequence.html
	Sequence struct {
		schema.Object
		// Fields used by the Identity schema attribute.
		Start     int64
		Increment int64
		// Last sequence value written to disk.
		// https://postgresql.org/docs/current/view-pg-sequences.html.
		Last int64

		// Field used when defining and managing independent
		// sequences (not part of IDENTITY or serial columns).
		Name     string         // Sequence name.
		Schema   *schema.Schema // Optional schema.
		Type     schema.Type    // Sequence type.
		Cache    int64          // Cache size.
		Min, Max *int64         // Min and max values.
		Cycle    bool           // Whether the sequence cycles.
		Attrs    []schema.Attr  // Additional attributes (e.g., comments),
		Owner    struct {       // Optional owner of the sequence.
			T *schema.Table
			C *schema.Column
		}
	}

	// Identity defines an identity column.
	Identity struct {
		schema.Attr
		Generation string // ALWAYS, BY DEFAULT.
		Sequence   *Sequence
	}

	// IndexType represents an index type.
	// https://postgresql.org/docs/current/indexes-types.html
	IndexType struct {
		schema.Attr
		T string // BTREE, BRIN, HASH, GiST, SP-GiST, GIN.
	}

	// IndexPredicate describes a partial index predicate.
	// https://postgresql.org/docs/current/catalog-pg-index.html
	IndexPredicate struct {
		schema.Attr
		P string
	}

	// IndexColumnProperty describes an index column property.
	// https://postgresql.org/docs/current/functions-info.html#FUNCTIONS-INFO-INDEX-COLUMN-PROPS
	IndexColumnProperty struct {
		schema.Attr
		// NullsFirst defaults to true for DESC indexes.
		NullsFirst bool
		// NullsLast defaults to true for ASC indexes.
		NullsLast bool
	}

	// IndexStorageParams describes index storage parameters add with the WITH clause.
	// https://postgresql.org/docs/current/sql-createindex.html#SQL-CREATEINDEX-STORAGE-PARAMETERS
	IndexStorageParams struct {
		schema.Attr
		// AutoSummarize defines the authsummarize storage parameter.
		AutoSummarize bool
		// PagesPerRange defines pages_per_range storage
		// parameter for BRIN indexes. Defaults to 128.
		PagesPerRange int64
	}

	// IndexInclude describes the INCLUDE clause allows specifying
	// a list of column which added to the index as non-key columns.
	// https://www.postgresql.org/docs/current/sql-createindex.html
	IndexInclude struct {
		schema.Attr
		Columns []*schema.Column
	}

	// IndexOpClass describers operator class of the index part.
	// https://www.postgresql.org/docs/current/indexes-opclass.html.
	IndexOpClass struct {
		schema.Attr
		Name    string                  // Name of the operator class.
		Default bool                    // If it is the default operator class.
		Params  []struct{ N, V string } // Optional parameters.
	}

	// IndexNullsDistinct describes the NULLS [NOT] DISTINCT clause.
	IndexNullsDistinct struct {
		schema.Attr
		V bool // NULLS [NOT] DISTINCT. Defaults to true.
	}

	// Concurrently describes the CONCURRENTLY clause to instruct Postgres to
	// build or drop the index concurrently without blocking the current table.
	// https://www.postgresql.org/docs/current/sql-createindex.html#SQL-CREATEINDEX-CONCURRENTLY
	Concurrently struct {
		schema.Clause
	}

	// NoInherit attribute defines the NO INHERIT flag for CHECK constraint.
	// https://postgresql.org/docs/current/catalog-pg-constraint.html
	NoInherit struct {
		schema.Attr
	}

	// CheckColumns attribute hold the column named used by the CHECK constraints.
	// This attribute is added on inspection for internal usage and has no meaning
	// on migration.
	CheckColumns struct {
		schema.Attr
		Columns []string
	}

	// Partition defines the spec of a partitioned table.
	Partition struct {
		schema.Attr
		// T defines the type/strategy of the partition.
		// Can be one of: RANGE, LIST, HASH.
		T string
		// Partition parts. The additional attributes
		// on each part can be used to control collation.
		Parts []*PartitionPart

		// Internal info returned from pg_partitioned_table.
		start, attrs, exprs string
	}

	// An PartitionPart represents an index part that
	// can be either an expression or a column.
	PartitionPart struct {
		X     schema.Expr
		C     *schema.Column
		Attrs []schema.Attr
	}

	// Cascade describes that a CASCADE clause should be added to the DROP [TABLE|SCHEMA]
	// operation. Note, this clause is automatically added to DROP SCHEMA by the planner.
	Cascade struct {
		schema.Clause
	}

	// ReferenceOption describes the ON DELETE and ON UPDATE options for foreign keys.
	ReferenceOption schema.ReferenceOption
)

// Ref returns a reference to the domain type.
func (d *DomainType) Ref() *schemahcl.Ref {
	return &schemahcl.Ref{V: "$domain." + d.T}
}

// Underlying returns the underlying type of the domain.
func (d *DomainType) Underlying() schema.Type {
	return d.Type
}

// Underlying returns the underlying type of the array.
func (a *ArrayType) Underlying() schema.Type {
	return a.Type
}

// String implements fmt.Stringer interface.
func (o ReferenceOption) String() string {
	return string(o)
}

// Scan implements sql.Scanner interface.
func (o *ReferenceOption) Scan(v any) error {
	var s sql.NullString
	if err := s.Scan(v); err != nil {
		return err
	}
	switch strings.ToLower(s.String) {
	case "a":
		*o = ReferenceOption(schema.NoAction)
	case "r":
		*o = ReferenceOption(schema.Restrict)
	case "c":
		*o = ReferenceOption(schema.Cascade)
	case "n":
		*o = ReferenceOption(schema.SetNull)
	case "d":
		*o = ReferenceOption(schema.SetDefault)
	default:
		return fmt.Errorf("unknown reference option: %q", s.String)
	}
	return nil
}

// IsUnique reports if the type is unique constraint.
func (c Constraint) IsUnique() bool { return strings.ToLower(c.T) == "u" }

// IntegerType returns the underlying integer type this serial type represents.
func (s *SerialType) IntegerType() *schema.IntegerType {
	t := &schema.IntegerType{T: TypeInteger}
	switch s.T {
	case TypeSerial2, TypeSmallSerial:
		t.T = TypeSmallInt
	case TypeSerial8, TypeBigSerial:
		t.T = TypeBigInt
	}
	return t
}

// SetType sets the serial type from the given integer type.
func (s *SerialType) SetType(t *schema.IntegerType) {
	switch t.T {
	case TypeSmallInt, TypeInt2:
		s.T = TypeSmallSerial
	case TypeInteger, TypeInt4, TypeInt:
		s.T = TypeSerial
	case TypeBigInt, TypeInt8:
		s.T = TypeBigSerial
	}
}

// sequence returns the inspected name of the sequence
// or the standard name defined by postgres.
func (s *SerialType) sequence(t *schema.Table, c *schema.Column) string {
	if s.SequenceName != "" {
		return s.SequenceName
	}
	return fmt.Sprintf("%s_%s_seq", t.Name, c.Name)
}

var (
	opsOnce    sync.Once
	defaultOps map[postgresop.Class]bool
)

// DefaultFor reports if the operator_class is the default for the index part.
func (o *IndexOpClass) DefaultFor(idx *schema.Index, part *schema.IndexPart) (bool, error) {
	// Explicitly defined as the default (Usually, it comes from the inspection).
	if o.Default && len(o.Params) == 0 {
		return true, nil
	}
	it := &IndexType{T: IndexTypeBTree}
	if sqlx.Has(idx.Attrs, it) {
		it.T = strings.ToUpper(it.T)
	}
	// The key type must be known to check if it is the default op_class.
	if part.X != nil || len(o.Params) > 0 {
		return false, nil
	}
	opsOnce.Do(func() {
		defaultOps = make(map[postgresop.Class]bool, len(postgresop.Classes))
		for _, op := range postgresop.Classes {
			if op.Default {
				defaultOps[postgresop.Class{Name: op.Name, Method: op.Method, Type: op.Type}] = true
			}
		}
	})
	var (
		t   string
		err error
	)
	switch typ := part.C.Type.Type.(type) {
	case *schema.EnumType:
		t = "anyenum"
	case *ArrayType:
		t = "anyarray"
	default:
		t, err = FormatType(typ)
		if err != nil {
			return false, fmt.Errorf("postgres: format operator-class type %T: %w", typ, err)
		}
	}
	return defaultOps[postgresop.Class{Name: o.Name, Method: it.T, Type: t}], nil
}

// Equal reports whether o and x are the same operator class.
func (o *IndexOpClass) Equal(x *IndexOpClass) bool {
	if o.Name != x.Name || o.Default != x.Default || len(o.Params) != len(x.Params) {
		return false
	}
	for i := range o.Params {
		if o.Params[i].N != x.Params[i].N || o.Params[i].V != x.Params[i].V {
			return false
		}
	}
	return true
}

// String returns the string representation of the operator class.
func (o *IndexOpClass) String() string {
	if len(o.Params) == 0 {
		return o.Name
	}
	var b strings.Builder
	b.WriteString(o.Name)
	b.WriteString("(")
	for i, p := range o.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.N)
		b.WriteString("=")
		b.WriteString(p.V)
	}
	b.WriteString(")")
	return b.String()
}

// UnmarshalText parses the operator class from its string representation.
func (o *IndexOpClass) UnmarshalText(text []byte) error {
	i := bytes.IndexByte(text, '(')
	if i == -1 {
		o.Name = string(text)
		return nil
	}
	o.Name = string(text[:i])
	return o.parseParams(string(text[i:]))
}

// parseParams parses index class parameters defined in HCL or returned
// from the database. For example: '{k=v}', '(k1=v1,k2=v2)'.
func (o *IndexOpClass) parseParams(kv string) error {
	switch {
	case kv == "":
	case strings.HasPrefix(kv, "(") && strings.HasSuffix(kv, ")"), strings.HasPrefix(kv, "{") && strings.HasSuffix(kv, "}"):
		for _, e := range strings.Split(kv[1:len(kv)-1], ",") {
			if kv := strings.Split(strings.TrimSpace(e), "="); len(kv) == 2 {
				o.Params = append(o.Params, struct{ N, V string }{N: kv[0], V: kv[1]})
			}
		}
	default:
		return fmt.Errorf("postgres: unexpected operator class parameters format: %q", kv)
	}
	return nil
}

// newIndexStorage parses and returns the index storage parameters.
func newIndexStorage(opts string) (*IndexStorageParams, error) {
	params := &IndexStorageParams{}
	for _, p := range strings.Split(strings.Trim(opts, "{}"), ",") {
		kv := strings.Split(p, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid index storage parameter: %s", p)
		}
		switch kv[0] {
		case "autosummarize":
			b, err := strconv.ParseBool(kv[1])
			if err != nil {
				return nil, fmt.Errorf("failed parsing autosummarize %q: %w", kv[1], err)
			}
			params.AutoSummarize = b
		case "pages_per_range":
			i, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed parsing pages_per_range %q: %w", kv[1], err)
			}
			params.PagesPerRange = i
		}
	}
	return params, nil
}

// reFmtType extracts the formatted type and an option schema qualifier.
var reFmtType = regexp.MustCompile(`^(?:(".+"|\w+)\.)?(".+"|\w+)$`)

// parseFmtType parses the formatted type returned from pg_catalog.format_type
// and extract the schema and type name.
func parseFmtType(t string) (s, n string) {
	n = t
	parts := reFmtType.FindStringSubmatch(t)
	r := func(s string) string {
		s = strings.ReplaceAll(s, `""`, `"`)
		if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return s
	}
	if len(parts) > 1 {
		s = r(parts[1])
	}
	if len(parts) > 2 {
		n = r(parts[2])
	}
	return s, n
}

func newEnumType(t string, id int64) *enumType {
	s, n := parseFmtType(t)
	return &enumType{T: n, Schema: s, ID: id}
}

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT setting FROM pg_settings WHERE name IN ('server_version_num', 'crdb_version') ORDER BY name DESC`

	// Query to list database schemas.
	schemasQuery = `
SELECT
	nspname AS schema_name,
	pg_catalog.obj_description(oid) AS comment
FROM
    pg_catalog.pg_namespace
WHERE
    nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast', 'crdb_internal', 'pg_extension')
    AND nspname NOT LIKE 'pg_%temp_%'
ORDER BY
    nspname`

	// Query to list database schemas.
	schemasQueryArgs = `
SELECT
	nspname AS schema_name,
	pg_catalog.obj_description(oid) AS comment
FROM
    pg_catalog.pg_namespace
WHERE
    nspname %s
ORDER BY
    nspname`

	// Query to list table information.
	tablesQuery = `
SELECT
	t3.oid,
	t1.table_schema,
	t1.table_name,
	pg_catalog.obj_description(t3.oid, 'pg_class') AS comment,
	t4.partattrs AS partition_attrs,
	t4.partstrat AS partition_strategy,
	pg_get_expr(t4.partexprs, t4.partrelid) AS partition_exprs
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN pg_catalog.pg_namespace AS t2 ON t2.nspname = t1.table_schema
	JOIN pg_catalog.pg_class AS t3 ON t3.relnamespace = t2.oid AND t3.relname = t1.table_name
	LEFT JOIN pg_catalog.pg_partitioned_table AS t4 ON t4.partrelid = t3.oid
	LEFT JOIN pg_depend AS t5 ON t5.objid = t3.oid AND t5.deptype = 'e'
WHERE
	t1.table_type = 'BASE TABLE'
	AND NOT COALESCE(t3.relispartition, false)
	AND t1.table_schema IN (%s)
	AND t5.objid IS NULL
ORDER BY
	t1.table_schema, t1.table_name
`
	tablesQueryArgs = `
SELECT
	t3.oid,
	t1.table_schema,
	t1.table_name,
	pg_catalog.obj_description(t3.oid, 'pg_class') AS comment,
	t4.partattrs AS partition_attrs,
	t4.partstrat AS partition_strategy,
	pg_get_expr(t4.partexprs, t4.partrelid) AS partition_exprs
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN pg_catalog.pg_namespace AS t2 ON t2.nspname = t1.table_schema
	JOIN pg_catalog.pg_class AS t3 ON t3.relnamespace = t2.oid AND t3.relname = t1.table_name
	LEFT JOIN pg_catalog.pg_partitioned_table AS t4 ON t4.partrelid = t3.oid
	LEFT JOIN pg_depend AS t5 ON t5.objid = t3.oid AND t5.deptype = 'e'
WHERE
	t1.table_type = 'BASE TABLE'
	AND NOT COALESCE(t3.relispartition, false)
	AND t1.table_schema IN (%s)
	AND t1.table_name IN (%s)
	AND t5.objid IS NULL
ORDER BY
	t1.table_schema, t1.table_name
`
	// Query to list table columns.
	columnsQuery = `
SELECT
	t1.table_name,
	t1.column_name,
	t1.data_type,
	pg_catalog.format_type(a.atttypid, a.atttypmod) AS format_type,
	t1.is_nullable,
	t1.column_default,
	t1.character_maximum_length,
	t1.numeric_precision,
	t1.datetime_precision,
	t1.numeric_scale,
	t1.interval_type,
	t1.character_set_name,
	t1.collation_name,
	t1.is_identity,
	t1.identity_start,
	t1.identity_increment,
	(CASE WHEN t1.is_identity = 'YES' THEN (SELECT last_value FROM pg_sequences WHERE quote_ident(schemaname) || '.' || quote_ident(sequencename) = pg_get_serial_sequence(quote_ident(t1.table_schema) || '.' || quote_ident(t1.table_name), t1.column_name)) END) AS identity_last,
	t1.identity_generation,
	t1.generation_expression,
	col_description(t3.oid, "ordinal_position") AS comment,
	t4.typtype,
	t4.typelem,
	(CASE WHEN t4.typcategory = 'A' AND t4.typelem <> 0 THEN (SELECT t.typtype FROM pg_catalog.pg_type t WHERE t.oid = t4.typelem) END) AS elemtyp,
	t4.oid
FROM
	"information_schema"."columns" AS t1
	JOIN pg_catalog.pg_namespace AS t2 ON t2.nspname = t1.table_schema
	JOIN pg_catalog.pg_class AS t3 ON t3.relnamespace = t2.oid AND t3.relname = t1.table_name
	JOIN pg_catalog.pg_attribute AS a ON a.attrelid = t3.oid AND a.attname = t1.column_name
	LEFT JOIN pg_catalog.pg_type AS t4 ON t4.oid = a.atttypid
WHERE
	t1.table_schema = $1 AND t1.table_name IN (%s)
ORDER BY
	t1.table_name, t1.ordinal_position
`
	// Query to list enum values.
	enumsQuery = `
SELECT
	n.nspname AS schema_name,
	e.enumtypid AS enum_id,
	t.typname AS enum_name,
	e.enumlabel AS enum_value
FROM
	pg_enum e
	JOIN pg_type t ON e.enumtypid = t.oid
	JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE
    n.nspname IN (%s)
ORDER BY
    n.nspname, e.enumtypid, e.enumsortorder
`
	// Query to list foreign-keys.
	fksQuery = `
SELECT 
    fk.constraint_name,
    fk.table_name,
    a1.attname AS column_name,
    fk.schema_name,
    fk.referenced_table_name,
    a2.attname AS referenced_column_name,
    fk.referenced_schema_name,
    fk.confupdtype,
    fk.confdeltype
	FROM 
	    (
	    	SELECT
	      		con.conname AS constraint_name,
	      		con.conrelid,
	      		con.confrelid,
	      		t1.relname AS table_name,
	      		ns1.nspname AS schema_name,
      			t2.relname AS referenced_table_name,
	      		ns2.nspname AS referenced_schema_name,
	      		generate_series(1,array_length(con.conkey,1)) as ord,
	      		unnest(con.conkey) AS conkey,
	      		unnest(con.confkey) AS confkey,
	      		con.confupdtype,
	      		con.confdeltype
	    	FROM pg_constraint con
	    	JOIN pg_class t1 ON t1.oid = con.conrelid
	    	JOIN pg_class t2 ON t2.oid = con.confrelid
	    	JOIN pg_namespace ns1 on t1.relnamespace = ns1.oid
	    	JOIN pg_namespace ns2 on t2.relnamespace = ns2.oid
	    	WHERE ns1.nspname = $1
	    	AND t1.relname IN (%s)
	    	AND con.contype = 'f'
	) AS fk
	JOIN pg_attribute a1 ON a1.attnum = fk.conkey AND a1.attrelid = fk.conrelid
	JOIN pg_attribute a2 ON a2.attnum = fk.confkey AND a2.attrelid = fk.confrelid
	ORDER BY
	    fk.conrelid, fk.constraint_name, fk.ord
`

	// Query to list table check constraints.
	checksQuery = `
SELECT
	rel.relname AS table_name,
	t1.conname AS constraint_name,
	pg_get_expr(t1.conbin, t1.conrelid) as expression,
	t2.attname as column_name,
	t1.conkey as column_indexes,
	t1.connoinherit as no_inherit
FROM
	pg_constraint t1
	JOIN pg_attribute t2
	ON t2.attrelid = t1.conrelid
	AND t2.attnum = ANY (t1.conkey)
	JOIN pg_class rel
	ON rel.oid = t1.conrelid
	JOIN pg_namespace nsp
	ON nsp.oid = t1.connamespace
WHERE
	t1.contype = 'c'
	AND nsp.nspname = $1
	AND rel.relname IN (%s)
ORDER BY
	t1.conname, array_position(t1.conkey, t2.attnum)
`
)

var (
	indexesBelow11   = fmt.Sprintf(indexesQueryTmpl, "false", "false", "%s")
	indexesAbove11   = fmt.Sprintf(indexesQueryTmpl, "(a.attname <> '' AND idx.indnatts > idx.indnkeyatts AND idx.ord > idx.indnkeyatts)", "false", "%s")
	indexesAbove15   = fmt.Sprintf(indexesQueryTmpl, "(a.attname <> '' AND idx.indnatts > idx.indnkeyatts AND idx.ord > idx.indnkeyatts)", "idx.indnullsnotdistinct", "%s")
	indexesQueryTmpl = `
SELECT
	t.relname AS table_name,
	i.relname AS index_name,
	am.amname AS index_type,
	a.attname AS column_name,
	%s AS included,
	idx.indisprimary AS primary,
	idx.indisunique AS unique,
	con.nametypes AS constraints,
	pg_get_expr(idx.indpred, idx.indrelid) AS predicate,
	pg_get_indexdef(idx.indexrelid, idx.ord, false) AS expression,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'desc') AS isdesc,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_first') AS nulls_first,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_last') AS nulls_last,
	obj_description(i.oid, 'pg_class') AS comment,
	i.reloptions AS options,
	op.opcname AS opclass_name,
	op.opcdefault AS opclass_default,
	a2.attoptions AS opclass_params,
    %s AS indnullsnotdistinct
FROM
	(
		select
			*,
			generate_series(1,array_length(i.indkey,1)) as ord,
			unnest(i.indkey) AS key
		from pg_index i
	) idx
	JOIN pg_class i ON i.oid = idx.indexrelid
	JOIN pg_class t ON t.oid = idx.indrelid
	JOIN pg_namespace n ON n.oid = t.relnamespace
	LEFT JOIN (
	    select conindid, jsonb_object_agg(conname, contype) AS nametypes
	    from pg_constraint
	    group by conindid
	) con ON con.conindid = idx.indexrelid
	LEFT JOIN pg_attribute a ON (a.attrelid, a.attnum) = (idx.indrelid, idx.key)
	JOIN pg_am am ON am.oid = i.relam
	LEFT JOIN pg_opclass op ON op.oid = idx.indclass[idx.ord-1]
	LEFT JOIN pg_attribute a2 ON (a2.attrelid, a2.attnum) = (idx.indexrelid, idx.ord)
WHERE
	n.nspname = $1
	AND t.relname IN (%s)
ORDER BY
	table_name, index_name, idx.ord
`
)
