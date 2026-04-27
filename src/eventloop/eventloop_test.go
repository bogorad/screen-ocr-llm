package eventloop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/llm"
)

func TestRefreshRuntimeConfigDisabledSkipsLoad(t *testing.T) {
	called := false
	l := &Loop{
		reloadConfigOnGrab: false,
		loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
			called = true
			return &config.Config{}, nil
		},
	}

	if err := l.refreshRuntimeConfig(); err != nil {
		t.Fatalf("refreshRuntimeConfig returned error: %v", err)
	}
	if called {
		t.Fatal("expected config loader to be skipped when reload is disabled")
	}
}

func TestRefreshRuntimeConfigAppliesUpdatedValues(t *testing.T) {
	var capturedLoadOpts config.LoadOptions
	var capturedLLM *llm.Config

	l := &Loop{
		reloadConfigOnGrab: true,
		defaultMode:        config.DefaultModeRect,
		deadline:           20 * time.Second,
		loadOptions: config.LoadOptions{
			APIKeyPathOverride:  "/tmp/override.key",
			DefaultModeOverride: "lasso",
		},
		loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
			capturedLoadOpts = opts
			return &config.Config{
				APIKey:             "new-api-key",
				APIKeyPath:         "/tmp/override.key",
				Model:              "qwen/qwen3-vl",
				Providers:          []string{"openrouter"},
				DefaultMode:        config.DefaultModeLasso,
				OCRDeadlineSec:     9,
				ReloadConfigOnGrab: true,
			}, nil
		},
		llmInit: func(cfg *llm.Config) {
			capturedLLM = cfg
		},
	}

	if err := l.refreshRuntimeConfig(); err != nil {
		t.Fatalf("refreshRuntimeConfig returned error: %v", err)
	}

	if capturedLoadOpts.APIKeyPathOverride != "/tmp/override.key" {
		t.Fatalf("expected APIKeyPathOverride to be propagated, got %q", capturedLoadOpts.APIKeyPathOverride)
	}
	if capturedLoadOpts.DefaultModeOverride != "lasso" {
		t.Fatalf("expected DefaultModeOverride to be propagated, got %q", capturedLoadOpts.DefaultModeOverride)
	}

	if capturedLLM == nil {
		t.Fatal("expected llm init to be called")
	}
	if capturedLLM.APIKey != "new-api-key" {
		t.Fatalf("expected updated api key, got %q", capturedLLM.APIKey)
	}
	if capturedLLM.Model != "qwen/qwen3-vl" {
		t.Fatalf("expected updated model, got %q", capturedLLM.Model)
	}
	if len(capturedLLM.Providers) != 1 || capturedLLM.Providers[0] != "openrouter" {
		t.Fatalf("expected updated providers [openrouter], got %v", capturedLLM.Providers)
	}

	if l.defaultMode != config.DefaultModeLasso {
		t.Fatalf("expected updated default mode %q, got %q", config.DefaultModeLasso, l.defaultMode)
	}
	if l.deadline != 9*time.Second {
		t.Fatalf("expected updated deadline 9s, got %v", l.deadline)
	}
}

func TestRefreshRuntimeConfigLoadError(t *testing.T) {
	l := &Loop{
		reloadConfigOnGrab: true,
		loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
			return nil, errors.New("boom")
		},
	}

	err := l.refreshRuntimeConfig()
	if err == nil {
		t.Fatal("expected reload error")
	}
}

