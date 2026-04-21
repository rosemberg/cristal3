package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/sitemap"
	"github.com/bergmaia/site-research/internal/canonical"
	"github.com/bergmaia/site-research/internal/config"
)

// URLCandidate is a discovered URL ready to be crawled.
type URLCandidate struct {
	URL     string
	LastMod time.Time
}

// DiscoveryCounts tracks how the raw sitemap was filtered.
type DiscoveryCounts struct {
	Total      int
	InScope    int
	OutOfScope int
	Excluded   int
	Invalid    int
}

// DiscoverInScope fetches the sitemap (from cfg.Sitemap.URL or fromFile if set),
// canonicalizes each entry and filters by cfg.Scope.Prefix. Returns the in-scope
// URLs (already canonicalized) and counts. Used by both `discover` and `crawl`.
//
// If no URLs end up in scope, returns an error per RF-01.
func DiscoverInScope(ctx context.Context, cfg *config.Config, fromFile string) ([]URLCandidate, DiscoveryCounts, error) {
	src := sitemap.New(sitemap.Options{
		URL:        cfg.Sitemap.URL,
		FromFile:   fromFile,
		HTTPClient: &http.Client{Timeout: time.Duration(cfg.Crawler.RequestTimeoutSeconds) * time.Second},
		UserAgent:  cfg.Crawler.UserAgent,
	})

	entries, err := src.Fetch(ctx)
	if err != nil {
		return nil, DiscoveryCounts{}, fmt.Errorf("discover: fetch sitemap: %w", err)
	}

	canon := canonical.New()
	canonPrefix, _, err := canon.Canonicalize(cfg.Scope.Prefix)
	if err != nil {
		return nil, DiscoveryCounts{}, fmt.Errorf("discover: invalid scope.prefix: %w", err)
	}

	var counts DiscoveryCounts
	counts.Total = len(entries)
	urls := make([]URLCandidate, 0, len(entries)/4)

	for _, e := range entries {
		c, excluded, err := canon.Canonicalize(e.Loc)
		if err != nil {
			counts.Invalid++
			continue
		}
		if excluded {
			counts.Excluded++
			continue
		}
		if c != canonPrefix && !strings.HasPrefix(c, canonPrefix+"/") {
			counts.OutOfScope++
			continue
		}
		counts.InScope++
		urls = append(urls, URLCandidate{URL: c, LastMod: e.LastMod})
	}

	if counts.InScope == 0 {
		return nil, counts, fmt.Errorf(
			"discover: no URLs in scope for prefix %q (total=%d, excluded=%d, invalid=%d)",
			canonPrefix, counts.Total, counts.Excluded, counts.Invalid,
		)
	}

	return urls, counts, nil
}
