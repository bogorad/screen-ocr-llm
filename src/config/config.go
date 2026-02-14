package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	DefaultAPIKeyPath = "/run/secrets/api_keys/openrouter"
	APIKeyPathEnvVar  = "OPENROUTER_API_KEY_FILE"
	DefaultModeEnvVar = "DEFAULT_MODE"
	DefaultModeRect   = "rectangle"
	DefaultModeLasso  = "lasso"
)

type LoadOptions struct {
	APIKeyPathOverride  string
	DefaultModeOverride string
}

type Config struct {
	APIKey            string
	APIKeyPath        string
	Model             string
	EnableFileLogging bool
	Hotkey            string
	DefaultMode       string
	Providers         []string
	OCRDeadlineSec    int
}

func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

func LoadWithOptions(opts LoadOptions) (*Config, error) {
	// Load configuration from sources in priority order:
	// 1) .env in the application (executable) directory
	// 2) If not found, use SCREEN_OCR_LLM env var as a path to a config file
	envPath := resolveEnvPath()
	dotenvValues := readDotenvValues(envPath)
	if envPath != "" {
		_ = godotenv.Load(envPath)
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

	// Resolve OCR deadline (seconds) with env override and sane default
	ocrDeadlineSec := 20
	if v := os.Getenv("OCR_DEADLINE_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ocrDeadlineSec = n
		}
	}

	apiKeyPath := resolveAPIKeyPath(opts, dotenvValues)

	cfg := &Config{
		APIKey:            resolveAPIKey(apiKeyPath),
		APIKeyPath:        apiKeyPath,
		Model:             os.Getenv("MODEL"),
		EnableFileLogging: strings.ToLower(os.Getenv("ENABLE_FILE_LOGGING")) == "true",
		Hotkey:            getEnvWithDefault("HOTKEY", "Ctrl+Alt+Q"),
		DefaultMode:       resolveDefaultModeValue(opts),
		Providers:         providers,
		OCRDeadlineSec:    ocrDeadlineSec,
	}

	return cfg, nil
}

func resolveEnvPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	execDir := filepath.Dir(execPath)
	exeEnv := filepath.Join(execDir, ".env")
	if _, err := os.Stat(exeEnv); err == nil {
		return exeEnv
	}

	if alt := os.Getenv("SCREEN_OCR_LLM"); alt != "" {
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}

	return ""
}

func readDotenvValues(envPath string) map[string]string {
	if envPath == "" {
		return map[string]string{}
	}

	values, err := godotenv.Read(envPath)
	if err != nil {
		return map[string]string{}
	}

	return values
}

func resolveAPIKeyPath(opts LoadOptions, dotenvValues map[string]string) string {
	keyPath := DefaultAPIKeyPath

	if envPath := strings.TrimSpace(os.Getenv(APIKeyPathEnvVar)); envPath != "" {
		keyPath = envPath
	}

	if dotenvPath := strings.TrimSpace(dotenvValues[APIKeyPathEnvVar]); dotenvPath != "" {
		keyPath = dotenvPath
	}

	if overridePath := strings.TrimSpace(opts.APIKeyPathOverride); overridePath != "" {
		keyPath = overridePath
	}

	return keyPath
}

func resolveAPIKey(keyPath string) string {
	if data, err := os.ReadFile(keyPath); err == nil {
		if fileKey := strings.TrimSpace(string(data)); fileKey != "" {
			return fileKey
		}
	}

	return os.Getenv("OPENROUTER_API_KEY")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func resolveDefaultMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "rect", DefaultModeRect:
		return DefaultModeRect
	case DefaultModeLasso:
		return DefaultModeLasso
	default:
		return DefaultModeRect
	}
}

func resolveDefaultModeValue(opts LoadOptions) string {
	if override := strings.TrimSpace(opts.DefaultModeOverride); override != "" {
		return resolveDefaultMode(override)
	}
	return resolveDefaultMode(os.Getenv(DefaultModeEnvVar))
}
