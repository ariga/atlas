package schema

type (
	// A Table represents a table definition.
	Table struct {
		Name        string
		Columns     []*Column
		Indexes     []*Index
		PrimaryKey  []*Column
		ForeignKeys []*ForeignKey
		Attrs       []Attr // Attributes, constraints and options.
	}

	// A Column represents a column definition.
	Column struct {
		Name        string
		Type        *ColumnType
		Attrs       []Attr
		Indexes     []*Index
		ForeignKeys []*ForeignKey
	}

	// ColumnType represents a column type that is implemented by the dialect.
	ColumnType struct {
		Type    Type
		Raw     string
		Null    bool
		Default Expr
	}

	// An Index represents an index definition.
	Index struct {
		Name    string
		Unique  bool
		Attrs   []Attr
		Table   *Table
		Columns []*Column
		Options []Expr
	}

	// A ForeignKey represents an index definition.
	ForeignKey struct {
		Symbol     string
		Columns    []*Column
		RefTable   *Table
		RefColumns []*Column
		OnUpdate   ReferenceOption
		OnDelete   ReferenceOption
	}
)

// ReferenceOption for constraint actions.
type ReferenceOption string

// Reference options (actions) specified by ON UPDATE and ON DELETE
// subclauses of the FOREIGN KEY clause.
const (
	NoAction   ReferenceOption = "NO ACTION"
	Restrict   ReferenceOption = "RESTRICT"
	Cascade    ReferenceOption = "CASCADE"
	SetNull    ReferenceOption = "SET NULL"
	SetDefault ReferenceOption = "SET DEFAULT"
)

type (
	// A Type represents a database type. The types below implements this
	// interface and can be used for describing schemas.
	//
	// The Type interface can also be implemented outside this package as follows:
	//
	//	type SpatialType struct {
	//		schema.Type
	//		T string
	//	}
	//
	//	var t schema.Type = &SpatialType{T: "point"}
	//
	Type interface {
		typ()
	}

	// EnumType represents an enum type.
	EnumType struct {
		Values []string
	}

	// BinaryType represents a type that stores a binary data.
	BinaryType struct {
		T    string
		Size int
	}

	// StringType represents a string type.
	StringType struct {
		T    string
		Size int
	}

	// BoolType represents a boolean type.
	BoolType struct {
		T string
	}

	// IntegerType represents an int type.
	IntegerType struct {
		T      string
		Size   int
		Signed bool
	}

	// DecimalType represents a fixed-point type that stores exact numeric values.
	DecimalType struct {
		T         string
		Precision int
		Scale     int
	}

	// DecimalType represents a floating-point type that stores approximate numeric values.
	FloatType struct {
		T         string
		Precision int
	}

	// UnsupportedType represents a type that is not supported by the drivers.
	UnsupportedType struct {
		T string
	}
)

type (
	// Expr defines an SQL expression in schema DDL.
	Expr interface {
		expr()
	}

	// Literal represents a basic literal expression like "1".
	Literal struct {
		V string
	}

	// RawExpr represents a raw expresion like "uuid()".
	RawExpr struct {
		X string
	}
)

type (
	// Attr represents the interface that all attributes implement.
	Attr interface {
		attr()
	}
)

// expressions.
func (*Literal) expr() {}
func (*RawExpr) expr() {}

// types.
func (*BoolType) typ()        {}
func (*EnumType) typ()        {}
func (*FloatType) typ()       {}
func (*StringType) typ()      {}
func (*BinaryType) typ()      {}
func (*IntegerType) typ()     {}
func (*DecimalType) typ()     {}
func (*UnsupportedType) typ() {}
