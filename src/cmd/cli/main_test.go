package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"screen-ocr-llm/src/config"
)

func TestCLIWithTestImage(t *testing.T) {
	// Load configuration to check if API key is available
	cfg, err := config.Load()
	if err != nil || cfg.APIKey == "" {
		t.Skip("Skipping integration test: no API key configured")
	}

	// Build the CLI tool
	binaryPath := filepath.Join(t.TempDir(), "ocr-tool")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI tool: %v\n%s", err, output)
	}

	// Path to existing test-image.png (2 directories up from cmd/cli)
	testImagePath := "../../test-image.png"
	if _, err := os.Stat(testImagePath); err != nil {
		t.Fatalf("test-image.png not found: %v", err)
	}

	// Test 1: Plain text output
	t.Run("PlainTextOutput", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-file", testImagePath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			t.Errorf("Command failed: %v\nStderr: %s", err, stderr.String())
		}

		text := stdout.String()
		if len(text) == 0 {
			t.Error("Expected output, got empty string")
		}

		// test-image.png successfully extracted 2,198 characters previously
		if len(text) < 1000 {
			t.Errorf("Expected substantial text output (previous run: 2198 chars), got %d chars", len(text))
		}

		t.Logf("OCR extracted %d characters from test-image.png", len(text))
	})

	// Test 2: JSON output
	t.Run("JSONOutput", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-file", testImagePath, "-json")
		output, err := cmd.Output()
		if err != nil {
			t.Errorf("Command failed: %v", err)
		}

		var result OCRResult
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
		}

		if result.Text == "" {
			t.Error("JSON result missing text field")
		}
		if result.CharCount == 0 {
			t.Error("JSON result missing character count")
		}
		if result.Source != testImagePath {
			t.Errorf("Expected source=%s, got %s", testImagePath, result.Source)
		}

		t.Logf("JSON output: %d chars, duration: %.2fs", result.CharCount, result.Duration)
	})

	// Test 3: Verbose mode
	t.Run("VerboseMode", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-file", testImagePath, "-v")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		cmd.Run()

		if !strings.Contains(stderr.String(), "[verbose]") {
			t.Error("Expected verbose output in stderr")
		}
	})

	// Test 4: Stdin input
	t.Run("StdinInput", func(t *testing.T) {
		imageData, _ := os.ReadFile(testImagePath)
		cmd := exec.Command(binaryPath, "-file", "-")
		cmd.Stdin = bytes.NewReader(imageData)

		output, err := cmd.Output()
		if err != nil {
			t.Errorf("Stdin test failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("Expected output from stdin input")
		}
	})
}

