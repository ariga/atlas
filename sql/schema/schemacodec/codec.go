package schemacodec

type Encoder interface {
	Encode(Spec) ([]byte, error)
}

type Decoder interface {
	Decode([]byte, Spec) error
}

type Codec interface {
	Encoder
	Decoder
}
