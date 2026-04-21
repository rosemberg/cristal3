package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bergmaia/cristal-chat/internal/mcp"
	"github.com/bergmaia/cristal-chat/internal/ui"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(run())
}

func run() int {
	// Flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	versionFlag := flag.Bool("version", false, "print version")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cristal v%s\n", version)
		return 0
	}

	// Logger
	logLevel := slog.LevelInfo
	if *debugFlag {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Por enquanto, config hardcoded (M4.1 vai ler YAML)
	_ = configPath

	// MCP Client config
	mcpCfg := mcp.Config{
		PythonPath: "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/.venv/bin/python",
		ScriptPath: "src.server",
		WorkingDir: "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp", // Caminho absoluto
		Logger:     logger,
	}

	// Conectar ao MCP
	logger.Info("conectando ao data-orchestrator-mcp")
	client, err := mcp.NewClient(mcpCfg)
	if err != nil {
		logger.Error("falha ao criar cliente MCP", "error", err)
		return 1
	}
	defer client.Close()

	if err := client.Initialize(); err != nil {
		logger.Error("falha ao inicializar MCP", "error", err)
		return 1
	}

	logger.Info("MCP inicializado com sucesso")

	// REPL
	repl := ui.NewREPL(client, logger)

	// Context com signal handling
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := repl.Run(ctx); err != nil {
		if err != context.Canceled {
			logger.Error("erro no REPL", "error", err)
			return 1
		}
	}

	return 0
}
