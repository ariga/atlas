// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package mysqlversion

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/internal/sqlx"
	"ariga.io/atlas/sql/schema"
	"golang.org/x/mod/semver"
)

// V provides information about MySQL versions.
type V string

// SupportsCheck reports if the version supports the CHECK
// clause, and return the querying for getting them.
func (v V) SupportsCheck() bool {
	u := "8.0.16"
	if v.Maria() {
		u = "10.2.1"
	}
	return v.GTE(u)
}

// SupportsIndexExpr reports if the version supports
// index expressions (functional key part).
func (v V) SupportsIndexExpr() bool {
	return !v.Maria() && v.GTE("8.0.13")
}

// SupportsDisplayWidth reports if the version supports getting
// the display width information from the information schema.
func (v V) SupportsDisplayWidth() bool {
	// MySQL v8.0.19 dropped the display width
	// information from the information schema
	return v.Maria() || v.LT("8.0.19")
}

// SupportsExprDefault reports if the version supports
// expressions in the DEFAULT clause on column definition.
func (v V) SupportsExprDefault() bool {
	u := "8.0.13"
	if v.Maria() {
		u = "10.2.1"
	}
	return v.GTE(u)
}

// SupportsEnforceCheck reports if the version supports
// the ENFORCED option in CHECK constraint syntax.
func (v V) SupportsEnforceCheck() bool {
	return !v.Maria() && v.GTE("8.0.16")
}

// SupportsGeneratedColumns reports if the version supports
// the generated columns in information schema.
func (v V) SupportsGeneratedColumns() bool {
	u := "5.7"
	if v.Maria() {
		u = "10.2"
	}
	return v.GTE(u)
}

// SupportsRenameColumn reports if the version supports
// the "RENAME COLUMN" clause.
func (v V) SupportsRenameColumn() bool {
	u := "8"
	if v.Maria() {
		u = "10.5.2"
	}
	return v.GTE(u)
}

// SupportsIndexComment reports if the version
// supports comments on indexes.
func (v V) SupportsIndexComment() bool {
	// According to Oracle release notes, comments on
	// indexes were added in version 5.5.3.
	return v.Maria() || v.GTE("5.5.3")
}

// SupportsViewUsage reports if the version supports
// querying the VIEW_TABLE_USAGE table.
func (v V) SupportsViewUsage() bool {
	return !v.Maria() && v.GTE("8.0.13")
}

// CharsetToCollate returns the mapping from charset to its default collation.
func (v V) CharsetToCollate(conn schema.ExecQuerier) (map[string]string, error) {
	name := "is/charset2collate"
	if v.Maria() {
		name += ".maria"
	}
	c2c, err := decode(name)
	if err != nil {
		return nil, err
	}
	if conn != nil {
		mayExtend(conn, "SELECT CHARACTER_SET_NAME, DEFAULT_COLLATE_NAME FROM INFORMATION_SCHEMA.CHARACTER_SETS", c2c)
	}
	return c2c, nil
}

// CollateToCharset returns the mapping from a collation to its charset.
func (v V) CollateToCharset(conn schema.ExecQuerier) (map[string]string, error) {
	name := "is/collate2charset"
	if v.Maria() {
		name += ".maria"
	}
	c2c, err := decode(name)
	if err != nil {
		return nil, err
	}
	if conn != nil {
		mayExtend(conn, "SELECT COLLATION_NAME, CHARACTER_SET_NAME FROM INFORMATION_SCHEMA.COLLATIONS", c2c)
	}
	return c2c, nil
}

// mayExtend the given collation/charset map from information schema.
func mayExtend(conn schema.ExecQuerier, query string, to map[string]string) {
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c1, c2 sql.NullString
		if err := rows.Scan(&c1, &c2); err != nil {
			return
		}
		if sqlx.ValidString(c1) && sqlx.ValidString(c2) {
			to[c1.String] = c2.String
		}
	}
}

// Maria reports if the MySQL version is MariaDB.
func (v V) Maria() bool {
	return strings.Index(string(v), "MariaDB") > 0
}

// TiDB reports if the MySQL version is TiDB.
func (v V) TiDB() bool {
	return strings.Index(string(v), "TiDB") > 0
}

// Compare returns an integer comparing two versions according to
// semantic version precedence.
func (v V) Compare(w string) int {
	u := string(v)
	switch idx := strings.Index(u, "-"); {
	case v.Maria():
		u = u[:strings.Index(u, "MariaDB")-1]
	case v.TiDB():
		u = u[:strings.Index(u, "TiDB")-1]
	case idx > 0:
		// Remove server build information, if any.
		u = u[:idx]
	}
	return semver.Compare("v"+u, "v"+w)
}

// GTE reports if the version is >= w.
func (v V) GTE(w string) bool { return v.Compare(w) >= 0 }

// LT reports if the version is < w.
func (v V) LT(w string) bool { return v.Compare(w) == -1 }

//go:embed is/*
var encoding embed.FS

func decode(name string) (map[string]string, error) {
	f, err := encoding.Open(name)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode %q", name)
	}
	return m, nil
}
