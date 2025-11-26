package api

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// HTTPError represents an error with an associated HTTP status code.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// ParseRequestBody decodes the request body into the provided value based on Content-Type.
func ParseRequestBody(r *http.Request, v interface{}) error {
	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}

	switch strings.ToLower(mediaType) {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid request body"}
		}
	case "application/msgpack":
		if err := msgpack.NewDecoder(r.Body).Decode(v); err != nil {
			return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid request body"}
		}
	case "multipart/form-data":
		if err := parseMultipart(r, v); err != nil {
			return err
		}
	default:
		return &HTTPError{Status: http.StatusUnsupportedMediaType, Message: "Unsupported content type"}
	}

	return nil
}

// ParseTTSRequest parses and validates a ServeTTSRequest from the HTTP request.
func ParseTTSRequest(r *http.Request) (*schema.ServeTTSRequest, error) {
	var req schema.ServeTTSRequest

	if err := ParseRequestBody(r, &req); err != nil {
		return nil, err
	}

	if err := req.Validate(0); err != nil {
		return nil, &HTTPError{Status: http.StatusBadRequest, Message: err.Error()}
	}

	return &req, nil
}

// parseMultipart attempts to decode a multipart/form-data request into the provided value.
func parseMultipart(r *http.Request, v interface{}) error {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid multipart form"}
	}

	if len(r.MultipartForm.Value) == 0 && len(r.MultipartForm.File) == 0 {
		return &HTTPError{Status: http.StatusBadRequest, Message: "Empty multipart form"}
	}

	// Prefer a "payload" field if provided containing JSON.
	if payloads, ok := r.MultipartForm.Value["payload"]; ok && len(payloads) > 0 {
		if err := json.Unmarshal([]byte(payloads[0]), v); err != nil {
			return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid multipart payload"}
		}
		return nil
	}

	// Fallback: build a map from form fields and decode via JSON for type inference.
	data := map[string]interface{}{}
	for key, values := range r.MultipartForm.Value {
		if len(values) == 0 {
			continue
		}
		val := values[0]

		var decoded interface{}
		if err := json.Unmarshal([]byte(val), &decoded); err == nil {
			data[key] = decoded
			continue
		}

		data[key] = val
	}

	// Handle file uploads for reference audio if present.
	for key, files := range r.MultipartForm.File {
		if len(files) == 0 {
			continue
		}
		file, err := files[0].Open()
		if err != nil {
			return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid file upload"}
		}
		defer file.Close()

		buf := make([]byte, files[0].Size)
		n, err := file.Read(buf)
		if err != nil {
			return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid file upload"}
		}
		data[key] = buf[:n]
	}

	marshaled, err := json.Marshal(data)
	if err != nil {
		return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid multipart data"}
	}

	if err := json.Unmarshal(marshaled, v); err != nil {
		return &HTTPError{Status: http.StatusBadRequest, Message: "Invalid multipart data"}
	}

	return nil
}

// IsHTTPError checks whether an error is an *HTTPError.
func IsHTTPError(err error) (*HTTPError, bool) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr, true
	}
	return nil, false
}
