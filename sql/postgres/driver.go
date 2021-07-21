package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"

	"golang.org/x/mod/semver"
)

// Driver represents a PostgreSQL driver for introspecting database schemas
// and apply migrations changes on them.
type Driver struct {
	schema.ExecQuerier
	// System variables that are set on `Open`.
	collate string
	ctype   string
	version string
}

// Open opens a new PostgreSQL driver.
func Open(db schema.ExecQuerier) (*Driver, error) {
	drv := &Driver{ExecQuerier: db}
	rows, err := db.Query(paramsQuery)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning system variables: %w", err)
	}
	defer rows.Close()
	params := make([]string, 0, 3)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("postgres: failed scanning row value: %w", err)
		}
		params = append(params, v)
	}
	if len(params) != 3 {
		return nil, fmt.Errorf("postgres: unexpected number of rows: %d", len(params))
	}
	drv.collate, drv.ctype, drv.version = params[0], params[1], params[2]
	if len(drv.version) != 6 {
		return nil, fmt.Errorf("postgres: malformed version: %s", drv.version)
	}
	drv.version = fmt.Sprintf("%s.%s.%s", drv.version[:2], drv.version[2:4], drv.version[4:])
	if semver.Compare("v"+drv.version, "v10.0.0") != -1 {
		return nil, fmt.Errorf("postgres: unsupported postgres version: %s", drv.version)
	}
	return drv, nil
}

// InspectRealm returns schema descriptions of all resources in the given realm.
func (d *Driver) InspectRealm(ctx context.Context, opts *schema.InspectRealmOption) (*schema.Realm, error) {
	schemas, err := d.schemas(ctx, opts)
	if err != nil {
		return nil, err
	}
	realm := &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Collation{V: d.collate}, &CType{V: d.ctype}}}
	for _, s := range schemas {
		names, err := d.tableNames(ctx, s.Name, nil)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			t, err := d.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
			if err != nil {
				return nil, err
			}
			s.Tables = append(s.Tables, t)
		}
		s.Realm = realm
	}
	sqlx.LinkSchemaTables(schemas)
	return realm, nil
}

