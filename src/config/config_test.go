package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("OPENROUTER_API_KEY", "test_api_key")
	os.Setenv("MODEL", "test_model")
	os.Setenv("ENABLE_FILE_LOGGING", "true")
	os.Setenv("HOTKEY", "Ctrl+Shift+T")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("MODEL")
		os.Unsetenv("ENABLE_FILE_LOGGING")
		os.Unsetenv("HOTKEY")
	}()

	// Load the configuration
	cfg, err := Load()
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
