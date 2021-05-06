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

	// Changeset describes a collection of schema change that are
	// applied (and can be reverted) as a group.
	Changeset []Change

	// AddTable describes a table creation change.
	AddTable struct {
		T     *Table
		Attrs []Attr
	}

	// DropTable describes a table removal change.
	DropTable struct {
		T     *Table
		Attrs []Attr
	}

	// ModifyTable holds a set of schema changes that modify a table
	// or elements that associate with it.
	ModifyTable struct {
		Columns     []Change
		Indexes     []Change
		ForeignKeys []Change
	}

	// AddColumn describes a column creation change.
	AddColumn struct {
		C     *Column
		Attrs []Attr
	}

	// DropColumn describes a column removal change.
	DropColumn struct {
		C     *Column
		Attrs []Attr
	}

	// DropColumn describes a change that modifies a column.
	ModifyColumn struct {
		From, To *Column
		Attrs    []Attr
		Reason   string
	}

	// AddIndex describes an index creation change.
	AddIndex struct {
		I     *Index
		Attrs []Attr
	}

	// DropIndex describes an index removal change.
	DropIndex struct {
		I     *Index
		Attrs []Attr
	}

	// ModifyIndex describes an index modification.
	ModifyIndex struct {
		I      *Index
		Attrs  []Attr
		Reason string
	}

	// AddForeignKey describes a foreign-key creation change.
	AddForeignKey struct {
		F     *ForeignKey
		Attrs []Attr
	}

	// DropForeignKey describes a foreign-key removal change.
	DropForeignKey struct {
		F     *ForeignKey
		Attrs []Attr
	}

	// ModifyForeignKey describes a change that modifies a foreign-key.
	ModifyForeignKey struct {
		From, To *ForeignKey
		Attrs    []Attr
		Reason   string
	}
)

type (
	// DiffReport defines a report for schema diffing.
	DiffReport struct {
		Changes    Changeset
		Compatible bool
	}

	// TableDiffer is the interface implemented by the different
	// migration drivers for comparing and diffing tables.
	TableDiffer interface {
		// TableDiff returns a diff report for migrating a table
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		TableDiff(from, to *Table) (*DiffReport, error)

		// TablesDiff returns a diff report for migrating a table
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		TablesDiff(from, to []*Table) (*DiffReport, error)
	}
)

// changes.
func (*AddTable) change()         {}
func (*DropTable) change()        {}
func (*AddIndex) change()         {}
func (*DropIndex) change()        {}
func (*ModifyIndex) change()      {}
func (*AddColumn) change()        {}
func (*DropColumn) change()       {}
func (*ModifyColumn) change()     {}
func (*AddForeignKey) change()    {}
func (*DropForeignKey) change()   {}
func (*ModifyForeignKey) change() {}
