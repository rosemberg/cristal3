package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/logging"
)

// seedInspectPage writes a single page at <scopePrefix>/a/b into the store.
func seedInspectPage(t *testing.T, dataDir, scopePrefix string) {
	t.Helper()
	ctx := context.Background()
	store, err := fsstore.New(fsstore.Options{
		RootDir:     dataDir,
		ScopePrefix: scopePrefix,
	})
	if err != nil {
		t.Fatalf("seedInspectPage: open store: %v", err)
	}
	p := &domain.Page{
		URL:          scopePrefix + "/a/b",
		CanonicalURL: scopePrefix + "/a/b",
		Title:        "Título da Página Teste",
		Section:      "Seção A",
		PageType:     domain.PageTypeArticle,
		Content: domain.Content{
			Summary:       "Este é um resumo de teste para a página.",
			ContentLength: 42,
		},
		Metadata: domain.Metadata{
			Depth:          2,
			ExtractedAt:    time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
			CrawlerVersion: "0.1.0",
			DiscoveredVia:  domain.DiscoveredViaSitemap,
		},
		Tags: []string{"transparência", "teste"},
		Links: domain.Links{
			Children: []domain.ChildLink{
				{Title: "Filho 1", URL: scopePrefix + "/a/b/c"},
			},
		},
	}
	if err := store.Put(ctx, p); err != nil {
		t.Fatalf("seedInspectPage: store.Put: %v", err)
	}
}

// TestInspect_ByPath verifies that a relative path target resolves correctly.
func TestInspect_ByPath(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedInspectPage(t, cfg.Storage.DataDir, cfg.Scope.Prefix)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	var buf bytes.Buffer
	err := app.Inspect(context.Background(), logger, cfg, app.InspectOptions{
		Target: "a/b",
		Full:   false,
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Title:") {
		t.Errorf("compact output should contain 'Title:', got:\n%s", out)
	}
	if !strings.Contains(out, "Título da Página Teste") {
		t.Errorf("compact output should contain page title, got:\n%s", out)
	}
}

// TestInspect_ByURL verifies that a full URL target works.
func TestInspect_ByURL(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedInspectPage(t, cfg.Storage.DataDir, cfg.Scope.Prefix)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	var buf bytes.Buffer
	err := app.Inspect(context.Background(), logger, cfg, app.InspectOptions{
		Target: cfg.Scope.Prefix + "/a/b",
		Full:   false,
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Inspect (by URL) returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Title:") {
		t.Errorf("compact output should contain 'Title:', got:\n%s", out)
	}
	if !strings.Contains(out, "Título da Página Teste") {
		t.Errorf("compact output should contain page title, got:\n%s", out)
	}
}

// TestInspect_NotFound verifies that a missing page returns an error containing "not found".
func TestInspect_NotFound(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	// Do NOT seed any pages.

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.Inspect(context.Background(), logger, cfg, app.InspectOptions{
		Target: "does/not/exist",
		Full:   false,
		Output: io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for missing page, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

// TestInspect_Full verifies that --full emits parseable JSON with the correct URL.
func TestInspect_Full(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedInspectPage(t, cfg.Storage.DataDir, cfg.Scope.Prefix)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	var buf bytes.Buffer
	err := app.Inspect(context.Background(), logger, cfg, app.InspectOptions{
		Target: "a/b",
		Full:   true,
		Output: &buf,
	})
	if err != nil {
		t.Fatalf("Inspect (full) returned error: %v", err)
	}

	var page domain.Page
	if err := json.Unmarshal(buf.Bytes(), &page); err != nil {
		t.Fatalf("full output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if page.URL != cfg.Scope.Prefix+"/a/b" {
		t.Errorf("page.URL = %q, want %q", page.URL, cfg.Scope.Prefix+"/a/b")
	}
	if page.Title != "Título da Página Teste" {
		t.Errorf("page.Title = %q, want 'Título da Página Teste'", page.Title)
	}
}
