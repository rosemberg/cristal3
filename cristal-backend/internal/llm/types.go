package llm

import "encoding/json"

// Message represents a conversation message
type Message struct {
	Role    string         `json:"role"` // "user" | "assistant"
	Content []ContentBlock `json:"content"`
}

// ContentBlock is the canonical, provider-neutral content unit.
// It can carry text, a tool invocation (tool_use), or a tool result.
type ContentBlock struct {
	Type string // BlockText | BlockToolUse | BlockToolResult

	// For Type == BlockText
	Text string

	// For Type == BlockToolUse
	ID    string
	Name  string
	Input map[string]interface{}

	// For Type == BlockToolResult
	ToolUseID string
	Content   string
	IsError   bool

	// ThoughtSignature is provider-specific opaque metadata that must
	// round-trip through the conversation history. Used by Gemini 3 on
	// function_call parts. Never serialized by the Anthropic adapter.
	ThoughtSignature []byte
}

// MarshalJSON emits the Anthropic Messages API wire format by default.
// The Vertex provider converts via its own adapter and does not rely on
// this method — so tailoring the JSON to Anthropic here is safe.
func (b ContentBlock) MarshalJSON() ([]byte, error) {
	switch b.Type {
	case BlockText:
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{b.Type, b.Text})
	case BlockToolUse:
		return json.Marshal(struct {
			Type  string                 `json:"type"`
			ID    string                 `json:"id"`
			Name  string                 `json:"name"`
			Input map[string]interface{} `json:"input"`
		}{b.Type, b.ID, b.Name, b.Input})
	case BlockToolResult:
		return json.Marshal(struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error,omitempty"`
		}{b.Type, b.ToolUseID, b.Content, b.IsError})
	default:
		return json.Marshal(struct {
			Type string `json:"type"`
		}{b.Type})
	}
}

// UnmarshalJSON parses an Anthropic Messages API content block into the
// canonical ContentBlock form.
func (b *ContentBlock) UnmarshalJSON(data []byte) error {
	var aux struct {
		Type      string                 `json:"type"`
		Text      string                 `json:"text,omitempty"`
		ID        string                 `json:"id,omitempty"`
		Name      string                 `json:"name,omitempty"`
		Input     map[string]interface{} `json:"input,omitempty"`
		ToolUseID string                 `json:"tool_use_id,omitempty"`
		Content   string                 `json:"content,omitempty"`
		IsError   bool                   `json:"is_error,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.Type = aux.Type
	b.Text = aux.Text
	b.ID = aux.ID
	b.Name = aux.Name
	b.Input = aux.Input
	b.ToolUseID = aux.ToolUseID
	b.Content = aux.Content
	b.IsError = aux.IsError
	return nil
}

// Tool is the canonical tool declaration (JSON Schema for inputs).
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// GenerateRequest is the provider-neutral request envelope.
type GenerateRequest struct {
	Model       string    `json:"model,omitempty"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// GenerateResponse is the provider-neutral response envelope.
// StopReason uses canonical constants (StopEndTurn, StopToolUse, ...).
type GenerateResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	Usage        Usage          `json:"usage"`
	TokensInput  int64          `json:"-"`
	TokensOutput int64          `json:"-"`
}

type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}
