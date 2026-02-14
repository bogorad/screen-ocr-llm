package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test_api_key")
	t.Setenv("MODEL", "test_model")
	t.Setenv("ENABLE_FILE_LOGGING", "true")
	t.Setenv("HOTKEY", "Ctrl+Shift+T")
	t.Setenv("DEFAULT_MODE", "lasso")

	// Load the configuration
	cfg, err := LoadWithOptions(LoadOptions{APIKeyPathOverride: filepath.Join(t.TempDir(), "missing.key")})
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Check the configuration values
	if cfg.APIKey != "test_api_key" {
		t.Errorf("Expected APIKey to be 'test_api_key', got '%s'", cfg.APIKey)
	}
	if cfg.Model != "test_model" {
		t.Errorf("Expected Model to be 'test_model', got '%s'", cfg.Model)
	}
	if !cfg.EnableFileLogging {
		t.Errorf("Expected EnableFileLogging to be true, got %v", cfg.EnableFileLogging)
	}
	if cfg.Hotkey != "Ctrl+Shift+T" {
		t.Errorf("Expected Hotkey to be 'Ctrl+Shift+T', got '%s'", cfg.Hotkey)
	}
	if cfg.DefaultMode != DefaultModeLasso {
		t.Errorf("Expected DefaultMode to be '%s', got '%s'", DefaultModeLasso, cfg.DefaultMode)
	}
}

func TestResolveDefaultMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty defaults to rectangle", input: "", want: DefaultModeRect},
		{name: "rect accepted", input: "rect", want: DefaultModeRect},
		{name: "lasso accepted", input: "lasso", want: DefaultModeLasso},
		{name: "lasso case insensitive", input: " LASSO ", want: DefaultModeLasso},
		{name: "invalid defaults to rectangle", input: "triangle", want: DefaultModeRect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveDefaultMode(tt.input); got != tt.want {
				t.Fatalf("resolveDefaultMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadWithOptionsDefaultModeOverride(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "env-key")
	t.Setenv("MODEL", "test-model")
	t.Setenv("DEFAULT_MODE", "lasso")

	t.Run("CLI override wins and normalizes rect", func(t *testing.T) {
		cfg, err := LoadWithOptions(LoadOptions{DefaultModeOverride: "rect"})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.DefaultMode != DefaultModeRect {
			t.Fatalf("Expected DefaultMode=%q, got %q", DefaultModeRect, cfg.DefaultMode)
		}
	})

	t.Run("Invalid CLI override falls back to rectangle", func(t *testing.T) {
		cfg, err := LoadWithOptions(LoadOptions{DefaultModeOverride: "blob"})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.DefaultMode != DefaultModeRect {
			t.Fatalf("Expected DefaultMode=%q, got %q", DefaultModeRect, cfg.DefaultMode)
		}
	})
}

func TestLoadWithOptionsAPIKeyPathPrecedence(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "fallback-env-key")
	t.Setenv("OPENROUTER_API_KEY_FILE", "/env/path.key")

	envPath := filepath.Join(t.TempDir(), ".env")
	envContent := "OPENROUTER_API_KEY_FILE=/dotenv/path.key\n"
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		t.Fatalf("Failed to write test .env: %v", err)
	}
	t.Setenv("SCREEN_OCR_LLM", envPath)

	t.Run("CLI override wins", func(t *testing.T) {
		cfg, err := LoadWithOptions(LoadOptions{APIKeyPathOverride: "/cli/path.key"})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKeyPath != "/cli/path.key" {
			t.Fatalf("Expected CLI key path, got %q", cfg.APIKeyPath)
		}
	})

	t.Run("Dotenv overrides env var", func(t *testing.T) {
		cfg, err := LoadWithOptions(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKeyPath != "/dotenv/path.key" {
			t.Fatalf("Expected dotenv key path, got %q", cfg.APIKeyPath)
		}
	})

	t.Run("Env var overrides default", func(t *testing.T) {
		t.Setenv("SCREEN_OCR_LLM", filepath.Join(t.TempDir(), "not-found.env"))
		cfg, err := LoadWithOptions(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKeyPath != "/env/path.key" {
			t.Fatalf("Expected env key path, got %q", cfg.APIKeyPath)
		}
	})

	t.Run("Default when no overrides", func(t *testing.T) {
		t.Setenv("SCREEN_OCR_LLM", filepath.Join(t.TempDir(), "not-found.env"))
		t.Setenv("OPENROUTER_API_KEY_FILE", "")
		cfg, err := LoadWithOptions(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKeyPath != DefaultAPIKeyPath {
			t.Fatalf("Expected default key path, got %q", cfg.APIKeyPath)
		}
	})
}

func TestLoadWithOptionsAPIKeyResolution(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "openrouter.key")
	if err := os.WriteFile(keyFile, []byte("file-key\n"), 0o600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	t.Run("Uses key file when present", func(t *testing.T) {
		t.Setenv("OPENROUTER_API_KEY", "env-key")
		cfg, err := LoadWithOptions(LoadOptions{APIKeyPathOverride: keyFile})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKey != "file-key" {
			t.Fatalf("Expected file API key, got %q", cfg.APIKey)
		}
	})

	t.Run("Falls back to OPENROUTER_API_KEY", func(t *testing.T) {
		t.Setenv("OPENROUTER_API_KEY", "env-key")
		cfg, err := LoadWithOptions(LoadOptions{APIKeyPathOverride: filepath.Join(t.TempDir(), "missing.key")})
		if err != nil {
			t.Fatalf("LoadWithOptions failed: %v", err)
		}
		if cfg.APIKey != "env-key" {
			t.Fatalf("Expected env API key, got %q", cfg.APIKey)
		}
	})
}
