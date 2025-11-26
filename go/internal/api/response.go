package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// WriteError writes an error response using upstream format.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(schema.ErrorResponse{Detail: message})
}

// WriteJSON writes the data structure as JSON.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteAudio writes binary audio data with the appropriate content type.
func WriteAudio(w http.ResponseWriter, format string, data []byte) {
	w.Header().Set("Content-Type", GetAudioContentType(format))
	w.Header().Set("Content-Disposition", "attachment; filename=audio."+strings.ToLower(format))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GetAudioContentType returns the MIME type for a given audio format.
func GetAudioContentType(format string) string {
	switch strings.ToLower(format) {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "pcm":
		return "audio/pcm"
	default:
		return "application/octet-stream"
	}
}
