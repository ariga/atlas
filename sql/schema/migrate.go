// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package schema

import (
	"context"
	"errors"
	"reflect"
	"time"
)

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

	// Clause carries additional information that can be added
	// to schema changes. The Clause interface can be implemented
	// outside this package as follows:
	//
	//	type Authorization struct {
	//		schema.Clause
	//		UserName string
	//	}
	//
	//	var c schema.Clause = &Authorization{UserName: "a8m"}
	//
	Clause interface {
		clause()
	}

	// AddSchema describes a schema (named database) creation change.
	// Unlike table creation, schemas and their elements are described
	// with separate changes. For example, "AddSchema" and "AddTable"
	AddSchema struct {
		S     *Schema
		Extra []Clause // Extra clauses and options.
	}

	// DropSchema describes a schema (named database) removal change.
	DropSchema struct {
		S     *Schema
		Extra []Clause // Extra clauses and options.
	}

	// ModifySchema describes a modification change for schema attributes.
	ModifySchema struct {
		S       *Schema
		Changes []Change
	}

	// AddTable describes a table creation change.
	AddTable struct {
		T     *Table
		Extra []Clause // Extra clauses and options.
	}

	// DropTable describes a table removal change.
	DropTable struct {
		T     *Table
		Extra []Clause // Extra clauses.
	}

	// ModifyTable describes a table modification change.
	ModifyTable struct {
		T       *Table
		Changes []Change
	}

	// RenameTable describes a table rename change.
	RenameTable struct {
		From, To *Table
	}

	// AddView describes a view creation change.
	AddView struct {
		V     *View
		Extra []Clause // Extra clauses and options.
	}

	// DropView describes a view removal change.
	DropView struct {
		V     *View
		Extra []Clause // Extra clauses.
	}

	// ModifyView describes a view modification change.
	ModifyView struct {
		From, To *View
		// Changes that are extra to the view definition.
		// For example, adding or dropping indexes.
		Changes []Change
	}

	// RenameView describes a view rename change.
	RenameView struct {
		From, To *View
	}

	// AddFunc describes a function creation change.
	AddFunc struct {
		F     *Func
		Extra []Clause // Extra clauses and options.
	}

	// DropFunc describes a function removal change.
	DropFunc struct {
		F     *Func
		Extra []Clause // Extra clauses.
	}

	// ModifyFunc describes a function modification change.
	ModifyFunc struct {
		From, To *Func
		// Changes that are extra to the function definition.
		// For example, adding, dropping, or modifying attributes.
		Changes []Change
	}

	// RenameFunc describes a function rename change.
	RenameFunc struct {
		From, To *Func
	}

	// AddProc describes a procedure creation change.
	AddProc struct {
		P     *Proc
		Extra []Clause // Extra clauses and options.
	}

	// DropProc describes a procedure removal change.
	DropProc struct {
		P     *Proc
		Extra []Clause // Extra clauses.
	}

	// ModifyProc describes a procedure modification change.
	ModifyProc struct {
		From, To *Proc
		// Changes that are extra to the procedure definition.
		// For example, adding, dropping, or modifying attributes.
		Changes []Change
	}

	// RenameProc describes a procedure rename change.
	RenameProc struct {
		From, To *Proc
	}

	// AddObject describes a generic object creation change.
	AddObject struct {
		O     Object
		Extra []Clause // Extra clauses and options.
	}

	// DropObject describes a generic object removal change.
	DropObject struct {
		O     Object
		Extra []Clause // Extra clauses.
	}

	// ModifyObject describes a generic object modification change.
	// Unlike tables changes, the diffing types are implemented by
	// the underlying driver.
	ModifyObject struct {
		From, To Object
	}

	// RenameObject describes a generic object rename change.
	RenameObject struct {
		From, To Object
	}

	// AddTrigger describes a trigger creation change.
	AddTrigger struct {
		T     *Trigger
		Extra []Clause // Extra clauses and options.
	}

	// DropTrigger describes a trigger removal change.
	DropTrigger struct {
		T     *Trigger
		Extra []Clause // Extra clauses.
	}

	// ModifyTrigger describes a trigger modification change.
	ModifyTrigger struct {
		From, To *Trigger
		// Changes that are extra to the trigger definition.
		// For example, adding, dropping, or modifying attributes.
		Changes []Change
	}

	// RenameTrigger describes a trigger rename change.
	RenameTrigger struct {
		From, To *Trigger
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
		Extra    []Clause // Extra clauses and options.
	}

	// RenameColumn describes a column rename change.
	RenameColumn struct {
		From, To *Column
	}

	// AddIndex describes an index creation change.
	AddIndex struct {
		I     *Index
		Extra []Clause // Extra clauses and options.
	}

	// DropIndex describes an index removal change.
	DropIndex struct {
		I     *Index
		Extra []Clause // Extra clauses and options.
	}

	// ModifyIndex describes an index modification.
	ModifyIndex struct {
		From, To *Index
		Change   ChangeKind
	}

	// RenameIndex describes an index rename change.
	RenameIndex struct {
		From, To *Index
	}

	// AddPrimaryKey describes a primary-key creation change.
	AddPrimaryKey struct {
		P *Index
	}

	// DropPrimaryKey describes a primary-key removal change.
	DropPrimaryKey struct {
		P *Index
	}

	// ModifyPrimaryKey describes a primary-key modification.
	ModifyPrimaryKey struct {
		From, To *Index
		Change   ChangeKind
	}

	// AddForeignKey describes a foreign-key creation change.
	AddForeignKey struct {
		F     *ForeignKey
		Extra []Clause // Extra clauses and options.
	}

	// DropForeignKey describes a foreign-key removal change.
	DropForeignKey struct {
		F *ForeignKey
		Extra []Clause // Extra clauses and options.
	}

	// ModifyForeignKey describes a change that modifies a foreign-key.
	ModifyForeignKey struct {
		From, To *ForeignKey
		Change   ChangeKind
	}

	// AddCheck describes a CHECK constraint creation change.
	AddCheck struct {
		C     *Check
		Extra []Clause // Extra clauses and options.
	}

	// DropCheck describes a CHECK constraint removal change.
	DropCheck struct {
		C *Check
	}

	// ModifyCheck describes a change that modifies a check.
	ModifyCheck struct {
		From, To *Check
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

	// IfExists represents a clause in a schema change that is commonly
	// supported by multiple statements (e.g. DROP TABLE or DROP SCHEMA).
	IfExists struct{}

	// IfNotExists represents a clause in a schema change that is commonly
	// supported by multiple statements (e.g. CREATE TABLE or CREATE SCHEMA).
	IfNotExists struct{}
)

