package llm

import "context"

// StopReason is a provider-neutral enumeration of why generation ended.
const (
	StopEndTurn   = "end_turn"
	StopToolUse   = "tool_use"
	StopMaxTokens = "max_tokens"
	StopOther     = "other"
)

// ContentBlock types (provider-neutral canonical form).
const (
	BlockText       = "text"
	BlockToolUse    = "tool_use"
	BlockToolResult = "tool_result"
)

// Roles (provider-neutral canonical form).
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Provider is the abstraction over LLM backends (Anthropic, Vertex AI, ...).
// Implementations translate the canonical GenerateRequest/GenerateResponse
// to and from their native wire formats and normalize StopReason.
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Name() string
}
