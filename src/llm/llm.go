package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Config struct {
	APIKey    string
	Model     string
	Providers []string
}

var activeConfig atomic.Pointer[Config]
var requestSequence atomic.Uint64

func Init(cfg *Config) {
	if cfg == nil {
		activeConfig.Store(nil)
		return
	}
	snapshot := *cfg
	snapshot.Providers = append([]string(nil), cfg.Providers...)
	activeConfig.Store(&snapshot)
	if len(snapshot.Providers) > 0 {
		log.Printf("event=llm_config_loaded model=%q provider_order=%q allow_fallbacks=false", snapshot.Model, snapshot.Providers)
	} else {
		log.Printf("event=llm_config_loaded model=%q provider_order=default", snapshot.Model)
	}
}

// OpenRouter API structures
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type ProviderPreferences struct {
	Order          []string `json:"order,omitempty"`
	Quantizations  []string `json:"quantizations,omitempty"`
	AllowFallbacks *bool    `json:"allow_fallbacks,omitempty"`
}

type ChatRequest struct {
	Model       string               `json:"model"`
	Messages    []Message            `json:"messages"`
	Temperature float64              `json:"temperature"`
	MaxTokens   int                  `json:"max_tokens"`
	Provider    *ProviderPreferences `json:"provider,omitempty"`
}

type ChatResponse struct {
	ID       string      `json:"id"`
	Model    string      `json:"model"`
	Provider string      `json:"provider"`
	Choices  []Choice    `json:"choices"`
	Usage    *TokenUsage `json:"usage,omitempty"`
	Error    *APIError   `json:"error,omitempty"`
}

type Choice struct {
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
	Error        *APIError       `json:"error,omitempty"`
}

type ResponseMessage struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
}

type TokenUsage struct {
	PromptTokens           int `json:"prompt_tokens"`
	CompletionTokens       int `json:"completion_tokens"`
	CompletionTokenDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details,omitempty"`
}

type APIError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Code    interface{} `json:"code"` // Can be string or number
}

var openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// getProviderPreferences returns provider preferences based on config
func getProviderPreferences(cfg *Config) *ProviderPreferences {
	if cfg == nil || len(cfg.Providers) == 0 {
		// No providers specified, use default OpenRouter routing
		return nil
	}

	// Use the providers exactly as specified by the user
	allowFallbacks := false
	prefs := &ProviderPreferences{
		Order:          append([]string(nil), cfg.Providers...),
		AllowFallbacks: &allowFallbacks,
	}
	return prefs
}

// QueryVision sends an image to OpenRouter vision model for OCR
func QueryVision(imageData []byte) (string, error) {
	cfg := activeConfig.Load()
	if cfg == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}
	if cfg.APIKey == "" {
		return "", fmt.Errorf("API key is required")
	}
	if cfg.Model == "" {
		return "", fmt.Errorf("model is required")
	}
	log.Printf("event=llm_vision_start model=%q image_bytes=%d", cfg.Model, len(imageData))

	// Encode image as base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	imageURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	// Create the request payload matching Python implementation
	request := ChatRequest{
		Model: cfg.Model,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: "Perform OCR on this image. Return ONLY the raw extracted text with:\n" +
							"- No formatting\n" +
							"- No XML/HTML tags\n" +
							"- No markdown\n" +
							"- No explanations\n" +
							"- Preserve line breaks accurately from the visual layout.\n" +
							"If no text found, return 'NO_TEXT_FOUND'",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: imageURL,
						},
					},
				},
			},
		},
		Temperature: 0.1,
		MaxTokens:   2000,
		Provider:    getProviderPreferences(cfg),
	}

	// Single attempt - no retries, hard fail on any error
	response, err := makeAPIRequest(cfg, request)
	if err != nil {
		log.Printf("LLM: API request failed: %v", err)
		return "", fmt.Errorf("API request failed: %w", err)
	}

	// Extract text from response
	if len(response.Choices) == 0 {
		log.Printf("LLM: API response has no choices")
		return "", fmt.Errorf("no choices in API response")
	}

	extractedText := response.Choices[0].Message.Content
	log.Printf("LLM: API returned text: %d characters", len(extractedText))
	if extractedText == "" || extractedText == "NO_TEXT_FOUND" {
		log.Printf("LLM: No text detected in image (response was: %q)", extractedText)
		return "", fmt.Errorf("no text detected in image")
	}

	// Clean up any remaining artifacts
	extractedText = cleanExtractedText(extractedText)
	log.Printf("LLM: Successfully extracted %d characters", len(extractedText))
	return extractedText, nil
}

func makeAPIRequest(cfg *Config, request ChatRequest) (*ChatResponse, error) {
	return sendAPIRequest(cfg, request, 45*time.Second, "vision")
}

func makeAPIRequestWithTimeout(cfg *Config, request ChatRequest, timeout time.Duration) (*ChatResponse, error) {
	return sendAPIRequest(cfg, request, timeout, "ping")
}

