package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/logging"
)

// buildCatalogCfg creates a Config pointing at tmp/data for store, catalog.json and sqlite.
func buildCatalogCfg(t *testing.T, tmp string) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Scope.SeedURL = "https://example.com/scope"
	cfg.Scope.Prefix = "https://example.com/scope"
	cfg.Storage.DataDir = filepath.Join(tmp, "data")
	cfg.Storage.CatalogPath = filepath.Join(tmp, "data", "catalog.json")
	cfg.Storage.SQLitePath = filepath.Join(tmp, "data", "catalog.sqlite")
	return cfg
}

// seedPages writes 3 test pages into the fsstore.
func seedPages(t *testing.T, cfg *config.Config) {
	t.Helper()
	ctx := context.Background()
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	urls := []string{"", "/a", "/a/b"}
	for i, u := range urls {
		p := &domain.Page{
			URL:                   cfg.Scope.Prefix + u,
			CanonicalURL:          cfg.Scope.Prefix + u,
			Title:                 fmt.Sprintf("Title %d", i),
			Section:               "Sec",
			PageType:              domain.PageTypeArticle,
			HasSubstantiveContent: true,
			Content: domain.Content{
				FullText:     fmt.Sprintf("Conteúdo %d sobre diárias de viagem e alimentação", i),
				FullTextHash: fmt.Sprintf("sha256:h%d", i),
			},
			Metadata: domain.Metadata{
				Depth:       i,
				ExtractedAt: time.Now(),
			},
		}
		if err := store.Put(ctx, p); err != nil {
			t.Fatalf("store.Put page %d: %v", i, err)
		}
	}
}

func TestBuildCatalog_EndToEnd(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.BuildCatalog(context.Background(), logger, cfg, app.BuildCatalogOptions{})
	if err != nil {
		t.Fatalf("BuildCatalog returned error: %v", err)
	}

	// catalog.json must exist.
	if _, err := os.Stat(cfg.Storage.CatalogPath); err != nil {
		t.Fatalf("catalog.json not found: %v", err)
	}

	// SQLite must exist.
	if _, err := os.Stat(cfg.Storage.SQLitePath); err != nil {
		t.Fatalf("catalog.sqlite not found: %v", err)
	}

	// Unmarshal catalog and check TotalPages.
	raw, err := os.ReadFile(cfg.Storage.CatalogPath)
	if err != nil {
		t.Fatalf("read catalog.json: %v", err)
	}
	var cat domain.Catalog
	if err := json.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("unmarshal catalog.json: %v", err)
	}
	if cat.Stats.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", cat.Stats.TotalPages)
	}
	if len(cat.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(cat.Entries))
	}
}

func TestBuildCatalog_EmptyStore_Errors(t *testing.T) {
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	// Do NOT seed any pages — store is empty.

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	err := app.BuildCatalog(context.Background(), logger, cfg, app.BuildCatalogOptions{})
	if err == nil {
		t.Fatal("expected error for empty store, got nil")
	}
	if !strings.Contains(err.Error(), "store is empty") {
		t.Errorf("error %q should contain \"store is empty\"", err.Error())
	}
}

func TestBuildCatalog_Idempotent(t *testing.T) {
	// Running BuildCatalog twice must succeed both times.
	tmp := t.TempDir()
	cfg := buildCatalogCfg(t, tmp)
	seedPages(t, cfg)

	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})
	ctx := context.Background()

	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("first BuildCatalog: %v", err)
	}
	if err := app.BuildCatalog(ctx, logger, cfg, app.BuildCatalogOptions{}); err != nil {
		t.Fatalf("second BuildCatalog: %v", err)
	}

	// File should still have 3 entries.
	raw, _ := os.ReadFile(cfg.Storage.CatalogPath)
	var cat domain.Catalog
	_ = json.Unmarshal(raw, &cat)
	if cat.Stats.TotalPages != 3 {
		t.Errorf("after second run, TotalPages = %d, want 3", cat.Stats.TotalPages)
	}
}
