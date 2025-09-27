package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey            string
	Model             string
	EnableFileLogging bool
	Hotkey            string
	Providers         []string
}

func Load() (*Config, error) {
	// Try to load .env file from current directory or executable directory
	envPaths := []string{".env"}

	// If running as executable, also try the executable's directory
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		envPaths = append(envPaths, filepath.Join(execDir, ".env"))
	}

	// Try to load .env file (ignore errors if file doesn't exist)
	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			godotenv.Load(envPath)
			break
		}
	}

	// Parse providers from comma-separated string
	var providers []string
	if providersStr := os.Getenv("PROVIDERS"); providersStr != "" {
		// Split by comma and trim whitespace
		for _, provider := range strings.Split(providersStr, ",") {
			if trimmed := strings.TrimSpace(provider); trimmed != "" {
				providers = append(providers, trimmed)
			}
		}
	}

	cfg := &Config{
		APIKey:            os.Getenv("OPENROUTER_API_KEY"),
		Model:             os.Getenv("MODEL"),
		EnableFileLogging: strings.ToLower(os.Getenv("ENABLE_FILE_LOGGING")) == "true",
		Hotkey:            getEnvWithDefault("HOTKEY", "Ctrl+Alt+Q"),
		Providers:         providers,
	}

	return cfg, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