// InspectSchema returns schema descriptions of all tables in the given schema.
func (d *Driver) InspectSchema(ctx context.Context, name string, opts *schema.InspectOptions) (*schema.Schema, error) {
	schemas, err := d.schemas(ctx, &schema.InspectRealmOption{
		Schemas: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, &schema.NotExistError{
			Err: fmt.Errorf("postgres: schema %q was not found", name),
		}
	}
	names, err := d.tableNames(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	s := schemas[0]
	for _, name := range names {
		t, err := d.inspectTable(ctx, name, &schema.InspectTableOptions{Schema: s.Name}, s)
		if err != nil {
			return nil, err
		}
		s.Tables = append(s.Tables, t)
	}
	sqlx.LinkSchemaTables(schemas)
	s.Realm = &schema.Realm{Schemas: schemas, Attrs: []schema.Attr{&schema.Collation{V: d.collate}, &CType{V: d.ctype}}}
	return s, nil
}

// InspectTable returns the schema description of the given table.
func (d *Driver) InspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	return d.inspectTable(ctx, name, opts, nil)
}

func (d *Driver) inspectTable(ctx context.Context, name string, opts *schema.InspectTableOptions, top *schema.Schema) (*schema.Table, error) {
	t, err := d.table(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	if top != nil {
		// Link the table to its top element if provided.
		t.Schema = top
	}
	if err := d.columns(ctx, t); err != nil {
		return nil, err
	}
	if err := d.indexes(ctx, t); err != nil {
		return nil, err
	}
	if err := d.fks(ctx, t); err != nil {
		return nil, err
	}
	if err := d.checks(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// table returns the table from the database, or a NotExistError if the table was not found.
func (d *Driver) table(ctx context.Context, name string, opts *schema.InspectTableOptions) (*schema.Table, error) {
	var (
		args  = []interface{}{name}
		query = tableQuery
	)
	if opts != nil && opts.Schema != "" {
		query = tableSchemaQuery
		args = append(args, opts.Schema)
	}
	row := d.QueryRowContext(ctx, query, args...)
	var tSchema, comment sql.NullString
	if err := row.Scan(&tSchema, &comment); err != nil {
		if err == sql.ErrNoRows {
			return nil, &schema.NotExistError{
				Err: fmt.Errorf("postgres: table %q was not found", name),
			}
		}
		return nil, err
	}
	t := &schema.Table{Name: name, Schema: &schema.Schema{Name: tSchema.String}}
	if sqlx.ValidString(comment) {
		t.Attrs = append(t.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	return t, nil
}

// columns queries and appends the columns of the given table.
func (d *Driver) columns(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, columnsQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q columns: %w", t.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := d.addColumn(t, rows); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := d.enumValues(ctx, t.Columns); err != nil {
		return err
	}
	return nil
}

// addColumn scans the current row and adds a new column from it to the table.
func (d *Driver) addColumn(t *schema.Table, rows *sql.Rows) error {
	var (
		typid, maxlen, precision, scale                                                    sql.NullInt64
		name, typ, nullable, defaults, udt, identity, charset, collation, comment, typtype sql.NullString
	)
	if err := rows.Scan(&name, &typ, &nullable, &defaults, &maxlen, &precision, &scale, &charset, &collation, &udt, &identity, &comment, &typtype, &typid); err != nil {
		return err
	}
	c := &schema.Column{
		Name: name.String,
		Type: &schema.ColumnType{
			Raw:  typ.String,
			Null: nullable.String == "YES",
		},
	}
	c.Type.Type = columnType(&columnMeta{
		typ:       typ.String,
		size:      maxlen.Int64,
		udt:       udt.String,
		precision: precision.Int64,
		scale:     scale.Int64,
		typtype:   typtype.String,
		typid:     typid.Int64,
	})
	if sqlx.ValidString(defaults) {
		c.Default = &schema.RawExpr{
			X: defaults.String,
		}
	}
	if sqlx.ValidString(comment) {
		c.Attrs = append(c.Attrs, &schema.Comment{
			Text: comment.String,
		})
	}
	if sqlx.ValidString(charset) {
		c.Attrs = append(c.Attrs, &schema.Charset{
			V: charset.String,
		})
	}
	if sqlx.ValidString(collation) {
		c.Attrs = append(c.Attrs, &schema.Collation{
			V: collation.String,
		})
	}
	t.Columns = append(t.Columns, c)
	return nil
}

func columnType(c *columnMeta) schema.Type {
	var typ schema.Type
	switch t := c.typ; t {
	case "bigint", "int8", "integer", "int", "int4", "smallint", "int2":
		typ = &schema.IntegerType{T: t}
	case "bit", "bit varying":
		typ = &BitType{T: t, Len: c.size}
	case "boolean", "bool":
		typ = &schema.BoolType{T: t}
	case "bytea":
		typ = &schema.BinaryType{T: t}
	case "character", "char", "character varying", "varchar", "text":
		// A `character` column without length specifier is equivalent to `character(1)`,
		// but `varchar` without length accepts strings of any size (same as `text`).
		typ = &schema.StringType{T: t, Size: int(c.size)}
	case "cidr", "inet", "macaddr", "macaddr8":
		typ = &NetworkType{T: t}
	case "circle", "line", "lseg", "box", "path", "polygon":
		typ = &schema.SpatialType{T: t}
	case "date", "time", "time with time zone", "time without time zone",
		"timestamp", "timestamp with time zone", "timestamp without time zone":
		typ = &schema.TimeType{T: t}
	case "interval":
		// TODO: get 'interval_type' from query above before implementing.
		typ = &schema.UnsupportedType{T: t}
	case "double precision", "float8", "real", "float4":
		typ = &schema.FloatType{T: t, Precision: int(c.precision)}
	case "json", "jsonb":
		typ = &schema.JSONType{T: t}
	case "money":
		typ = &CurrencyType{T: t}
	case "numeric", "decimal":
		typ = &schema.DecimalType{T: t, Precision: int(c.precision), Scale: int(c.scale)}
	case "smallserial", "serial2", "serial", "serial4", "bigserial", "serial8":
		typ = &SerialType{T: t, Precision: int(c.precision)}
	case "uuid":
		typ = &UUIDType{T: t}
	case "xml":
		typ = &XMLType{T: t}
	case "ARRAY":
		// Note that for ARRAY types, the 'udt_name' column holds the array type
		// prefixed with '_'. For example, for 'integer[]' the result is '_int',
		// and for 'text[N][M]' the result is also '_text'. That's because, the
		// database ignores any size or multi-dimensions constraints.
		typ = &ArrayType{T: strings.TrimPrefix(c.udt, "_")}
	case "USER-DEFINED":
		typ = &UserDefinedType{T: c.udt}
		// The `typtype` column is set to 'e' for enum types, and the
		// values are filled in batch after the rows above is closed.
		// https://www.postgresql.org/docs/current/catalog-pg-type.html
		if c.typtype == "e" {
			typ = &EnumType{T: c.udt, ID: c.typid}
		}
	default:
		typ = &schema.UnsupportedType{T: t}
	}
	return typ
}

// enumValues fills enum columns with their values from the database.
func (d *Driver) enumValues(ctx context.Context, columns []*schema.Column) error {
	var (
		args []interface{}
		ids  = make(map[int64][]*EnumType)
	)
	for _, c := range columns {
		if enum, ok := c.Type.Type.(*EnumType); ok {
			if _, ok := ids[enum.ID]; !ok {
				args = append(args, enum.ID)
			}
			ids[enum.ID] = append(ids[enum.ID], enum)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	query := `SELECT enumtypid, enumlabel FROM pg_enum WHERE enumtypid IN (` + strings.Repeat("?, ", len(ids)-1) + "?)"
	rows, err := d.QueryContext(ctx, query, args...)
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
func (d *Driver) indexes(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, indexesQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q indexes: %w", t.Name, err)
	}
	defer rows.Close()
	if err := d.addIndexes(t, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addIndexes scans the rows and adds the indexes to the table.
func (d *Driver) addIndexes(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			name, typ                            string
			uniq, primary                        bool
			asc, desc, nullsfirst, nullslast     sql.NullBool
			column, contype, pred, expr, comment sql.NullString
		)
		if err := rows.Scan(&name, &typ, &column, &primary, &uniq, &contype, &pred, &expr, &asc, &desc, &nullsfirst, &nullslast, &comment); err != nil {
			return fmt.Errorf("postgres: scanning index: %w", err)
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
			names[name] = idx
			if primary {
				t.PrimaryKey = idx
			} else {
				t.Indexes = append(t.Indexes, idx)
			}
		}
		part := &schema.IndexPart{
			SeqNo: len(idx.Parts) + 1,
			Attrs: []schema.Attr{
				&IndexColumnProperty{
					Asc:        asc.Bool,
					Desc:       desc.Bool,
					NullsFirst: nullsfirst.Bool,
					NullsLast:  nullslast.Bool,
				},
			},
		}
		switch {
		case sqlx.ValidString(expr):
			part.X = &schema.RawExpr{
				X: expr.String,
			}
		case sqlx.ValidString(column):
			part.C, ok = t.Column(column.String)
			if !ok {
				return fmt.Errorf("postgres: column %q was not found for index %q", column.String, idx.Name)
			}
		default:
			return fmt.Errorf("postgres: invalid part for index %q", idx.Name)
		}
		idx.Parts = append(idx.Parts, part)
	}
	return nil
}

// fks queries and appends the foreign keys of the given table.
func (d *Driver) fks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, fksQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("postgres: querying %q foreign keys: %w", t.Name, err)
	}
	defer rows.Close()
	if err := sqlx.ScanFKs(t, rows); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	return rows.Err()
}

// checks queries and appends the check constraints of the given table.
func (d *Driver) checks(ctx context.Context, t *schema.Table) error {
	rows, err := d.QueryContext(ctx, checksQuery, t.Schema.Name, t.Name)
	if err != nil {
		return fmt.Errorf("mysql: querying %q check constraints: %w", t.Name, err)
	}
	defer rows.Close()
	if err := d.addChecks(t, rows); err != nil {
		return err
	}
	return rows.Err()
}

// addChecks scans the rows and adds the checks to the table.
func (d *Driver) addChecks(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*Check)
	for rows.Next() {
		var name, column, clause, indexes string
		if err := rows.Scan(&name, &clause, &column, &indexes); err != nil {
			return fmt.Errorf("postgres: scanning check: %w", err)
		}
		if _, ok := t.Column(column); !ok {
			return fmt.Errorf("postgres: column %q was not found for check %q", column, name)
		}
		check, ok := names[name]
		if !ok {
			check = &Check{Name: name, Clause: clause}
			names[name] = check
			t.Attrs = append(t.Attrs, check)
		}
		check.Columns = append(check.Columns, column)
	}
	return nil
}

// schemas returns the list of the schemas in the database.
func (d *Driver) schemas(ctx context.Context, opts *schema.InspectRealmOption) ([]*schema.Schema, error) {
	var (
		args  []interface{}
		query = schemasQuery + " WHERE schema_name "
	)
	if opts != nil && len(opts.Schemas) > 0 {
		query, args = inStrings(opts.Schemas, query, args)
	} else {
		query += "NOT IN ('information_schema', 'pg_catalog', 'pg_toast')"
	}
	rows, err := d.QueryContext(ctx, query, args...)
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
	return schemas, nil
}

// tableNames returns a list of all tables exist in the schema.
func (d *Driver) tableNames(ctx context.Context, schema string, opts *schema.InspectOptions) ([]string, error) {
	query, args := tablesQuery, []interface{}{schema}
	if opts != nil && len(opts.Tables) > 0 {
		query += " AND table_name "
		query, args = inStrings(opts.Tables, query, args)
	}
	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: querying schema tables: %w", err)
	}
	names, err := sqlx.ScanStrings(rows)
	if err != nil {
		return nil, fmt.Errorf("postgres: scanning table names: %w", err)
	}
	return names, nil
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

	// EnumType represents an enum type.
	EnumType struct {
		schema.Type
		T      string // Type name.
		ID     int64  // Type id.
		Values []string
	}

	// ArrayType defines an array type.
	// https://www.postgresql.org/docs/current/arrays.html
	ArrayType struct {
		schema.Type
		T string
	}

	// BitType defines a bit type.
	// https://www.postgresql.org/docs/current/datatype-bit.html
	BitType struct {
		schema.Type
		T   string
		Len int64
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
		Asc        bool
		Desc       bool
		NullsFirst bool
		NullsLast  bool
	}

	// Check attributes defines a CHECK constraint.
	Check struct {
		schema.Attr
		Name    string
		Clause  string
		Columns []string
	}
)

const (
	// Query to list runtime parameters.
	paramsQuery = `SELECT setting FROM pg_settings WHERE name IN ('lc_collate', 'lc_ctype', 'server_version_num') ORDER BY name`

	// Query to list database schemas.
	schemasQuery = "SELECT schema_name FROM information_schema.schemata"

	// Query to list schema tables.
	tablesQuery = "SELECT table_name FROM information_schema.tables WHERE table_schema = $1"

	// Query to list table information.
	tableQuery = `
SELECT
	t1.table_schema,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	information_schema.tables AS t1
	INNER JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.table_type = 'BASE TABLE'
	AND t1.table_name = $1
	AND t1.table_schema = (CURRENT_SCHEMA())
`
	tableSchemaQuery = `
SELECT
	t1.TABLE_SCHEMA,
	pg_catalog.obj_description(t2.oid, 'pg_class') AS COMMENT
FROM
	INFORMATION_SCHEMA.TABLES AS t1
	JOIN pg_catalog.pg_class AS t2
	ON t1.table_name = t2.relname
WHERE
	t1.TABLE_TYPE = 'BASE TABLE'
	AND t1.TABLE_NAME = $1
	AND t1.TABLE_SCHEMA = $2
`
	// Query to list table columns.
	columnsQuery = `
SELECT
	t1.column_name,
	t1.data_type,
	t1.is_nullable,
	t1.column_default,
	t1.character_maximum_length,
	t1.numeric_precision,
	t1.numeric_scale,
	t1.character_set_name,
	t1.collation_name,
	t1.udt_name,
	t1.is_identity,
	col_description(to_regclass("table_schema" || '.' || "table_name")::oid, "ordinal_position") AS comment,
	t2.typtype,
	t2.oid
FROM
	"information_schema"."columns" AS t1
	LEFT JOIN pg_catalog.pg_type AS t2
	ON t1.udt_name = t2.typname
WHERE
	TABLE_SCHEMA = $1 AND TABLE_NAME = $2
`

	// Query to list table indexes.
	indexesQuery = `
SELECT
	i.relname AS index_name,
	am.amname AS index_type,
	a.attname AS column_name,
	idx.indisprimary AS primary,
	idx.indisunique AS unique,
	c.contype AS constraint_type,
	pg_get_expr(idx.indpred, idx.indrelid) AS predicate,
	pg_get_expr(idx.indexprs, idx.indrelid) AS expression,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'asc') AS asc,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'desc') AS desc,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'nulls_first') AS nulls_first,
	pg_index_column_has_property(idx.indexrelid, a.attnum, 'nulls_last') AS nulls_last,
	obj_description(to_regclass($1 || i.relname)::oid) AS comment
FROM
	pg_index idx
	JOIN pg_class i
	ON i.oid = idx.indexrelid
	LEFT JOIN pg_constraint c
	ON idx.indexrelid = c.conindid
	LEFT JOIN pg_attribute a
	ON a.attrelid = idx.indexrelid
	JOIN pg_am am
	ON am.oid = i.relam
WHERE
	idx.indrelid = to_regclass($1 || '.' || $2)::oid
	AND COALESCE(c.contype, '') <> 'f'
ORDER BY
	index_name, a.attnum
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
    AND t1.table_name = $2
ORDER BY
    t1.constraint_name,
    t2.ordinal_position
`

	// Query to list table check constraints.
	checksQuery = `
SELECT
	t1.conname AS constraint_name,
	pg_get_expr(t1.conbin, to_regclass($1 || '.' || $2)::oid) as expression,
	t2.attname as column_name,
	t1.conkey as column_indexes
FROM
	pg_catalog.pg_constraint t1
	JOIN pg_attribute t2
	ON t2.attrelid = t1.conrelid
	AND t2.attnum = ANY (t1.conkey)
WHERE
	t1.contype = 'c'
	AND t1.conrelid = to_regclass($1 || '.' || $2)::oid
ORDER BY
	t1.conname, array_position(t1.conkey, t2.attnum)
`
)

func inStrings(s []string, query string, args []interface{}) (string, []interface{}) {
	query += "IN ("
	for i := range s {
		args = append(args, s[i])
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", len(args))
	}
	query += ")"
	return query, args
}
