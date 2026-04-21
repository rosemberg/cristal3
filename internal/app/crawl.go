package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/adapters/htmlextract"
	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
	"github.com/bergmaia/site-research/internal/canonical"
	"github.com/bergmaia/site-research/internal/classify"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

// priorRecord holds the incremental-crawl metadata for a previously-stored page.
type priorRecord struct {
	ETag         string             // value from Metadata.ETag
	LastModified string             // raw HTTP header string (for If-Modified-Since)
	ContentHash  string             // "sha256:..." from Page.Content.ContentHash
	MiniSummary  domain.MiniSummary // preserved when content is unchanged
	FullPage     *domain.Page       // used on 304 reload path
}

// loadPriorIndex walks the store and returns canonicalURL -> priorRecord.
// Uses Canonicalize on page.URL to build the key so comparisons work against
// canonicalized candidate URLs. Errors from Walk are logged and swallowed —
// a corrupted prior record should not abort a fresh crawl.
func loadPriorIndex(ctx context.Context, store *fsstore.Store, canon *canonical.Canonicalizer, logger *slog.Logger) map[string]priorRecord {
	index := make(map[string]priorRecord)
	err := store.Walk(ctx, func(p *domain.Page) error {
		if p.URL == "" {
			return nil
		}
		key, _, err := canon.Canonicalize(p.URL)
		if err != nil {
			logger.Warn("prior-index: canonicalize failed; skipping", "url", p.URL, "err", err)
			return nil
		}
		// Shallow-copy the page so the pointer is stable.
		pageCopy := *p
		index[key] = priorRecord{
			ETag:         p.Metadata.ETag,
			LastModified: p.Metadata.LastModified,
			ContentHash:  p.Content.ContentHash,
			MiniSummary:  p.MiniSummary,
			FullPage:     &pageCopy,
		}
		return nil
	})
	if err != nil {
		logger.Warn("prior-index: walk error; proceeding without prior index", "err", err)
		return map[string]priorRecord{}
	}
	return index
}

// CrawlerVersion is stamped into Page.Metadata.CrawlerVersion.
const CrawlerVersion = "0.1.0"

// CrawlOptions configures the Crawl application service.
type CrawlOptions struct {
	// FromFile overrides cfg.Sitemap.URL when non-empty (for offline testing/validation).
	FromFile string
	// DryRun skips writes to PageStore while still performing fetch/extract/classify.
	DryRun bool
	// MaxURLs caps the number of URLs processed; 0 means unlimited.
	MaxURLs int
	// OverrideURLs, when non-nil, bypasses DiscoverInScope entirely (used by tests).
	// Intended for integration testing; CLI should leave this nil.
	OverrideURLs []URLCandidate
	// PurgeStale, when true, deletes pages that have been stale longer than
	// cfg.Recrawl.StaleRetentionDays. Requires Confirm == true (destructive operation).
	PurgeStale bool
	// Confirm must be set to true alongside PurgeStale to authorize the destructive
	// purge operation (RNF-05).
	Confirm bool
}

// crawlSession bundles all shared state for one crawl run, making per-URL processing
// available as a method to avoid code duplication between the sitemap and orphan passes.
type crawlSession struct {
	ctx          context.Context
	logger       *slog.Logger
	cfg          *config.Config
	opts         CrawlOptions
	fetcher      ports.Fetcher
	store        *fsstore.Store
	canon        *canonical.Canonicalizer
	extractor    *htmlextract.Extractor
	classifier   *classify.Classifier
	breaker      *httpfetch.CircuitBreaker
	suspDetector *httpfetch.SuspiciousDetector
	priorIndex   map[string]priorRecord
	// visited tracks all URLs whose fetch attempt concluded (success, 304, 4xx).
	visited map[string]struct{}
	// linked accumulates in-scope links discovered during extraction.
	linked    map[string]struct{}
	report    *domain.CrawlReport
	processed int // total fetch attempts counted toward MaxURLs
	aborted   bool
}

