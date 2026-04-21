package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bergmaia/site-research/internal/adapters/catalog"
	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/adapters/sqlitefts"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
)

// BuildCatalogOptions configures the build-catalog run (reserved for future flags).
type BuildCatalogOptions struct{}

// BuildCatalog consolidates the page tree at cfg.Storage.DataDir into
// cfg.Storage.CatalogPath (JSON) AND a fresh SQLite/FTS database at
// cfg.Storage.SQLitePath. The SQLite DB is rebuilt from scratch (idempotent).
//
// Returns an error if the store is empty (no pages to consolidate).
func BuildCatalog(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts BuildCatalogOptions) error {
	// 1. Open fsstore.
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		return fmt.Errorf("build-catalog: open store: %w", err)
	}

	// 2. Walk the store once; collect pages.
	var pages []*domain.Page
	if err := store.Walk(ctx, func(p *domain.Page) error {
		pages = append(pages, p)
		return nil
	}); err != nil {
		return fmt.Errorf("build-catalog: walking store: %w", err)
	}

	// 3. If empty, return error.
	if len(pages) == 0 {
		return fmt.Errorf("build-catalog: store is empty; run crawl first (datadir=%q)", cfg.Storage.DataDir)
	}

	// 4. Build catalog via catalog.New + builder.Build.
	builder, err := catalog.New(catalog.Options{
		Store:   store,
		RootURL: cfg.Scope.Prefix,
	})
	if err != nil {
		return fmt.Errorf("build-catalog: create catalog builder: %w", err)
	}

	cat, err := builder.Build(ctx)
	if err != nil {
		return fmt.Errorf("build-catalog: build catalog: %w", err)
	}

	// 5. Log summary.
	logger.Info("catalog built",
		"total_pages", cat.Stats.TotalPages,
		"by_depth", cat.Stats.ByDepth,
		"by_page_type", cat.Stats.ByPageType,
	)

	// 6. Ensure parent dirs exist for CatalogPath and SQLitePath.
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.CatalogPath), 0o755); err != nil {
		return fmt.Errorf("build-catalog: create catalog dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.SQLitePath), 0o755); err != nil {
		return fmt.Errorf("build-catalog: create sqlite dir: %w", err)
	}

	// 7. WriteFile.
	if err := builder.WriteFile(ctx, cat, cfg.Storage.CatalogPath); err != nil {
		return fmt.Errorf("build-catalog: write catalog file: %w", err)
	}

	// 8. Open SQLite.
	ftsStore, err := sqlitefts.Open(sqlitefts.Options{Path: cfg.Storage.SQLitePath})
	if err != nil {
		return fmt.Errorf("build-catalog: open sqlite: %w", err)
	}
	defer ftsStore.Close()

	// 9. Rebuild FTS.
	if err := ftsStore.Rebuild(ctx, cat, pages); err != nil {
		return fmt.Errorf("build-catalog: rebuild fts: %w", err)
	}

	// 10. Log FTS rebuild.
	logger.Info("fts rebuilt", "path", cfg.Storage.SQLitePath, "rows", len(pages))

	return nil
}
