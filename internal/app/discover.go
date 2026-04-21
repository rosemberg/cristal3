package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/bergmaia/site-research/internal/config"
)

// DiscoverOptions configures the discover command behavior beyond the shared Config.
type DiscoverOptions struct {
	// FromFile, when non-empty, reads the sitemap from a local file path
	// instead of the URL configured in cfg.Sitemap.URL. Useful for offline validation.
	FromFile string
	// Format is "text" (default) — URLs one per line to Output — or "json"
	// (structured blob with summary + urls to Output).
	Format string
	// Output receives the primary payload (URLs or JSON). Defaults to os.Stdout when nil.
	Output io.Writer
}

// DiscoverResult summarizes a discover run (returned for JSON output and logging).
type DiscoverResult struct {
	Total      int      `json:"total"`
	InScope    int      `json:"in_scope"`
	OutOfScope int      `json:"out_of_scope"`
	Excluded   int      `json:"excluded"`
	Invalid    int      `json:"invalid"`
	URLs       []string `json:"urls"`
}

// Discover downloads the sitemap (or reads it from disk when opts.FromFile is set),
// applies RF-03 canonicalization to every URL, filters entries by cfg.Scope.Prefix,
// and writes the resulting URLs to opts.Output (one per line for "text", or a JSON
// blob for "json"). Summary and warnings go via logger. Per RF-01, returns an
// explicit error if no URLs fall within scope.
func Discover(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts DiscoverOptions) error {
	// 1. Default Output.
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	if opts.Format == "" {
		opts.Format = "text"
	}
	if opts.Format != "text" && opts.Format != "json" {
		return fmt.Errorf("discover: unsupported format %q (use text|json)", opts.Format)
	}

	// 2. Discover in-scope URLs using shared helper.
	candidates, counts, err := DiscoverInScope(ctx, cfg, opts.FromFile)
	if err != nil {
		return err
	}

	logger.Info("sitemap fetched",
		"total_entries", counts.Total,
		"source", sourceLabel(opts.FromFile, cfg.Sitemap.URL),
	)

	// 3. Build result from candidates.
	res := DiscoverResult{
		Total:      counts.Total,
		InScope:    counts.InScope,
		OutOfScope: counts.OutOfScope,
		Excluded:   counts.Excluded,
		Invalid:    counts.Invalid,
		URLs:       make([]string, 0, len(candidates)),
	}
	for _, c := range candidates {
		res.URLs = append(res.URLs, c.URL)
	}

	// 4. Emit output.
	switch opts.Format {
	case "text":
		for _, u := range res.URLs {
			fmt.Fprintln(out, u)
		}
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			return fmt.Errorf("discover: write json: %w", err)
		}
	}

	// 5. Summary via logger (structured).
	logger.Info("discover complete",
		"total", res.Total,
		"in_scope", res.InScope,
		"out_of_scope", res.OutOfScope,
		"excluded", res.Excluded,
		"invalid", res.Invalid,
		"format", opts.Format,
	)
	return nil
}

func sourceLabel(fromFile, url string) string {
	if fromFile != "" {
		return "file:" + fromFile
	}
	return url
}
