package app

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/adapters/llm"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

//go:embed prompts/landing.md
var landingTmpl string

//go:embed prompts/article.md
var articleTmpl string

//go:embed prompts/listing.md
var listingTmpl string

const (
	systemPrompt   = "Você é um classificador de páginas web; produz descrições objetivas curtas em português."
	maxSummaryLen  = 500 // safety cap; log warn if > 240
	warnSummaryLen = 240
	excerptRunes   = 1500
	maxChildren    = 15
)

// SummarizeOptions configures the summarize run.
type SummarizeOptions struct {
	// Force regeneration even if source_hash matches.
	Force bool
	// MaxPages caps total processed pages; 0 = unlimited.
	MaxPages int
}

// Summarize walks the store, identifies pages needing a mini_summary,
// calls the provider concurrently (cfg.LLM.Concurrency), stores results,
// and returns a report with tokens + estimated cost.
// Failures are isolated: a single provider error does NOT abort the run;
// it's recorded in SummarizeReport.FailedReasons.
func Summarize(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts SummarizeOptions, provider ports.LLMProvider) (*domain.SummarizeReport, error) {
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		return nil, err
	}

	// Step 1: Walk and collect all pages.
	var pages []*domain.Page
	if err := store.Walk(ctx, func(p *domain.Page) error {
		pages = append(pages, p)
		return nil
	}); err != nil {
		return nil, err
	}

	report := &domain.SummarizeReport{
		StartedAt:     time.Now(),
		TotalPages:    len(pages),
		Provider:      provider.Name(),
		Model:         provider.Model(),
		FailedReasons: map[string]int{},
	}

	// Step 2: Filter pages into todo list; handle immediate skips.
	var todo []*domain.Page
	for _, page := range pages {
		skippedReason := shouldSkip(page, opts.Force)
		if skippedReason != "" {
			page.MiniSummary.Skipped = &skippedReason
			if err := store.Put(ctx, page); err != nil {
				logger.Warn("summarize: failed to store skip marker", "url", page.URL, "err", err)
			}
			report.Skipped++
			continue
		}
		todo = append(todo, page)
	}

	// Step 3: Apply MaxPages cap.
	if opts.MaxPages > 0 && len(todo) > opts.MaxPages {
		todo = todo[:opts.MaxPages]
	}

	// Step 4: Worker pool.
	concurrency := cfg.LLM.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	pageCh := make(chan *domain.Page, len(todo))
	for _, p := range todo {
		pageCh <- p
	}
	close(pageCh)

	var (
		muReport sync.Mutex
		wg       sync.WaitGroup

		atomicGenerated    int64
		atomicFailed       int64
		atomicTokensInput  int64
		atomicTokensOutput int64
	)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range pageCh {
				if ctx.Err() != nil {
					return
				}

				system, user := buildPrompt(page)
				req := ports.GenerateRequest{
					System:      system,
					User:        user,
					MaxTokens:   256,
					Temperature: 0.2,
				}

				resp, err := provider.Generate(ctx, req)
				if err != nil {
					atomic.AddInt64(&atomicFailed, 1)
					cls := classifyErr(err)
					muReport.Lock()
					report.FailedReasons[cls]++
					muReport.Unlock()
					logger.Warn("summarize: provider error", "url", page.URL, "class", cls, "err", err)
					continue
				}

				text := strings.TrimSpace(resp.Text)
				if len([]rune(text)) > maxSummaryLen {
					text = string([]rune(text)[:maxSummaryLen])
				}
				if len([]rune(text)) > warnSummaryLen {
					logger.Warn("summarize: summary exceeds 240 chars", "url", page.URL, "length", len([]rune(text)))
				}

				page.MiniSummary.Text = text
				page.MiniSummary.GeneratedAt = time.Now()
				page.MiniSummary.Model = resp.Model
				page.MiniSummary.SourceHash = page.Content.FullTextHash
				page.MiniSummary.Skipped = nil

				if err := store.Put(ctx, page); err != nil {
					logger.Warn("summarize: failed to store page", "url", page.URL, "err", err)
				}

				atomic.AddInt64(&atomicGenerated, 1)
				atomic.AddInt64(&atomicTokensInput, resp.TokensInput)
				atomic.AddInt64(&atomicTokensOutput, resp.TokensOutput)
			}
		}()
	}

	wg.Wait()

	report.Generated = int(atomic.LoadInt64(&atomicGenerated))
	report.Failed = int(atomic.LoadInt64(&atomicFailed))
	report.TokensInput = atomic.LoadInt64(&atomicTokensInput)
	report.TokensOutput = atomic.LoadInt64(&atomicTokensOutput)
	report.EstimatedCost = llm.EstimateCost(report.Provider, report.Model, report.TokensInput, report.TokensOutput)
	report.CompletedAt = time.Now()

	logger.Info("summarize complete",
		"total", report.TotalPages,
		"generated", report.Generated,
		"skipped", report.Skipped,
		"failed", report.Failed,
		"tokens_in", report.TokensInput,
		"tokens_out", report.TokensOutput,
		"estimated_cost_usd", report.EstimatedCost,
	)

	return report, nil
}

