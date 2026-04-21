package httpfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bergmaia/site-research/internal/domain/ports"
)

// maxBodyBytes caps body reads to 10 MiB to avoid runaway allocations.
const maxBodyBytes = 10 * 1024 * 1024

// ErrBlockedByRobots is returned when the target URL is disallowed by robots.txt.
// Callers may match with errors.Is to distinguish from other errors.
var ErrBlockedByRobots = errors.New("httpfetch: blocked by robots.txt")

// Options configures a Client.
type Options struct {
	UserAgent               string        // required; empty means a generic Go UA
	HTTPClient              *http.Client  // optional; if nil a client with Timeout is built
	Timeout                 time.Duration // per-request timeout; default 30s
	Limiter                 *Limiter      // required
	Backoff                 BackoffParams // optional; zero → defaults
	Robots                  *RobotsCache  // optional; if nil, robots are not enforced
	RespectRobotsTxt        bool          // if false, Robots is ignored
	HonorRetryAfter         bool          // if true, 429/503 Retry-After is honored per RF-07
	LongRetryAfterThreshold time.Duration // long pause threshold; default 60s
	Logger                  *slog.Logger  // optional; if nil, a no-op logger
	Rng                     *rand.Rand    // optional; if nil, one is seeded from time.Now
}

// Client is a rate-limited, retrying HTTP fetcher with robots.txt enforcement,
// Retry-After honoring and structured observability. Implements ports.Fetcher.
type Client struct {
	opts       Options
	backoff    BackoffParams
	classifier RetryClassifier
	rng        *rand.Rand
	logger     *slog.Logger
	httpClient *http.Client
}

// New builds a Client. Returns error if Limiter is nil (the only hard requirement).
func New(opts Options) (*Client, error) {
	if opts.Limiter == nil {
		return nil, errors.New("httpfetch: Limiter is required")
	}

	// Apply defaults
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.LongRetryAfterThreshold <= 0 {
		opts.LongRetryAfterThreshold = 60 * time.Second
	}

	// Build HTTP client if not provided
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: opts.Timeout,
		}
	}

	// Build RNG if not provided
	rng := opts.Rng
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// Build logger if not provided
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	backoff := opts.Backoff.WithDefaults()

	return &Client{
		opts:       opts,
		backoff:    backoff,
		classifier: RetryClassifier{},
		rng:        rng,
		logger:     logger,
		httpClient: httpClient,
	}, nil
}

