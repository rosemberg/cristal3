package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

const (
	defaultAnthropicEndpoint = "https://api.anthropic.com"
	defaultAnthropicVersion  = "2023-06-01"
	defaultAnthropicTimeout  = 60 * time.Second
	defaultMaxTokens         = 256

	// retryAfterCap is the maximum Retry-After delay we will honour without warning.
	retryAfterCap = 60 * time.Second
)

// AnthropicOptions configures an Anthropic Messages API client.
type AnthropicOptions struct {
	APIKey     string        // required
	Model      string        // e.g., "claude-haiku-4-5"
	Endpoint   string        // default "https://api.anthropic.com"
	Version    string        // default "2023-06-01"
	Timeout    time.Duration // default 60s
	HTTPClient *http.Client  // optional; if nil, a client with Timeout is built

	// MaxRetries bounds the number of attempts on transient failures. Zero → default 3.
	MaxRetries int
	// BackoffBase is the base delay for exponential backoff; zero → default 500ms.
	BackoffBase time.Duration
	// BackoffCeil is the max delay per attempt; zero → default 10s.
	BackoffCeil time.Duration
}

// AnthropicProvider implements ports.LLMProvider targeting Anthropic Messages API.
type AnthropicProvider struct {
	opts   AnthropicOptions
	http   *http.Client
	backoff httpfetch.BackoffParams
	rng    *rand.Rand
}

// NewAnthropic builds an AnthropicProvider. Returns error if APIKey is empty.
func NewAnthropic(opts AnthropicOptions) (*AnthropicProvider, error) {
	if opts.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}
	if opts.Endpoint == "" {
		opts.Endpoint = defaultAnthropicEndpoint
	}
	if opts.Version == "" {
		opts.Version = defaultAnthropicVersion
	}
	if opts.Timeout == 0 {
		opts.Timeout = defaultAnthropicTimeout
	}

	var httpClient *http.Client
	if opts.HTTPClient != nil {
		httpClient = opts.HTTPClient
	} else {
		httpClient = &http.Client{Timeout: opts.Timeout}
	}

	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	backoff := httpfetch.BackoffParams{
		Base:     opts.BackoffBase,
		Factor:   2,
		Ceil:     opts.BackoffCeil,
		Attempts: maxRetries,
	}.WithDefaults()

	return &AnthropicProvider{
		opts:    opts,
		http:    httpClient,
		backoff: backoff,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Name returns "anthropic".
func (p *AnthropicProvider) Name() string { return "anthropic" }

// Model returns the configured model ID.
func (p *AnthropicProvider) Model() string { return p.opts.Model }

// anthropicMessage is a single message in the messages array.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the Anthropic Messages API request body.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

// anthropicContentBlock is a single block in the response content array.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicUsage holds token usage from the response.
type anthropicUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// anthropicResponse is the Anthropic Messages API response.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Usage   anthropicUsage          `json:"usage"`
	Model   string                  `json:"model"`
}

// doOneRequest performs a single HTTP attempt against the Anthropic Messages API.
// On success (2xx), returns the parsed GenerateResponse and nil error.
// On HTTP non-2xx, returns (nil, body bytes, statusCode, response headers, nil).
// On transport/network error, returns (nil, nil, 0, nil, err).
func (p *AnthropicProvider) doOneRequest(ctx context.Context, req ports.GenerateRequest) (*ports.GenerateResponse, []byte, int, http.Header, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	body := anthropicRequest{
		Model:     p.opts.Model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.User},
		},
	}
	if req.Temperature != 0 {
		body.Temperature = req.Temperature
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, 0, nil, fmt.Errorf("anthropic: marshaling request: %w", err)
	}

	url := p.opts.Endpoint + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, 0, nil, fmt.Errorf("anthropic: creating HTTP request: %w", err)
	}

	httpReq.Header.Set("x-api-key", p.opts.APIKey)
	httpReq.Header.Set("anthropic-version", p.opts.Version)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, nil, 0, nil, fmt.Errorf("%w: %w", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, resp.Header, fmt.Errorf("anthropic: reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, respBytes, resp.StatusCode, resp.Header, nil
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, nil, 0, nil, fmt.Errorf("anthropic: unmarshaling response: %w", err)
	}

	// Find first text content block.
	var text string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	if text == "" {
		return nil, nil, 0, nil, ErrEmptyResponse
	}

	model := apiResp.Model
	if model == "" {
		model = p.opts.Model
	}

	return &ports.GenerateResponse{
		Text:         text,
		TokensInput:  apiResp.Usage.InputTokens,
		TokensOutput: apiResp.Usage.OutputTokens,
		Provider:     "anthropic",
		Model:        model,
	}, nil, 0, nil, nil
}

