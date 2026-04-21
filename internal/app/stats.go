package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
)

// StatsOptions configures the stats run.
type StatsOptions struct {
	Output io.Writer
	Format string // "text" (default) | "json"
}

// StatsReport is the aggregate struct returned by GetStats.
// JSON tags are preserved for backward-compatible serialization.
type StatsReport struct {
	GeneratedAt             string         `json:"generated_at"`
	RootURL                 string         `json:"root_url"`
	SchemaVersion           int            `json:"schema_version"`
	TotalPages              int            `json:"total_pages"`
	PagesWithoutMiniSummary int            `json:"pages_without_mini_summary"`
	PagesWithDocs           int            `json:"pages_with_docs"`
	TotalDocuments          int            `json:"total_documents"`
	StalePages              int            `json:"stale_pages"`
	ByDepth                 map[int]int    `json:"by_depth"`
	ByPageType              map[string]int `json:"by_page_type"`
	TopSections             []SectionCount `json:"top_sections"`
}

// SectionCount is a (section, count) pair used in StatsReport.TopSections.
type SectionCount struct {
	Section string `json:"section"`
	Count   int    `json:"count"`
}

// statsReport is an alias kept for compatibility with writeStatsText/writeStatsJSON.
type statsReport = StatsReport

// sectionCount is an alias kept for internal use.
type sectionCount = SectionCount

// GetStats aggregates catalog metrics and returns a StatsReport.
// It is the structured variant used by the MCP layer; the CLI uses Stats.
func GetStats(ctx context.Context, logger *slog.Logger, cfg *config.Config) (StatsReport, error) {
	_ = logger

	// 1. Read catalog.json.
	data, err := os.ReadFile(cfg.Storage.CatalogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return StatsReport{}, fmt.Errorf("stats: catalog.json not found; run build-catalog first")
		}
		return StatsReport{}, fmt.Errorf("stats: read catalog: %w", err)
	}

	var cat domain.Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return StatsReport{}, fmt.Errorf("stats: parse catalog: %w", err)
	}

	// 2. Aggregate from catalog entries.
	pagesWithoutMini := 0
	pagesWithDocs := 0
	sectionMap := make(map[string]int)

	for _, e := range cat.Entries {
		if e.MiniSummary == "" {
			pagesWithoutMini++
		}
		if e.HasDocs {
			pagesWithDocs++
		}
		sec := e.Section
		if sec == "" {
			sec = "(root)"
		}
		sectionMap[sec]++
	}

	// 3. Walk fsstore for total documents and stale pages.
	totalDocuments := 0
	stalePages := 0
	if _, statErr := os.Stat(cfg.Storage.DataDir); statErr == nil {
		store, storeErr := fsstore.New(fsstore.Options{
			RootDir:     cfg.Storage.DataDir,
			ScopePrefix: cfg.Scope.Prefix,
		})
		if storeErr == nil {
			_ = store.Walk(ctx, func(p *domain.Page) error {
				totalDocuments += len(p.Documents)
				if p.Metadata.StaleSince != nil {
					stalePages++
				}
				return nil
			})
		}
	}

	// 4. Sort top sections.
	type kv struct {
		k string
		v int
	}
	var sections []kv
	for k, v := range sectionMap {
		sections = append(sections, kv{k, v})
	}
	sort.Slice(sections, func(i, j int) bool {
		if sections[i].v != sections[j].v {
			return sections[i].v > sections[j].v
		}
		return sections[i].k < sections[j].k
	})
	if len(sections) > 15 {
		sections = sections[:15]
	}

	topSections := make([]SectionCount, len(sections))
	for i, s := range sections {
		topSections[i] = SectionCount{Section: s.k, Count: s.v}
	}

	// Build by_page_type as map[string]int.
	byPageType := make(map[string]int, len(cat.Stats.ByPageType))
	for pt, c := range cat.Stats.ByPageType {
		byPageType[string(pt)] = c
	}

	report := StatsReport{
		GeneratedAt:             cat.GeneratedAt.Format("2006-01-02 15:04:05 UTC"),
		RootURL:                 cat.RootURL,
		SchemaVersion:           cat.SchemaVersion,
		TotalPages:              cat.Stats.TotalPages,
		PagesWithoutMiniSummary: pagesWithoutMini,
		PagesWithDocs:           pagesWithDocs,
		TotalDocuments:          totalDocuments,
		StalePages:              stalePages,
		ByDepth:                 cat.Stats.ByDepth,
		ByPageType:              byPageType,
		TopSections:             topSections,
	}

	return report, nil
}

// Stats reads cfg.Storage.CatalogPath and prints aggregate metrics to opts.Output.
func Stats(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts StatsOptions) error {
	// 1. Default output/format.
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.Format == "" {
		opts.Format = "text"
	}

	report, err := GetStats(ctx, logger, cfg)
	if err != nil {
		return err
	}

	// 2. Output.
	switch opts.Format {
	case "json":
		return writeStatsJSON(opts.Output, report)
	default:
		return writeStatsText(opts.Output, report)
	}
}

func writeStatsText(w io.Writer, r statsReport) error {
	fmt.Fprintln(w, "Site Research — Catalog Stats")
	fmt.Fprintln(w, "=============================")
	fmt.Fprintf(w, "Generated at: %s\n", r.GeneratedAt)
	fmt.Fprintf(w, "Root URL:     %s\n", r.RootURL)
	fmt.Fprintf(w, "Schema:       v%d\n", r.SchemaVersion)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total pages:             %d\n", r.TotalPages)
	fmt.Fprintf(w, "Pages without summary:   %d\n", r.PagesWithoutMiniSummary)
	fmt.Fprintf(w, "Pages with documents:    %d\n", r.PagesWithDocs)
	fmt.Fprintf(w, "Total documents:         %d\n", r.TotalDocuments)
	fmt.Fprintf(w, "Stale pages:             %d\n", r.StalePages)
	fmt.Fprintln(w)

	// By depth: sort keys.
	fmt.Fprintln(w, "By depth:")
	depthKeys := make([]int, 0, len(r.ByDepth))
	for k := range r.ByDepth {
		depthKeys = append(depthKeys, k)
	}
	sort.Ints(depthKeys)
	for _, d := range depthKeys {
		fmt.Fprintf(w, "  depth %d: %d\n", d, r.ByDepth[d])
	}
	fmt.Fprintln(w)

	// By page type: canonical order.
	fmt.Fprintln(w, "By page type:")
	pageTypeOrder := []string{"landing", "article", "listing", "redirect", "empty"}
	for _, pt := range pageTypeOrder {
		if n, ok := r.ByPageType[pt]; ok {
			fmt.Fprintf(w, "  %-9s %d\n", pt+":", n)
		}
	}
	// Any extra types not in the canonical order.
	known := map[string]bool{"landing": true, "article": true, "listing": true, "redirect": true, "empty": true}
	for pt, n := range r.ByPageType {
		if !known[pt] {
			fmt.Fprintf(w, "  %-9s %d\n", pt+":", n)
		}
	}
	fmt.Fprintln(w)

	// Top sections.
	if len(r.TopSections) > 0 {
		fmt.Fprintln(w, "Top sections (by page count):")
		for _, s := range r.TopSections {
			fmt.Fprintf(w, "  %4d  %s\n", s.Count, s.Section)
		}
	}

	return nil
}

func writeStatsJSON(w io.Writer, r statsReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