// A ChangeKind describes a change kind that can be combined
// using a set of flags. The zero kind is no change.
//
//go:generate stringer -type ChangeKind
type ChangeKind uint

const (
	// NoChange holds the zero value of a change kind.
	NoChange ChangeKind = 0

	// Common changes.

	// ChangeAttr describes attributes change of an element.
	// For example, a table CHECK was added or changed.
	ChangeAttr ChangeKind = 1 << (iota - 1)
	// ChangeCharset describes character-set change.
	ChangeCharset
	// ChangeCollate describes collation/encoding change.
	ChangeCollate
	// ChangeComment describes comment chang (of any element).
	ChangeComment

	// Column specific changes.

	// ChangeNull describe a change to the NULL constraint.
	ChangeNull
	// ChangeType describe a column type change.
	ChangeType
	// ChangeDefault describe a column default change.
	ChangeDefault
	// ChangeGenerated describe a change to the generated expression.
	ChangeGenerated

	// Index specific changes.

	// ChangeUnique describes a change to the uniqueness constraint.
	// For example, an index was changed from non-unique to unique.
	ChangeUnique
	// ChangeParts describes a change to one or more of the index parts.
	// For example, index keeps its previous name, but the columns order
	// was changed.
	ChangeParts

	// Foreign key specific changes.

	// ChangeColumn describes a change to the foreign-key (child) columns.
	ChangeColumn
	// ChangeRefColumn describes a change to the foreign-key (parent) columns.
	ChangeRefColumn
	// ChangeRefTable describes a change to the foreign-key (parent) table.
	ChangeRefTable
	// ChangeUpdateAction describes a change to the foreign-key update action.
	ChangeUpdateAction
	// ChangeDeleteAction describes a change to the foreign-key delete action.
	ChangeDeleteAction
)

