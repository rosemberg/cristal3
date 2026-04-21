package httpfetch_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
	"github.com/bergmaia/site-research/internal/domain/ports"
	"github.com/bergmaia/site-research/internal/logging"
)

// newTestClient builds a Client suitable for fast tests:
// 100 req/s limiter with no jitter, small backoff (10ms base, 3 attempts).
func newTestClient(t *testing.T, opts httpfetch.Options) *httpfetch.Client {
	t.Helper()
	if opts.Limiter == nil {
		opts.Limiter = httpfetch.NewLimiter(100, 0)
	}
	if opts.Backoff == (httpfetch.BackoffParams{}) {
		opts.Backoff = httpfetch.BackoffParams{
			Base:     10 * time.Millisecond,
			Factor:   2.0,
			Ceil:     50 * time.Millisecond,
			Attempts: 3,
		}
	}
	c, err := httpfetch.New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestClient_Success200 verifies a 200 response returns the body and correct fields.
func TestClient_Success200(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello")
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{UserAgent: "test-agent"})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if string(result.Body) != "hello" {
		t.Errorf("Body = %q, want %q", result.Body, "hello")
	}
	if result.OriginalURL != srv.URL {
		t.Errorf("OriginalURL = %q, want %q", result.OriginalURL, srv.URL)
	}
	if result.NotModified {
		t.Error("NotModified should be false for 200")
	}
	if result.DurationMs <= 0 {
		t.Errorf("DurationMs = %d, want > 0", result.DurationMs)
	}
}

// TestClient_304NotModified verifies 304 returns NotModified=true.
func TestClient_304NotModified(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "abc" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "body")
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{
		URL:         srv.URL,
		IfNoneMatch: "abc",
	})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if !result.NotModified {
		t.Error("NotModified should be true for 304")
	}
	if result.StatusCode != 304 {
		t.Errorf("StatusCode = %d, want 304", result.StatusCode)
	}
}

// TestClient_SetsUserAgent verifies the User-Agent header is sent.
func TestClient_SetsUserAgent(t *testing.T) {
	t.Parallel()

	var capturedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{UserAgent: "MyCrawler/1.0"})
	_, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if capturedUA != "MyCrawler/1.0" {
		t.Errorf("User-Agent = %q, want %q", capturedUA, "MyCrawler/1.0")
	}
}

// TestClient_SetsCacheControl verifies Cache-Control: no-cache is always sent (RNF-06).
func TestClient_SetsCacheControl(t *testing.T) {
	t.Parallel()

	var capturedCC string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCC = r.Header.Get("Cache-Control")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{})
	_, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if capturedCC != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", capturedCC, "no-cache")
	}
}

// TestClient_IfModifiedSince verifies the If-Modified-Since header is set correctly.
func TestClient_IfModifiedSince(t *testing.T) {
	t.Parallel()

	var capturedIMS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIMS = r.Header.Get("If-Modified-Since")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	modTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	c := newTestClient(t, httpfetch.Options{})
	_, err := c.Fetch(context.Background(), ports.FetchRequest{
		URL:             srv.URL,
		IfModifiedSince: modTime,
	})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	expected := modTime.UTC().Format(http.TimeFormat)
	if capturedIMS != expected {
		t.Errorf("If-Modified-Since = %q, want %q", capturedIMS, expected)
	}
}

// TestClient_NonTransient4xxNoRetry verifies 404 returns immediately with exactly 1 call.
func TestClient_NonTransient4xxNoRetry(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.StatusCode)
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Errorf("server called %d times, want exactly 1", n)
	}
}

// TestClient_Transient503RetriesAndSucceeds verifies 503 is retried and 200 eventually returned.
func TestClient_Transient503RetriesAndSucceeds(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: 5,
		},
	})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if n := atomic.LoadInt32(&calls); n != 3 {
		t.Errorf("server called %d times, want 3", n)
	}
}

// TestClient_MaxRetriesExceeded verifies that when server always returns 503, error is returned.
func TestClient_MaxRetriesExceeded(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	attempts := 3
	c := newTestClient(t, httpfetch.Options{
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: attempts,
		},
	})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("result should be nil on max retries exceeded, got %+v", result)
	}
	if n := atomic.LoadInt32(&calls); int(n) != attempts {
		t.Errorf("server called %d times, want %d (Backoff.Attempts)", n, attempts)
	}
}

