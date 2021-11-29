package sqlspec

import (
	"ariga.io/atlas/schema/schemaspec"
)

type (
	// Change is the interface implemented by change specifications. Change instances are supposed
	// to be mappable to schema.Change instances.
	Change interface {
		change()
	}

	// AddTable is a specification for a schema.AddTable.
	AddTable struct {
		Change
		Table *Table `spec:"table"`
	}

	// DropTable is a specification for a schema.DropTable.
	DropTable struct {
		Change
		Table string `spec:"table"`
	}

	// ModifyTable is a specification for a schema.ModifyTable.
	ModifyTable struct {
		Change
		Table   string   `spec:"table"`
		Changes []Change `spec:""`
	}

	// AddColumn is a specification for a schema.AddColumn.
	AddColumn struct {
		Change
		Column *Column `spec:"column"`
	}
)

func init() {
	schemaspec.Register("modify_table", &ModifyTable{})
	schemaspec.Register("add_column", &AddColumn{})
	schemaspec.Register("add_table", &AddTable{})
	schemaspec.Register("drop_table", &DropTable{})
}
