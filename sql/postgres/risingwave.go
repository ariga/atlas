// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
)

type (
	risingwaveDiff    struct{ diff }
	risingwaveInspect struct{ inspect }
)

var _ sqlx.DiffDriver = (*risingwaveDiff)(nil)

func (c *conn) isRisingWaveConn() (bool, error) {
	rows, err := c.QueryContext(context.Background(), "select version()")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var version string
	err = sqlx.ScanOne(rows, &version)
	if err != nil {
		return false, fmt.Errorf("postgres: failed scanning rows: %w", err)
	}
	return strings.Contains(strings.ToLower(version), "risingwave"), nil
}

func (i *inspect) risingwaveIndexes(ctx context.Context, s *schema.Schema) error {
	rows, err := i.querySchema(ctx, risingwaveIndexesQuery, s)
	if err != nil {
		return fmt.Errorf("postgres: querying schema %q indexes: %w", s.Name, err)
	}
	defer rows.Close()
	if err := i.risingwaveAddIndexes(s, rows); err != nil {
		return err
	}
	return rows.Err()
}

// RisingWave doesn't support:
// - Unique Indexes
// - Index Constraints
// - Partial Indexes or Partial Index Predicates
// - Indexes on Expressions
func (i *inspect) risingwaveAddIndexes(s *schema.Schema, rows *sql.Rows) error {
	names := make(map[string]*schema.Index)
	for rows.Next() {
		var (
			primary                     bool
			table, name                 string
			desc, nullsfirst, nullslast sql.NullBool
			column, comment             sql.NullString
		)
		if err := rows.Scan(&table, &name, &column, &primary, &desc, &nullsfirst, &nullslast, &comment); err != nil {
			return fmt.Errorf("risingwave: scanning indexes for schema %q: %w", s.Name, err)
		}

		t, ok := s.Table(table)
		if !ok {
			return fmt.Errorf("table %q was not found in schema", table)
		}

		idx, ok := names[name]
		if !ok {
			idx = &schema.Index{
				Name:   name,
				Unique: primary,
				Table:  t,
			}
			if sqlx.ValidString(comment) {
				idx.Attrs = append(idx.Attrs, &schema.Comment{Text: comment.String})
			}
			if primary {
				t.PrimaryKey = idx
			} else {
				t.Indexes = append(t.Indexes, idx)
			}
			names[name] = idx
		}
		// TODO: Extract isdesc from RisingWave indexes.
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
				return fmt.Errorf("risingwave: column %q was not found for index %q", column.String, idx.Name)
			}
			part.C.Indexes = append(part.C.Indexes, idx)
		default:
			return fmt.Errorf("risingwave: invalid part for index %q", idx.Name)
		}
		idx.Parts = append(idx.Parts, part)
	}
	return nil
}

const (
	/// table, name, typ, column, primary, comment
	risingwaveIndexesQuery = `
SELECT
	t.relname AS table_name,
	i.relname AS index_name,
	a.attname AS column_name,
	idx.indisprimary AS primary,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'desc') AS isdesc,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_first') AS nulls_first,
	pg_index_column_has_property(idx.indexrelid, idx.ord, 'nulls_last') AS nulls_last,
	obj_description(i.oid, 'pg_class') AS comment
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
WHERE
	n.nspname = $1
	AND t.relname IN (%s)
ORDER BY
	table_name, index_name, idx.ord
`
)
