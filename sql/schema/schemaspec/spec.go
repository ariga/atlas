package schemaspec

// Column holds a specification for a column in an SQL table.
type Column struct {
	Name     string
	TypeName string
	Default  *string
	Null     bool
	Attrs    []*Attr
	Blocks   []*Block
}

type Attr struct {
	K string
	V Literal
}

type Block struct {
	Type   string
	Labels []string
	Attrs  []*Attr
	Blocks []*Block
}

type Literal interface {
	lit()
}

type String string
type Number float64
type Bool bool

func (String) lit() {}
func (Number) lit() {}
func (Bool) lit()   {}
