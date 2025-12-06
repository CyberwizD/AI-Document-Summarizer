package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const openRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type AnalysisResult struct {
	Summary      string                 `json:"summary"`
	DocumentType string                 `json:"document_type"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func NewClient(apiKey, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// AnalyzeDocument sends the extracted text to OpenRouter and expects a JSON payload back.
func (c *Client) AnalyzeDocument(ctx context.Context, text string) (AnalysisResult, error) {
	if c.apiKey == "" {
		return AnalysisResult{}, fmt.Errorf("missing OPENROUTER_API_KEY")
	}

	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You are a document analysis assistant. Given the document text, return a concise JSON object with fields: summary (max 5 sentences), document_type (invoice, CV, report, letter, etc.), metadata (key-value pairs for dates, sender/receiver, totals, subject). Respond with JSON only.",
			},
			{
				"role":    "user",
				"content": text,
			},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("call openrouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return AnalysisResult{}, fmt.Errorf("openrouter returned status %d", resp.StatusCode)
	}

	var raw struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return AnalysisResult{}, fmt.Errorf("decode response: %w", err)
	}
	if len(raw.Choices) == 0 {
		return AnalysisResult{}, fmt.Errorf("no choices returned from LLM")
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(raw.Choices[0].Message.Content), &result); err != nil {
		return AnalysisResult{}, fmt.Errorf("parse llm content: %w", err)
	}
	return result, nil
}
