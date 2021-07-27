package sqlx

import (
	"database/sql"
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// ValidString reports if the given string is not null and valid.
func ValidString(s sql.NullString) bool {
	return s.Valid && s.String != "" && strings.ToLower(s.String) != "null"
}

// ScanFKs scans the rows and adds the foreign-key to the table.
// Reference elements are added as stubs and should be linked
// manually by the caller.
func ScanFKs(t *schema.Table, rows *sql.Rows) error {
	names := make(map[string]*schema.ForeignKey)
	for rows.Next() {
		var name, table, column, tSchema, refTable, refColumn, refSchema, updateRule, deleteRule string
		if err := rows.Scan(&name, &table, &column, &tSchema, &refTable, &refColumn, &refSchema, &updateRule, &deleteRule); err != nil {
			return err
		}
		fk, ok := names[name]
		if !ok {
			fk = &schema.ForeignKey{
				Symbol:   name,
				Table:    t,
				RefTable: t,
				OnDelete: schema.ReferenceOption(deleteRule),
				OnUpdate: schema.ReferenceOption(updateRule),
			}
			if refTable != t.Name || tSchema != refSchema {
				fk.RefTable = &schema.Table{Name: refTable, Schema: &schema.Schema{Name: refSchema}}
			}
			names[name] = fk
			t.ForeignKeys = append(t.ForeignKeys, fk)
		}
		c, ok := t.Column(column)
		if !ok {
			return fmt.Errorf("column %q was not found for fk %q", column, fk.Symbol)
		}
		// Rows are ordered by ORDINAL_POSITION that specifies
		// the position of the column in the FK definition.
		if _, ok := fk.Column(c.Name); !ok {
			fk.Columns = append(fk.Columns, c)
			c.ForeignKeys = append(c.ForeignKeys, fk)
		}

		// Stub referenced columns or link if it's a self-reference.
		var rc *schema.Column
		if fk.Table != fk.RefTable {
			rc = &schema.Column{Name: refColumn}
		} else if c, ok := t.Column(refColumn); ok {
			rc = c
		} else {
			return fmt.Errorf("referenced column %q was not found for fk %q", refColumn, fk.Symbol)
		}
		if _, ok := fk.RefColumn(rc.Name); !ok {
			fk.RefColumns = append(fk.RefColumns, rc)
		}
	}
	return nil
}

// LinkSchemaTables links foreign-key stub tables/columns to actual elements.
func LinkSchemaTables(schemas []*schema.Schema) {
	byName := make(map[string]map[string]*schema.Table)
	for _, s := range schemas {
		byName[s.Name] = make(map[string]*schema.Table)
		for _, t := range s.Tables {
			t.Schema = s
			byName[s.Name][t.Name] = t
		}
	}
	for _, s := range schemas {
		for _, t := range s.Tables {
			for _, fk := range t.ForeignKeys {
				rs, ok := byName[fk.RefTable.Schema.Name]
				if !ok {
					continue
				}
				ref, ok := rs[fk.RefTable.Name]
				if !ok {
					continue
				}
				fk.RefTable = ref
				for i, c := range fk.RefColumns {
					rc, ok := ref.Column(c.Name)
					if ok {
						fk.RefColumns[i] = rc
					}
				}
			}
		}
	}
}

// ScanStrings scans sql.Rows into a slice of strings and closes it at the end.
func ScanStrings(rows *sql.Rows) ([]string, error) {
	defer rows.Close()
	var vs []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

// ValuesEqual checks if the 2 string slices are equal (including their order).
func ValuesEqual(v1, v2 []string) bool {
	if len(v1) != len(v2) {
		return false
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			return false
		}
	}
	return true
}

// VersionPermutations returns permutations of the dialect version sorted
// from coarse to fine grained. For example:
//
//   VersionPermutations("mysql", "1.2.3") => ["mysql", "mysql 1", "mysql 1.2", "mysql 1.2.3"]
//
// VersionPermutations will split the version number by ".", " ", "-" or "_", and rejoin them
// with ".". The output slice can be used by drivers to generate a list of permutations
// for searching for relevant overrides in schema element specs.
func VersionPermutations(dialect, version string) []string {
	parts := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == ' ' || r == '-' || r == '_'
	})
	names := []string{dialect}
	for i := range parts {
		version := strings.Join(parts[0:i+1], ".")
		names = append(names, dialect+" "+version)
	}
	return names
}