func TestRefreshRuntimeConfigSkipsReloadWhenUnchanged(t *testing.T) {
	t.Run("content hash unchanged", func(t *testing.T) {
		t.Setenv("SCREEN_OCR_LLM", filepath.Join(t.TempDir(), ".env"))
		cfgPath := os.Getenv("SCREEN_OCR_LLM")

		initial := "OPENROUTER_API_KEY=initial-key\nMODEL=qwen/qwen3-vl\nOCR_DEADLINE_SEC=10\n"
		if err := os.WriteFile(cfgPath, []byte(initial), 0o600); err != nil {
			t.Fatalf("Failed to write env file: %v", err)
		}

		callCount := 0
		l := &Loop{
			reloadConfigOnGrab: true,
			loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
				callCount++
				return &config.Config{APIKey: "updated-key", Model: "qwen/qwen3-vl", Providers: []string{"openrouter"}, OCRDeadlineSec: 10, ReloadConfigOnGrab: true}, nil
			},
			llmInit: func(cfg *llm.Config) {
				_ = cfg
			},
		}

		if err := l.refreshRuntimeConfig(); err != nil {
			t.Fatalf("first refreshRuntimeConfig failed: %v", err)
		}
		if callCount != 1 {
			t.Fatalf("expected first refresh to load config, got %d calls", callCount)
		}

		if err := os.WriteFile(cfgPath, []byte(initial), 0o600); err != nil {
			t.Fatalf("Failed to rewrite env file: %v", err)
		}
		if err := l.refreshRuntimeConfig(); err != nil {
			t.Fatalf("second refreshRuntimeConfig failed: %v", err)
		}
		if callCount != 1 {
			t.Fatalf("expected no reload when content hash unchanged, got %d calls", callCount)
		}
	})
}

func TestRefreshRuntimeConfigReloadsOnContentChange(t *testing.T) {
	t.Setenv("SCREEN_OCR_LLM", filepath.Join(t.TempDir(), ".env"))
	cfgPath := os.Getenv("SCREEN_OCR_LLM")

	if err := os.WriteFile(cfgPath, []byte("OPENROUTER_API_KEY=initial\nMODEL=one\n\n"), 0o600); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	callCount := 0
	l := &Loop{
		reloadConfigOnGrab: true,
		loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
			callCount++
			if callCount == 1 {
				return &config.Config{APIKey: "initial-key", Model: "one", Providers: []string{"openrouter"}, OCRDeadlineSec: 10, ReloadConfigOnGrab: true}, nil
			}
			return &config.Config{APIKey: "updated-key", Model: "two", Providers: []string{"openrouter"}, OCRDeadlineSec: 10, ReloadConfigOnGrab: true}, nil
		},
		llmInit: func(cfg *llm.Config) {
			_ = cfg
		},
	}

	if err := l.refreshRuntimeConfig(); err != nil {
		t.Fatalf("first refreshRuntimeConfig failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected first refresh to load config, got %d calls", callCount)
	}

	if err := os.WriteFile(cfgPath, []byte("OPENROUTER_API_KEY=updated-key\nMODEL=two-changed\n\n"), 0o600); err != nil {
		t.Fatalf("Failed to rewrite env file: %v", err)
	}

	if err := l.refreshRuntimeConfig(); err != nil {
		t.Fatalf("second refreshRuntimeConfig failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected reload on content change, got %d calls", callCount)
	}
}

func TestLoadConfigSourceStateDetectsConfigFileChange(t *testing.T) {
	t.Setenv("SCREEN_OCR_LLM", filepath.Join(t.TempDir(), ".env"))
	cfgPath := os.Getenv("SCREEN_OCR_LLM")

	if err := os.WriteFile(cfgPath, []byte("OPENROUTER_API_KEY=initial\nMODEL=one\n\n"), 0o600); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	first, err := loadConfigSourceState(configSourceState{}, cfgPath)
	if err != nil {
		t.Fatalf("initial config source read failed: %v", err)
	}

	if err := os.WriteFile(cfgPath, []byte("OPENROUTER_API_KEY=updated-key\nMODEL=two-changed\n\n"), 0o600); err != nil {
		t.Fatalf("Failed to rewrite env file: %v", err)
	}

	second, err := loadConfigSourceState(first, cfgPath)
	if err != nil {
		t.Fatalf("updated config source read failed: %v", err)
	}

	if configSourceStateEqual(first, second) {
		t.Fatal("expected config source states to differ after content change")
	}
}

func TestRefreshRuntimeConfigValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "missing api key",
			cfg: &config.Config{
				APIKeyPath:         "/tmp/missing.key",
				Model:              "qwen/qwen3-vl",
				ReloadConfigOnGrab: true,
			},
		},
		{
			name: "missing model",
			cfg: &config.Config{
				APIKey:             "api-key",
				ReloadConfigOnGrab: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Loop{
				reloadConfigOnGrab: true,
				loadConfig: func(opts config.LoadOptions) (*config.Config, error) {
					return tt.cfg, nil
				},
			}

			err := l.refreshRuntimeConfig()
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
