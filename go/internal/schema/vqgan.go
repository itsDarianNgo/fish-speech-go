package schema

// ServeVQGANEncodeRequest represents a request to encode audio with VQGAN.
type ServeVQGANEncodeRequest struct {
	Audios [][]byte `json:"audios" msgpack:"audios"`
}

// ServeVQGANEncodeResponse represents the encoded token response from VQGAN.
type ServeVQGANEncodeResponse struct {
	Tokens [][][]int `json:"tokens" msgpack:"tokens"`
}

// ServeVQGANDecodeRequest represents a request to decode tokens with VQGAN.
type ServeVQGANDecodeRequest struct {
	Tokens [][][]int `json:"tokens" msgpack:"tokens"`
}

// ServeVQGANDecodeResponse represents decoded audio payloads from VQGAN.
type ServeVQGANDecodeResponse struct {
	Audios [][]byte `json:"audios" msgpack:"audios"`
}
