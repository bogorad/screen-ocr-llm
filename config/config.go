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
	// Load configuration from sources in priority order:
	// 1) .env in the application (executable) directory
	// 2) If not found, use SCREEN_OCR_LLM env var as a path to a config file
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		exeEnv := filepath.Join(execDir, ".env")
		if _, err := os.Stat(exeEnv); err == nil {
			_ = godotenv.Load(exeEnv)
		} else {
			if alt := os.Getenv("SCREEN_OCR_LLM"); alt != "" {
				if _, err := os.Stat(alt); err == nil {
					_ = godotenv.Load(alt)
				}
			}
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
