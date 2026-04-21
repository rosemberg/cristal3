package httpfetch

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryClassifier decides whether a response/error indicates a transient failure.
type RetryClassifier struct{}

// IsTransient returns true if the outcome should trigger a retry.
// Transient statuses: 429, 500, 502, 503, 504.
// Transient errors: net.Error.Timeout() || connection reset/refused.
func (RetryClassifier) IsTransient(statusCode int, err error) bool {
	if err != nil {
		return IsNetworkError(err)
	}
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// BackoffParams is full-jitter exponential backoff configuration.
type BackoffParams struct {
	Base     time.Duration // default 500ms when zero
	Factor   float64       // default 2.0 when zero
	Ceil     time.Duration // default 10s when zero
	Attempts int           // max total attempts; default 3 when zero
}

// WithDefaults returns a copy with zero fields replaced by defaults.
func (p BackoffParams) WithDefaults() BackoffParams {
	if p.Base == 0 {
		p.Base = 500 * time.Millisecond
	}
	if p.Factor == 0 {
		p.Factor = 2.0
	}
	if p.Ceil == 0 {
		p.Ceil = 10 * time.Second
	}
	if p.Attempts == 0 {
		p.Attempts = 3
	}
	return p
}

// Delay returns the backoff delay for the given 0-indexed attempt number.
// Uses full jitter: rand.Float64() * min(Ceil, Base * Factor^attempt).
// Panics if rng is nil.
func (p BackoffParams) Delay(attempt int, rng *rand.Rand) time.Duration {
	if rng == nil {
		panic("httpfetch: BackoffParams.Delay called with nil rng")
	}
	base := float64(p.Base) * math.Pow(p.Factor, float64(attempt))
	ceil := float64(p.Ceil)
	window := math.Min(ceil, base)
	return time.Duration(rng.Float64() * window)
}

// ParseRetryAfter parses an HTTP Retry-After header value.
// Supports integer-seconds form and HTTP-date form (RFC 7231).
// Returns the delay from "now" until the retry time, and ok=true on success.
// Negative delays clamp to zero.
func ParseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)

	// Try integer seconds form first
	if n, err := strconv.Atoi(value); err == nil {
		d := time.Duration(n) * time.Second
		if d < 0 {
			d = 0
		}
		return d, true
	}

	// Try HTTP-date form (RFC 7231)
	if t, err := http.ParseTime(value); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}

	return 0, false
}

// IsNetworkError returns true if err is a net.Error Timeout, connection refused/reset,
// or wraps one. Exposed for testability.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Structural check: net.Error with Timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Structural check: *net.OpError for connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Fallback: substring check for common network error messages
	msg := err.Error()
	for _, substr := range []string{
		"connection reset",
		"connection refused",
		"no such host",
		"i/o timeout",
	} {
		if strings.Contains(strings.ToLower(msg), substr) {
			return true
		}
	}

	return false
}
