package schema

// Encoder takes a Spec and returns a byte slice representing that Spec in some configuration
// format (for instance, HCL).
type Encoder interface {
	Encode(Spec) ([]byte, error)
}

// Decoder takes a byte slice representing a Spec and decodes it into a Spec.
type Decoder interface {
	Decode([]byte, Spec) error
}

type Codec interface {
	Encoder
	Decoder
}
