package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultEndpoint = "https://api.anthropic.com"
	defaultVersion  = "2023-06-01"
	defaultTimeout  = 60 * time.Second
)

// ClaudeConfig configures the Claude API client
type ClaudeConfig struct {
	APIKey      string
	Model       string
	Endpoint    string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

// ClaudeProvider implements Provider against the Anthropic Messages API.
type ClaudeProvider struct {
	cfg    ClaudeConfig
	client *http.Client
}

// Name returns the provider identifier.
func (p *ClaudeProvider) Name() string { return "anthropic" }

// NewClaude creates a new Claude provider
func NewClaude(cfg ClaudeConfig) (*ClaudeProvider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	return &ClaudeProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Generate sends a message with optional tools and returns the response
func (p *ClaudeProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// Build request
	reqBody := map[string]interface{}{
		"model":       p.cfg.Model,
		"max_tokens":  p.cfg.MaxTokens,
		"messages":    req.Messages,
		"temperature": p.cfg.Temperature,
	}

	if req.System != "" {
		reqBody["system"] = req.System
	}

	if len(req.Tools) > 0 {
		reqBody["tools"] = req.Tools
	}

	// Marshal request
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := p.cfg.Endpoint + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", p.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", defaultVersion)
	httpReq.Header.Set("content-type", "application/json")

	// Execute request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBytes))
	}

	// Parse response
	var result GenerateResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Extract token counts
	result.TokensInput = result.Usage.InputTokens
	result.TokensOutput = result.Usage.OutputTokens

	// Normalize stop reason to canonical values.
	switch result.StopReason {
	case StopEndTurn, StopToolUse, StopMaxTokens:
		// already canonical
	case "":
		result.StopReason = StopOther
	default:
		// Claude may emit "stop_sequence" etc. Treat as other.
		result.StopReason = StopOther
	}

	return &result, nil
}

// ExtractText extracts text content from response
func ExtractText(resp *GenerateResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}

// HasToolUse checks if response contains tool_use blocks
func HasToolUse(resp *GenerateResponse) bool {
	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			return true
		}
	}
	return false
}

// ExtractToolUse extracts all tool_use blocks from response
func ExtractToolUse(resp *GenerateResponse) []ContentBlock {
	var tools []ContentBlock
	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			tools = append(tools, block)
		}
	}
	return tools
}
