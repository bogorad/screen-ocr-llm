package llm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Config struct {
	APIKey    string
	Model     string
	Providers []string
}

var config *Config

func Init(cfg *Config) {
	config = cfg
	if len(cfg.Providers) > 0 {
		log.Printf("LLM: Initialized with %d provider(s): %v", len(cfg.Providers), cfg.Providers)
	} else {
		log.Printf("LLM: Initialized with no specific providers (using OpenRouter default routing)")
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
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	Content string `json:"content"`
}

type APIError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Code    interface{} `json:"code"` // Can be string or number
}

const (
	openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
)

// getProviderPreferences returns provider preferences based on config
func getProviderPreferences() *ProviderPreferences {
	if config == nil || len(config.Providers) == 0 {
		// No providers specified, use default OpenRouter routing
		log.Printf("LLM: No provider preferences configured, using OpenRouter default routing")
		return nil
	}

	// Use the providers exactly as specified by the user
	allowFallbacks := false
	prefs := &ProviderPreferences{
		Order:          config.Providers,
		AllowFallbacks: &allowFallbacks,
	}
	log.Printf("LLM: Using provider preferences: order=%v, allow_fallbacks=%v", prefs.Order, *prefs.AllowFallbacks)
	return prefs
}

// QueryVision sends an image to OpenRouter vision model for OCR
func QueryVision(imageData []byte) (string, error) {
	if config == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}
	if config.APIKey == "" {
		return "", fmt.Errorf("API key is required")
	}
	if config.Model == "" {
		return "", fmt.Errorf("model is required")
	}

	// Encode image as base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	imageURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	// Create the request payload matching Python implementation
	request := ChatRequest{
		Model: config.Model,
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
		Provider:    getProviderPreferences(),
	}

	// Single attempt - no retries, hard fail on any error
	response, err := makeAPIRequest(request)
	if err != nil {
		log.Printf("LLM: API request failed: %v", err)
		return "", fmt.Errorf("API request failed: %v", err)
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

func makeAPIRequest(request ChatRequest) (*ChatResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Log the request for debugging (only log provider preferences, not the full request with image data)
	if request.Provider != nil {
		log.Printf("LLM: API request includes provider preferences: %+v", request.Provider)
	} else {
		log.Printf("LLM: API request without provider preferences (using default routing)")
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/cherjr/screen-ocr-llm")
	req.Header.Set("X-Title", "Screen OCR Tool")

	// Make the request
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("LLM: API response status: %d %s", resp.StatusCode, resp.Status)

	// Parse response
	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for API errors
	if response.Error != nil {
		log.Printf("LLM: API error response: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
		return nil, fmt.Errorf("API error: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	log.Printf("LLM: API response parsed successfully, %d choices", len(response.Choices))
	return &response, nil
}

// makeAPIRequestWithTimeout is like makeAPIRequest but allows a custom HTTP timeout (used by Ping)
func makeAPIRequestWithTimeout(request ChatRequest, timeout time.Duration) (*ChatResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Log the request for debugging (only log provider preferences)
	if request.Provider != nil {
		log.Printf("LLM: API request includes provider preferences: %+v", request.Provider)
	} else {
		log.Printf("LLM: API request without provider preferences (using default routing)")
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/cherjr/screen-ocr-llm")
	req.Header.Set("X-Title", "Screen OCR Tool")

	// Make the request with custom timeout
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("LLM: API response status: %d %s", resp.StatusCode, resp.Status)

	// Parse response
	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for API errors
	if response.Error != nil {
		log.Printf("LLM: API error response: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
		return nil, fmt.Errorf("API error: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	log.Printf("LLM: API response parsed successfully, %d choices", len(response.Choices))
	return &response, nil
}

// Ping performs a minimal LLM validation request with MaxTokens=1
// It logs success/failure and returns an error on failure. Intended to be fast.
func Ping() error {
	if config == nil {
		return fmt.Errorf("LLM client not initialized")
	}
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if config.Model == "" {
		return fmt.Errorf("model is required")
	}

	req := ChatRequest{
		Model: config.Model,
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
		Provider:    getProviderPreferences(),
	}

	start := time.Now()
	resp, err := makeAPIRequestWithTimeout(req, 8*time.Second)
	latency := time.Since(start)
	if err != nil {
		log.Printf("LLM: Ping failed after %dms: %v", latency.Milliseconds(), err)
		return err
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		log.Printf("LLM: Ping returned empty response after %dms", latency.Milliseconds())
		return fmt.Errorf("empty response")
	}
	log.Printf("LLM: Ping successful in %dms", latency.Milliseconds())
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