// Generate posts a messages request and returns the first text content block.
// Retries on transient failures (429/5xx/network errors) with exponential backoff + jitter.
// Wraps HTTP/rate errors into ErrRateLimited/ErrProviderUnavailable/ErrInvalidAPIKey.
func (p *AnthropicProvider) Generate(ctx context.Context, req ports.GenerateRequest) (*ports.GenerateResponse, error) {
	for attempt := 0; attempt < p.backoff.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, body, statusCode, header, err := p.doOneRequest(ctx, req)

		// Success
		if err == nil && result != nil {
			return result, nil
		}

		// Non-retryable errors from doOneRequest itself (marshal/unmarshal/empty response)
		if err != nil && statusCode == 0 {
			// Check for context cancel/deadline first
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			// Network errors are transient
			if httpfetch.IsNetworkError(err) {
				// Fall through to transient handling below
			} else {
				// Non-network, non-transport errors (e.g. marshal, unmarshal, empty response)
				return nil, err
			}
		}

		// 401 is non-transient — return immediately, no retry
		if statusCode == http.StatusUnauthorized {
			return nil, ErrInvalidAPIKey
		}

		// Context cancel: bubble up immediately
		if errors.Is(err, context.Canceled) {
			return nil, err
		}

		// Determine if transient
		transient := false
		switch statusCode {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			transient = true
		}
		if err != nil && httpfetch.IsNetworkError(err) {
			transient = true
		}

		if !transient {
			// Other 4xx (non-401, non-429): return immediately
			if statusCode >= 500 {
				return nil, ErrProviderUnavailable
			}
			if statusCode == http.StatusTooManyRequests {
				// shouldn't reach here since 429 is transient above
				return nil, ErrRateLimited
			}
			if err != nil {
				return nil, err
			}
			return nil, ErrProviderUnavailable
		}

		// Transient. If last attempt, return the best sentinel error.
		if attempt >= p.backoff.Attempts-1 {
			switch statusCode {
			case http.StatusTooManyRequests:
				return nil, ErrRateLimited
			case http.StatusInternalServerError,
				http.StatusBadGateway,
				http.StatusServiceUnavailable,
				http.StatusGatewayTimeout:
				return nil, ErrProviderUnavailable
			}
			if err != nil {
				return nil, err
			}
			return nil, ErrProviderUnavailable
		}

		// Compute delay: prefer Retry-After header on 429/503; else exponential backoff.
		delay := p.backoff.Delay(attempt, p.rng)
		if (statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable) && header != nil {
			if ra := header.Get("Retry-After"); ra != "" {
				if raDelay, ok := httpfetch.ParseRetryAfter(ra, time.Now()); ok {
					// Cap at BackoffCeil to avoid extreme waits
					if raDelay > p.backoff.Ceil {
						raDelay = p.backoff.Ceil
					}
					delay = raDelay
				}
			}
		}

		// Suppress unused variable warning when body is not used
		_ = body

		t := time.NewTimer(delay)
		select {
		case <-t.C:
			t.Stop()
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		}
	}

	// Should be unreachable (last attempt returns inside loop)
	return nil, ErrProviderUnavailable
}
