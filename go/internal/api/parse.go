package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// ParseError represents a request parsing error.
type ParseError struct {
	Status  int
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

// NewParseError creates a new parse error.
func NewParseError(status int, message string) *ParseError {
	return &ParseError{Status: status, Message: message}
}

// ParseRequestBody decodes the request body into the provided value based on Content-Type.
func ParseRequestBody(r *http.Request, v interface{}) error {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(contentType, "application/msgpack"):
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return NewParseError(http.StatusBadRequest, "Failed to read request body")
		}
		if err := msgpack.Unmarshal(body, v); err != nil {
			return NewParseError(http.StatusBadRequest, "Invalid MessagePack body")
		}
		return nil
	case strings.HasPrefix(contentType, "application/json"):
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return NewParseError(http.StatusBadRequest, "Invalid JSON body")
		}
		return nil
	case strings.HasPrefix(contentType, "multipart/form-data"):
		return NewParseError(http.StatusBadRequest, "Use specific handler for multipart")
	case contentType == "":
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return NewParseError(http.StatusBadRequest, "Invalid request body")
		}
		return nil
	default:
		return NewParseError(http.StatusUnsupportedMediaType, "Unsupported content type")
	}
}

// ParseTTSRequest parses and validates a ServeTTSRequest from the HTTP request.
func ParseTTSRequest(r *http.Request) (*schema.ServeTTSRequest, error) {
	var req schema.ServeTTSRequest

	if err := ParseRequestBody(r, &req); err != nil {
		return nil, err
	}

	if err := req.Validate(0); err != nil {
		return nil, NewParseError(http.StatusBadRequest, err.Error())
	}

	return &req, nil
}
