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