// processOne executes the full per-URL pipeline for cand using discoveredVia as the
// source stamp. Returns true to continue, false to stop the current pass (circuit
// aborted, ctx canceled, or MaxURLs reached).
func (s *crawlSession) processOne(cand URLCandidate, discoveredVia domain.DiscoverySource) bool {
	if err := s.ctx.Err(); err != nil {
		return false
	}

	// Honor MaxURLs across both passes.
	if s.opts.MaxURLs > 0 && s.processed >= s.opts.MaxURLs {
		return false
	}

	// Circuit breaker gate.
	wait, cbErr := s.breaker.Allow()
	if errors.Is(cbErr, httpfetch.ErrCircuitAborted) {
		s.logger.Error("circuit breaker aborted", "processed", s.processed)
		s.aborted = true
		return false
	}
	if wait > 0 {
		s.logger.Warn("circuit breaker open; pausing", "wait", wait.String())
		t := time.NewTimer(wait)
		select {
		case <-t.C:
			t.Stop()
		case <-s.ctx.Done():
			t.Stop()
			return false
		}
	}

	// Mark as processed (counts toward MaxURLs) before the fetch.
	s.processed++

	// Canonicalize candidate URL for prior-index lookup.
	canonURL, _, _ := s.canon.Canonicalize(cand.URL)

	// Build fetch request with conditional headers when a prior record exists.
	fetchReq := ports.FetchRequest{URL: cand.URL}
	if prior, ok := s.priorIndex[canonURL]; ok {
		fetchReq.IfNoneMatch = prior.ETag
		if prior.LastModified != "" {
			if t, err := http.ParseTime(prior.LastModified); err == nil {
				fetchReq.IfModifiedSince = t
			}
		}
	}

	// Fetch.
	result, err := s.fetcher.Fetch(s.ctx, fetchReq)
	if err != nil {
		s.breaker.RecordFailure()
		s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{URL: cand.URL, Error: err.Error()})
		s.logger.Warn("fetch failed", "url", cand.URL, "err", err)
		s.visited[canonURL] = struct{}{}
		return true
	}
	s.report.HTTPStatusHistogram[result.StatusCode]++

	// 304 short-circuit: reload prior page, stamp fresh timing metadata, re-write.
	// 304 does NOT count toward TotalCrawled because the full pipeline was skipped.
	if result.NotModified {
		s.visited[canonURL] = struct{}{}
		prior, ok := s.priorIndex[canonURL]
		if ok && prior.FullPage != nil {
			refreshed := *prior.FullPage // shallow copy
			refreshed.Metadata.ExtractedAt = time.Now()
			refreshed.Metadata.FetchDurationMs = result.DurationMs
			refreshed.Metadata.HTTPStatus = 304
			if !s.opts.DryRun {
				if err := s.store.Put(s.ctx, &refreshed); err != nil {
					s.breaker.RecordFailure()
					s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{URL: cand.URL, Error: err.Error()})
					s.logger.Warn("store failed (304 refresh)", "url", cand.URL, "err", err)
					return true
				}
			}
			s.breaker.RecordSuccess()
			s.report.UnchangedPages++
			s.report.HTTPStatusHistogram[304]++
			return true
		}
		// No prior page but got 304 (unusual) → log warn and skip.
		s.logger.Warn("304 without prior record; skipping", "url", canonURL)
		s.breaker.RecordSuccess()
		s.report.UnchangedPages++
		return true
	}

	// Non-success status → record and continue (don't retry; Fetcher already did).
	if result.StatusCode >= 400 {
		// Don't count as circuit failure (non-transient 4xx).
		s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{
			URL:    cand.URL,
			Status: result.StatusCode,
			Error:  fmt.Sprintf("HTTP %d", result.StatusCode),
		})
		s.visited[canonURL] = struct{}{}
		return true
	}

	// Extract.
	page, err := s.extractor.Extract(s.ctx, result.URL, result.Body)
	if err != nil {
		s.breaker.RecordFailure()
		s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{URL: result.URL, Error: err.Error()})
		s.logger.Warn("extract failed", "url", result.URL, "err", err)
		return true
	}

	// Collect in-scope links for orphan detection.
	scopePrefix := strings.TrimRight(s.cfg.Scope.Prefix, "/")
	for _, child := range page.Links.Children {
		u := strings.TrimRight(child.URL, "/")
		if u != "" && u != scopePrefix && strings.HasPrefix(u, scopePrefix+"/") {
			if c, exc, err := s.canon.Canonicalize(u); err == nil && !exc {
				s.linked[c] = struct{}{}
			}
		}
	}
	for _, internal := range page.Links.Internal {
		u := strings.TrimRight(internal.URL, "/")
		if u != "" && u != scopePrefix && strings.HasPrefix(u, scopePrefix+"/") {
			if c, exc, err := s.canon.Canonicalize(u); err == nil && !exc {
				s.linked[c] = struct{}{}
			}
		}
	}

	// Suspicious response check (per RF-07).
	hdr := http.Header{}
	for k, v := range result.Headers {
		hdr.Set(k, v)
	}
	reason := s.suspDetector.Check(page.Title, len(result.Body), hdr, false)
	if reason != httpfetch.ReasonNone {
		s.breaker.RecordFailure()
		s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{
			URL:   result.URL,
			Error: fmt.Sprintf("suspicious_response:%s", reason),
		})
		s.logger.Warn("suspicious response dropped", "url", result.URL, "reason", string(reason))
		return true
	}

	// Stamp metadata + classify + canonical.
	// Re-canonicalize using result.URL (may differ after redirects).
	canonURL, _, _ = s.canon.Canonicalize(result.URL)
	page.URL = result.URL
	page.CanonicalURL = canonURL
	page.Schema = "page-node-v2"
	page.SchemaVersion = 2
	page.PageType = s.classifier.Classify(page)
	page.HasSubstantiveContent = page.PageType == domain.PageTypeArticle || page.PageType == domain.PageTypeListing
	page.Metadata.CrawlerVersion = CrawlerVersion
	page.Metadata.ExtractedAt = time.Now()
	page.Metadata.HTTPStatus = result.StatusCode
	page.Metadata.ContentType = headerGet(result.Headers, "Content-Type")
	page.Metadata.ETag = result.ETag
	page.Metadata.LastModified = headerGet(result.Headers, "Last-Modified")
	page.Metadata.FetchDurationMs = result.DurationMs
	page.Metadata.DiscoveredVia = discoveredVia
	page.Metadata.Depth = depthFor(canonURL, s.cfg.Scope.Prefix)
	page.Metadata.ParentURL = parentURLFor(canonURL, s.cfg.Scope.Prefix)
	page.Metadata.IsPloneCopy = canonical.IsPloneCopy(page.URL)
	if result.OriginalURL != "" && result.OriginalURL != result.URL {
		orig := result.OriginalURL
		page.Metadata.RedirectedFrom = &orig
	}

	// Content-hash diff: compare with prior to decide new/updated/unchanged (RF-08).
	prior, havePrior := s.priorIndex[canonURL]
	unchangedDueToHash := false
	if havePrior {
		if prior.ContentHash != "" && prior.ContentHash == page.Content.ContentHash {
			page.MiniSummary = prior.MiniSummary
			unchangedDueToHash = true
		} else {
			page.MiniSummary = domain.MiniSummary{}
		}
		// If the page was previously stale, clear StaleSince now that it's back.
		if prior.FullPage != nil && prior.FullPage.Metadata.StaleSince != nil {
			page.Metadata.StaleSince = nil
			s.logger.Info("page returned from stale", "url", page.URL)
		}
	}

	// Store.
	if !s.opts.DryRun {
		if err := s.store.Put(s.ctx, page); err != nil {
			s.breaker.RecordFailure()
			s.report.FailedURLs = append(s.report.FailedURLs, domain.FailedURL{URL: page.URL, Error: err.Error()})
			s.logger.Warn("store failed", "url", page.URL, "err", err)
			return true
		}
	} else {
		s.logger.Debug("dry-run: skipped store", "url", page.URL)
	}

	s.breaker.RecordSuccess()
	s.visited[canonURL] = struct{}{}
	// TotalCrawled = NewPages + UpdatedPages + unchanged-via-hash.
	// 304 paths do NOT increment TotalCrawled (full pipeline was skipped).
	if havePrior {
		if unchangedDueToHash {
			s.report.UnchangedPages++
		} else {
			s.report.UpdatedPages++
		}
	} else {
		s.report.NewPages++
	}
	s.report.TotalCrawled++

	return true
}

