package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// AuthMiddleware enforces bearer token authentication when an API key is configured.
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if token != apiKey {
				WriteError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs request method, path, status, and duration using zerolog.
func LoggingMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.status).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}

// RequestIDMiddleware injects a X-Request-ID header when missing.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			r.Header.Set("X-Request-ID", requestID)
		}
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware allows cross-origin requests similar to upstream behavior.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseRecorder captures status codes for logging.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.status = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
