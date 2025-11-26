package main

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	viper.Reset()
	initConfig()

	cfg, err := loadConfig(rootCmd)
	assert.NoError(t, err)

	assert.Equal(t, "0.0.0.0:8080", cfg.Server.Listen)
	assert.Equal(t, "http://127.0.0.1:8081", cfg.Backend.URL)
	assert.Equal(t, "", cfg.Auth.APIKey)
	assert.Equal(t, 0, cfg.Limits.MaxTextLength)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestConfigFromEnv(t *testing.T) {
	viper.Reset()
	os.Setenv("FISH_LISTEN", "0.0.0.0:9090")
	os.Setenv("FISH_BACKEND", "http://backend:8081")
	os.Setenv("FISH_API_KEY", "test-key")
	os.Setenv("FISH_MAX_TEXT_LENGTH", "5000")
	os.Setenv("FISH_LOG_LEVEL", "debug")

	defer func() {
		os.Unsetenv("FISH_LISTEN")
		os.Unsetenv("FISH_BACKEND")
		os.Unsetenv("FISH_API_KEY")
		os.Unsetenv("FISH_MAX_TEXT_LENGTH")
		os.Unsetenv("FISH_LOG_LEVEL")
	}()

	initConfig()

	cfg, err := loadConfig(rootCmd)
	assert.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9090", cfg.Server.Listen)
	assert.Equal(t, "http://backend:8081", cfg.Backend.URL)
	assert.Equal(t, "test-key", cfg.Auth.APIKey)
	assert.Equal(t, 5000, cfg.Limits.MaxTextLength)
	assert.Equal(t, "debug", cfg.Logging.Level)
}