// markStale walks the store and marks pages absent from both visited and linked sets
// as stale by setting Metadata.StaleSince to now (if not already set). Returns the
// number of pages newly marked stale this run (nil → non-nil transitions only).
//
// NOTE: we mark stale on sitemap/orphan absence alone; BRIEF suggests also requiring
// 404 on re-crawl. Adding an HTTP HEAD verification is a future hardening step
// (tracked in plan M5 notes).
func (s *crawlSession) markStale() int {
	if s.opts.DryRun {
		return 0
	}
	now := time.Now()
	newlyStale := 0
	err := s.store.Walk(s.ctx, func(p *domain.Page) error {
		if p.URL == "" {
			return nil
		}
		canonURL, _, err := s.canon.Canonicalize(p.URL)
		if err != nil {
			return nil
		}
		_, inVisited := s.visited[canonURL]
		_, inLinked := s.linked[canonURL]
		if inVisited || inLinked {
			// URL was seen this run — not stale.
			return nil
		}
		// URL was absent this run.
		if p.Metadata.StaleSince != nil {
			// Already marked stale; preserve original timestamp.
			return nil
		}
		// Newly stale: stamp it.
		p.Metadata.StaleSince = &now
		if err := s.store.Put(s.ctx, p); err != nil {
			s.logger.Warn("markStale: store failed", "url", p.URL, "err", err)
			return nil
		}
		newlyStale++
		s.logger.Info("page marked stale", "url", p.URL)
		return nil
	})
	if err != nil {
		s.logger.Warn("markStale: walk error", "err", err)
	}
	return newlyStale
}

