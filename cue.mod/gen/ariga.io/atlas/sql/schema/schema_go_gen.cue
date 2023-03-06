// Code generated by cue get go. DO NOT EDIT.

//cue:generate cue get go ariga.io/atlas/sql/schema

package schema

#Realm: {
	Schemas: [...null | #Schema] @go(,[]*Schema)
	Attrs: [...#Attr] @go(,[]Attr)
}

#Schema: {
	Name:   string
	Realm?: null | #Realm @go(,*Realm)
	Tables: [...null | #Table] @go(,[]*Table)
	Attrs: [...#Attr] @go(,[]Attr)
}

#Table: {
	Name:    string
	Schema?: null | #Schema @go(,*Schema)
	Columns: [...null | #Column] @go(,[]*Column)
	Indexes: [...null | #Index] @go(,[]*Index)
	PrimaryKey?: null | #Index @go(,*Index)
	ForeignKeys: [...null | #ForeignKey] @go(,[]*ForeignKey)
	Attrs: [...#Attr] @go(,[]Attr)
}

#Column: {
	Name:    string
	Type?:   null | #ColumnType @go(,*ColumnType)
	Default: #Expr
	Attrs: [...#Attr] @go(,[]Attr)
	Indexes: [...null | #Index] @go(,[]*Index)

	// Foreign keys that this column is
	// part of their child columns.
	ForeignKeys: [...null | #ForeignKey] @go(,[]*ForeignKey)
}

#ColumnType: {
	Type: #Type
	Raw:  string
	Null: bool
}

#Index: {
	Name:   string
	Unique: bool
	Table?: null | #Table @go(,*Table)
	Attrs: [...#Attr] @go(,[]Attr)
	Parts: [...null | #IndexPart] @go(,[]*IndexPart)
}

#IndexPart: {
	// SeqNo represents the sequence number of the key part
	// in the index.
	SeqNo: int

	// Desc indicates if the key part is stored in descending
	// order. All databases use ascending order as default.
	Desc: bool
	X:    #Expr
	C?:   null | #Column @go(,*Column)
	Attrs: [...#Attr] @go(,[]Attr)
}

#ForeignKey: {
	Symbol: string
	Table?: null | #Table @go(,*Table)
	Columns: [...null | #Column] @go(,[]*Column)
	RefTable?: null | #Table @go(,*Table)
	RefColumns: [...null | #Column] @go(,[]*Column)
	OnUpdate: #ReferenceOption
	OnDelete: #ReferenceOption
}

// ReferenceOption for constraint actions.
#ReferenceOption: string // #enumReferenceOption

#enumReferenceOption:
	#NoAction |
	#Restrict |
	#Cascade |
	#SetNull |
	#SetDefault

#NoAction:   #ReferenceOption & "NO ACTION"
#Restrict:   #ReferenceOption & "RESTRICT"
#Cascade:    #ReferenceOption & "CASCADE"
#SetNull:    #ReferenceOption & "SET NULL"
#SetDefault: #ReferenceOption & "SET DEFAULT"

#Type: _

#EnumType: {
	T: string
	Values: [...string] @go(,[]string)
	Schema?: null | #Schema @go(,*Schema)
}

#BinaryType: {
	T:     string
	Size?: null | int @go(,*int)
}

#StringType: {
	T:    string
	Size: int
}

#BoolType: T: string

#IntegerType: {
	T:        string
	Unsigned: bool
	Attrs: [...#Attr] @go(,[]Attr)
}

#DecimalType: {
	T:         string
	Precision: int
	Scale:     int
	Unsigned:  bool
}

#FloatType: {
	T:         string
	Unsigned:  bool
	Precision: int
}

#TimeType: {
	T:          string
	Precision?: null | int @go(,*int)
}

#JSONType: T: string

#SpatialType: T: string

#UnsupportedType: T: string

#Expr: _

#Literal: V: string

#RawExpr: X: string

#Attr: _

#Comment: Text: string

#Charset: V: string

#Collation: V: string

#Check: {
	Name: string
	Expr: string
	Attrs: [...#Attr] @go(,[]Attr)
}

#GeneratedExpr: {
	Expr: string
	Type: string
}
