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

// TestStats_Text verifies text output contains expected headers and depth section.
func TestStats_Text(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	ctx := context.Background()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	// Build catalog.json first.
	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}

	var buf bytes.Buffer
	err := app.Stats(ctx, logger, cfg, app.StatsOptions{
		Output: &buf,
		Format: "text",
	})
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total pages:") {
		t.Errorf("output should contain 'Total pages:', got:\n%s", out)
	}
	if !strings.Contains(out, "By depth:") {
		t.Errorf("output should contain 'By depth:', got:\n%s", out)
	}
	if !strings.Contains(out, "By page type:") {
		t.Errorf("output should contain 'By page type:', got:\n%s", out)
	}
	if !strings.Contains(out, "Site Research") {
		t.Errorf("output should contain 'Site Research', got:\n%s", out)
	}
	// seeded 3 pages.
	if !strings.Contains(out, "3") {
		t.Errorf("output should mention total of 3 pages, got:\n%s", out)
	}
}

// TestStats_Json verifies JSON output is parseable with a total_pages key.
func TestStats_Json(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	ctx := context.Background()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}

	var buf bytes.Buffer
	err := app.Stats(ctx, logger, cfg, app.StatsOptions{
		Output: &buf,
		Format: "json",
	})
	if err != nil {
		t.Fatalf("Stats (json) returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if _, ok := result["total_pages"]; !ok {
		t.Errorf("JSON output missing 'total_pages' key: %s", buf.String())
	}
	// Verify total_pages == 3.
	if v, ok := result["total_pages"].(float64); ok {
		if int(v) != 3 {
			t.Errorf("total_pages = %v, want 3", v)
		}
	}
}

// TestStats_NoCatalog_Errors verifies error when catalog.json is missing.
func TestStats_NoCatalog_Errors(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	// Do NOT run BuildCatalog → catalog.json is absent.

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.Stats(context.Background(), logger, cfg, app.StatsOptions{
		Output: io.Discard,
		Format: "text",
	})
	if err == nil {
		t.Fatal("expected error when catalog.json does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "build-catalog") {
		t.Errorf("error %q should mention 'build-catalog'", err.Error())
	}
}