func sendAPIRequest(cfg *Config, request ChatRequest, timeout time.Duration, operation string) (*ChatResponse, error) {
	id := requestSequence.Add(1)
	order := []string(nil)
	allowFallbacks := true
	if request.Provider != nil {
		order = request.Provider.Order
		if request.Provider.AllowFallbacks != nil {
			allowFallbacks = *request.Provider.AllowFallbacks
		}
	}
	log.Printf("event=llm_request_start request_id=%d operation=%s model=%q max_tokens=%d provider_order=%q allow_fallbacks=%t timeout=%s",
		id, operation, request.Model, request.MaxTokens, order, allowFallbacks, timeout)

	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Printf("event=llm_request_failed request_id=%d stage=marshal error=%q", id, err)
		return nil, fmt.Errorf("marshal API request: %w", err)
	}
	log.Printf("event=llm_request_encoded request_id=%d body_bytes=%d", id, len(jsonData))

	req, err := http.NewRequest(http.MethodPost, openRouterURL, bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("event=llm_request_failed request_id=%d stage=create error=%q", id, err)
		return nil, fmt.Errorf("create API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/cherjr/screen-ocr-llm")
	req.Header.Set("X-Title", "Screen OCR Tool")

	started := time.Now()
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	elapsed := time.Since(started)
	if err != nil {
		log.Printf("event=llm_request_failed request_id=%d stage=transport elapsed_ms=%d error=%q",
			id, elapsed.Milliseconds(), sanitizeDiagnostic(err.Error(), cfg.APIKey))
		return nil, fmt.Errorf("send API request: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("event=llm_http_response request_id=%d status=%d elapsed_ms=%d content_length=%d",
		id, resp.StatusCode, elapsed.Milliseconds(), resp.ContentLength)

	var response ChatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&response); err != nil {
		log.Printf("event=llm_request_failed request_id=%d stage=decode status=%d error=%q", id, resp.StatusCode, err)
		return nil, fmt.Errorf("decode API response (HTTP %d): %w", resp.StatusCode, err)
	}
	if response.Error != nil {
		message := sanitizeDiagnostic(response.Error.Message, cfg.APIKey)
		log.Printf("event=llm_api_error request_id=%d status=%d type=%q code=%v message=%q",
			id, resp.StatusCode, response.Error.Type, response.Error.Code, message)
		return nil, fmt.Errorf("API error (HTTP %d, type %s, code %v): %s",
			resp.StatusCode, response.Error.Type, response.Error.Code, message)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("event=llm_request_failed request_id=%d stage=http_status status=%d", id, resp.StatusCode)
		return nil, fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	finishReason := ""
	visibleChars := 0
	reasoningChars := 0
	if len(response.Choices) > 0 {
		choice := response.Choices[0]
		finishReason = choice.FinishReason
		visibleChars = len(choice.Message.Content)
		reasoningChars = len(choice.Message.Reasoning)
		if choice.Error != nil {
			message := sanitizeDiagnostic(choice.Error.Message, cfg.APIKey)
			log.Printf("event=llm_choice_error request_id=%d type=%q code=%v message=%q",
				id, choice.Error.Type, choice.Error.Code, message)
			return nil, fmt.Errorf("provider error (type %s, code %v): %s", choice.Error.Type, choice.Error.Code, message)
		}
		if finishReason == "error" {
			log.Printf("event=llm_request_failed request_id=%d stage=finish_reason finish_reason=error", id)
			return nil, fmt.Errorf("provider returned finish_reason=error")
		}
	}
	promptTokens, completionTokens, reasoningTokens := 0, 0, 0
	if response.Usage != nil {
		promptTokens = response.Usage.PromptTokens
		completionTokens = response.Usage.CompletionTokens
		if response.Usage.CompletionTokenDetails != nil {
			reasoningTokens = response.Usage.CompletionTokenDetails.ReasoningTokens
		}
	}
	log.Printf("event=llm_request_complete request_id=%d response_id=%q response_model=%q provider=%q choices=%d finish_reason=%q visible_chars=%d reasoning_chars=%d prompt_tokens=%d completion_tokens=%d reasoning_tokens=%d",
		id, response.ID, response.Model, response.Provider, len(response.Choices), finishReason,
		visibleChars, reasoningChars, promptTokens, completionTokens, reasoningTokens)
	return &response, nil
}

func sanitizeDiagnostic(message, apiKey string) string {
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[REDACTED]")
	}
	message = strings.ReplaceAll(message, "\r", "\\r")
	message = strings.ReplaceAll(message, "\n", "\\n")
	if len(message) > 500 {
		message = message[:500] + "..."
	}
	return message
}

// Ping performs a minimal LLM validation request with MaxTokens=1
// It logs success/failure and returns an error on failure. Intended to be fast.
func Ping() error {
	cfg := activeConfig.Load()
	if cfg == nil {
		return fmt.Errorf("LLM client not initialized")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}

	req := ChatRequest{
		Model: cfg.Model,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{Type: "text", Text: "Reply with a single '.' and nothing else."},
				},
			},
		},
		Temperature: 0,
		MaxTokens:   1,
		Provider:    getProviderPreferences(cfg),
	}

	start := time.Now()
	resp, err := makeAPIRequestWithTimeout(cfg, req, 8*time.Second)
	latency := time.Since(start)
	if err != nil {
		log.Printf("event=llm_ping_failed elapsed_ms=%d error=%q", latency.Milliseconds(), sanitizeDiagnostic(err.Error(), cfg.APIKey))
		return err
	}
	if len(resp.Choices) == 0 {
		log.Printf("event=llm_ping_failed elapsed_ms=%d reason=no_choices", latency.Milliseconds())
		return fmt.Errorf("startup ping returned no choices")
	}
	log.Printf("event=llm_ping_succeeded elapsed_ms=%d visible_chars=%d finish_reason=%q", latency.Milliseconds(), len(resp.Choices[0].Message.Content), resp.Choices[0].FinishReason)
	return nil
}

func cleanExtractedText(text string) string {
	// Remove any remaining image tags or artifacts
	// This matches the Python implementation
	if text == "</image>" {
		return ""
	}
	// Remove </image> tags if present
	if len(text) > 8 && text[len(text)-8:] == "</image>" {
		text = text[:len(text)-8]
	}
	return text
}