func TestPNGValidation(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "ValidPNG",
			data:    []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00},
			wantErr: false,
		},
		{
			name:    "InvalidMagic",
			data:    []byte{0x00, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a},
			wantErr: true,
		},
		{
			name:    "TooShort",
			data:    []byte{0x89, 'P', 'N', 'G'},
			wantErr: true,
		},
		{
			name:    "Empty",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePNG(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePNG() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAPIKeyLoadOrder tests the API key loading priority.
// Note: This test may be affected by real secret files in /run/secrets/
// In a real deployment, the secret file takes priority as expected.
func TestAPIKeyLoadOrder(t *testing.T) {
	// This test verifies that when no secret file exists,
	// the environment variable and config file priorities work correctly.

	// Save original environment
	originalEnv := os.Getenv("OPENROUTER_API_KEY")
	defer func() {
		if originalEnv != "" {
			os.Setenv("OPENROUTER_API_KEY", originalEnv)
		} else {
			os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	// Test priority when secret file doesn't exist
	// (We'll temporarily rename the real secret file if it exists)
	secretExists := false
	backupPath := ""
	if _, err := os.Stat(secretFilePath); err == nil {
		secretExists = true
		backupPath = secretFilePath + ".backup"
		err := os.Rename(secretFilePath, backupPath)
		if err != nil {
			t.Skipf("Cannot test without secret file backup: %v", err)
		}
		defer func() {
			if backupPath != "" {
				os.Rename(backupPath, secretFilePath)
			}
		}()
	}

	// Now test without secret file
	os.Unsetenv("OPENROUTER_API_KEY")

	// Test 1: Environment variable takes priority over config
	os.Setenv("OPENROUTER_API_KEY", "test-key-from-env")
	cfg := &config.Config{APIKey: "test-key-from-config"}
	key, err := loadAPIKey(cfg, false)
	if err != nil {
		t.Errorf("Expected to load from env, got error: %v", err)
	}
	if key != "test-key-from-env" {
		t.Errorf("Expected 'test-key-from-env', got '%s'", key)
	}

	// Test 2: Config file used when env var is missing
	os.Unsetenv("OPENROUTER_API_KEY")
	cfg = &config.Config{APIKey: "test-key-from-config"}
	key, err = loadAPIKey(cfg, false)
	if err != nil {
		t.Errorf("Expected to load from config, got error: %v", err)
	}
	if key != "test-key-from-config" {
		t.Errorf("Expected 'test-key-from-config', got '%s'", key)
	}

	// Test 3: No API key found
	os.Unsetenv("OPENROUTER_API_KEY")
	cfg = &config.Config{}
	_, err = loadAPIKey(cfg, false)
	if err == nil {
		t.Error("Expected error when no API key found")
	}
	expectedError := "OPENROUTER_API_KEY not found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}

	// Restore secret file if it existed
	if secretExists && backupPath != "" {
		os.Rename(backupPath, secretFilePath)
	}
}

func TestTruncateSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		maxLen int
		want   string
	}{
		{
			name:   "Secret longer than maxLen",
			secret: "sk-or-v1-1234567890abcdef",
			maxLen: 10,
			want:   "sk-or-v1-1...",
		},
		{
			name:   "Secret shorter than maxLen",
			secret: "short",
			maxLen: 10,
			want:   "short...",
		},
		{
			name:   "Secret equal to maxLen",
			secret: "exactly10c",
			maxLen: 10,
			want:   "exactly10c...",
		},
		{
			name:   "Empty secret",
			secret: "",
			maxLen: 10,
			want:   "...",
		},
		{
			name:   "Single character",
			secret: "x",
			maxLen: 10,
			want:   "x...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSecret(tt.secret, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateSecret() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestStdoutStderrSeparation verifies that only OCR result goes to stdout,
// and all other output (errors, verbose logs) goes to stderr
func TestStdoutStderrSeparation(t *testing.T) {
	cfg, err := config.Load()
	if err != nil || cfg.APIKey == "" {
		t.Skip("Skipping integration test: no API key configured")
	}

	binaryPath := filepath.Join(t.TempDir(), "ocr-tool")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI tool: %v\n%s", err, output)
	}

	testImagePath := "../../test-image.png"
	if _, err := os.Stat(testImagePath); err != nil {
		t.Fatalf("test-image.png not found: %v", err)
	}

	// Test 1: Without -v flag, stderr should be empty (no verbose output)
	t.Run("NoVerboseNoStderr", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-file", testImagePath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			t.Errorf("Command failed: %v", err)
		}

		// stdout should have OCR result
		if stdout.Len() == 0 {
			t.Error("Expected OCR result in stdout")
		}

		// stderr should be EMPTY (no verbose output)
		if stderr.Len() > 0 {
			t.Errorf("Expected empty stderr without -v flag, got: %s", stderr.String())
		}
	})

	// Test 2: With -v flag, stderr should have verbose logs, stdout only OCR result
	t.Run("VerboseToStderrOnly", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-file", testImagePath, "-v")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			t.Errorf("Command failed: %v", err)
		}

		// stdout should ONLY have OCR result (no [verbose] tags)
		if strings.Contains(stdout.String(), "[verbose]") {
			t.Error("Found [verbose] in stdout - should only be in stderr")
		}

		// stderr should have [verbose] tags
		if !strings.Contains(stderr.String(), "[verbose]") {
			t.Error("Expected [verbose] logs in stderr with -v flag")
		}
	})

	// Test 3: Error should go to stderr, not stdout
	t.Run("ErrorToStderr", func(t *testing.T) {
		// Try to read non-existent file
		cmd := exec.Command(binaryPath, "-file", "/nonexistent/file.png")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Command should fail
		if err := cmd.Run(); err == nil {
			t.Error("Expected command to fail for non-existent file")
		}

		// stdout should be EMPTY
		if stdout.Len() > 0 {
			t.Errorf("Expected empty stdout on error, got: %s", stdout.String())
		}

		// stderr should have error message
		if !strings.Contains(stderr.String(), "Error:") {
			t.Error("Expected error message in stderr")
		}
	})
}

func validatePNG(data []byte) error {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		return fmt.Errorf("invalid PNG")
	}
	return nil
}
