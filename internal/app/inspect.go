package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
)

// InspectOptions configures the inspect run.
type InspectOptions struct {
	// Target is either a full URL or a path relative to cfg.Scope.Prefix.
	// Examples: "contabilidade/balancetes" or "https://www.tre-pi.jus.br/.../contabilidade/balancetes".
	Target string
	// Full, when true, emits the entire Page as pretty-printed JSON.
	// When false, emits a compact human-readable summary.
	Full bool
	// Output where to write; defaults to os.Stdout.
	Output io.Writer
}

// Inspect loads a single page from fsstore and prints it to opts.Output.
// Resolves relative targets against cfg.Scope.Prefix.
// Returns an error if the page is not found.
func Inspect(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts InspectOptions) error {
	_ = logger

	// 1. Resolve output.
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	// 2. Resolve URL.
	target := opts.Target
	var url string
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		url = target
	} else {
		prefix := strings.TrimSuffix(cfg.Scope.Prefix, "/")
		if target == "" {
			url = prefix
		} else {
			url = prefix + "/" + strings.TrimPrefix(target, "/")
		}
	}

	// 3. Open fsstore.
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		return fmt.Errorf("inspect: open store: %w", err)
	}

	// 4. Get page.
	page, err := store.Get(ctx, url)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect: page not found at %q", url)
		}
		return fmt.Errorf("inspect: get page: %w", err)
	}

	// 5. Output.
	if opts.Full {
		return writeInspectFull(out, page)
	}
	return writeInspectCompact(out, page)
}

// ErrPageNotFound is returned by InspectPage when the page does not exist.
var ErrPageNotFound = errors.New("page not found")

// InspectPage resolves target to a canonical URL and returns the *domain.Page.
// Returns ErrPageNotFound (wrapped) when the page is absent from the store.
func InspectPage(ctx context.Context, logger *slog.Logger, cfg *config.Config, target string) (*domain.Page, error) {
	_ = logger

	var url string
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		url = target
	} else {
		prefix := strings.TrimSuffix(cfg.Scope.Prefix, "/")
		if target == "" {
			url = prefix
		} else {
			url = prefix + "/" + strings.TrimPrefix(target, "/")
		}
	}

	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("inspect: open store: %w", err)
	}

	page, err := store.Get(ctx, url)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrPageNotFound, url)
		}
		return nil, fmt.Errorf("inspect: get page: %w", err)
	}

	return page, nil
}

func writeInspectFull(w io.Writer, page *domain.Page) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(page)
}

func writeInspectCompact(w io.Writer, page *domain.Page) error {
	parentURL := page.Metadata.ParentURL
	if parentURL == "" {
		parentURL = "—"
	}

	// Breadcrumb: path_titles joined by " › "
	breadcrumb := "—"
	if len(page.PathTitles) > 0 {
		breadcrumb = strings.Join(page.PathTitles, " › ")
	} else if len(page.Breadcrumb) > 0 {
		titles := make([]string, 0, len(page.Breadcrumb))
		for _, b := range page.Breadcrumb {
			if b.Title != "" {
				titles = append(titles, b.Title)
			}
		}
		if len(titles) > 0 {
			breadcrumb = strings.Join(titles, " › ")
		}
	}

	// Summary: trim to 300 chars.
	summary := page.Content.Summary
	if utf8.RuneCountInString(summary) > 300 {
		runes := []rune(summary)
		summary = string(runes[:300]) + "…"
	}
	if summary == "" {
		summary = "(not available)"
	}

	// Mini-summary.
	miniSummaryText := page.MiniSummary.Text
	if miniSummaryText == "" {
		if page.MiniSummary.Skipped != nil {
			miniSummaryText = fmt.Sprintf("(not generated — skipped: %s)", *page.MiniSummary.Skipped)
		} else {
			miniSummaryText = "(not generated)"
		}
	}

	// Canonical-of.
	canonicalOf := "—"
	if page.Metadata.CanonicalOf != nil {
		canonicalOf = *page.Metadata.CanonicalOf
	}

	// Stale-since.
	staleSince := "—"
	if page.Metadata.StaleSince != nil {
		staleSince = page.Metadata.StaleSince.Format("2006-01-02 15:04:05 UTC")
	}

	// Tags.
	tagsStr := "—"
	if len(page.Tags) > 0 {
		tagsStr = strings.Join(page.Tags, ", ")
	}

	// Extracted at.
	extractedAt := page.Metadata.ExtractedAt.Format("2006-01-02 15:04:05 UTC")

	fmt.Fprintf(w, "URL:          %s\n", page.URL)
	fmt.Fprintf(w, "Title:        %s\n", page.Title)
	fmt.Fprintf(w, "Section:      %s\n", page.Section)
	fmt.Fprintf(w, "Type:         %s\n", page.PageType)
	fmt.Fprintf(w, "Depth:        %d\n", page.Metadata.Depth)
	fmt.Fprintf(w, "Parent:       %s\n", parentURL)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Breadcrumb:   %s\n", breadcrumb)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary:      %s\n", summary)
	fmt.Fprintf(w, "Content Len:  %d chars\n", page.Content.ContentLength)
	fmt.Fprintf(w, "Mini-Summary: %s\n", miniSummaryText)
	fmt.Fprintln(w)

	// Children.
	fmt.Fprintf(w, "Children:     %d\n", len(page.Links.Children))
	if len(page.Links.Children) > 0 {
		limit := 10
		shown := page.Links.Children
		if len(shown) > limit {
			shown = shown[:limit]
		}
		for _, c := range shown {
			fmt.Fprintf(w, "  - %s — %s\n", c.Title, c.URL)
		}
		if len(page.Links.Children) > limit {
			fmt.Fprintf(w, "  (+ %d more)\n", len(page.Links.Children)-limit)
		}
	}
	fmt.Fprintln(w)

	// Links.
	fmt.Fprintf(w, "Internal:     %d\n", len(page.Links.Internal))
	fmt.Fprintf(w, "External:     %d\n", len(page.Links.External))

	// Documents.
	fmt.Fprintf(w, "Documents:    %d\n", len(page.Documents))
	if len(page.Documents) > 0 {
		limit := 10
		shown := page.Documents
		if len(shown) > limit {
			shown = shown[:limit]
		}
		for _, d := range shown {
			fmt.Fprintf(w, "  - %s [%s] — %s\n", d.Title, d.Type, d.URL)
		}
		if len(page.Documents) > limit {
			fmt.Fprintf(w, "  (+ %d more)\n", len(page.Documents)-limit)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Tags:         %s\n", tagsStr)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Extracted at: %s\n", extractedAt)
	fmt.Fprintf(w, "Last-Modified:%s\n", page.Metadata.LastModified)
	fmt.Fprintf(w, "ETag:         %s\n", page.Metadata.ETag)
	fmt.Fprintf(w, "Crawler:      %s\n", page.Metadata.CrawlerVersion)
	fmt.Fprintf(w, "Discovered:   %s\n", page.Metadata.DiscoveredVia)
	fmt.Fprintf(w, "Plone copy:   %v\n", page.Metadata.IsPloneCopy)
	fmt.Fprintf(w, "Canonical of: %s\n", canonicalOf)
	fmt.Fprintf(w, "Stale since:  %s\n", staleSince)

	return nil
}
