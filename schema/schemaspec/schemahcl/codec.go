package schemahcl

import (
	"fmt"
	"io"
	"io/ioutil"
)

// Decoder implements the schemaspec.Decoder interface for Atlas HCL documents.
type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new Decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode decodes the Atlas HCL document and stores it in target.
func (d *Decoder) Decode(target interface{}) error {
	all, err := ioutil.ReadAll(d.r)
	if err != nil {
		return fmt.Errorf("schemahcl: failed reading while decoding: %w", err)
	}
	spec, err := Decode(all)
	if err != nil {
		return fmt.Errorf("schemahcl: failed decoding: %w", err)
	}
	return spec.As(target)
}
