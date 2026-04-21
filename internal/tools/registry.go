// Package tools contains the MCP tool registry and handlers.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bergmaia/site-research/internal/config"
)

// CallToolResult is the result of a tool invocation.
// IsError signals that the tool itself encountered a domain error
// (distinct from a JSON-RPC transport error).
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a single content item in a CallToolResult.
// Only type "text" is used in this server.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Handler is the function signature every tool must implement.
// args is the raw JSON object sent by the client as tools/call arguments.
type Handler func(ctx context.Context, args json.RawMessage) (CallToolResult, error)

// Tool holds the metadata and handler for a single MCP tool.
// InputSchema is stored as json.RawMessage so it is embedded verbatim in the
// tools/list response without double-marshalling.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// Registry maps tool names to Tool definitions and their Handlers.
type Registry struct {
	tools    []Tool
	handlers map[string]Handler
}

// Tools returns the list of registered tools in registration order.
func (r *Registry) Tools() []Tool {
	return r.tools
}

// Handler returns the Handler for the named tool and a bool indicating whether
// the tool exists.
func (r *Registry) Handler(name string) (Handler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}

// register adds a tool and its handler.
func (r *Registry) register(t Tool, h Handler) {
	r.tools = append(r.tools, t)
	r.handlers[t.Name] = h
}

// RegisterTool is the exported variant of register, intended for tests and
// custom registries assembled outside DefaultRegistry.
func (r *Registry) RegisterTool(t Tool, h Handler) {
	r.register(t, h)
}

// NewRegistry returns an empty Registry ready for use.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// ---------------------------------------------------------------------------
// DefaultRegistry constructs the registry with the three standard tools.
// In M1 every handler returns a "not implemented" response.
// ---------------------------------------------------------------------------

// DefaultRegistry returns a Registry pre-loaded with the three MCP tools.
func DefaultRegistry(cfg *config.Config, logger *slog.Logger) *Registry {
	r := &Registry{handlers: make(map[string]Handler)}

	r.register(searchTool(), searchHandler(cfg, logger))
	r.register(inspectPageTool(), inspectHandler(cfg, logger))
	r.register(catalogStatsTool(), statsHandler(cfg, logger))

	return r
}

// okResult wraps markdown text in a successful CallToolResult.
func okResult(md string) CallToolResult {
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: md}},
	}
}

// errorResult wraps an error message in a CallToolResult with IsError=true.
func errorResult(msg string) CallToolResult {
	return CallToolResult{
		IsError: true,
		Content: []ContentBlock{{Type: "text", Text: msg}},
	}
}

// stubHandler is kept for test use only.
func stubHandler(name string, logger *slog.Logger) Handler {
	return func(_ context.Context, _ json.RawMessage) (CallToolResult, error) {
		logger.Warn("stub tool called", "tool", name)
		return CallToolResult{
			IsError: true,
			Content: []ContentBlock{
				{Type: "text", Text: fmt.Sprintf("tool %s stub called", name)},
			},
		}, nil
	}
}

// ---------------------------------------------------------------------------
// Tool definitions — schemas are Go struct literals serialised to JSON.
// ---------------------------------------------------------------------------

func searchTool() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Consulta em português, em linguagem natural (ex: 'balancetes 2025', 'diárias magistrados'). Obrigatório.",
				"minLength":   1,
				"maxLength":   500,
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Número máximo de resultados. Default 10, máximo 50.",
				"minimum":     1,
				"maximum":     50,
				"default":     10,
			},
			"section": map[string]any{
				"type":        "string",
				"description": "Filtra resultados à seção indicada (ex: 'Contabilidade', 'Recursos Humanos'). Opcional.",
				"maxLength":   120,
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
	raw := mustMarshal(schema)
	return Tool{
		Name: "search",
		Description: "Busca páginas no catálogo do portal de transparência do TRE-PI que respondam a uma consulta em linguagem natural. " +
			"Use quando o usuário quiser descobrir conteúdo oficial sobre um tópico (ex: \"balancetes de março\", \"diárias de servidores\", \"contratos vigentes\"). " +
			"Retorna top-N páginas com título, mini-resumo e URL oficial. " +
			"Não use para consultar dados tabulares dentro de anexos — este servidor apenas localiza páginas; o usuário deve abrir o PDF/XLSX pela URL retornada.",
		InputSchema: raw,
	}
}

func inspectPageTool() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{
				"type":        "string",
				"description": "URL completa (https://...) ou path relativo ao escopo do catálogo (ex: 'contabilidade/balancetes'). Obrigatório.",
				"minLength":   1,
				"maxLength":   500,
			},
		},
		"required":             []string{"target"},
		"additionalProperties": false,
	}
	raw := mustMarshal(schema)
	return Tool{
		Name: "inspect_page",
		Description: "Retorna metadados completos de uma única página do catálogo: título, seção, breadcrumb, mini-resumo, tipo de página, " +
			"documentos anexos listados, datas de publicação/atualização, e URLs de páginas filhas. " +
			"Use quando o usuário perguntar detalhes sobre uma página específica retornada anteriormente por search, " +
			"ou quando precisar entender a estrutura hierárquica ao redor de uma página. " +
			"Aceita URL completa ou path relativo ao escopo (ex: 'contabilidade/balancetes').",
		InputSchema: raw,
	}
}

func catalogStatsTool() Tool {
	schema := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	raw := mustMarshal(schema)
	return Tool{
		Name: "catalog_stats",
		Description: "Retorna estatísticas agregadas sobre o catálogo: total de páginas, distribuição por profundidade hierárquica, " +
			"distribuição por tipo (landing/article/listing/empty), top seções por volume, páginas sem mini-resumo, " +
			"documentos anexos detectados, e páginas marcadas como stale. " +
			"Use quando o usuário quiser uma visão geral do tamanho e cobertura do conteúdo indexado antes de formular buscas mais específicas.",
		InputSchema: raw,
	}
}

// mustMarshal marshals v to JSON and panics on error.
// Schemas are compile-time constants so a marshal failure is a programming error.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("tools: marshal schema: %v", err))
	}
	return json.RawMessage(b)
}
