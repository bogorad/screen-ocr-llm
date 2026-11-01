package config

import (
	"fmt"
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
	APIKey             string
	APIKeyPath         string
	Model              string
	SourcePath         string
	EnableFileLogging  bool
	ReloadConfigOnGrab bool
	Hotkey             string
	DefaultMode        string
	Providers          []string
	OCRDeadlineSec     int
}

func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

// LoadWithOptions reads a fresh snapshot. It does not put file values into the
// process environment, so later reads can see edits to the selected file.
func LoadWithOptions(opts LoadOptions) (*Config, error) {
	envPath := resolveEnvPath()
	dotenvValues, err := readDotenvValues(envPath)
	if err != nil {
		return nil, fmt.Errorf("read configuration %q: %w", envPath, err)
	}
	value := func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return dotenvValues[key]
	}

	var providers []string
	for _, provider := range strings.Split(value("PROVIDERS"), ",") {
		if trimmed := strings.TrimSpace(provider); trimmed != "" {
			providers = append(providers, trimmed)
		}
	}

	ocrDeadlineSec := 20
	if v := value("OCR_DEADLINE_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ocrDeadlineSec = n
		}
	}

	apiKeyPath := resolveAPIKeyPath(opts, dotenvValues)
	return &Config{
		APIKey:             resolveAPIKey(apiKeyPath, value("OPENROUTER_API_KEY")),
		APIKeyPath:         apiKeyPath,
		Model:              value("MODEL"),
		SourcePath:         envPath,
		EnableFileLogging:  strings.EqualFold(value("ENABLE_FILE_LOGGING"), "true"),
		ReloadConfigOnGrab: !strings.EqualFold(value("RELOAD_CONFIG_ON_GRAB"), "false"),
		Hotkey:             withDefault(value("HOTKEY"), "Ctrl+Alt+Q"),
		DefaultMode:        resolveDefaultModeValue(opts, value(DefaultModeEnvVar)),
		Providers:          providers,
		OCRDeadlineSec:     ocrDeadlineSec,
	}, nil
}

// SourcePath returns the watched path. Keep a missing SCREEN_OCR_LLM target so
// the resident can report its loss and detect when it reappears.
func SourcePath() string {
	execPath, err := os.Executable()
	if err == nil {
		exeEnv := filepath.Join(filepath.Dir(execPath), ".env")
		if _, err := os.Stat(exeEnv); err == nil {
			return exeEnv
		}
	}
	return os.Getenv("SCREEN_OCR_LLM")
}

// Value reads one effective setting without changing the process environment.
// The single-instance package uses this for its startup port range.
func Value(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	values, err := readDotenvValues(resolveEnvPath())
	if err != nil {
		return ""
	}
	return values[key]
}

// FileLoggingRequested works even when a malformed file cannot be parsed.
func FileLoggingRequested() bool {
	if value, ok := os.LookupEnv("ENABLE_FILE_LOGGING"); ok {
		return strings.EqualFold(value, "true")
	}
	path := resolveEnvPath()
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && parts[0] == "ENABLE_FILE_LOGGING" {
			return strings.EqualFold(strings.TrimSpace(parts[1]), "true")
		}
	}
	return false
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

func readDotenvValues(envPath string) (map[string]string, error) {
	if envPath == "" {
		return map[string]string{}, nil
	}
	values, err := godotenv.Read(envPath)
	if err != nil {
		// godotenv parse errors can include the remaining file contents,
		// including credentials. Never pass them to logs or dialogs.
		return nil, fmt.Errorf("cannot parse dotenv file")
	}
	return values, nil
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

func resolveAPIKey(keyPath, fallback string) string {
	if data, err := os.ReadFile(keyPath); err == nil {
		if fileKey := strings.TrimSpace(string(data)); fileKey != "" {
			return fileKey
		}
	}
	return fallback
}

func withDefault(value, defaultValue string) string {
	if value != "" {
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

func resolveDefaultModeValue(opts LoadOptions, value string) string {
	if override := strings.TrimSpace(opts.DefaultModeOverride); override != "" {
		return resolveDefaultMode(override)
	}
	return resolveDefaultMode(value)
}