// purgeStale walks the store and deletes pages that have been stale longer than
// cfg.Recrawl.StaleRetentionDays. It increments report.RemovedPages for each deletion.
// This is a destructive operation and must only be called when opts.PurgeStale &&
// opts.Confirm are both true.
func (s *crawlSession) purgeStale() {
	retention := time.Duration(s.cfg.Recrawl.StaleRetentionDays) * 24 * time.Hour
	if retention <= 0 {
		retention = 30 * 24 * time.Hour
	}
	err := s.store.Walk(s.ctx, func(p *domain.Page) error {
		if p.URL == "" || p.Metadata.StaleSince == nil {
			return nil
		}
		staleDuration := time.Since(*p.Metadata.StaleSince)
		if staleDuration <= retention {
			return nil
		}
		if err := s.store.Delete(s.ctx, p.URL); err != nil {
			s.logger.Warn("purgeStale: delete failed", "url", p.URL, "err", err)
			return nil
		}
		s.report.RemovedPages++
		s.logger.Info("purged stale page", "url", p.URL, "stale_for", staleDuration.String())
		return nil
	})
	if err != nil {
		s.logger.Warn("purgeStale: walk error", "err", err)
	}
}

// runSitemapPass processes all sitemap URL candidates in order.
func runSitemapPass(s *crawlSession, urls []URLCandidate) {
	for i, cand := range urls {
		if !s.processOne(cand, domain.DiscoveredViaSitemap) {
			break
		}
		if (i+1)%50 == 0 {
			s.logger.Info("crawl progress", "processed", i+1, "total", len(urls))
		}
	}
}

// collectOrphans computes the set of in-scope linked URLs that were not visited
// during the sitemap pass. Returns them as URLCandidate slices (zero LastMod).
func collectOrphans(s *crawlSession) []URLCandidate {
	var orphans []URLCandidate
	for u := range s.linked {
		if _, seen := s.visited[u]; seen {
			continue
		}
		orphans = append(orphans, URLCandidate{URL: u})
	}
	return orphans
}

// runOrphanPass processes a list of orphan candidates, stamping each with
// DiscoveredViaLink. It respects MaxURLs (checked inside processOne).
func runOrphanPass(s *crawlSession, orphans []URLCandidate) {
	failed := 0
	for _, cand := range orphans {
		if !s.processOne(cand, domain.DiscoveredViaLink) {
			failed++
			break
		}
	}
	processed := s.processed
	s.logger.Info("orphan pass complete", "processed", processed, "failed", failed)
}