// shouldSkip returns the skip reason string, or "" if the page should be summarized.
func shouldSkip(page *domain.Page, force bool) string {
	switch page.PageType {
	case domain.PageTypeEmpty:
		return "empty_content"
	case domain.PageTypeRedirect:
		return "redirect"
	}
	if !force &&
		page.MiniSummary.Text != "" &&
		page.MiniSummary.SourceHash == page.Content.FullTextHash {
		return "up_to_date"
	}
	return ""
}

// buildPrompt builds the system and user prompt for a page based on its type.
func buildPrompt(page *domain.Page) (system, user string) {
	system = systemPrompt

	var tmpl string
	switch page.PageType {
	case domain.PageTypeLanding:
		tmpl = landingTmpl
	case domain.PageTypeListing:
		tmpl = listingTmpl
	default:
		// article and any unknown types
		tmpl = articleTmpl
	}

	// Build children list for landing pages
	childrenStr := ""
	if len(page.Links.Children) > 0 {
		limit := len(page.Links.Children)
		if limit > maxChildren {
			limit = maxChildren
		}
		titles := make([]string, limit)
		for i := 0; i < limit; i++ {
			titles[i] = page.Links.Children[i].Title
		}
		childrenStr = strings.Join(titles, ", ")
	}

	excerpt := safeExcerpt(page.Content.FullText, excerptRunes)

	r := strings.NewReplacer(
		"{{TITLE}}", page.Title,
		"{{SECTION}}", page.Section,
		"{{PATH_TITLES}}", strings.Join(page.PathTitles, " › "),
		"{{CHILDREN}}", childrenStr,
		"{{DOC_COUNT}}", strconv.Itoa(len(page.Documents)),
		"{{CHILD_COUNT}}", strconv.Itoa(len(page.Links.Children)),
		"{{EXCERPT}}", excerpt,
	)

	user = r.Replace(tmpl)
	return system, user
}

// safeExcerpt returns the first n runes of s, safely cut at a rune boundary.
func safeExcerpt(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	// Walk runes to cut at boundary.
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// classifyErr maps an error to a string category for reporting.
func classifyErr(err error) string {
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "context_timeout"
	}
	if errors.Is(err, llm.ErrRateLimited) {
		return "rate_limited"
	}
	if errors.Is(err, llm.ErrProviderUnavailable) {
		return "provider_unavailable"
	}
	if errors.Is(err, llm.ErrInvalidAPIKey) {
		return "invalid_api_key"
	}
	if errors.Is(err, llm.ErrEmptyResponse) {
		return "empty_response"
	}
	return "other"
}
