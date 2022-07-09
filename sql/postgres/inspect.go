// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

// A diff provides a PostgreSQL implementation for schema.Inspector.
type inspect struct{ conn }

var _ schema.Inspector = (*inspect)(nil)

// InspectRealm returns schema descriptions of all resources in the given realm.
func (i *inspect) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := i.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	r := schema.NewRealm(schemas...).SetCollation(i.collate)
	r.Attrs = append(r.Attrs, &CType{V: i.ctype})
	if len(schemas) == 0 || !sqlx.ModeInspectRealm(opts).Is(schema.InspectTables) {
		return r, nil
	}
	if err := i.inspectTables(ctx, r, nil); err != nil {
		return nil, err
	}
	sqlx.LinkSchemaTables(schemas)
	return r, nil
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
		return nil, &schema.NotExistError{Err: fmt.Errorf("postgres: schema %q was not found", name)}
	case n > 1:
		return nil, fmt.Errorf("postgres: %d schemas were found for %q", n, name)
	}
	r := schema.NewRealm(schemas...).SetCollation(i.collate)
	r.Attrs = append(r.Attrs, &CType{V: i.ctype})
	if sqlx.ModeInspectSchema(opts).Is(schema.InspectTables) {
		if err := i.inspectTables(ctx, r, opts); err != nil {
			return nil, err
		}
		sqlx.LinkSchemaTables(schemas)
	}
	return r.Schemas[0], nil
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
		args  []interface{}
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
		var tSchema, name, comment, partattrs, partstart, partexprs sql.NullString
		if err := rows.Scan(&tSchema, &name, &comment, &partattrs, &partstart, &partexprs); err != nil {
			return fmt.Errorf("scan table information: %w", err)
		}
		if !sqlx.ValidString(tSchema) || !sqlx.ValidString(name) {
			return fmt.Errorf("invalid schema or table name: %q.%q", tSchema.String, name.String)
		}
		s, ok := realm.Schema(tSchema.String)
		if !ok {
			return fmt.Errorf("schema %q was not found in realm", tSchema.String)
		}
		t := &schema.Table{Name: name.String}
		s.AddTables(t)
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
	if err := rows.Close(); err != nil {
		return err
	}
	if err := i.enumValues(ctx, s); err != nil {
		return err
	}
	return nil
}

// addColumn scans the current row and adds a new column from it to the table.
func (i *inspect) addColumn(s *schema.Schema, rows *sql.Rows) (err error) {
	var (
		typid, maxlen, precision, timeprecision, scale, seqstart, seqinc, seqlast                                                  sql.NullInt64
		table, name, typ, fmtype, nullable, defaults, identity, genidentity, genexpr, charset, collate, comment, typtype, interval sql.NullString
	)
	if err = rows.Scan(
		&table, &name, &typ, &fmtype, &nullable, &defaults, &maxlen, &precision, &timeprecision, &scale, &interval,
		&charset, &collate, &identity, &seqstart, &seqinc, &seqlast, &genidentity, &genexpr, &comment, &typtype, &typid,
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
			Raw:  typ.String,
			Null: nullable.String == "YES",
		},
	}
	c.Type.Type, err = columnType(&columnDesc{
		typ:           typ.String,
		fmtype:        fmtype.String,
		size:          maxlen.Int64,
		scale:         scale.Int64,
		typtype:       typtype.String,
		typid:         typid.Int64,
		interval:      interval.String,
		precision:     precision.Int64,
		timePrecision: &timeprecision.Int64,
	})
	if defaults.Valid {
		c.Default = defaultExpr(c, defaults.String)
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
	t.Columns = append(t.Columns, c)
	return nil
}

