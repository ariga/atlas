package schemahcl

import "ariga.io/atlas/schema/schemaspec"

// Codec implements schemaspec.Codec for Atlas HCL files.
type Codec struct {
}

// New returns a new Codec.
func New() *Codec {
	return &Codec{}
}

// encode implements schemaspec.Encoder.
func (*Codec) Encode(f *schemaspec.File) ([]byte, error) {
	return encode(&schemaspec.Resource{
		Attrs:    f.Attrs,
		Children: f.Resources,
	})
}

// decode implements schemaspec.Decoder.
func (*Codec) Decode(body []byte) (*schemaspec.File, error) {
	r, err := decode(body)
	if err != nil {
		return nil, err
	}
	return &schemaspec.File{
		Attrs:     r.Attrs,
		Resources: r.Children,
	}, nil
}