// Fetch executes one HTTP GET for req.URL with full M3 semantics:
//   - robots.txt check (if RespectRobotsTxt and Robots != nil) — returns wrapped ErrBlockedByRobots on denial.
//   - Rate limit + jitter via Limiter before each attempt.
//   - User-Agent header set (if non-empty).
//   - If req.IfNoneMatch != "" → set "If-None-Match" header.
//   - If !req.IfModifiedSince.IsZero() → set "If-Modified-Since" to HTTP date.
//   - Set "Cache-Control: no-cache" per RNF-06.
//   - Transient statuses retried up to backoff.Attempts times.
//   - Retry-After (if present and HonorRetryAfter):
//   - If parsed delay ≤ LongRetryAfterThreshold → use as backoff delay for the next attempt.
//   - If parsed delay > LongRetryAfterThreshold → emit logger.Warn + Limiter.PauseUntil(now+delay);
//     current attempt returns the error; next attempts will block in Limiter.Wait for the pause window.
//   - 304 Not Modified → returns FetchResult with NotModified=true, Body=nil, StatusCode=304.
//   - Final URL after redirects is recorded in FetchResult.URL; OriginalURL holds the original req.URL.
//   - Non-transient (4xx other than 429) → returned in FetchResult with no retry and no error.
//   - Network/timeout after max attempts → returns nil result + wrapped error.
//   - Body is fully read (up to 10 MiB) and returned in FetchResult.Body.
func (c *Client) Fetch(ctx context.Context, req ports.FetchRequest) (*ports.FetchResult, error) {
	originalURL := req.URL

	// robots.txt check before any HTTP call
	if c.opts.RespectRobotsTxt && c.opts.Robots != nil {
		allowed, err := c.opts.Robots.Allowed(ctx, req.URL)
		if err != nil {
			c.logger.Warn("robots.txt fetch error; allowing by default", "url", req.URL, "error", err)
		}
		if !allowed {
			return nil, fmt.Errorf("%w: %s", ErrBlockedByRobots, req.URL)
		}
	}

	var lastErr error

	for attempt := 0; attempt < c.backoff.Attempts; attempt++ {
		// Check context cancellation before each attempt
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Rate limit before each attempt
		if err := c.opts.Limiter.Wait(ctx); err != nil {
			return nil, err
		}

		// Re-check context after limiter wait
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		start := time.Now()

		// Build HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("httpfetch: build request: %w", err)
		}

		// Set headers
		if c.opts.UserAgent != "" {
			httpReq.Header.Set("User-Agent", c.opts.UserAgent)
		}
		if req.IfNoneMatch != "" {
			httpReq.Header.Set("If-None-Match", req.IfNoneMatch)
		}
		if !req.IfModifiedSince.IsZero() {
			httpReq.Header.Set("If-Modified-Since", req.IfModifiedSince.UTC().Format(http.TimeFormat))
		}
		// RNF-06: always send Cache-Control: no-cache
		httpReq.Header.Set("Cache-Control", "no-cache")

		// Execute request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			// Context cancellation: bubble up immediately without retry
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			// Network error: check if transient and retry
			if c.classifier.IsTransient(0, err) {
				lastErr = fmt.Errorf("httpfetch: attempt %d network error: %w", attempt+1, err)
				c.logger.Warn("transient network error", "url", req.URL, "attempt", attempt+1, "error", err)

				if attempt < c.backoff.Attempts-1 {
					delay := c.backoff.Delay(attempt, c.rng)
					c.logger.Debug("backing off", "url", req.URL, "delay", delay, "attempt", attempt+1)
					t := time.NewTimer(delay)
					select {
					case <-t.C:
						t.Stop()
					case <-ctx.Done():
						t.Stop()
						return nil, ctx.Err()
					}
				}
				continue
			}
			return nil, fmt.Errorf("httpfetch: request failed: %w", err)
		}

		durationMs := time.Since(start).Milliseconds()

		// Handle 304 Not Modified
		if resp.StatusCode == http.StatusNotModified {
			resp.Body.Close()
			return &ports.FetchResult{
				URL:         resp.Request.URL.String(),
				OriginalURL: originalURL,
				StatusCode:  http.StatusNotModified,
				Headers:     collectHeaders(resp.Header),
				FetchedAt:   time.Now(),
				DurationMs:  durationMs,
				NotModified: true,
			}, nil
		}

		// Read body with cap at maxBodyBytes
		limitedReader := io.LimitReader(resp.Body, maxBodyBytes+1)
		body, readErr := io.ReadAll(limitedReader)
		resp.Body.Close()

		if readErr != nil {
			c.logger.Warn("body read error", "url", req.URL, "error", readErr)
		}
		// If we read more than max, truncate and warn
		if int64(len(body)) > maxBodyBytes {
			body = body[:maxBodyBytes]
			c.logger.Warn("body truncated at max size", "url", req.URL, "max_bytes", maxBodyBytes)
		}

		// Parse ETag and Last-Modified
		etag := resp.Header.Get("ETag")
		var lastModified time.Time
		if lm := resp.Header.Get("Last-Modified"); lm != "" {
			if t, err := http.ParseTime(lm); err == nil {
				lastModified = t
			}
		}

		finalURL := resp.Request.URL.String()

		// Check if the status is transient (and should be retried)
		if c.classifier.IsTransient(resp.StatusCode, nil) {
			lastErr = fmt.Errorf("httpfetch: transient status %d on attempt %d", resp.StatusCode, attempt+1)
			c.logger.Warn("transient status", "url", req.URL, "status", resp.StatusCode, "attempt", attempt+1)

			if attempt < c.backoff.Attempts-1 {
				// Check Retry-After header if HonorRetryAfter is enabled
				sleepDur := c.backoff.Delay(attempt, c.rng)

				if c.opts.HonorRetryAfter {
					if raHeader := resp.Header.Get("Retry-After"); raHeader != "" {
						now := time.Now()
						if ra, ok := ParseRetryAfter(raHeader, now); ok {
							if ra <= c.opts.LongRetryAfterThreshold {
								// Short path: use as backoff delay
								sleepDur = ra
								c.logger.Debug("retry-after short path",
									"url", req.URL,
									"retry_after", ra,
									"attempt", attempt+1,
								)
							} else {
								// Long path: set global pause + log warn
								c.logger.Warn("retry_after long pause registered",
									"url", req.URL,
									"retry_after", ra,
									"threshold", c.opts.LongRetryAfterThreshold,
									"pause_until", now.Add(ra),
									"attempt", attempt+1,
								)
								c.opts.Limiter.PauseUntil(now.Add(ra))
								// Return an error for the current attempt; next call will block in Wait
								return nil, fmt.Errorf("httpfetch: long retry-after %v registered, status %d: %w",
									ra, resp.StatusCode, lastErr)
							}
						}
					}
				}

				t := time.NewTimer(sleepDur)
				select {
				case <-t.C:
					t.Stop()
				case <-ctx.Done():
					t.Stop()
					return nil, ctx.Err()
				}
			}
			continue
		}

		// Non-transient: return result (including 4xx, 5xx non-transient)
		return &ports.FetchResult{
			URL:          finalURL,
			OriginalURL:  originalURL,
			StatusCode:   resp.StatusCode,
			Headers:      collectHeaders(resp.Header),
			Body:         body,
			ETag:         etag,
			LastModified: lastModified,
			FetchedAt:    time.Now(),
			DurationMs:   durationMs,
			NotModified:  false,
		}, nil
	}

	// All attempts exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("httpfetch: all %d attempts failed: %w", c.backoff.Attempts, lastErr)
	}
	return nil, fmt.Errorf("httpfetch: all %d attempts failed", c.backoff.Attempts)
}

// collectHeaders copies response headers into a plain map[string]string.
// Only the first value per key is kept. Keys are kept in their original canonical form
// (e.g., "Content-Type", not "content-type") as returned by net/http's canonical header map.
func collectHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for key, vals := range h {
		if len(vals) > 0 {
			result[strings.ToLower(key)] = vals[0]
		}
	}
	return result
}
