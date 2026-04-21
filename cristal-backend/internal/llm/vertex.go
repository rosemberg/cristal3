package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/genai"
)

// VertexConfig configures the Vertex AI / Gemini client.
type VertexConfig struct {
	ProjectID       string
	Location        string // e.g. "global" or "us-central1"
	Model           string // e.g. "gemini-3-flash-preview"
	CredentialsFile string // optional; otherwise ADC is used
	MaxTokens       int
	Temperature     float64
	Timeout         time.Duration
}

// VertexProvider implements Provider against Vertex AI via the google.golang.org/genai SDK.
type VertexProvider struct {
	cfg    VertexConfig
	client *genai.Client
}

// Name returns the provider identifier.
func (p *VertexProvider) Name() string { return "vertex" }

// NewVertex constructs a Vertex AI provider. Authentication uses Application
// Default Credentials (ADC). If CredentialsFile is set, its path is exported as
// GOOGLE_APPLICATION_CREDENTIALS for the genai SDK to pick up.
func NewVertex(ctx context.Context, cfg VertexConfig) (*VertexProvider, error) {
	if cfg.ProjectID == "" {
		return nil, errors.New("project_id is required")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	if cfg.Location == "" {
		cfg.Location = "global"
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	if cfg.CredentialsFile != "" {
		// genai reads GOOGLE_APPLICATION_CREDENTIALS via ADC.
		if err := setenvIfEmpty("GOOGLE_APPLICATION_CREDENTIALS", cfg.CredentialsFile); err != nil {
			return nil, fmt.Errorf("set credentials env: %w", err)
		}
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  cfg.ProjectID,
		Location: cfg.Location,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &VertexProvider{cfg: cfg, client: client}, nil
}

// Generate sends the canonical request through Vertex AI, normalizing the
// response back to the canonical GenerateResponse form.
func (p *VertexProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	contents := toVertexContents(req.Messages)

	// Temperature: use per-request value if non-zero, otherwise provider default.
	temperature := req.Temperature
	if temperature == 0 {
		temperature = p.cfg.Temperature
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.cfg.MaxTokens
	}

	genConfig := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(maxTokens),
	}
	if temperature != 0 {
		t := float32(temperature)
		genConfig.Temperature = &t
	}
	if req.System != "" {
		genConfig.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: req.System}},
		}
	}
	if len(req.Tools) > 0 {
		decls := make([]*genai.FunctionDeclaration, 0, len(req.Tools))
		for _, t := range req.Tools {
			decls = append(decls, &genai.FunctionDeclaration{
				Name:                 t.Name,
				Description:          t.Description,
				ParametersJsonSchema: sanitizeSchemaForVertex(t.InputSchema),
			})
		}
		genConfig.Tools = []*genai.Tool{{FunctionDeclarations: decls}}
	}

	callCtx, cancel := context.WithTimeout(ctx, p.cfg.Timeout)
	defer cancel()

	resp, err := p.client.Models.GenerateContent(callCtx, p.cfg.Model, contents, genConfig)
	if err != nil {
		return nil, fmt.Errorf("vertex generate: %w", err)
	}

	return fromVertexResponse(resp, p.cfg.Model), nil
}

// toVertexContents converts canonical Messages to Vertex AI Contents.
func toVertexContents(msgs []Message) []*genai.Content {
	out := make([]*genai.Content, 0, len(msgs))
	for _, m := range msgs {
		role := m.Role
		if role == RoleAssistant {
			role = "model"
		}
		parts := make([]*genai.Part, 0, len(m.Content))
		for _, b := range m.Content {
			switch b.Type {
			case BlockText:
				if b.Text == "" {
					continue
				}
				parts = append(parts, &genai.Part{
					Text:             b.Text,
					ThoughtSignature: b.ThoughtSignature,
				})
			case BlockToolUse:
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   b.ID,
						Name: b.Name,
						Args: b.Input,
					},
					ThoughtSignature: b.ThoughtSignature,
				})
			case BlockToolResult:
				response := map[string]any{}
				if b.IsError {
					response["error"] = b.Content
				} else {
					response["result"] = b.Content
				}
				parts = append(parts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						ID:       b.ToolUseID,
						Name:     b.Name,
						Response: response,
					},
				})
			}
		}
		if len(parts) == 0 {
			continue
		}
		out = append(out, &genai.Content{Role: role, Parts: parts})
	}
	return out
}

// fromVertexResponse converts a genai response to the canonical form.
// StopReason is derived from FinishReason plus presence of function calls.
func fromVertexResponse(resp *genai.GenerateContentResponse, model string) *GenerateResponse {
	out := &GenerateResponse{
		Model:      model,
		Role:       RoleAssistant,
		StopReason: StopOther,
	}
	if resp == nil || len(resp.Candidates) == 0 {
		return out
	}
	cand := resp.Candidates[0]

	hasToolUse := false
	if cand.Content != nil {
		for i, part := range cand.Content.Parts {
			if part == nil {
				continue
			}
			if part.FunctionCall != nil {
				id := part.FunctionCall.ID
				if id == "" {
					// Gemini does not always return an id. Synthesize a stable one
					// so our orchestrator's tool_use_id plumbing stays coherent.
					id = fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, i)
				}
				out.Content = append(out.Content, ContentBlock{
					Type:             BlockToolUse,
					ID:               id,
					Name:             part.FunctionCall.Name,
					Input:            part.FunctionCall.Args,
					ThoughtSignature: part.ThoughtSignature,
				})
				hasToolUse = true
				continue
			}
			if part.Text != "" {
				out.Content = append(out.Content, ContentBlock{
					Type:             BlockText,
					Text:             part.Text,
					ThoughtSignature: part.ThoughtSignature,
				})
			}
		}
	}

	switch cand.FinishReason {
	case genai.FinishReasonStop:
		out.StopReason = StopEndTurn
	case genai.FinishReasonMaxTokens:
		out.StopReason = StopMaxTokens
	}
	if hasToolUse {
		out.StopReason = StopToolUse
	}

	if resp.UsageMetadata != nil {
		out.Usage.InputTokens = int64(resp.UsageMetadata.PromptTokenCount)
		out.Usage.OutputTokens = int64(resp.UsageMetadata.CandidatesTokenCount)
		out.TokensInput = out.Usage.InputTokens
		out.TokensOutput = out.Usage.OutputTokens
	}
	return out
}

// sanitizeSchemaForVertex returns a deep copy of the JSON Schema with keys
// that Gemini's OpenAPI subset does not accept stripped out.
func sanitizeSchemaForVertex(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}
	unsupported := map[string]bool{
		"$schema":              true,
		"$id":                  true,
		"$defs":                true,
		"$ref":                 true,
		"$comment":             true,
		"definitions":          true,
		"additionalProperties": true, // Gemini rejects this in practice
	}
	out := make(map[string]interface{}, len(schema))
	for k, v := range schema {
		if unsupported[k] {
			continue
		}
		out[k] = sanitizeValue(v)
	}
	return out
}

func sanitizeValue(v interface{}) interface{} {
	switch vv := v.(type) {
	case map[string]interface{}:
		return sanitizeSchemaForVertex(vv)
	case []interface{}:
		arr := make([]interface{}, len(vv))
		for i, item := range vv {
			arr[i] = sanitizeValue(item)
		}
		return arr
	default:
		return v
	}
}
