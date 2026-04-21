package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientInitialize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Assumindo que data-orchestrator-mcp está no parent dir
	orchDir, err := filepath.Abs(filepath.Join("..", "..", "..", "data-orchestrator-mcp"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Skip("data-orchestrator-mcp not found")
	}

	// Usar Python do venv se disponível
	pythonPath := filepath.Join(orchDir, ".venv", "bin", "python")
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		pythonPath = "python3"
	}

	cfg := Config{
		PythonPath: pythonPath,
		ScriptPath: "src.server",
		WorkingDir: orchDir,
		Timeout:    30 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Dar tempo para o servidor iniciar
	time.Sleep(2 * time.Second)

	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
}

func TestListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupTestClient(t)
	defer client.Close()

	tools, err := client.ListTools()
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	expectedTools := []string{"research", "get_cached", "get_document", "metrics"}
	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !toolMap[name] {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestCallToolMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupTestClient(t)
	defer client.Close()

	result, err := client.CallTool("metrics", nil)
	if err != nil {
		t.Fatalf("CallTool metrics: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	if result.Content[0].Type != "text" {
		t.Errorf("expected text content, got %s", result.Content[0].Type)
	}
}

func setupTestClient(t *testing.T) *Client {
	orchDir, err := filepath.Abs(filepath.Join("..", "..", "..", "data-orchestrator-mcp"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	// Usar Python do venv se disponível
	pythonPath := filepath.Join(orchDir, ".venv", "bin", "python")
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		pythonPath = "python3"
	}

	cfg := Config{
		PythonPath: pythonPath,
		ScriptPath: "src.server",
		WorkingDir: orchDir,
		Timeout:    30 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Dar tempo para o servidor iniciar
	time.Sleep(2 * time.Second)

	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	return client
}
