package api

import (
	"net/http"
)

// Health Handlers
func handleHealthGet(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleHealthPost(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TTS Handler (partial implementation for parsing/validation)
func handleTTS(w http.ResponseWriter, r *http.Request) {
	req, err := ParseTTSRequest(r)
	if err != nil {
		if httpErr, ok := IsHTTPError(err); ok {
			WriteError(w, httpErr.Status, httpErr.Message)
			return
		}
		WriteError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Streaming && req.Format != "wav" {
		WriteError(w, http.StatusBadRequest, "Streaming only supports WAV format")
		return
	}

	WriteError(w, http.StatusNotImplemented, "TTS backend not connected")
}

// Stub handlers
func handleVQGANEncode(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func handleVQGANDecode(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func handleAddReference(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func handleListReferences(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}

func handleDeleteReference(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "Not implemented")
}
