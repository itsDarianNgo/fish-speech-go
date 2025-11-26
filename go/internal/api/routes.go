package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/fish-speech-go/fish-speech-go/internal/config"
)

// NewRouter constructs the HTTP router with middleware and routes.
func NewRouter(cfg *config.Config, logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()

	r.Use(RequestIDMiddleware)
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware)
	r.Use(AuthMiddleware(cfg.Auth.APIKey))

	r.Get("/v1/health", handleHealthGet)
	r.Post("/v1/health", handleHealthPost)

	r.Post("/v1/tts", handleTTS)

	r.Post("/v1/vqgan/encode", handleVQGANEncode)
	r.Post("/v1/vqgan/decode", handleVQGANDecode)

	r.Post("/v1/references/add", handleAddReference)
	r.Get("/v1/references", handleListReferences)
	r.Delete("/v1/references/{id}", handleDeleteReference)

	return r
}