// Crawl executes the discover → fetch → extract → classify → store pipeline
// described by RF-01, RF-03, RF-04, RF-05, RF-06 and RF-07. Incremental logic
// (RF-08) and BFS orphan discovery (RF-02) are included.
//
// The function returns only after all URLs are processed or ctx is canceled
// or the circuit breaker aborts. A summary CrawlReport is logged.
func Crawl(ctx context.Context, logger *slog.Logger, cfg *config.Config, opts CrawlOptions) (*domain.CrawlReport, error) {
	// 0. Validate destructive-operation guard (RNF-05).
	if opts.PurgeStale && !opts.Confirm {
		return nil, fmt.Errorf("purge-stale requires --confirm (destructive operation)")
	}

	// 1. URL list.
	var urls []URLCandidate
	if opts.OverrideURLs != nil {
		urls = opts.OverrideURLs
	} else {
		discovered, _, err := DiscoverInScope(ctx, cfg, opts.FromFile)
		if err != nil {
			return nil, err
		}
		urls = discovered
	}

	// 2. Apply MaxURLs to sitemap slice only (orphan pass manages its own limit).
	sitemapURLs := urls
	if opts.MaxURLs > 0 && opts.MaxURLs < len(sitemapURLs) {
		sitemapURLs = sitemapURLs[:opts.MaxURLs]
	}

	// 3. Build adapters.
	ua := cfg.Crawler.UserAgent
	jitter := time.Duration(cfg.Crawler.JitterMS) * time.Millisecond
	limiter := httpfetch.NewLimiter(cfg.Crawler.RateLimitPerSecond, jitter)
	httpClient := &http.Client{Timeout: time.Duration(cfg.Crawler.RequestTimeoutSeconds) * time.Second}

	robotsFetcher := func(ctx context.Context, host string) ([]byte, error) {
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://"+host+"/robots.txt", nil)
		req.Header.Set("User-Agent", ua)
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return []byte(""), nil
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("robots.txt: status %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	robots := httpfetch.NewRobotsCache(ua, robotsFetcher)

	fetcher, err := httpfetch.New(httpfetch.Options{
		UserAgent:               ua,
		HTTPClient:              httpClient,
		Timeout:                 time.Duration(cfg.Crawler.RequestTimeoutSeconds) * time.Second,
		Limiter:                 limiter,
		Backoff:                 httpfetch.BackoffParams{Attempts: cfg.Crawler.MaxRetries},
		Robots:                  robots,
		RespectRobotsTxt:        cfg.Crawler.RespectRobotsTxt,
		HonorRetryAfter:         cfg.Crawler.HonorRetryAfter,
		LongRetryAfterThreshold: 60 * time.Second,
		Logger:                  logger,
	})
	if err != nil {
		return nil, fmt.Errorf("crawl: build fetcher: %w", err)
	}

	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("crawl: build store: %w", err)
	}

	canon := canonical.New()
	extractor := htmlextract.New(htmlextract.Options{
		CrawlerVersion: CrawlerVersion,
		ScopePrefix:    cfg.Scope.Prefix,
	})
	classifier := classify.New(classify.Config{})
	breaker := httpfetch.NewCircuitBreaker(httpfetch.CircuitBreakerConfig{
		MaxConsecutive: cfg.Crawler.CircuitBreaker.MaxConsecutiveFailures,
		PauseDuration:  time.Duration(cfg.Crawler.CircuitBreaker.PauseMinutes) * time.Minute,
		AbortThreshold: cfg.Crawler.CircuitBreaker.AbortThreshold,
	}, nil)
	suspDetector := httpfetch.NewSuspiciousDetector(httpfetch.SuspiciousDetectorConfig{
		MinBodyBytes:       cfg.Crawler.SuspiciousResponse.MinBodyBytes,
		BlockTitlePatterns: cfg.Crawler.SuspiciousResponse.BlockTitlePatterns,
	})

	// 4. Build prior index for incremental crawl (RF-08).
	priorIndex := loadPriorIndex(ctx, store, canon, logger)

	// 5. Report init.
	report := &domain.CrawlReport{
		StartedAt:           time.Now(),
		SitemapTotal:        len(urls),
		HTTPStatusHistogram: map[int]int{},
	}

	// 6. Build session.
	sess := &crawlSession{
		ctx:          ctx,
		logger:       logger,
		cfg:          cfg,
		opts:         opts,
		fetcher:      fetcher,
		store:        store,
		canon:        canon,
		extractor:    extractor,
		classifier:   classifier,
		breaker:      breaker,
		suspDetector: suspDetector,
		priorIndex:   priorIndex,
		visited:      make(map[string]struct{}),
		linked:       make(map[string]struct{}),
		report:       report,
	}

	// 7. Sitemap pass.
	runSitemapPass(sess, sitemapURLs)

	// 8. Orphan pass — only when the sitemap pass completed normally
	//    (no circuit abort, no ctx cancellation, not dry-run-only skip).
	if !sess.aborted && ctx.Err() == nil {
		orphans := collectOrphans(sess)
		report.OrphansFound = len(orphans)

		if len(orphans) > 0 {
			// Check whether MaxURLs is already exhausted.
			if opts.MaxURLs > 0 && sess.processed >= opts.MaxURLs {
				logger.Info("orphan pass skipped: MaxURLs reached", "orphans", len(orphans))
			} else {
				logger.Info("orphan pass starting", "count", len(orphans))
				runOrphanPass(sess, orphans)
			}
		}
	} else if sess.aborted {
		logger.Info("orphan pass skipped: circuit breaker aborted main loop")
	}

	// 9. Dedup pass (content_hash for duplicate pages) — runs after BOTH loops.
	if !opts.DryRun {
		dedupCount := dedupContentHashes(ctx, store, logger)
		if dedupCount > 0 {
			logger.Info("dedup complete", "dedup_pairs_count", dedupCount)
		}
	}

	// 10. Stale-marking pass — walk the store and stamp pages absent from both
	// visited and linked sets. Only runs when not dry-run and when the session
	// was not aborted (to avoid false stale marks from incomplete runs).
	if !opts.DryRun && !sess.aborted && ctx.Err() == nil {
		newlyStale := sess.markStale()
		report.StalePages = newlyStale
		if newlyStale > 0 {
			logger.Info("stale marking complete", "newly_stale", newlyStale)
		}
	}

	// 11. Purge-stale pass — delete stale pages older than retention threshold.
	if opts.PurgeStale && opts.Confirm && !opts.DryRun {
		sess.purgeStale()
		if report.RemovedPages > 0 {
			logger.Info("purge stale complete", "removed", report.RemovedPages)
		}
	}

	// 12. Final report.
	report.CompletedAt = time.Now()
	durationMs := report.CompletedAt.Sub(report.StartedAt).Milliseconds()
	logger.Info("crawl complete",
		"total", len(urls),
		"crawled", report.TotalCrawled,
		"orphans_found", report.OrphansFound,
		"unchanged", report.UnchangedPages,
		"failed", len(report.FailedURLs),
		"duration_ms", durationMs,
	)

	return report, nil
}

// dedupContentHashes walks the store, groups pages by ContentHash, and for each
// group with ≥ 2 pages picks the shortest URL as canonical; others get
// Metadata.CanonicalOf set and are written back. Returns the number of dedup pairs.
func dedupContentHashes(ctx context.Context, store *fsstore.Store, logger *slog.Logger) int {
	// Group pages by content hash.
	byHash := map[string][]*domain.Page{}
	err := store.Walk(ctx, func(p *domain.Page) error {
		if p.Content.ContentHash == "" {
			return nil
		}
		byHash[p.Content.ContentHash] = append(byHash[p.Content.ContentHash], p)
		return nil
	})
	if err != nil {
		logger.Warn("dedup: walk error", "err", err)
		return 0
	}

	dedupPairs := 0
	for _, pages := range byHash {
		if len(pages) < 2 {
			continue
		}
		// Pick the shortest URL as canonical.
		canonical := pages[0]
		for _, p := range pages[1:] {
			if len(p.URL) < len(canonical.URL) {
				canonical = p
			}
		}
		canonURL := canonical.URL
		// Mark all non-canonical pages.
		for _, p := range pages {
			if p.URL == canonURL {
				continue
			}
			p.Metadata.CanonicalOf = &canonURL
			if err := store.Put(ctx, p); err != nil {
				logger.Warn("dedup: store error", "url", p.URL, "err", err)
			}
			dedupPairs++
		}
	}
	return dedupPairs
}

// headerGet returns the first value of a header, case-insensitively matching.
// Headers come from FetchResult.Headers as map[string]string with original casing.
func headerGet(headers map[string]string, key string) string {
	// Try exact match first.
	if v, ok := headers[key]; ok {
		return v
	}
	// Case-insensitive fallback (headers are stored lowercased by collectHeaders).
	lower := strings.ToLower(key)
	if v, ok := headers[lower]; ok {
		return v
	}
	// Full scan for any remaining case variant.
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

// depthFor returns the number of path segments below scopePrefix.
// depthFor(prefix, prefix) = 0; depthFor(prefix+"/a", prefix) = 1; etc.
func depthFor(u, prefix string) int {
	prefix = strings.TrimRight(prefix, "/")
	u = strings.TrimRight(u, "/")
	if u == prefix {
		return 0
	}
	if !strings.HasPrefix(u, prefix+"/") {
		return 0
	}
	rel := u[len(prefix)+1:]
	if rel == "" {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

// parentURLFor returns the URL of the parent page (one segment up), or scopePrefix
// when already at depth 1, or "" when u == prefix.
func parentURLFor(u, prefix string) string {
	prefix = strings.TrimRight(prefix, "/")
	u = strings.TrimRight(u, "/")
	if u == prefix {
		return ""
	}
	if !strings.HasPrefix(u, prefix+"/") {
		return ""
	}
	rel := u[len(prefix)+1:]
	if !strings.Contains(rel, "/") {
		// Already at depth 1; parent is the prefix itself.
		return prefix
	}
	// Strip last segment.
	lastSlash := strings.LastIndex(u, "/")
	if lastSlash <= len(prefix) {
		return prefix
	}
	return u[:lastSlash]
}
