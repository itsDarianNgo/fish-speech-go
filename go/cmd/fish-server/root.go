package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "fish-server",
	Short: "High-performance Fish-Speech API server",
	Long: `Fish-Speech-Go is a high-performance Go wrapper for the Fish-Speech
text-to-speech system. It provides better streaming, concurrency, and
deployment compared to the Python server.

Start the server:
  fish-server

Start with custom settings:
  fish-server --listen 0.0.0.0:8080 --backend http://localhost:8081

Use environment variables:
  FISH_LISTEN=0.0.0.0:8080 FISH_BACKEND=http://localhost:8081 fish-server`,
	RunE: runServer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fish-server %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")

	rootCmd.Flags().String("listen", "0.0.0.0:8080", "Server listen address")
	rootCmd.Flags().Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	rootCmd.Flags().Duration("write-timeout", 120*time.Second, "HTTP write timeout")

	rootCmd.Flags().String("backend", "http://127.0.0.1:8081", "Python backend URL")
	rootCmd.Flags().Duration("backend-timeout", 60*time.Second, "Backend request timeout")

	rootCmd.Flags().String("api-key", "", "API key for authentication (empty = no auth)")
	rootCmd.Flags().Int("max-text-length", 0, "Maximum text length (0 = unlimited)")

	rootCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().String("log-format", "json", "Log format (json, text)")

	bindFlags()

	rootCmd.AddCommand(versionCmd)
}

func bindFlags() {
	bindings := []struct {
		key  string
		flag string
	}{
		{"server.listen", "listen"},
		{"server.read_timeout", "read-timeout"},
		{"server.write_timeout", "write-timeout"},
		{"backend.url", "backend"},
		{"backend.timeout", "backend-timeout"},
		{"auth.api_key", "api-key"},
		{"limits.max_text_length", "max-text-length"},
		{"logging.level", "log-level"},
		{"logging.format", "log-format"},
	}

	for _, b := range bindings {
		flag := rootCmd.Flags().Lookup(b.flag)
		if flag == nil {
			continue
		}
		_ = viper.BindPFlag(b.key, flag)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("FISH")
	viper.AutomaticEnv()

	viper.BindEnv("server.listen", "FISH_LISTEN")
	viper.BindEnv("backend.url", "FISH_BACKEND")
	viper.BindEnv("backend.timeout", "FISH_BACKEND_TIMEOUT")
	viper.BindEnv("auth.api_key", "FISH_API_KEY")
	viper.BindEnv("limits.max_text_length", "FISH_MAX_TEXT_LENGTH")
	viper.BindEnv("logging.level", "FISH_LOG_LEVEL")
	viper.BindEnv("logging.format", "FISH_LOG_FORMAT")

	viper.SetDefault("server.listen", "0.0.0.0:8080")
	viper.SetDefault("server.read_timeout", 30*time.Second)
	viper.SetDefault("server.write_timeout", 120*time.Second)
	viper.SetDefault("backend.url", "http://127.0.0.1:8081")
	viper.SetDefault("backend.timeout", 60*time.Second)
	viper.SetDefault("backend.max_connections", 100)
	viper.SetDefault("auth.api_key", "")
	viper.SetDefault("limits.max_text_length", 0)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	bindFlags()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