// TestClient_RetryAfterSeconds_Short verifies Retry-After integer seconds is honored for short delays.
func TestClient_RetryAfterSeconds_Short(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{
		HonorRetryAfter:         true,
		LongRetryAfterThreshold: 5 * time.Second,
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: 3,
		},
	})

	start := time.Now()
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Errorf("server called %d times, want 2", n)
	}
	// Retry-After: 1 second → should sleep ~1s
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed %v, want >= 900ms (Retry-After: 1)", elapsed)
	}
}

// TestClient_RetryAfterHTTPDate_Short verifies Retry-After HTTP-date form is honored.
// We use an HTTP date 2s in the future from within the handler (ensuring ≥1s sleep
// regardless of HTTP-date second truncation), then assert elapsed >= 900ms.
func TestClient_RetryAfterHTTPDate_Short(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			// Add 2s from now to guarantee ≥1s remains after HTTP-date second truncation
			future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
			w.Header().Set("Retry-After", future)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{
		HonorRetryAfter:         true,
		LongRetryAfterThreshold: 10 * time.Second,
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: 5,
		},
	})

	start := time.Now()
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Errorf("server called %d times, want 2", n)
	}
	// Retry-After date is 2s from handler invocation; even with second truncation
	// the remaining sleep should be >= ~1s, so total elapsed >= 900ms.
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed %v, want >= 900ms (Retry-After: HTTP-date 2s)", elapsed)
	}
}

// TestClient_RetryAfterLong_TriggersPauseAndWarn verifies that a long Retry-After
// registers a limiter pause and emits a warning log.
//
// Design choice: when Retry-After > LongRetryAfterThreshold, the current Fetch call
// returns an error immediately after registering the global limiter pause. Subsequent
// Fetch calls will block in Limiter.Wait until the pause window expires.
// We use Retry-After: 800ms with threshold 500ms to keep the test fast.
func TestClient_RetryAfterLong_TriggersPauseAndWarn(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// 800ms > 500ms threshold → triggers long path
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	logger := logging.New(logging.Config{Output: &buf, Level: "warn", Format: "json"})

	limiter := httpfetch.NewLimiter(100, 0)
	c, err := httpfetch.New(httpfetch.Options{
		Limiter:                 limiter,
		Logger:                  logger,
		HonorRetryAfter:         true,
		LongRetryAfterThreshold: 500 * time.Millisecond, // 1s > 500ms → long path
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: 3,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fetchStart := time.Now()
	result, fetchErr := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})

	// The current Fetch call should return an error (long Retry-After path)
	if fetchErr == nil {
		t.Error("expected error from long Retry-After path, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result from long Retry-After, got %+v", result)
	}

	// Verify the warning log was emitted
	logOutput := buf.String()
	if !strings.Contains(logOutput, "retry_after") {
		t.Errorf("expected log to contain 'retry_after', got: %s", logOutput)
	}

	// Verify the limiter pause was registered
	pausedUntil := limiter.PausedUntil()
	expectedPause := fetchStart.Add(1 * time.Second)
	// Allow ±200ms slack
	slack := 200 * time.Millisecond
	if pausedUntil.Before(expectedPause.Add(-slack)) || pausedUntil.After(expectedPause.Add(slack)) {
		t.Errorf("PausedUntil = %v, want ≈ %v (±%v)", pausedUntil, expectedPause, slack)
	}
}

// TestClient_RobotsDisallow verifies that a disallowed URL returns ErrBlockedByRobots.
func TestClient_RobotsDisallow(t *testing.T) {
	t.Parallel()

	var targetCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&targetCalls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	robotsFetcher := httpfetch.RobotsFetcher(func(ctx context.Context, host string) ([]byte, error) {
		return []byte("User-agent: *\nDisallow: /private/\n"), nil
	})
	robots := httpfetch.NewRobotsCache("test-agent", robotsFetcher)

	c := newTestClient(t, httpfetch.Options{
		UserAgent:        "test-agent",
		Robots:           robots,
		RespectRobotsTxt: true,
	})

	// Fetch a disallowed path
	_, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL + "/private/x"})
	if err == nil {
		t.Fatal("expected error for disallowed robots path, got nil")
	}
	if !errors.Is(err, httpfetch.ErrBlockedByRobots) {
		t.Errorf("error = %v, want ErrBlockedByRobots", err)
	}
	if n := atomic.LoadInt32(&targetCalls); n != 0 {
		t.Errorf("target server got %d calls, want 0 (blocked before HTTP)", n)
	}
}

