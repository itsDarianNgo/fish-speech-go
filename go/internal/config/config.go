package config

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Backend BackendConfig `mapstructure:"backend"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Limits  LimitsConfig  `mapstructure:"limits"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Listen       string        `mapstructure:"listen"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// BackendConfig holds Python backend settings.
type BackendConfig struct {
	URL            string        `mapstructure:"url"`
	Timeout        time.Duration `mapstructure:"timeout"`
	MaxConnections int           `mapstructure:"max_connections"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// LimitsConfig holds request limit settings.
type LimitsConfig struct {
	MaxTextLength int `mapstructure:"max_text_length"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Listen:       "0.0.0.0:8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 120 * time.Second,
		},
		Backend: BackendConfig{
			URL:            "http://127.0.0.1:8081",
			Timeout:        60 * time.Second,
			MaxConnections: 100,
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
}

// Load returns a Config populated with defaults and environment overrides.
func Load() (*Config, error) {
	return LoadWithDefaults(nil)
}

// LoadWithDefaults loads configuration using defaults and optional overrides map (for tests).
func LoadWithDefaults(overrides map[string]interface{}) (*Config, error) {
	cfg := Default()
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
	if v := os.Getenv("FISH_LISTEN"); v != "" {
		cfg.Server.Listen = v
	}
	if v := os.Getenv("FISH_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("FISH_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("FISH_BACKEND"); v != "" {
		cfg.Backend.URL = v
	}
	if v := os.Getenv("FISH_BACKEND_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Backend.Timeout = d
		}
	}
	if v := os.Getenv("FISH_BACKEND_MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Backend.MaxConnections = n
		}
	}
	if v := os.Getenv("FISH_API_KEY"); v != "" {
		cfg.Auth.APIKey = v
	}
	if v := os.Getenv("FISH_MAX_TEXT_LENGTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Limits.MaxTextLength = n
		}
	}
	if v := os.Getenv("FISH_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("FISH_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
}
