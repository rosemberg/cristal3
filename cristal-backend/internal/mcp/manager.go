package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bergmaia/cristal-backend/internal/config"
)

// Manager manages multiple MCP servers and routes tool calls
type Manager struct {
	dataOrchestrator *Client
	siteResearch     *Client
	toolRoutes       map[string]*Client
	tools            []Tool
	logger           *slog.Logger
}

// NewManager creates a manager for both MCP servers
func NewManager(cfg config.MCPConfig, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create data-orchestrator client
	dataOrch, err := NewClient(ClientConfig{
		Command:    cfg.DataOrchestrator.Command,
		Args:       cfg.DataOrchestrator.Args,
		WorkingDir: cfg.DataOrchestrator.WorkingDir,
		Env:        cfg.DataOrchestrator.Env,
		Timeout:    cfg.DataOrchestrator.Timeout,
		Logger:     logger.With("server", "data-orchestrator"),
	})
	if err != nil {
		return nil, fmt.Errorf("create data-orchestrator client: %w", err)
	}

	// Create site-research client
	siteRes, err := NewClient(ClientConfig{
		Command:    cfg.SiteResearch.Command,
		Args:       cfg.SiteResearch.Args,
		WorkingDir: cfg.SiteResearch.WorkingDir,
		Env:        cfg.SiteResearch.Env,
		Timeout:    cfg.SiteResearch.Timeout,
		Logger:     logger.With("server", "site-research"),
	})
	if err != nil {
		dataOrch.Close()
		return nil, fmt.Errorf("create site-research client: %w", err)
	}

	m := &Manager{
		dataOrchestrator: dataOrch,
		siteResearch:     siteRes,
		toolRoutes:       make(map[string]*Client),
		logger:           logger,
	}

	return m, nil
}

// Initialize performs handshake with both servers and collects tools
func (m *Manager) Initialize(ctx context.Context) error {
	// Initialize data-orchestrator
	if err := m.dataOrchestrator.Initialize(); err != nil {
		return fmt.Errorf("initialize data-orchestrator: %w", err)
	}

	// Initialize site-research
	if err := m.siteResearch.Initialize(); err != nil {
		return fmt.Errorf("initialize site-research: %w", err)
	}

	// List tools from data-orchestrator
	dataTools, err := m.dataOrchestrator.ListTools()
	if err != nil {
		return fmt.Errorf("list data-orchestrator tools: %w", err)
	}

	// List tools from site-research
	siteTools, err := m.siteResearch.ListTools()
	if err != nil {
		return fmt.Errorf("list site-research tools: %w", err)
	}

	// Build tool routes
	for _, tool := range dataTools {
		m.toolRoutes[tool.Name] = m.dataOrchestrator
		m.tools = append(m.tools, tool)
	}

	for _, tool := range siteTools {
		m.toolRoutes[tool.Name] = m.siteResearch
		m.tools = append(m.tools, tool)
	}

	m.logger.Info("MCP manager initialized",
		"data_orchestrator_tools", len(dataTools),
		"site_research_tools", len(siteTools),
		"total_tools", len(m.tools))

	return nil
}

// GetTools returns all available tools from both servers
func (m *Manager) GetTools() []Tool {
	return m.tools
}

// CallTool routes the tool call to the appropriate server
func (m *Manager) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	client, ok := m.toolRoutes[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	m.logger.Debug("routing tool call", "tool", name, "args", args)

	result, err := client.CallTool(ctx, name, args)
	if err != nil {
		return nil, fmt.Errorf("tool %s failed: %w", name, err)
	}

	return result, nil
}

// Close shuts down both servers
func (m *Manager) Close() error {
	var err1, err2 error

	if m.dataOrchestrator != nil {
		err1 = m.dataOrchestrator.Close()
	}

	if m.siteResearch != nil {
		err2 = m.siteResearch.Close()
	}

	if err1 != nil {
		return fmt.Errorf("close data-orchestrator: %w", err1)
	}
	if err2 != nil {
		return fmt.Errorf("close site-research: %w", err2)
	}

	return nil
}
