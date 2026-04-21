package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/logging"
)

func TestSearch_DiariasReturnsHits(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg) // pages contain "diárias" in FullText

	ctx := context.Background()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}

	var buf bytes.Buffer
	err := app.Search(ctx, logger, cfg, app.SearchOptions{
		Query:  "diarias",
		Limit:  5,
		Format: "text",
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "#1") {
		t.Errorf("output should contain \"#1\", got:\n%s", out)
	}
	// At least one of our seeded URLs should appear.
	if !strings.Contains(out, cfg.Scope.Prefix) {
		t.Errorf("output should contain seeded URL prefix %q, got:\n%s", cfg.Scope.Prefix, out)
	}
}

func TestSearch_NoStore_Errors(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	// No BuildCatalog → no SQLite file.

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.Search(context.Background(), logger, cfg, app.SearchOptions{
		Query:  "anything",
		Limit:  5,
		Format: "text",
		Output: io.Discard,
	})
	if err == nil {
		t.Fatal("expected error when SQLite does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "build-catalog first") {
		t.Errorf("error %q should contain \"build-catalog first\"", err.Error())
	}
}

func TestSearch_EmptyQuery_Errors(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.Search(context.Background(), logger, cfg, app.SearchOptions{
		Query:  "",
		Limit:  5,
		Format: "text",
		Output: io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Errorf("error %q should contain \"query is required\"", err.Error())
	}
}

func TestSearch_JSONFormat(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	ctx := context.Background()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}

	var buf bytes.Buffer
	err := app.Search(ctx, logger, cfg, app.SearchOptions{
		Query:  "diarias",
		Limit:  5,
		Format: "json",
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Search (json) returned error: %v", err)
	}

	// Must be valid JSON with a "hits" field.
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if _, ok := result["hits"]; !ok {
		t.Errorf("JSON output missing \"hits\" field: %s", buf.String())
	}
	if _, ok := result["query"]; !ok {
		t.Errorf("JSON output missing \"query\" field: %s", buf.String())
	}
}

func TestSearch_InvalidFormat_Errors(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.Search(context.Background(), logger, cfg, app.SearchOptions{
		Query:  "something",
		Limit:  5,
		Format: "csv", // invalid
		Output: io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error %q should contain \"invalid format\"", err.Error())
	}
}

func TestSearch_DefaultLimit(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	ctx := context.Background()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}

	// Limit=0 should default to 10 (no error).
	var buf bytes.Buffer
	err := app.Search(ctx, logger, cfg, app.SearchOptions{
		Query:  "diarias",
		Limit:  0,
		Format: "text",
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Search with Limit=0: %v", err)
	}
}
