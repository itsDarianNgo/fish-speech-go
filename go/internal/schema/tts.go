package schema

import "fmt"

const (
	defaultChunkLength       = 200
	defaultFormat            = "wav"
	defaultMaxNewTokens      = 1024
	defaultTopP              = 0.8
	defaultRepetitionPenalty = 1.1
	defaultTemperature       = 0.8
	defaultUseMemoryCache    = "off"
	defaultNormalize         = true
)

// ServeReferenceAudio represents an inline reference audio payload.
type ServeReferenceAudio struct {
	Audio []byte `json:"audio" msgpack:"audio"`
	Text  string `json:"text" msgpack:"text"`
}

// ServeTTSRequest represents the upstream ServeTTSRequest schema.
type ServeTTSRequest struct {
	Text string `json:"text" msgpack:"text"`

	ChunkLength       int     `json:"chunk_length" msgpack:"chunk_length"`
	Format            string  `json:"format" msgpack:"format"`
	MaxNewTokens      int     `json:"max_new_tokens" msgpack:"max_new_tokens"`
	TopP              float64 `json:"top_p" msgpack:"top_p"`
	RepetitionPenalty float64 `json:"repetition_penalty" msgpack:"repetition_penalty"`
	Temperature       float64 `json:"temperature" msgpack:"temperature"`

	References  []ServeReferenceAudio `json:"references" msgpack:"references"`
	ReferenceID *string               `json:"reference_id,omitempty" msgpack:"reference_id,omitempty"`

	Seed           *int   `json:"seed,omitempty" msgpack:"seed,omitempty"`
	UseMemoryCache string `json:"use_memory_cache" msgpack:"use_memory_cache"`
	Normalize      bool   `json:"normalize" msgpack:"normalize"`
	Streaming      bool   `json:"streaming" msgpack:"streaming"`
}

// Validate applies default values and validates the request against upstream rules.
func (r *ServeTTSRequest) Validate(maxTextLength int) error {
	r.applyDefaults()

	if maxTextLength > 0 && len(r.Text) > maxTextLength {
		return fmt.Errorf("Text is too long, max length is %d", maxTextLength)
	}

	if r.ChunkLength < 100 || r.ChunkLength > 300 {
		return fmt.Errorf("chunk_length must be between 100 and 300")
	}

	if r.TopP < 0.1 || r.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0. 1 and 1. 0")
	}

	if r.Temperature < 0.1 || r.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.1 and 1. 0")
	}

	if r.RepetitionPenalty < 0.9 || r.RepetitionPenalty > 2.0 {
		return fmt.Errorf("repetition_penalty must be between 0. 9 and 2. 0")
	}

	if r.Streaming && r.Format != "wav" {
		return fmt.Errorf("Streaming only supports WAV format")
	}

	return nil
}

func (r *ServeTTSRequest) applyDefaults() {
	if r.ChunkLength == 0 {
		r.ChunkLength = defaultChunkLength
	}

	if r.Format == "" {
		r.Format = defaultFormat
	}

	if r.MaxNewTokens == 0 {
		r.MaxNewTokens = defaultMaxNewTokens
	}

	if r.TopP == 0 {
		r.TopP = defaultTopP
	}

	if r.RepetitionPenalty == 0 {
		r.RepetitionPenalty = defaultRepetitionPenalty
	}

	if r.Temperature == 0 {
		r.Temperature = defaultTemperature
	}

	if r.References == nil {
		r.References = []ServeReferenceAudio{}
	}

	if r.UseMemoryCache == "" {
		r.UseMemoryCache = defaultUseMemoryCache
	}

	r.Normalize = r.Normalize || defaultNormalize
}