// Is reports whether c is match the given change kind.
func (k ChangeKind) Is(c ChangeKind) bool {
	return k == c || k&c != 0
}

type (
	// Differ is the interface implemented by the different
	// drivers for comparing and diffing schema top elements.
	Differ interface {
		// RealmDiff returns a diff report for migrating a realm
		// (or a database) from state "from" to state "to". An error
		// is returned if such step is not possible.
		RealmDiff(from, to *Realm, opts ...DiffOption) ([]Change, error)

		// SchemaDiff returns a diff report for migrating a schema
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		SchemaDiff(from, to *Schema, opts ...DiffOption) ([]Change, error)

		// TableDiff returns a diff report for migrating a table
		// from state "from" to state "to". An error is returned
		// if such step is not possible.
		TableDiff(from, to *Table, opts ...DiffOption) ([]Change, error)
	}

	// DiffOptions defines the standard and per-driver configuration
	// for the schema diffing process.
	DiffOptions struct {
		// SkipChanges defines a list of change types to skip.
		SkipChanges []Change

		// Extra defines per-driver configuration. If not
		// nil, should be set to schemahcl.Extension.
		Extra any // avoid circular dependency with schemahcl.

		// AskFunc can be implemented by the caller to
		// make diff process interactive.
		AskFunc func(string, []string) (string, error)
	}

	// DiffOption allows configuring the DiffOptions using functional options.
	DiffOption func(*DiffOptions)
)

// NewDiffOptions creates a new DiffOptions from the given configuration.
func NewDiffOptions(opts ...DiffOption) *DiffOptions {
	o := &DiffOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// DiffSkipChanges returns a DiffOption that skips the given change types.
// For example, in order to skip all destructive changes, use:
//
//	DiffSkipChanges(&DropSchema{}, &DropTable{}, &DropColumn{}, &DropIndex{}, &DropForeignKey{})
func DiffSkipChanges(changes ...Change) DiffOption {
	return func(o *DiffOptions) {
		o.SkipChanges = append(o.SkipChanges, changes...)
	}
}

// Skipped reports whether the given change should be skipped.
func (o *DiffOptions) Skipped(c Change) bool {
	for _, s := range o.SkipChanges {
		if reflect.TypeOf(c) == reflect.TypeOf(s) {
			return true
		}
	}
	return false
}

// AddOrSkip adds the given change to the list of changes if it is not skipped.
func (o *DiffOptions) AddOrSkip(changes Changes, cs ...Change) Changes {
	for _, c := range cs {
		if !o.Skipped(c) {
			changes = append(changes, c)
		}
	}
	return changes

}

// ErrLocked is returned on Lock calls which have failed to obtain the lock.
var ErrLocked = errors.New("sql/schema: lock is held by other session")

type (
	// UnlockFunc is returned by the Locker to explicitly
	// release the named "advisory lock".
	UnlockFunc func() error

	// Locker is an interface that is optionally implemented by the different drivers
	// for obtaining an "advisory lock" with the given name.
	Locker interface {
		// Lock acquires a named "advisory lock", using the given timeout. Negative value means no timeout,
		// and the zero value means a "try lock" mode. i.e. return immediately if the lock is already taken.
		// The returned unlock function is used to release the advisory lock acquired by the session.
		//
		// An ErrLocked is returned if the operation failed to obtain the lock in all different timeout modes.
		Lock(ctx context.Context, name string, timeout time.Duration) (UnlockFunc, error)
	}
)

// Changes is a list of changes allow for searching and mutating changes.
type Changes []Change

// IndexAddTable returns the index of the first AddTable in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexAddTable(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*AddTable)
		return ok && a.T.Name == name
	})
}

// IndexDropTable returns the index of the first DropTable in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexDropTable(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*DropTable)
		return ok && a.T.Name == name
	})
}

// LastIndexAddTable returns the index of the last AddTable in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) LastIndexAddTable(name string) int {
	return c.rsearch(func(c Change) bool {
		a, ok := c.(*AddTable)
		return ok && a.T.Name == name
	})
}

