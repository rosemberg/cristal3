package tools

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
)

func TestDefaultRegistryToolCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := DefaultRegistry(nil, logger)

	ts := r.Tools()
	if len(ts) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(ts))
	}

	names := make(map[string]bool)
	for _, tool := range ts {
		names[tool.Name] = true
	}

	for _, want := range []string{"search", "inspect_page", "catalog_stats"} {
		if !names[want] {
			t.Errorf("tool %q not found in registry", want)
		}
	}
}

func TestDefaultRegistryInputSchemaValid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := DefaultRegistry(nil, logger)

	for _, tool := range r.Tools() {
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %q has empty InputSchema", tool.Name)
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(tool.InputSchema, &obj); err != nil {
			t.Errorf("tool %q InputSchema is not valid JSON: %v", tool.Name, err)
		}
	}
}

func TestDefaultRegistryHandlerExists(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := DefaultRegistry(nil, logger)

	for _, want := range []string{"search", "inspect_page", "catalog_stats"} {
		if _, ok := r.Handler(want); !ok {
			t.Errorf("no handler registered for tool %q", want)
		}
	}
}

func TestDefaultRegistryHandlerNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := DefaultRegistry(nil, logger)

	if _, ok := r.Handler("nonexistent"); ok {
		t.Error("expected no handler for nonexistent tool")
	}
}
