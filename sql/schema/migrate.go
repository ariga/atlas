package schema

type (
	// A Change represents a schema change. The types below implement this
	// interface and can be used for describing schema changes.
	//
	// The Change interface can also be implemented outside this package
	// as follows:
	//
	//	type RenameType struct {
	//		schema.Change
	//		From, To string
	//	}
	//
	//	var t schema.Change = &RenameType{From: "old", To: "new"}
	//
	Change interface {
		change()
	}

	// AddTable describes a table creation change.
	AddTable struct {
		T     *Table
		Extra []Expr // Extra clauses and options.
	}

	// DropTable describes a table removal change.
	DropTable struct {
		T     *Table
		Extra []Expr // Extra clauses.
	}

	// ModifyTable describes a table modification change.
	ModifyTable struct {
		T       *Table
		Changes []Change
	}

	// AddColumn describes a column creation change.
	AddColumn struct {
		C *Column
	}

	// DropColumn describes a column removal change.
	DropColumn struct {
		C *Column
	}

	// ModifyColumn describes a change that modifies a column.
	ModifyColumn struct {
		From, To *Column
		Change   ChangeKind
	}

	// AddIndex describes an index creation change.
	AddIndex struct {
		I *Index
	}

	// DropIndex describes an index removal change.
	DropIndex struct {
		I *Index
	}

	// ModifyIndex describes an index modification.
	ModifyIndex struct {
		From, To *Index
		Change   ChangeKind
	}

	// AddForeignKey describes a foreign-key creation change.
	AddForeignKey struct {
		F *ForeignKey
	}

	// DropForeignKey describes a foreign-key removal change.
	DropForeignKey struct {
		F *ForeignKey
	}

	// ModifyForeignKey describes a change that modifies a foreign-key.
	ModifyForeignKey struct {
		From, To *ForeignKey
		Change   ChangeKind
	}

	// AddAttr describes an attribute addition.
	AddAttr struct {
		A Attr
	}

	// DropAttr describes an attribute removal.
	DropAttr struct {
		A Attr
	}

	// ModifyAttr describes a change that modifies an element attribute.
	ModifyAttr struct {
		From, To Attr
	}
)

// A ChangeKind describes a change kind that can be combined
// using a set of flags. The zero kind is no change.
type ChangeKind uint

const (
	// Basic changes.
	NoChange ChangeKind = 0
	ChangeAttr

	// Table and common changes.
	ChangeCharset ChangeKind = 1 << iota
	ChangeCollation
	ChangeComment

	// Column specific changes.
	ChangeNull
	ChangeType
	ChangeDefault

	// Index specific changes.
	ChangeUnique
	ChangeParts

	// Foreign key specific changes.
	ChangeColumn
	ChangeRefTable
	ChangeRefColumn
	ChangeUpdateAction
	ChangeDeleteAction
)

// Is reports whether c is match the given change kind.
func (k ChangeKind) Is(c ChangeKind) bool { return k&c != 0 }

type (
	// Differ is the interface implemented by the different
	// migration drivers for comparing and diffing schemas.
	Differ interface {
		// SchemaDiff returns a diff report for migrating a schema
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		SchemaDiff(from, to *Schema) ([]Change, error)
	}

	// TableDiffer is the interface implemented by the different
	// migration drivers for comparing and diffing tables.
	TableDiffer interface {
		// TableDiff returns a diff report for migrating a table
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		TableDiff(from, to *Table) ([]Change, error)
	}
)

// changes.
func (*AddAttr) change()          {}
func (*DropAttr) change()         {}
func (*ModifyAttr) change()       {}
func (*AddTable) change()         {}
func (*DropTable) change()        {}
func (*ModifyTable) change()      {}
func (*AddIndex) change()         {}
func (*DropIndex) change()        {}
func (*ModifyIndex) change()      {}
func (*AddColumn) change()        {}
func (*DropColumn) change()       {}
func (*ModifyColumn) change()     {}
func (*AddForeignKey) change()    {}
func (*DropForeignKey) change()   {}
func (*ModifyForeignKey) change() {}