// LastIndexDropTable returns the index of the last DropTable in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) LastIndexDropTable(name string) int {
	return c.rsearch(func(c Change) bool {
		a, ok := c.(*DropTable)
		return ok && a.T.Name == name
	})
}

// IndexAddColumn returns the index of the first AddColumn in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexAddColumn(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*AddColumn)
		return ok && a.C.Name == name
	})
}

// IndexDropColumn returns the index of the first DropColumn in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexDropColumn(name string) int {
	return c.search(func(c Change) bool {
		d, ok := c.(*DropColumn)
		return ok && d.C.Name == name
	})
}

// IndexModifyColumn returns the index of the first ModifyColumn in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexModifyColumn(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*ModifyColumn)
		return ok && a.From.Name == name
	})
}

// IndexAddIndex returns the index of the first AddIndex in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexAddIndex(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*AddIndex)
		return ok && a.I.Name == name
	})
}

// IndexDropIndex returns the index of the first DropIndex in the changes
// with the given name, or -1 if there is no such change in the Changes.
func (c Changes) IndexDropIndex(name string) int {
	return c.search(func(c Change) bool {
		a, ok := c.(*DropIndex)
		return ok && a.I.Name == name
	})
}

// RemoveIndex removes elements in the given indexes from the Changes.
func (c *Changes) RemoveIndex(indexes ...int) {
	changes := make([]Change, 0, len(*c)-len(indexes))
Loop:
	for i := range *c {
		for _, idx := range indexes {
			if i == idx {
				continue Loop
			}
		}
		changes = append(changes, (*c)[i])
	}
	*c = changes
}

// search returns the index of the first call to f that returns true, or -1.
func (c Changes) search(f func(Change) bool) int {
	for i := range c {
		if f(c[i]) {
			return i
		}
	}
	return -1
}

// rsearch is the reversed version of search. It returns the
// index of the last call to f that returns true, or -1.
func (c Changes) rsearch(f func(Change) bool) int {
	for i := len(c) - 1; i >= 0; i-- {
		if f(c[i]) {
			return i
		}
	}
	return -1
}

// changes.
func (*AddAttr) change()          {}
func (*DropAttr) change()         {}
func (*ModifyAttr) change()       {}
func (*AddSchema) change()        {}
func (*DropSchema) change()       {}
func (*ModifySchema) change()     {}
func (*AddTable) change()         {}
func (*DropTable) change()        {}
func (*ModifyTable) change()      {}
func (*RenameTable) change()      {}
func (*AddView) change()          {}
func (*DropView) change()         {}
func (*ModifyView) change()       {}
func (*RenameView) change()       {}
func (*AddFunc) change()          {}
func (*DropFunc) change()         {}
func (*ModifyFunc) change()       {}
func (*RenameFunc) change()       {}
func (*AddProc) change()          {}
func (*DropProc) change()         {}
func (*ModifyProc) change()       {}
func (*RenameProc) change()       {}
func (*AddObject) change()        {}
func (*DropObject) change()       {}
func (*ModifyObject) change()     {}
func (*RenameObject) change()     {}
func (*AddTrigger) change()       {}
func (*DropTrigger) change()      {}
func (*ModifyTrigger) change()    {}
func (*RenameTrigger) change()    {}
func (*AddIndex) change()         {}
func (*DropIndex) change()        {}
func (*ModifyIndex) change()      {}
func (*RenameIndex) change()      {}
func (*AddPrimaryKey) change()    {}
func (*DropPrimaryKey) change()   {}
func (*ModifyPrimaryKey) change() {}
func (*AddCheck) change()         {}
func (*DropCheck) change()        {}
func (*ModifyCheck) change()      {}
func (*AddColumn) change()        {}
func (*DropColumn) change()       {}
func (*ModifyColumn) change()     {}
func (*RenameColumn) change()     {}
func (*AddForeignKey) change()    {}
func (*DropForeignKey) change()   {}
func (*ModifyForeignKey) change() {}

// clauses.
func (*IfExists) clause()    {}
func (*IfNotExists) clause() {}
