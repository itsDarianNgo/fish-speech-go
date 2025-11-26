package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
)

// NewRouter constructs the HTTP router with middleware and routes.
func NewRouter(cfg *config.Config, backendClient backend.Backend, logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()

	r.Use(RequestIDMiddleware)
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware)
	if cfg.Auth.APIKey != "" {
		r.Use(AuthMiddleware(cfg.Auth.APIKey))
	}

	h := NewHandler(backendClient, cfg, logger)

	r.Get("/v1/health", h.HandleHealthGet)
	r.Post("/v1/health", h.HandleHealthPost)

	r.Post("/v1/tts", h.HandleTTS)

	r.Post("/v1/vqgan/encode", h.HandleVQGANEncode)
	r.Post("/v1/vqgan/decode", h.HandleVQGANDecode)

	r.Post("/v1/references/add", h.HandleAddReference)
	r.Get("/v1/references", h.HandleListReferences)
	r.Delete("/v1/references/{id}", h.HandleDeleteReference)

	return r
}
