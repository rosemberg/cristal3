// Package catalog builds the consolidated catalog.json from a populated PageStore.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/domain"
)

// Options configures the Builder.
type Options struct {
	Store   *fsstore.Store
	RootURL string         // the scope root, e.g., "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
	Now     func() time.Time // optional; defaults to time.Now
}

// Builder consolidates an fsstore-backed tree into a domain.Catalog.
type Builder struct {
	opts Options
}

// New returns a Builder. Returns error if Store or RootURL is empty.
func New(opts Options) (*Builder, error) {
	if opts.Store == nil {
		return nil, fmt.Errorf("catalog: Store must not be nil")
	}
	if opts.RootURL == "" {
		return nil, fmt.Errorf("catalog: RootURL must not be empty")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Builder{opts: opts}, nil
}

// Build walks the store, assembles CatalogEntries, computes stats, and
// returns a populated *domain.Catalog ready to be written.
func (b *Builder) Build(ctx context.Context) (*domain.Catalog, error) {
	// First pass: collect all pages.
	var pages []*domain.Page
	if err := b.opts.Store.Walk(ctx, func(p *domain.Page) error {
		pages = append(pages, p)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("catalog: walking store: %w", err)
	}

	// Build a set of parent URLs to count children per URL.
	childCounts := make(map[string]int, len(pages))
	for _, p := range pages {
		parent := p.Metadata.ParentURL
		if parent != "" {
			childCounts[parent]++
		}
	}

	rootURL := strings.TrimRight(b.opts.RootURL, "/")

	// Second pass: build CatalogEntry for each page.
	entries := make([]domain.CatalogEntry, 0, len(pages))
	byDepth := make(map[int]int)
	byPageType := make(map[domain.PageType]int)

	for _, p := range pages {
		url := p.URL

		// Compute relative path.
		rel := ""
		trimmed := strings.TrimRight(url, "/")
		if trimmed != rootURL {
			if strings.HasPrefix(trimmed, rootURL+"/") {
				rel = strings.TrimLeft(trimmed[len(rootURL):], "/")
			} else {
				rel = trimmed
			}
		}

		// MiniSummary text is authoritative: Skipped may mark metadata reasons
		// (e.g., "up_to_date" from a re-run) while Text remains valid.
		miniSummary := p.MiniSummary.Text

		entry := domain.CatalogEntry{
			Path:                  rel,
			URL:                   url,
			Title:                 p.Title,
			Depth:                 p.Metadata.Depth,
			Parent:                p.Metadata.ParentURL,
			Section:               p.Section,
			PageType:              p.PageType,
			HasSubstantiveContent: p.HasSubstantiveContent,
			MiniSummary:           miniSummary,
			ChildCount:            childCounts[url],
			HasDocs:               len(p.Documents) > 0,
			ContentDate:           p.Dates.ContentDate,
		}

		entries = append(entries, entry)

		byDepth[p.Metadata.Depth]++
		byPageType[p.PageType]++
	}

	// Sort entries by Path lexicographically.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	catalog := &domain.Catalog{
		GeneratedAt:   b.opts.Now(),
		RootURL:       b.opts.RootURL,
		SchemaVersion: 2,
		Stats: domain.CatalogStats{
			TotalPages: len(entries),
			ByDepth:    byDepth,
			ByPageType: byPageType,
		},
		Entries: entries,
	}

	return catalog, nil
}

// WriteFile serializes the catalog as pretty-printed JSON (2-space indent)
// and writes it atomically (write-then-rename) to path. Creates parent dirs
// if missing. Does NOT call b.Build; caller passes the catalog.
func (b *Builder) WriteFile(ctx context.Context, catalog *domain.Catalog, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("catalog: marshaling catalog: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("catalog: creating parent dirs: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("catalog: writing temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("catalog: renaming temp file: %w", err)
	}

	return nil
}
