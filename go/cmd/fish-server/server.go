package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/fish-speech-go/fish-speech-go/internal/api"
	"github.com/fish-speech-go/fish-speech-go/internal/backend"
	"github.com/fish-speech-go/fish-speech-go/internal/config"
)

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := setupLogger(cfg.Logging)

	logger.Info().
		Str("listen", cfg.Server.Listen).
		Str("backend", cfg.Backend.URL).
		Str("log_level", cfg.Logging.Level).
		Msg("Starting Fish-Speech-Go server")

	backendClient := backend.NewBackendClient(&cfg.Backend)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := backendClient.Health(ctx); err != nil {
		logger.Warn().Err(err).Msg("Backend health check failed - server will start but TTS may not work")
	} else {
		logger.Info().Str("backend", cfg.Backend.URL).Msg("Backend connection verified")
	}
	cancel()

	router := api.NewRouter(cfg, backendClient, logger)

	srv := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", cfg.Server.Listen).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("Shutting down server...")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	logger.Info().Msg("Server stopped")
	return nil
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	defaults := config.Default()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Listen:       viper.GetString("server.listen"),
			ReadTimeout:  viper.GetDuration("server.read_timeout"),
			WriteTimeout: viper.GetDuration("server.write_timeout"),
		},
		Backend: config.BackendConfig{
			URL:            viper.GetString("backend.url"),
			Timeout:        viper.GetDuration("backend.timeout"),
			MaxConnections: viper.GetInt("backend.max_connections"),
		},
		Auth: config.AuthConfig{
			APIKey: viper.GetString("auth.api_key"),
		},
		Limits: config.LimitsConfig{
			MaxTextLength: viper.GetInt("limits.max_text_length"),
		},
		Logging: config.LoggingConfig{
			Level:  viper.GetString("logging.level"),
			Format: viper.GetString("logging.format"),
		},
	}

	if env := os.Getenv("FISH_LISTEN"); env != "" {
		cfg.Server.Listen = env
	}
	if env := os.Getenv("FISH_READ_TIMEOUT"); env != "" {
		if d, err := time.ParseDuration(env); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if env := os.Getenv("FISH_WRITE_TIMEOUT"); env != "" {
		if d, err := time.ParseDuration(env); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if env := os.Getenv("FISH_BACKEND"); env != "" {
		cfg.Backend.URL = env
	}
	if env := os.Getenv("FISH_BACKEND_TIMEOUT"); env != "" {
		if d, err := time.ParseDuration(env); err == nil {
			cfg.Backend.Timeout = d
		}
	}
	if env := os.Getenv("FISH_BACKEND_MAX_CONNECTIONS"); env != "" {
		if n, err := strconv.Atoi(env); err == nil {
			cfg.Backend.MaxConnections = n
		}
	}
	if env := os.Getenv("FISH_API_KEY"); env != "" {
		cfg.Auth.APIKey = env
	}
	if env := os.Getenv("FISH_MAX_TEXT_LENGTH"); env != "" {
		if n, err := strconv.Atoi(env); err == nil {
			cfg.Limits.MaxTextLength = n
		}
	}
	if env := os.Getenv("FISH_LOG_LEVEL"); env != "" {
		cfg.Logging.Level = env
	}
	if env := os.Getenv("FISH_LOG_FORMAT"); env != "" {
		cfg.Logging.Format = env
	}

	if cfg.Server.Listen == "" {
		cfg.Server.Listen = defaults.Server.Listen
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = defaults.Server.ReadTimeout
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = defaults.Server.WriteTimeout
	}
	if cfg.Backend.URL == "" {
		cfg.Backend.URL = defaults.Backend.URL
	}
	if cfg.Backend.Timeout == 0 {
		cfg.Backend.Timeout = defaults.Backend.Timeout
	}
	if cfg.Backend.MaxConnections == 0 {
		cfg.Backend.MaxConnections = defaults.Backend.MaxConnections
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = defaults.Logging.Level
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = defaults.Logging.Format
	}

	if cmd != nil {
		if flag := cmd.Flags().Lookup("listen"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetString("listen"); err == nil && v != "" {
				cfg.Server.Listen = v
			}
		}
		if flag := cmd.Flags().Lookup("read-timeout"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetDuration("read-timeout"); err == nil && v != 0 {
				cfg.Server.ReadTimeout = v
			}
		}
		if flag := cmd.Flags().Lookup("write-timeout"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetDuration("write-timeout"); err == nil && v != 0 {
				cfg.Server.WriteTimeout = v
			}
		}
		if flag := cmd.Flags().Lookup("backend"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetString("backend"); err == nil && v != "" {
				cfg.Backend.URL = v
			}
		}
		if flag := cmd.Flags().Lookup("backend-timeout"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetDuration("backend-timeout"); err == nil && v != 0 {
				cfg.Backend.Timeout = v
			}
		}
		if flag := cmd.Flags().Lookup("api-key"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetString("api-key"); err == nil {
				cfg.Auth.APIKey = v
			}
		}
		if flag := cmd.Flags().Lookup("max-text-length"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetInt("max-text-length"); err == nil {
				cfg.Limits.MaxTextLength = v
			}
		}
		if flag := cmd.Flags().Lookup("log-level"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetString("log-level"); err == nil && v != "" {
				cfg.Logging.Level = v
			}
		}
		if flag := cmd.Flags().Lookup("log-format"); flag != nil && flag.Changed {
			if v, err := cmd.Flags().GetString("log-format"); err == nil && v != "" {
				cfg.Logging.Format = v
			}
		}
	}

	return cfg, nil
}

func setupLogger(cfg config.LoggingConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Format == "text" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}

	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
