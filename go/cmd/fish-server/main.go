package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/fish-speech-go/fish-speech-go/internal/api"
	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	backendClient := backend.NewBackendClient(&cfg.Backend)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := backendClient.Health(ctx); err != nil {
		logger.Warn().Err(err).Msg("Backend health check failed - server will start but TTS may not work")
	} else {
		logger.Info().Str("backend", cfg.Backend.URL).Msg("Backend connection verified")
	}

	router := api.NewRouter(cfg, backendClient, logger)

	srv := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info().Str("addr", cfg.Server.Listen).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Graceful shutdown failed")
	}
}