func columnType(c *columnDesc) (schema.Type, error) {
	var typ schema.Type
	switch t := c.typ; strings.ToLower(t) {
	case TypeBigInt, TypeInt8, TypeInt, TypeInteger, TypeInt4, TypeSmallInt, TypeInt2, TypeInt64:
		typ = &schema.IntegerType{T: t}
	case TypeBit, TypeBitVar:
		typ = &BitType{T: t, Len: c.size}
	case TypeBool, TypeBoolean:
		typ = &schema.BoolType{T: t}
	case TypeBytea:
		typ = &schema.BinaryType{T: t}
	case TypeCharacter, TypeChar, TypeCharVar, TypeVarChar, TypeText:
		// A `character` column without length specifier is equivalent to `character(1)`,
		// but `varchar` without length accepts strings of any size (same as `text`).
		typ = &schema.StringType{T: t, Size: int(c.size)}
	case TypeCIDR, TypeInet, TypeMACAddr, TypeMACAddr8:
		typ = &NetworkType{T: t}
	case TypeCircle, TypeLine, TypeLseg, TypeBox, TypePath, TypePolygon, TypePoint, TypeGeometry:
		typ = &schema.SpatialType{T: t}
	case TypeDate:
		typ = &schema.TimeType{T: t}
	case TypeTime, TypeTimeWOTZ, TypeTimeTZ, TypeTimeWTZ, TypeTimestamp,
		TypeTimestampTZ, TypeTimestampWTZ, TypeTimestampWOTZ:
		p := defaultTimePrecision
		if c.timePrecision != nil {
			p = int(*c.timePrecision)
		}
		typ = &schema.TimeType{T: t, Precision: &p}
	case TypeInterval:
		p := defaultTimePrecision
		if c.timePrecision != nil {
			p = int(*c.timePrecision)
		}
		typ = &IntervalType{T: t, Precision: &p}
		if c.interval != "" {
			f, ok := intervalField(c.interval)
			if !ok {
				return &schema.UnsupportedType{T: c.interval}, nil
			}
			typ.(*IntervalType).F = f
		}
	case TypeReal, TypeDouble, TypeFloat4, TypeFloat8:
		typ = &schema.FloatType{T: t, Precision: int(c.precision)}
	case TypeJSON, TypeJSONB:
		typ = &schema.JSONType{T: t}
	case TypeMoney:
		typ = &CurrencyType{T: t}
	case TypeDecimal, TypeNumeric:
		typ = &schema.DecimalType{T: t, Precision: int(c.precision), Scale: int(c.scale)}
	case TypeSmallSerial, TypeSerial, TypeBigSerial, TypeSerial2, TypeSerial4, TypeSerial8:
		typ = &SerialType{T: t, Precision: int(c.precision)}
	case TypeUUID:
		typ = &UUIDType{T: t}
	case TypeXML:
		typ = &XMLType{T: t}
	case TypeArray:
		// Ignore multi-dimensions or size constraints
		// as they are ignored by the database.
		typ = &ArrayType{T: c.fmtype}
		if t, ok := arrayType(c.fmtype); ok {
			tt, err := ParseType(t)
			if err != nil {
				return nil, err
			}
			typ.(*ArrayType).Type = tt
		}
	case TypeUserDefined:
		typ = &UserDefinedType{T: c.fmtype}
		// The `typtype` column is set to 'e' for enum types, and the
		// values are filled in batch after the rows above is closed.
		// https://www.postgresql.org/docs/current/catalog-pg-type.html
		if c.typtype == "e" {
			typ = &enumType{T: c.fmtype, ID: c.typid}
		}
	default:
		typ = &schema.UnsupportedType{T: t}
	}
	return typ, nil
}

