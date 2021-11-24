package migratespec

import (
	"ariga.io/atlas/schema/schemaspec"
	"ariga.io/atlas/sql/sqlspec"
)

type (
	Change interface {
		c()
	}
	ModifyTable struct {
		Change
		Table   string   `spec:"table"`
		Changes []Change `spec:""`
	}
	AddColumn struct {
		Change
		Column *sqlspec.Column `spec:"column"`
	}
)

func init() {
	schemaspec.Register("modify_table", &ModifyTable{})
	schemaspec.Register("add_column", &AddColumn{})
}
