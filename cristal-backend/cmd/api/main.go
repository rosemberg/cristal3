package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bergmaia/cristal-backend/internal/config"
	"github.com/bergmaia/cristal-backend/internal/llm"
	"github.com/bergmaia/cristal-backend/internal/mcp"
	"github.com/bergmaia/cristal-backend/internal/orchestrator"
	"github.com/bergmaia/cristal-backend/internal/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting cristal-backend")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return 1
	}

	logger.Info("config loaded",
		"port", cfg.Server.Port,
		"llm_provider", cfg.LLM.Provider)

	// Initialize MCP manager
	logger.Info("initializing MCP manager...")
	mcpManager, err := mcp.NewManager(cfg.MCP, logger)
	if err != nil {
		logger.Error("failed to create MCP manager", "error", err)
		return 1
	}
	defer func() {
		logger.Info("closing MCP manager...")
		if err := mcpManager.Close(); err != nil {
			logger.Error("failed to close MCP manager", "error", err)
		}
	}()

	// Initialize MCP servers (with timeout)
	initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mcpManager.Initialize(initCtx); err != nil {
		logger.Error("failed to initialize MCP servers", "error", err)
		return 1
	}

	logger.Info("MCP servers initialized", "tools", len(mcpManager.GetTools()))

	// Initialize LLM provider
	provider, err := buildLLMProvider(initCtx, cfg, logger)
	if err != nil {
		logger.Error("failed to create llm provider", "error", err)
		return 1
	}

	// Initialize orchestrator
	logger.Info("initializing orchestrator...")
	orch, err := orchestrator.New(orchestrator.Config{
		LLM:        provider,
		MCPManager: mcpManager,
		Logger:     logger,
	})
	if err != nil {
		logger.Error("failed to create orchestrator", "error", err)
		return 1
	}

	// Create and start server
	srv := server.New(server.Config{
		Orchestrator: orch,
		Logger:       logger,
		Port:         cfg.Server.Port,
	})

	logger.Info("starting HTTP server", "port", cfg.Server.Port)
	fmt.Printf("\n🚀 Cristal Backend running on http://localhost:%d (llm=%s)\n\n",
		cfg.Server.Port, provider.Name())
	fmt.Println("Endpoints:")
	fmt.Println("  POST /chat    - Send chat message")
	fmt.Println("  GET  /health  - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		return 1
	}

	return 0
}

// buildLLMProvider selects and constructs the LLM provider based on config.
func buildLLMProvider(ctx context.Context, cfg *config.Config, logger *slog.Logger) (llm.Provider, error) {
	switch cfg.LLM.Provider {
	case config.ProviderAnthropic:
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required when llm.provider=anthropic")
		}
		logger.Info("initializing Anthropic provider", "model", cfg.Anthropic.Model)
		return llm.NewClaude(llm.ClaudeConfig{
			APIKey:      apiKey,
			Model:       cfg.Anthropic.Model,
			Endpoint:    cfg.Anthropic.Endpoint,
			MaxTokens:   cfg.LLM.MaxTokens,
			Temperature: cfg.LLM.Temperature,
			Timeout:     cfg.LLM.Timeout,
		})
	case config.ProviderVertex:
		logger.Info("initializing Vertex AI provider",
			"project", cfg.Vertex.ProjectID,
			"location", cfg.Vertex.Location,
			"model", cfg.Vertex.Model)
		return llm.NewVertex(ctx, llm.VertexConfig{
			ProjectID:       cfg.Vertex.ProjectID,
			Location:        cfg.Vertex.Location,
			Model:           cfg.Vertex.Model,
			CredentialsFile: cfg.Vertex.CredentialsFile,
			MaxTokens:       cfg.LLM.MaxTokens,
			Temperature:     cfg.LLM.Temperature,
			Timeout:         cfg.LLM.Timeout,
		})
	default:
		return nil, fmt.Errorf("unknown llm.provider %q", cfg.LLM.Provider)
	}
}
