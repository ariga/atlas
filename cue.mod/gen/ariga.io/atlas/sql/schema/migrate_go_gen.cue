// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go ariga.io/atlas/sql/schema

package schema

#Change: _

#Clause: _

#AddSchema: {
	S?: null | #Schema @go(,*Schema)
	Extra: [...#Clause] @go(,[]Clause)
}

#DropSchema: {
	S?: null | #Schema @go(,*Schema)
	Extra: [...#Clause] @go(,[]Clause)
}

#ModifySchema: {
	S?: null | #Schema @go(,*Schema)
	Changes: [...#Change] @go(,[]Change)
}

#AddTable: {
	T?: null | #Table @go(,*Table)
	Extra: [...#Clause] @go(,[]Clause)
}

#DropTable: {
	T?: null | #Table @go(,*Table)
	Extra: [...#Clause] @go(,[]Clause)
}

#ModifyTable: {
	T?: null | #Table @go(,*Table)
	Changes: [...#Change] @go(,[]Change)
}

#RenameTable: {
	From?: null | #Table @go(,*Table)
	To?:   null | #Table @go(,*Table)
}

#AddColumn: {
	C?: null | #Column @go(,*Column)
}

#DropColumn: {
	C?: null | #Column @go(,*Column)
}

#ModifyColumn: {
	From?:  null | #Column @go(,*Column)
	To?:    null | #Column @go(,*Column)
	Change: #ChangeKind
}

#RenameColumn: {
	From?: null | #Column @go(,*Column)
	To?:   null | #Column @go(,*Column)
}

#AddIndex: {
	I?: null | #Index @go(,*Index)
}

#DropIndex: {
	I?: null | #Index @go(,*Index)
}

#ModifyIndex: {
	From?:  null | #Index @go(,*Index)
	To?:    null | #Index @go(,*Index)
	Change: #ChangeKind
}

#RenameIndex: {
	From?: null | #Index @go(,*Index)
	To?:   null | #Index @go(,*Index)
}

#AddForeignKey: {
	F?: null | #ForeignKey @go(,*ForeignKey)
}

#DropForeignKey: {
	F?: null | #ForeignKey @go(,*ForeignKey)
}

#ModifyForeignKey: {
	From?:  null | #ForeignKey @go(,*ForeignKey)
	To?:    null | #ForeignKey @go(,*ForeignKey)
	Change: #ChangeKind
}

#AddCheck: {
	C?: null | #Check @go(,*Check)
}

#DropCheck: {
	C?: null | #Check @go(,*Check)
}

#ModifyCheck: {
	From?:  null | #Check @go(,*Check)
	To?:    null | #Check @go(,*Check)
	Change: #ChangeKind
}

#AddAttr: A: #Attr

#DropAttr: A: #Attr

#ModifyAttr: {
	From: #Attr
	To:   #Attr
}

#IfExists: {
}

#IfNotExists: {
}

// A ChangeKind describes a change kind that can be combined
// using a set of flags. The zero kind is no change.
//
//go:generate stringer -type ChangeKind
#ChangeKind: uint // #enumChangeKind

#enumChangeKind:
	#NoChange |
	#ChangeAttr |
	#ChangeCharset |
	#ChangeCollate |
	#ChangeComment |
	#ChangeNull |
	#ChangeType |
	#ChangeDefault |
	#ChangeGenerated |
	#ChangeUnique |
	#ChangeParts |
	#ChangeColumn |
	#ChangeRefColumn |
	#ChangeRefTable |
	#ChangeUpdateAction |
	#ChangeDeleteAction

#values_ChangeKind: {
	NoChange:           #NoChange
	ChangeAttr:         #ChangeAttr
	ChangeCharset:      #ChangeCharset
	ChangeCollate:      #ChangeCollate
	ChangeComment:      #ChangeComment
	ChangeNull:         #ChangeNull
	ChangeType:         #ChangeType
	ChangeDefault:      #ChangeDefault
	ChangeGenerated:    #ChangeGenerated
	ChangeUnique:       #ChangeUnique
	ChangeParts:        #ChangeParts
	ChangeColumn:       #ChangeColumn
	ChangeRefColumn:    #ChangeRefColumn
	ChangeRefTable:     #ChangeRefTable
	ChangeUpdateAction: #ChangeUpdateAction
	ChangeDeleteAction: #ChangeDeleteAction
}

// NoChange holds the zero value of a change kind.
#NoChange: #ChangeKind & 0

// ChangeAttr describes attributes change of an element.
// For example, a table CHECK was added or changed.
#ChangeAttr: #ChangeKind & 1

// ChangeCharset describes character-set change.
#ChangeCharset: #ChangeKind & 2

// ChangeCollate describes collation/encoding change.
#ChangeCollate: #ChangeKind & 4

// ChangeComment describes comment chang (of any element).
#ChangeComment: #ChangeKind & 8

// ChangeNull describe a change to the NULL constraint.
#ChangeNull: #ChangeKind & 16

// ChangeType describe a column type change.
#ChangeType: #ChangeKind & 32

// ChangeDefault describe a column default change.
#ChangeDefault: #ChangeKind & 64

// ChangeGenerated describe a change to the generated expression.
#ChangeGenerated: #ChangeKind & 128

// ChangeUnique describes a change to the uniqueness constraint.
// For example, an index was changed from non-unique to unique.
#ChangeUnique: #ChangeKind & 256

// ChangeParts describes a change to one or more of the index parts.
// For example, index keeps its previous name, but the columns order
// was changed.
#ChangeParts: #ChangeKind & 512

// ChangeColumn describes a change to the foreign-key (child) columns.
#ChangeColumn: #ChangeKind & 1024

// ChangeRefColumn describes a change to the foreign-key (parent) columns.
#ChangeRefColumn: #ChangeKind & 2048

// ChangeRefTable describes a change to the foreign-key (parent) table.
#ChangeRefTable: #ChangeKind & 4096

// ChangeUpdateAction describes a change to the foreign-key update action.
#ChangeUpdateAction: #ChangeKind & 8192

// ChangeDeleteAction describes a change to the foreign-key delete action.
#ChangeDeleteAction: #ChangeKind & 16384

// Differ is the interface implemented by the different
// drivers for comparing and diffing schema top elements.
#Differ: _

#Locker: _

// Changes is a list of changes allow for searching and mutating changes.
#Changes: [...#Change]
