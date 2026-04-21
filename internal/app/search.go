package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"unicode/utf8"

	"github.com/bergmaia/site-research/internal/adapters/sqlitefts"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

// SearchOptions configures a search run.
type SearchOptions struct {
	Query  string
	Limit  int      // 0 → default 10
	Format string   // "text" (default) | "json"
	Output io.Writer // defaults to os.Stdout when nil
}

// Search opens the FTS store at cfg.Storage.SQLitePath, executes the query,
// and writes hits to opts.Output.
// Returns an error if the store does not exist (typically: run build-catalog first).
func Search(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts SearchOptions) error {
	// 1. Default output.
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// 2. Default format.
	if opts.Format == "" {
		opts.Format = "text"
	}

	// 3. Validate format.
	if opts.Format != "text" && opts.Format != "json" {
		return fmt.Errorf("search: invalid format %q; must be \"text\" or \"json\"", opts.Format)
	}

	// 4. Require query.
	if opts.Query == "" {
		return fmt.Errorf("search: query is required")
	}

	// 5. Default limit.
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// 6. Check SQLite exists.
	if _, err := os.Stat(cfg.Storage.SQLitePath); os.IsNotExist(err) {
		return fmt.Errorf("search: SQLite not found at %q; run build-catalog first", cfg.Storage.SQLitePath)
	}

	// 7. Open FTS store.
	ftsStore, err := sqlitefts.Open(sqlitefts.Options{Path: cfg.Storage.SQLitePath})
	if err != nil {
		return fmt.Errorf("search: open sqlite: %w", err)
	}
	defer ftsStore.Close()

	// 8. Execute search.
	hits, err := ftsStore.Search(ctx, opts.Query, limit)
	if err != nil {
		return fmt.Errorf("search: fts query: %w", err)
	}

	// 9/10. Format and write output.
	switch opts.Format {
	case "text":
		if err := writeTextOutput(opts.Output, hits); err != nil {
			return fmt.Errorf("search: write output: %w", err)
		}
	case "json":
		if err := writeJSONOutput(opts.Output, opts.Query, hits); err != nil {
			return fmt.Errorf("search: write json output: %w", err)
		}
	}

	// 11. Log completion.
	logger.Info("search complete", "query", opts.Query, "hits", len(hits), "limit", limit)

	return nil
}

func writeTextOutput(w io.Writer, hits []ports.SearchHit) error {
	for i, h := range hits {
		summary := truncateSummary(h.MiniSummary, 200)
		_, err := fmt.Fprintf(w, "#%d  [score=%.3f]  %s\n    URL: %s\n    Path: %s\n    %s\n\n",
			i+1, h.Score, h.Title, h.URL, h.Path, summary)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeJSONOutput(w io.Writer, query string, hits []ports.SearchHit) error {
	type jsonOutput struct {
		Query string            `json:"query"`
		Count int               `json:"count"`
		Hits  []ports.SearchHit `json:"hits"`
	}
	out := jsonOutput{
		Query: query,
		Count: len(hits),
		Hits:  hits,
	}
	if hits == nil {
		out.Hits = []ports.SearchHit{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// truncateSummary truncates s to at most maxRunes runes, appending "…" if truncated.
func truncateSummary(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "…"
}

// SearchHits opens the FTS store and executes the query, returning the raw hits.
// This is the structured variant used by the MCP layer; the CLI uses Search.
func SearchHits(ctx context.Context, logger *slog.Logger, cfg *config.Config, query string, limit int) ([]ports.SearchHit, error) {
	if query == "" {
		return nil, fmt.Errorf("search: query is required")
	}
	if limit <= 0 {
		limit = 10
	}

	if _, err := os.Stat(cfg.Storage.SQLitePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("search: SQLite not found at %q; run build-catalog first", cfg.Storage.SQLitePath)
	}

	ftsStore, err := sqlitefts.Open(sqlitefts.Options{Path: cfg.Storage.SQLitePath})
	if err != nil {
		return nil, fmt.Errorf("search: open sqlite: %w", err)
	}
	defer ftsStore.Close()

	hits, err := ftsStore.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search: fts query: %w", err)
	}

	logger.Debug("search hits", "query", query, "hits", len(hits), "limit", limit)
	return hits, nil
}
