package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"screen-ocr-llm/config"
	"screen-ocr-llm/llm"
)

const (
	maxFileSizeMB  = 10
	maxFileSize    = maxFileSizeMB * 1024 * 1024
	secretFilePath = "/run/secrets/api_keys/openrouter"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define flags
	filePath := flag.String("file", "", "Path to PNG file (use '-' for stdin)")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	verbose := flag.Bool("v", false, "Verbose output to stderr")
	flag.Parse()

	// Validate required flags
	if *filePath == "" {
		return fmt.Errorf("required flag -file not specified\nUsage: ocr-tool -file <path|-> [-json] [-v]")
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Starting OCR tool\n")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Config loaded: Model=%s\n", cfg.Model)
	}

	// Load API key from multiple sources
	apiKey, err := loadAPIKey(cfg, *verbose)
	if err != nil {
		return err
	}

	// Validate model is required
	if cfg.Model == "" {
		return fmt.Errorf("MODEL is required in .env file")
	}

	// Initialize LLM package
	llm.Init(&llm.Config{
		APIKey:    apiKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	if *verbose {
		fmt.Fprintf(os.Stderr, "[verbose] LLM initialized\n")
	}

	return processOCR(*filePath, *jsonOutput, *verbose)
}

// loadAPIKey attempts to load the API key from multiple sources in priority order:
// 1. /run/secrets/api_keys/openrouter (SOPS/Kubernetes secret mount)
// 2. OPENROUTER_API_KEY environment variable
// 3. Config file (.env)
func loadAPIKey(cfg *config.Config, verbose bool) (string, error) {
	// Priority 1: Check SOPS secret file
	if data, err := os.ReadFile(secretFilePath); err == nil {
		apiKey := strings.TrimSpace(string(data))
		if apiKey != "" {
			if verbose {
				fmt.Fprintf(os.Stderr, "[verbose] API key loaded from: %s\n", secretFilePath)
			}
			return apiKey, nil
		}
	}

	// Priority 2: Check environment variable
	if envKey := os.Getenv("OPENROUTER_API_KEY"); envKey != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] API key loaded from: OPENROUTER_API_KEY env var (value: %s...)\n", envKey[:10])
		}
		return envKey, nil
	}

	// Priority 3: Check config file (already loaded by config.Load())
	if cfg.APIKey != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] API key loaded from: config file (value: %s...)\n", cfg.APIKey[:10])
		}
		return cfg.APIKey, nil
	}

	return "", fmt.Errorf("OPENROUTER_API_KEY not found. Checked:\n  1. %s\n  2. OPENROUTER_API_KEY env var\n  3. .env config file", secretFilePath)
}

func processOCR(filePath string, jsonOutput bool, verbose bool) error {
	// Read image data
	var imageData []byte
	var err error

	if filePath == "-" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] Reading image from stdin\n")
		}
		imageData, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] Reading image from file: %s\n", filePath)
		}
		imageData, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
	}

	// Validate file size
	if len(imageData) == 0 {
		return fmt.Errorf("input file is empty")
	}
	if len(imageData) > maxFileSize {
		return fmt.Errorf("input file exceeds maximum size of %d MB", maxFileSizeMB)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Read %d bytes\n", len(imageData))
	}

	// Validate PNG format
	if len(imageData) < 8 || !bytes.Equal(imageData[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		return fmt.Errorf("input is not a valid PNG file (invalid magic number)")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] PNG validation passed\n")
	}

	return performOCR(imageData, filePath, jsonOutput, verbose)
}

func performOCR(imageData []byte, sourcePath string, jsonOutput bool, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Starting OCR with model via llm.QueryVision\n")
	}

	// Use existing battle-tested implementation
	startTime := time.Now()
	text, err := llm.QueryVision(imageData)
	elapsed := time.Since(startTime)

	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] OCR failed after %v: %v\n", elapsed, err)
		}
		return fmt.Errorf("OCR failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] OCR completed in %v, extracted %d characters\n", elapsed, len(text))
	}

	return outputResult(text, sourcePath, elapsed, jsonOutput)
}

type OCRResult struct {
	Text      string  `json:"text"`
	Source    string  `json:"source"`
	Timestamp string  `json:"timestamp"`
	Duration  float64 `json:"duration_seconds"`
	CharCount int     `json:"character_count"`
}

func outputResult(text string, sourcePath string, elapsed time.Duration, jsonOutput bool) error {
	if jsonOutput {
		result := OCRResult{
			Text:      text,
			Source:    sourcePath,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Duration:  elapsed.Seconds(),
			CharCount: len(text),
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON output: %w", err)
		}
	} else {
		// Plain text output - no trailing newline
		fmt.Print(text)
	}

	return nil
}
