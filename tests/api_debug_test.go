package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"screen-ocr-llm/src/config"
)

// TestDirectAPICall tests the OpenRouter API directly to debug the issue
func TestDirectAPICall(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a simple test request to check API connectivity
	payload := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Hello, can you see this message?",
					},
				},
			},
		},
		"temperature": 0.1,
		"max_tokens":  100,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/cherjr/screen-ocr-llm")
	req.Header.Set("X-Title", "Screen OCR Tool")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	t.Logf("Response Status: %d", resp.StatusCode)
	t.Logf("Response Body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}
}

// TestAPIWithImage tests the API with a simple image
func TestAPIWithImage(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a simple 10x10 white PNG image (larger than our previous test)
	// This is a valid 10x10 white PNG
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAAdgAAAHYBTnsmCAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAABYSURBVBiVY/z//z8DJQAggBhJVQcQQIykqgMIIEZS1QEEECO56gACiJFcdQABxEiuOoAAYiRXHUAAMZKrDiCAGMlVBxBAjOSqAwggRnLVAQQQI7nqAAKIEQAOHwEJAJWVmwAAAABJRU5ErkJggg=="

	payload := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "What do you see in this image?",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/png;base64,%s", pngData),
						},
					},
				},
			},
		},
		"temperature": 0.1,
		"max_tokens":  200,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/cherjr/screen-ocr-llm")
	req.Header.Set("X-Title", "Screen OCR Tool")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	t.Logf("Image API Response Status: %d", resp.StatusCode)
	t.Logf("Image API Response Body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Image API returned non-200 status: %d", resp.StatusCode)
	}
}
