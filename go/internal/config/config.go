package config

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server  ServerConfig
	Backend BackendConfig
	Auth    AuthConfig
	Limits  LimitsConfig
	Logging LoggingConfig
}

type ServerConfig struct {
	Listen       string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type BackendConfig struct {
	URL     string
	Timeout time.Duration
}

type AuthConfig struct {
	APIKey string
}

type LimitsConfig struct {
	MaxTextLength int
}

type LoggingConfig struct {
	Level  string
	Format string
}

// Load returns a Config populated with defaults and environment overrides.
func Load() (*Config, error) {
	return LoadWithDefaults(nil)
}

// LoadWithDefaults loads configuration using defaults and optional overrides map (for tests).
func LoadWithDefaults(overrides map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Listen:       "0.0.0.0:8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 120 * time.Second,
		},
		Backend: BackendConfig{
			URL:     "http://127.0.0.1:8081",
			Timeout: 60 * time.Second,
		},
		Auth: AuthConfig{
			APIKey: "",
		},
		Limits: LimitsConfig{
			MaxTextLength: 0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	applyEnvOverrides(cfg)

	if overrides != nil {
		raw, err := json.Marshal(overrides)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("FISH_SERVER_LISTEN"); v != "" {
		cfg.Server.Listen = v
	}
	if v := os.Getenv("FISH_SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("FISH_SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("FISH_BACKEND_URL"); v != "" {
		cfg.Backend.URL = v
	}
	if v := os.Getenv("FISH_BACKEND_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Backend.Timeout = d
		}
	}
	if v := os.Getenv("FISH_API_KEY"); v != "" {
		cfg.Auth.APIKey = v
	}
	if v := os.Getenv("FISH_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("FISH_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("FISH_MAX_TEXT_LENGTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Limits.MaxTextLength = n
		}
	}
}
