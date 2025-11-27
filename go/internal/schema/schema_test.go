package schema

import (
	"encoding/json"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

func TestServeTTSRequestDefaults(t *testing.T) {
	req := &ServeTTSRequest{Text: "hello"}

	if err := req.Validate(0); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.ChunkLength != 200 {
		t.Fatalf("expected default chunk_length 200, got %d", req.ChunkLength)
	}
	if req.Format != "wav" {
		t.Fatalf("expected default format wav, got %s", req.Format)
	}
	if req.MaxNewTokens != 1024 {
		t.Fatalf("expected default max_new_tokens 1024, got %d", req.MaxNewTokens)
	}
	if req.TopP != 0.8 {
		t.Fatalf("expected default top_p 0.8, got %f", req.TopP)
	}
	if req.RepetitionPenalty != 1.1 {
		t.Fatalf("expected default repetition_penalty 1.1, got %f", req.RepetitionPenalty)
	}
	if req.Temperature != 0.8 {
		t.Fatalf("expected default temperature 0.8, got %f", req.Temperature)
	}
	if req.UseMemoryCache != "off" {
		t.Fatalf("expected default use_memory_cache off, got %s", req.UseMemoryCache)
	}
	if !req.Normalize {
		t.Fatalf("expected default normalize true")
	}
	if req.Streaming {
		t.Fatalf("expected default streaming false")
	}
	if req.References == nil {
		t.Fatalf("expected references to default to empty slice")
	}
}

func TestServeTTSRequestValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		req           ServeTTSRequest
		maxTextLength int
		expectedError string
	}{
		{
			name:          "chunk length below range",
			req:           ServeTTSRequest{Text: "hi", ChunkLength: 50},
			expectedError: "chunk_length must be between 100 and 300",
		},
		{
			name:          "top_p below range",
			req:           ServeTTSRequest{Text: "hi", TopP: 0.05},
			expectedError: "top_p must be between 0. 1 and 1. 0",
		},
		{
			name:          "temperature above range",
			req:           ServeTTSRequest{Text: "hi", Temperature: 1.5},
			expectedError: "temperature must be between 0.1 and 1. 0",
		},
		{
			name:          "repetition penalty below range",
			req:           ServeTTSRequest{Text: "hi", RepetitionPenalty: 0.5},
			expectedError: "repetition_penalty must be between 0. 9 and 2. 0",
		},
		{
			name:          "streaming with non wav format",
			req:           ServeTTSRequest{Text: "hi", Streaming: true, Format: "mp3"},
			expectedError: "Streaming only supports WAV format",
		},
		{
			name:          "text too long",
			req:           ServeTTSRequest{Text: "hello world"},
			maxTextLength: 5,
			expectedError: "Text is too long, max length is 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(tt.maxTextLength)
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if err.Error() != tt.expectedError {
				t.Fatalf("expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestServeTTSRequestJSONTags(t *testing.T) {
	referenceID := "ref-1"
	seed := 42
	req := ServeTTSRequest{
		Text:              "hello",
		ChunkLength:       150,
		Format:            "mp3",
		MaxNewTokens:      500,
		TopP:              0.9,
		RepetitionPenalty: 1.0,
		Temperature:       0.6,
		References: []ServeReferenceAudio{{
			Audio: []byte{0x01, 0x02},
			Text:  "ref text",
		}},
		ReferenceID:    &referenceID,
		Seed:           &seed,
		UseMemoryCache: "on",
		Normalize:      true,
		Streaming:      true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal to json: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}

	expectedKeys := []string{
		"text", "chunk_length", "format", "max_new_tokens", "top_p", "repetition_penalty",
		"temperature", "references", "reference_id", "seed", "use_memory_cache", "normalize", "streaming",
	}

	for _, key := range expectedKeys {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %s in json output", key)
		}
	}
}

func TestServeTTSRequestMsgpackTags(t *testing.T) {
	referenceID := "ref-1"
	seed := 42
	req := ServeTTSRequest{
		Text:              "hello",
		ChunkLength:       150,
		Format:            "mp3",
		MaxNewTokens:      500,
		TopP:              0.9,
		RepetitionPenalty: 1.0,
		Temperature:       0.6,
		References: []ServeReferenceAudio{{
			Audio: []byte{0x01, 0x02},
			Text:  "ref text",
		}},
		ReferenceID:    &referenceID,
		Seed:           &seed,
		UseMemoryCache: "on",
		Normalize:      true,
		Streaming:      true,
	}

	data, err := msgpack.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal to msgpack: %v", err)
	}

	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal msgpack: %v", err)
	}

	expectedKeys := []string{
		"text", "chunk_length", "format", "max_new_tokens", "top_p", "repetition_penalty",
		"temperature", "references", "reference_id", "seed", "use_memory_cache", "normalize", "streaming",
	}

	for _, key := range expectedKeys {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %s in msgpack output", key)
		}
	}
}
