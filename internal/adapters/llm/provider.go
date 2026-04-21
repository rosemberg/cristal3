// Package llm contains LLM provider adapters implementing ports.LLMProvider.
package llm

import "errors"

// Sentinel errors returned by LLM adapters.
var (
	ErrProviderUnavailable = errors.New("llm: provider unavailable")
	ErrRateLimited         = errors.New("llm: rate limited")
	ErrEmptyResponse       = errors.New("llm: empty response text")
	ErrInvalidAPIKey       = errors.New("llm: invalid or missing API key")
)
