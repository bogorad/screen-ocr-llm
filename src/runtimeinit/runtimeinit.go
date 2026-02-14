package runtimeinit

import (
	"fmt"
	"log"

	"screen-ocr-llm/src/clipboard"
	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/llm"
	"screen-ocr-llm/src/notification"
	"screen-ocr-llm/src/ocr"
	"screen-ocr-llm/src/screenshot"
)

type Options struct {
	LoadOptions          config.LoadOptions
	SetupLogging         func(bool)
	ShowBlockingLLMError bool
}

func Bootstrap(opts Options) (*config.Config, error) {
	cfg, err := config.LoadWithOptions(opts.LoadOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if opts.SetupLogging != nil {
		opts.SetupLogging(cfg.EnableFileLogging)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required. Checked key file %s and OPENROUTER_API_KEY env var", cfg.APIKeyPath)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("MODEL is required. Please set it in your .env file")
	}

	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})
	if err := llm.Ping(); err != nil {
		if opts.ShowBlockingLLMError {
			notification.ShowBlockingError("LLM unavailable", fmt.Sprintf("Startup check failed: %v\n\nPlease verify your API key and network connectivity.", err))
		}
		return nil, fmt.Errorf("startup check failed: %w", err)
	}
	log.Printf("LLM ping succeeded")

	screenshot.Init()
	ocr.Init()
	if err := clipboard.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	return cfg, nil
}
