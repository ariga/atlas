package schemacodec

type Element interface {
	elem()
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

func (String) elem()      {}
func (Number) elem()      {}
func (Bool) elem()        {}
func (*Block) elem()      {}
func (*Attr) elem()       {}
func (*ColumnSpec) elem() {}
func (*TableSpec) elem()  {}
func (*SchemaSpec) elem() {}