// enumValues fills enum columns with their values from the database.
func (i *inspect) enumValues(ctx context.Context, s *schema.Schema) error {
	var (
		args  []interface{}
		ids   = make(map[int64][]*schema.EnumType)
		query = "SELECT enumtypid, enumlabel FROM pg_enum WHERE enumtypid IN (%s)"
	)
	for _, t := range s.Tables {
		for _, c := range t.Columns {
			if enum, ok := c.Type.Type.(*enumType); ok {
				if _, ok := ids[enum.ID]; !ok {
					args = append(args, enum.ID)
				}
				// Convert the intermediate type to the
				// standard schema.EnumType.
				e := &schema.EnumType{T: enum.T}
				c.Type.Type = e
				c.Type.Raw = enum.T
				ids[enum.ID] = append(ids[enum.ID], e)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := i.QueryContext(ctx, fmt.Sprintf(query, nArgs(0, len(args))), args...)
	if err != nil {
		return fmt.Errorf("postgres: querying enum values: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id int64
			v  string
		)
		if err := rows.Scan(&id, &v); err != nil {
			return fmt.Errorf("postgres: scanning enum label: %w", err)
		}
		for _, enum := range ids[id] {
			enum.Values = append(enum.Values, v)
		}
	}
	return nil
}

// indexes queries and appends the indexes of the given table.
func (i *inspect) indexes(ctx context.Context, s *schema.Schema) error {
	if i.conn.crdb {
		return i.crdbIndexes(ctx, s)
	}
	rows, err := i.querySchema(ctx, indexesQuery, s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q indexes: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.addIndexes(s, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addIndexes scans the rows and adds the indexes to the table.
func (i *inspect) addIndexes(s *schema.Schema, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			uniq, primary                                 bool
			table, name, typ                              string
			desc, nullsfirst, nullslast                   sql.NullBool
			column, contype, pred, expr, comment, options sql.NullString
		)
		if err := rows.Scan(&table, &name, &typ, &column, &primary, &uniq, &contype, &pred, &expr, &desc, &nullsfirst, &nullslast, &comment, &options); err != nil {
			return fmt.Errorf("postgres: scanning indexes for schema %q: %w", s.Name, err)
		}
		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", table)
		}
		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: uniq,
				Table:  t,
				Attrs: []schema.Attr{
					&IndexType{T: typ},
				},
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(idx.Attrs, &schema.Comment{Text: comment.String})
			}
			if sqlx.ValidString(contype) {
				idx.Attrs = append(idx.Attrs, &ConType{T: contype.String})
			}
			if sqlx.ValidString(pred) {
				idx.Attrs = append(idx.Attrs, &IndexPredicate{P: pred.String})
			}
			if sqlx.ValidString(options) {
				p, err := newIndexStorage(options.String)
				if err != nil {
					return err
				}
				idx.Attrs = append(idx.Attrs, p)
			}
			names[name] = idx
			if primary {
				t.PrimaryKey = idx
			} else {
				t.Indexes = append(t.Indexes, idx)
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
		case sqlx.ValidString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("postgres: column %q was not found for index %q", column.String, idx.Name)
			}
			part.C.Indexes = append(part.C.Indexes, idx)
		case sqlx.ValidString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
		default:
			return fmt.Errorf("postgres: invalid part for index %q", idx.Name)
		}
		idx.Parts = append(idx.Parts, part)
	}
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
	if err := sqlx.SchemaFKs(s, rows); err != nil {
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
		args  []interface{}
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
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, &schema.Schema{
			Name: name,
		})
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return schemas, nil
}

func (i *inspect) querySchema(ctx context.Context, query string, s *schema.Schema) (*sql.Rows, error) {
	args := []interface{}{s.Name}
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

func defaultExpr(c *schema.Column, x string) schema.Expr {
	switch {
	case sqlx.IsLiteralBool(x), sqlx.IsLiteralNumber(x), sqlx.IsQuoted(x, '\''):
		return &schema.Literal{V: x}
	default:
		// Try casting or fallback to raw expressions (e.g. column text[] has the default of '{}':text[]).
		if v, ok := canConvert(c.Type, x); ok {
			return &schema.Literal{V: v}
		}
		return &schema.RawExpr{X: x}
	}
}

func canConvert(t *schema.ColumnType, x string) (string, bool) {
	r := t.Raw
	if t, ok := t.Type.(*ArrayType); ok {
		r = t.T
	}
	i := strings.Index(x, "::"+r)
	if i == -1 || !sqlx.IsQuoted(x[:i], '\'') {
		return "", false
	}
	q := x[0:i]
	x = x[1 : i-1]
	switch t.Type.(type) {
	case *schema.BoolType:
		if sqlx.IsLiteralBool(x) {
			return x, true
		}
	case *schema.DecimalType, *schema.IntegerType, *schema.FloatType:
		if sqlx.IsLiteralNumber(x) {
			return x, true
		}
	case *ArrayType, *schema.BinaryType, *schema.JSONType, *NetworkType, *schema.SpatialType, *schema.StringType, *schema.TimeType, *UUIDType, *XMLType:
		return q, true
	}
	return "", false
}

type (
	// CType describes the character classification setting (LC_CTYPE).
	CType struct {
		schema.Attr
		V string
	}

	// UserDefinedType defines a user-defined type attribute.
	UserDefinedType struct {
		schema.Type
		T string
	}

	// enumType represents an enum type. It serves aa intermediate representation of a Postgres enum type,
	// to temporary save TypeID and TypeName of an enum column until the enum values can be extracted.
	enumType struct {
		schema.Type
		T      string // Type name.
		ID     int64  // Type id.
		Values []string
	}

	// ArrayType defines an array type.
	// https://www.postgresql.org/docs/current/arrays.html
	ArrayType struct {
		schema.Type        // Underlying items type (e.g. varchar(255)).
		T           string // Formatted type (e.g. int[]).
	}

	// BitType defines a bit type.
	// https://www.postgresql.org/docs/current/datatype-bit.html
	BitType struct {
		schema.Type
		T   string
		Len int64
	}

	// IntervalType defines an interval type.
	// https://www.postgresql.org/docs/current/datatype-datetime.html
	IntervalType struct {
		schema.Type
		T         string // Type name.
		F         string // Optional field. YEAR, MONTH, ..., MINUTE TO SECOND.
		Precision *int   // Optional precision.
	}

	// A NetworkType defines a network type.
	// https://www.postgresql.org/docs/current/datatype-net-types.html
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

	// A SerialType defines a serial type.
	SerialType struct {
		schema.Type
		T         string
		Precision int
	}

	// A UUIDType defines a UUID type.
	UUIDType struct {
		schema.Type
		T string
	}

	// A XMLType defines an XML type.
	XMLType struct {
		schema.Type
		T string
	}

	// ConType describes constraint type.
	// https://www.postgresql.org/docs/current/catalog-pg-constraint.html
	ConType struct {
		schema.Attr
		T string // c, f, p, u, t, x.
	}

	// Sequence defines (the supported) sequence options.
	// https://www.postgresql.org/docs/current/sql-createsequence.html
	Sequence struct {
		Start, Increment int64
		// Last sequence value written to disk.
		// https://www.postgresql.org/docs/current/view-pg-sequences.html.
		Last int64
	}

	// Identity defines an identity column.
	Identity struct {
		schema.Attr
		Generation string // ALWAYS, BY DEFAULT.
		Sequence   *Sequence
	}

	// IndexType represents an index type.
	// https://www.postgresql.org/docs/current/indexes-types.html
	IndexType struct {
		schema.Attr
		T string // BTREE, BRIN, HASH, GiST, SP-GiST, GIN.
	}

	// IndexPredicate describes a partial index predicate.
	// https://www.postgresql.org/docs/current/catalog-pg-index.html
	IndexPredicate struct {
		schema.Attr
		P string
	}

	// IndexColumnProperty describes an index column property.
	// https://www.postgresql.org/docs/current/functions-info.html#FUNCTIONS-INFO-INDEX-COLUMN-PROPS
	IndexColumnProperty struct {
		schema.Attr
		// NullsFirst defaults to true for DESC indexes.
		NullsFirst bool
		// NullsLast defaults to true for ASC indexes.
		NullsLast bool
	}

	// IndexStorageParams describes index storage parameters add with the WITH clause.
	// https://www.postgresql.org/docs/current/sql-createindex.html#SQL-CREATEINDEX-STORAGE-PARAMETERS
	IndexStorageParams struct {
		schema.Attr
		// AutoSummarize defines the authsummarize storage parameter.
		AutoSummarize bool
		// PagesPerRange defines pages_per_range storage
		// parameter for BRIN indexes. Defaults to 128.
		PagesPerRange int64
	}

	// NoInherit attribute defines the NO INHERIT flag for CHECK constraint.
	// https://www.postgresql.org/docs/current/catalog-pg-constraint.html
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
)

// IsUnique reports if the type is unique constraint.
func (c ConType) IsUnique() bool { return strings.ToLower(c.T) == "u" }

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

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT setting FROM pg_settings WHERE name IN ('lc_collate', 'lc_ctype', 'server_version_num', 'crdb_version') ORDER BY name DESC`

	// Query to list database schemas.
	schemasQuery = "SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast', 'crdb_internal', 'pg_extension') AND schema_name NOT LIKE 'pg_%temp_%' ORDER BY schema_name"

	// Query to list specific database schemas.
	schemasQueryArgs = "SELECT schema_name FROM information_schema.schemata WHERE schema_name %s ORDER BY schema_name"

	// Query to list table information.
	tablesQuery = `
SELECT
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
WHERE
	t1.table_type = 'BASE TABLE'
	AND NOT COALESCE(t3.relispartition, false)
	AND t1.table_schema IN (%s)
ORDER BY
	t1.table_schema, t1.table_name
`
	tablesQueryArgs = `
SELECT
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
WHERE
	t1.table_type = 'BASE TABLE'
	AND NOT COALESCE(t3.relispartition, false)
	AND t1.table_schema IN (%s)
	AND t1.table_name IN (%s)
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
	t4.oid
FROM
	"information_schema"."columns" AS t1
	JOIN pg_catalog.pg_namespace AS t2 ON t2.nspname = t1.table_schema
	JOIN pg_catalog.pg_class AS t3 ON t3.relnamespace = t2.oid AND t3.relname = t1.table_name
	JOIN pg_catalog.pg_attribute AS a ON a.attrelid = t3.oid AND a.attname = t1.column_name
	LEFT JOIN pg_catalog.pg_type AS t4 ON t1.udt_name = t4.typname
WHERE
	t1.table_schema = $1 AND t1.table_name IN (%s)
ORDER BY
	t1.table_name, t1.ordinal_position
`

	// Query to list table indexes.
	indexesQuery = `
SELECT
	t.relname AS table_name,
	i.relname AS index_name,
	am.amname AS index_type,
	a.attname AS column_name,
	idx.indisprimary AS primary,
	idx.indisunique AS unique,
	c.contype AS constraint_type,
	pg_get_expr(idx.indpred, idx.indrelid) AS predicate,
	pg_get_indexdef(idx.indexrelid, idx.ord, false) AS expression,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'desc') AS desc,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_first') AS nulls_first,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_last') AS nulls_last,
	obj_description(i.oid, 'pg_class') AS comment,
	i.reloptions AS options
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
	LEFT JOIN pg_constraint c ON idx.indexrelid = c.conindid
	LEFT JOIN pg_attribute a ON (a.attrelid, a.attnum) = (idx.indrelid, idx.key)
	JOIN pg_am am ON am.oid = i.relam
WHERE
	n.nspname = $1
	AND t.relname IN (%s)
	AND COALESCE(c.contype, '') <> 'f'
ORDER BY
	table_name, index_name, idx.ord
`
	fksQuery = `
SELECT
    t1.constraint_name,
    t1.table_name,
    t2.column_name,
    t1.table_schema,
    t3.table_name AS referenced_table_name,
    t3.column_name AS referenced_column_name,
    t3.table_schema AS referenced_schema_name,
    t4.update_rule,
    t4.delete_rule
FROM
    information_schema.table_constraints t1
    JOIN information_schema.key_column_usage t2
    ON t1.constraint_name = t2.constraint_name
    AND t1.table_schema = t2.constraint_schema
    JOIN information_schema.constraint_column_usage t3
    ON t1.constraint_name = t3.constraint_name
    AND t1.table_schema = t3.constraint_schema
    JOIN information_schema.referential_constraints t4
    ON t1.constraint_name = t4.constraint_name
    AND t1.table_schema = t4.constraint_schema
WHERE
    t1.constraint_type = 'FOREIGN KEY'
    AND t1.table_schema = $1
    AND t1.table_name IN (%s)
ORDER BY
    t1.constraint_name,
    t2.ordinal_position
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