// TestClient_RobotsAllow verifies allowed robots paths proceed normally.
func TestClient_RobotsAllow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "allowed")
	}))
	defer srv.Close()

	robotsFetcher := httpfetch.RobotsFetcher(func(ctx context.Context, host string) ([]byte, error) {
		return []byte("User-agent: *\nAllow: /\n"), nil
	})
	robots := httpfetch.NewRobotsCache("test-agent", robotsFetcher)

	c := newTestClient(t, httpfetch.Options{
		UserAgent:        "test-agent",
		Robots:           robots,
		RespectRobotsTxt: true,
	})

	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL + "/page"})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
}

// TestClient_RobotsDisabled_NoRobotsCall verifies that RespectRobotsTxt=false bypasses robots.
func TestClient_RobotsDisabled_NoRobotsCall(t *testing.T) {
	t.Parallel()

	var robotsCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	robotsFetcher := httpfetch.RobotsFetcher(func(ctx context.Context, host string) ([]byte, error) {
		atomic.AddInt32(&robotsCalls, 1)
		return []byte("User-agent: *\nDisallow: /\n"), nil
	})
	robots := httpfetch.NewRobotsCache("test-agent", robotsFetcher)

	c := newTestClient(t, httpfetch.Options{
		UserAgent:        "test-agent",
		Robots:           robots,
		RespectRobotsTxt: false, // robots disabled
	})

	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL + "/blocked"})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	// The robots fetcher should never have been called since RespectRobotsTxt=false
	if n := atomic.LoadInt32(&robotsCalls); n != 0 {
		t.Errorf("robotsFetcher called %d times, want 0 (RespectRobotsTxt=false)", n)
	}
}

// TestClient_RateLimitRespected verifies that 5 req/s limiter enforces ~200ms between calls.
func TestClient_RateLimitRespected(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 5 req/s → 200ms base interval, no jitter
	limiter := httpfetch.NewLimiter(5, 0)
	c, err := httpfetch.New(httpfetch.Options{
		Limiter: limiter,
		Backoff: httpfetch.BackoffParams{
			Base:     5 * time.Millisecond,
			Factor:   1.0,
			Ceil:     10 * time.Millisecond,
			Attempts: 3,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	start := time.Now()
	for i := 0; i < 2; i++ {
		_, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL})
		if err != nil {
			t.Fatalf("Fetch[%d] error: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// 2 calls at 5 req/s → at least 200ms between them
	if elapsed < 200*time.Millisecond {
		t.Errorf("2 calls elapsed %v, want >= 200ms (5 req/s limiter)", elapsed)
	}
}

// TestClient_ContextCancel verifies that a canceled context returns quickly.
func TestClient_ContextCancel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // server hangs
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately
	cancel()

	start := time.Now()
	result, err := c.Fetch(ctx, ports.FetchRequest{URL: srv.URL})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if result != nil {
		t.Errorf("result should be nil on canceled context")
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("cancel took %v, want < 50ms", elapsed)
	}
}

// TestClient_RedirectRecordsFinalURL verifies that FetchResult.URL captures the final URL after redirects.
func TestClient_RedirectRecordsFinalURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			http.Redirect(w, r, "/b", http.StatusFound)
		case "/b":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "hi")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := newTestClient(t, httpfetch.Options{})
	result, err := c.Fetch(context.Background(), ports.FetchRequest{URL: srv.URL + "/a"})
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	if result.OriginalURL != srv.URL+"/a" {
		t.Errorf("OriginalURL = %q, want %q", result.OriginalURL, srv.URL+"/a")
	}
	if !strings.HasSuffix(result.URL, "/b") {
		t.Errorf("URL = %q, want to end with /b (final URL after redirect)", result.URL)
	}
	if string(result.Body) != "hi" {
		t.Errorf("Body = %q, want %q", result.Body, "hi")
	}
}
