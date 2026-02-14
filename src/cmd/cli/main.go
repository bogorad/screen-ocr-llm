package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/llm"
)

const (
	maxFileSizeMB = 10
	maxFileSize   = maxFileSizeMB * 1024 * 1024
)

type cliOptions struct {
	filePath   string
	jsonOutput bool
	verbose    bool
	apiKeyPath string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return runWithArgs(normalizeLegacyArgs(os.Args))
}

func runWithArgs(args []string) error {
	if len(args) == 0 {
		args = []string{"ocr-tool"}
	}

	opts := &cliOptions{}
	cmd := newRootCmd(opts)
	cmd.SetArgs(args[1:])
	return cmd.Execute()
}

func newRootCmd(opts *cliOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ocr-tool",
		Short:         "Run OCR on PNG input",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithOptions(*opts)
		},
	}

	cmd.Flags().StringVar(&opts.filePath, "file", "", "Path to PNG file (use '-' for stdin)")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Verbose output to stderr")
	cmd.Flags().StringVar(&opts.apiKeyPath, "api-key-path", "", "Path to API key file (highest precedence)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runWithOptions(opts cliOptions) error {
	// Configure logging BEFORE any other operations.
	if !opts.verbose {
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stderr)
		fmt.Fprintf(os.Stderr, "[verbose] Starting OCR tool\n")
	}

	loadOptions := config.LoadOptions{APIKeyPathOverride: opts.apiKeyPath}
	cfg, err := config.LoadWithOptions(loadOptions)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Config loaded: Model=%s\n", cfg.Model)
		fmt.Fprintf(os.Stderr, "[verbose] Effective API key path: %s\n", cfg.APIKeyPath)
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not found. Checked key file %s and OPENROUTER_API_KEY env var", cfg.APIKeyPath)
	}

	if cfg.Model == "" {
		return fmt.Errorf("MODEL is required in .env file")
	}

	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "[verbose] LLM initialized\n")
	}

	return processOCR(opts.filePath, opts.jsonOutput, opts.verbose)
}

func normalizeLegacyArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	normalized := make([]string, len(args))
	copy(normalized, args)

	for i := 1; i < len(normalized); i++ {
		arg := normalized[i]
		switch {
		case arg == "-file":
			normalized[i] = "--file"
		case strings.HasPrefix(arg, "-file="):
			normalized[i] = "--file=" + arg[len("-file="):]
		case arg == "-json":
			normalized[i] = "--json"
		case strings.HasPrefix(arg, "-json="):
			normalized[i] = "--json=" + arg[len("-json="):]
		case arg == "-verbose":
			normalized[i] = "--verbose"
		case strings.HasPrefix(arg, "-verbose="):
			normalized[i] = "--verbose=" + arg[len("-verbose="):]
		case arg == "-api-key-path":
			normalized[i] = "--api-key-path"
		case strings.HasPrefix(arg, "-api-key-path="):
			normalized[i] = "--api-key-path=" + arg[len("-api-key-path="):]
		}
	}

	return normalized
}

// truncateSecret safely truncates a secret for display, showing only first N characters.
func truncateSecret(secret string, maxLen int) string {
	if len(secret) <= maxLen {
		return secret + "..."
	}
	return secret[:maxLen] + "..."
}

func processOCR(filePath string, jsonOutput bool, verbose bool) error {
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

	if len(imageData) == 0 {
		return fmt.Errorf("input file is empty")
	}
	if len(imageData) > maxFileSize {
		return fmt.Errorf("input file exceeds maximum size of %d MB", maxFileSizeMB)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] Read %d bytes\n", len(imageData))
	}

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
		fmt.Print(text)
	}

	return nil
}
