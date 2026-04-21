package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/domain/ports"
)

func TestAnthropic_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "This is the summary."},
			},
			"usage": map[string]any{
				"input_tokens":  int64(100),
				"output_tokens": int64(50),
			},
			"model": "claude-haiku-4-5-20250514",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:   "test-key",
		Model:    "claude-haiku-4-5",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	resp, err := provider.Generate(context.Background(), ports.GenerateRequest{
		System: "you are a classifier",
		User:   "classify this page",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Text != "This is the summary." {
		t.Errorf("Text = %q, want %q", resp.Text, "This is the summary.")
	}
	if resp.TokensInput != 100 {
		t.Errorf("TokensInput = %d, want 100", resp.TokensInput)
	}
	if resp.TokensOutput != 50 {
		t.Errorf("TokensOutput = %d, want 50", resp.TokensOutput)
	}
	if resp.Model != "claude-haiku-4-5-20250514" {
		t.Errorf("Model = %q, want %q", resp.Model, "claude-haiku-4-5-20250514")
	}
	if resp.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "anthropic")
	}
}

func TestAnthropic_401_InvalidAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:   "bad-key",
		Model:    "claude-haiku-4-5",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

// TestAnthropic_429_RateLimited tests that MaxRetries=1 returns ErrRateLimited after 1 attempt.
func TestAnthropic_429_RateLimited(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:     "test-key",
		Model:      "claude-haiku-4-5",
		Endpoint:   srv.URL,
		MaxRetries: 1, // single attempt — no retry
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 call with MaxRetries=1, got %d", got)
	}
}

func TestAnthropic_500_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"internal server error"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "test-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  1,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestAnthropic_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{},
			"usage": map[string]any{
				"input_tokens":  int64(10),
				"output_tokens": int64(0),
			},
			"model": "claude-haiku-4-5",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:   "test-key",
		Model:    "claude-haiku-4-5",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrEmptyResponse) {
		t.Errorf("expected ErrEmptyResponse, got %v", err)
	}
}

func TestAnthropic_HeadersAndBody(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		body, _ := io.ReadAll(r.Body)
		capturedBody = body

		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "summary"},
			},
			"usage": map[string]any{
				"input_tokens":  int64(10),
				"output_tokens": int64(5),
			},
			"model": "claude-haiku-4-5",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:   "my-api-key",
		Model:    "claude-haiku-4-5",
		Endpoint: srv.URL,
		Version:  "2023-06-01",
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{
		User:      "classify this",
		MaxTokens: 128,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check headers
	if got := capturedReq.Header.Get("x-api-key"); got != "my-api-key" {
		t.Errorf("x-api-key header = %q, want %q", got, "my-api-key")
	}
	if got := capturedReq.Header.Get("anthropic-version"); got != "2023-06-01" {
		t.Errorf("anthropic-version header = %q, want %q", got, "2023-06-01")
	}
	if got := capturedReq.Header.Get("content-type"); got != "application/json" {
		t.Errorf("content-type header = %q, want %q", got, "application/json")
	}

	// Check body
	var reqBody map[string]any
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if model, ok := reqBody["model"]; !ok || model != "claude-haiku-4-5" {
		t.Errorf("body model = %v, want claude-haiku-4-5", model)
	}
	if msgs, ok := reqBody["messages"]; !ok || msgs == nil {
		t.Error("body messages is missing")
	}
	if maxTok, ok := reqBody["max_tokens"]; !ok || maxTok == nil {
		t.Error("body max_tokens is missing")
	}
}

func TestAnthropic_EmptyAPIKey_New(t *testing.T) {
	_, err := NewAnthropic(AnthropicOptions{
		APIKey: "",
		Model:  "claude-haiku-4-5",
	})
	if err == nil {
		t.Fatal("expected error for empty APIKey, got nil")
	}
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestAnthropic_ModelFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Response omits model field
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "result"},
			},
			"usage": map[string]any{
				"input_tokens":  int64(5),
				"output_tokens": int64(3),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:   "test-key",
		Model:    "claude-haiku-4-5",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	resp, err := provider.Generate(context.Background(), ports.GenerateRequest{User: "test"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Model != "claude-haiku-4-5" {
		t.Errorf("Model fallback = %q, want %q", resp.Model, "claude-haiku-4-5")
	}
}

// successResponse is a helper that writes a valid Anthropic 200 response with the given text.
func successResponse(w http.ResponseWriter, text string) {
	resp := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"usage": map[string]any{
			"input_tokens":  int64(10),
			"output_tokens": int64(5),
		},
		"model": "claude-haiku-4-5",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TestAnthropic_Retry_OnRateLimit verifies that the provider retries on 429
// and succeeds on the third call.
func TestAnthropic_Retry_OnRateLimit(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		successResponse(w, "retry succeeded")
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "test-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  3,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	resp, err := provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if err != nil {
		t.Fatalf("Generate: unexpected error: %v", err)
	}
	if resp.Text != "retry succeeded" {
		t.Errorf("Text = %q, want %q", resp.Text, "retry succeeded")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected exactly 3 calls, got %d", got)
	}
}

// TestAnthropic_Retry_HonorsRetryAfter verifies that Retry-After: 0 is honoured
// and the provider succeeds on the second call.
func TestAnthropic_Retry_HonorsRetryAfter(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		successResponse(w, "after retry-after")
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "test-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  3,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	resp, err := provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if err != nil {
		t.Fatalf("Generate: unexpected error: %v", err)
	}
	if resp.Text != "after retry-after" {
		t.Errorf("Text = %q, want %q", resp.Text, "after retry-after")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected exactly 2 calls, got %d", got)
	}
}

// TestAnthropic_Retry_ExhaustReturnsError verifies that exhausting retries on 429
// returns ErrRateLimited.
func TestAnthropic_Retry_ExhaustReturnsError(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "test-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  2,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected exactly 2 calls with MaxRetries=2, got %d", got)
	}
}

// TestAnthropic_400_BadRequest verifies that a non-transient 4xx (other than 401/429)
// returns ErrProviderUnavailable immediately without retrying.
func TestAnthropic_400_BadRequest(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "test-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  3,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 call (no retry on 400), got %d", got)
	}
}

// TestAnthropic_401_NoRetry verifies that 401 is not retried.
func TestAnthropic_401_NoRetry(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	provider, err := NewAnthropic(AnthropicOptions{
		APIKey:      "bad-key",
		Model:       "claude-haiku-4-5",
		Endpoint:    srv.URL,
		MaxRetries:  3,
		BackoffBase: 10 * time.Millisecond,
		BackoffCeil: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}

	_, err = provider.Generate(context.Background(), ports.GenerateRequest{User: "hello"})
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 call (no retry on 401), got %d", got)
	}
}
