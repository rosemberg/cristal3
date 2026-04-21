package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bergmaia/cristal-backend/internal/llm"
	"github.com/bergmaia/cristal-backend/internal/mcp"
)

const maxIterations = 20

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Config configures the orchestrator
type Config struct {
	LLM        llm.Provider
	MCPManager *mcp.Manager
	Logger     *slog.Logger
}

// Citation represents a cited page
type Citation struct {
	ID         int
	Title      string
	Breadcrumb string
	URL        string
}

// QueryResult represents the result of processing a query
type QueryResult struct {
	Response  string
	Citations []Citation
}

// Orchestrator coordinates an LLM Provider with MCP tools
type Orchestrator struct {
	llm        llm.Provider
	mcpManager *mcp.Manager
	tools      []llm.Tool
	logger     *slog.Logger
}

// New creates a new orchestrator
func New(cfg Config) (*Orchestrator, error) {
	if cfg.LLM == nil {
		return nil, fmt.Errorf("llm provider is required")
	}
	if cfg.MCPManager == nil {
		return nil, fmt.Errorf("mcp manager is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Get tools from MCP manager and convert to canonical LLM tool form.
	mcpTools := cfg.MCPManager.GetTools()
	tools := ConvertMCPToolsToLLM(mcpTools)

	cfg.Logger.Info("orchestrator initialized",
		"provider", cfg.LLM.Name(),
		"tools", len(tools))

	return &Orchestrator{
		llm:        cfg.LLM,
		mcpManager: cfg.MCPManager,
		tools:      tools,
		logger:     cfg.Logger,
	}, nil
}

// ProcessQuery processes a user query through Claude and MCP tools
// Returns response text with inline citations and array of citation metadata
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (*QueryResult, error) {
	o.logger.Info("processing query", "query", query)

	// Track citations from tool results
	var citations []Citation
	citationMap := make(map[string]int) // URL -> citation ID

	// Initialize conversation with user message
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			Content: []llm.ContentBlock{
				{Type: llm.BlockText, Text: query},
			},
		},
	}

	// Conversation loop
	for iteration := 0; iteration < maxIterations; iteration++ {
		o.logger.Debug("iteration", "n", iteration+1)

		// Send to LLM
		resp, err := o.llm.Generate(ctx, llm.GenerateRequest{
			System:   SystemPrompt,
			Messages: messages,
			Tools:    o.tools,
		})
		if err != nil {
			return nil, fmt.Errorf("llm generate: %w", err)
		}

		o.logger.Debug("llm response",
			"provider", o.llm.Name(),
			"stop_reason", resp.StopReason,
			"input_tokens", resp.TokensInput,
			"output_tokens", resp.TokensOutput)

		// If the model finished, extract text response.
		// Treat end_turn as terminal; if the model also emitted tool_use blocks
		// alongside end_turn we still honor them (fall through below).
		if resp.StopReason == llm.StopEndTurn && !llm.HasToolUse(resp) {
			text := llm.ExtractText(resp)
			if text == "" {
				return nil, fmt.Errorf("empty response from llm")
			}

			// Format inline citations [text](url) -> [text]^N
			formattedText := formatInlineCitations(text, citationMap)

			o.logger.Info("query completed", "iterations", iteration+1, "citations", len(citations))
			return &QueryResult{
				Response:  formattedText,
				Citations: citations,
			}, nil
		}

		// If the model wants to use tools
		if resp.StopReason == llm.StopToolUse || llm.HasToolUse(resp) {
			// Add assistant message with tool_use
			messages = append(messages, llm.Message{
				Role:    llm.RoleAssistant,
				Content: resp.Content,
			})

			// Execute tools and collect results
			toolResults := []llm.ContentBlock{}
			toolUses := llm.ExtractToolUse(resp)

			for _, toolUse := range toolUses {
				o.logger.Info("executing tool",
					"tool", toolUse.Name,
					"id", toolUse.ID,
					"args", toolUse.Input)

				result := o.executeTool(ctx, toolUse)
				toolResults = append(toolResults, result)

				o.logger.Info("tool result",
					"tool", toolUse.Name,
					"is_error", result.IsError,
					"content_len", len(result.Content),
					"content_preview", truncate(result.Content, 200))

				if result.IsError {
					o.logger.Warn("tool execution failed", "tool", toolUse.Name, "error", result.Content)
				} else {
					// Extract citations from site-research tools
					if toolUse.Name == "search" || toolUse.Name == "inspect_page" {
						extractCitationsFromMarkdown(result.Content, citationMap, &citations)
						o.logger.Debug("citations extracted", "total", len(citations))
					}
				}
			}

			// Add user message with tool results
			messages = append(messages, llm.Message{
				Role:    llm.RoleUser,
				Content: toolResults,
			})

			continue
		}

		// Other stop reasons (max_tokens, etc)
		o.logger.Warn("unexpected stop reason", "stop_reason", resp.StopReason)
		return nil, fmt.Errorf("unexpected stop reason: %s", resp.StopReason)
	}

	return nil, fmt.Errorf("max iterations (%d) reached without completion", maxIterations)
}

// executeTool executes a single tool via MCP
func (o *Orchestrator) executeTool(ctx context.Context, toolUse llm.ContentBlock) llm.ContentBlock {
	// Call MCP tool
	result, err := o.mcpManager.CallTool(ctx, toolUse.Name, toolUse.Input)
	if err != nil {
		return llm.ContentBlock{
			Type:      llm.BlockToolResult,
			ToolUseID: toolUse.ID,
			Name:      toolUse.Name,
			Content:   fmt.Sprintf("Error executing tool: %v", err),
			IsError:   true,
		}
	}

	// Convert MCP result to string
	content := o.formatMCPResult(result)

	return llm.ContentBlock{
		Type:      llm.BlockToolResult,
		ToolUseID: toolUse.ID,
		Name:      toolUse.Name,
		Content:   content,
		IsError:   result.IsError,
	}
}

// formatMCPResult converts MCP result to string
func (o *Orchestrator) formatMCPResult(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}

	// If single text content, return as is
	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text
	}

	// Multiple content items - format as JSON
	data, err := json.Marshal(result.Content)
	if err != nil {
		return fmt.Sprintf("Error formatting result: %v", err)
	}
	return string(data)
}
