package backend

import (
	"errors"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// EncodeMsgpack encodes a value to MessagePack format.
func EncodeMsgpack(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

// DecodeMsgpack decodes MessagePack data into the provided value.
func DecodeMsgpack(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

// EncodeTTSRequest encodes a TTS request ensuring defaults and validation are applied.
func EncodeTTSRequest(req *schema.ServeTTSRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	if err := req.Validate(0); err != nil {
		return nil, err
	}

	return EncodeMsgpack(req)
}
