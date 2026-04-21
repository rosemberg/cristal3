package httpfetch_test

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
)

// TestRetryClassifier_Statuses verifies which HTTP status codes trigger retries.
func TestRetryClassifier_Statuses(t *testing.T) {
	rc := httpfetch.RetryClassifier{}
	tests := []struct {
		code    int
		want    bool
		comment string
	}{
		{200, false, "OK"},
		{204, false, "No Content"},
		{301, false, "Moved Permanently"},
		{400, false, "Bad Request"},
		{401, false, "Unauthorized"},
		{403, false, "Forbidden"},
		{404, false, "Not Found"},
		{429, true, "Too Many Requests"},
		{500, true, "Internal Server Error"},
		{502, true, "Bad Gateway"},
		{503, true, "Service Unavailable"},
		{504, true, "Gateway Timeout"},
	}
	for _, tt := range tests {
		got := rc.IsTransient(tt.code, nil)
		if got != tt.want {
			t.Errorf("IsTransient(%d [%s]) = %v, want %v", tt.code, tt.comment, got, tt.want)
		}
	}
}

// mockNetErr is a mock implementing net.Error for testing.
type mockNetErr struct {
	timeout   bool
	temporary bool
}

func (m mockNetErr) Error() string   { return "mock net error" }
func (m mockNetErr) Timeout() bool   { return m.timeout }
func (m mockNetErr) Temporary() bool { return m.temporary }

// TestRetryClassifier_Errors verifies error-based transient detection.
func TestRetryClassifier_Errors(t *testing.T) {
	rc := httpfetch.RetryClassifier{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"plain error", errors.New("some unexpected error"), false},
		{"timeout net.Error", mockNetErr{timeout: true}, true},
		{"non-timeout net.Error", mockNetErr{timeout: false}, false},
		{
			"net.OpError ECONNREFUSED",
			&net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: syscall.ECONNREFUSED,
			},
			true,
		},
	}
	for _, tt := range tests {
		got := rc.IsTransient(0, tt.err)
		if got != tt.want {
			t.Errorf("IsTransient(0, %q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestBackoffParams_Delay verifies that delay values are within expected bounds.
func TestBackoffParams_Delay(t *testing.T) {
	p := httpfetch.BackoffParams{
		Base:     500 * time.Millisecond,
		Factor:   2.0,
		Ceil:     10 * time.Second,
		Attempts: 3,
	}
	rng := rand.New(rand.NewSource(12345))

	// attempt=0: window = min(10s, 500ms * 2^0) = 500ms; delay in [0, 500ms]
	const samples = 100
	var sum0 float64
	for i := 0; i < samples; i++ {
		d := p.Delay(0, rng)
		if d < 0 || d > 500*time.Millisecond {
			t.Errorf("attempt=0 delay %v out of [0, 500ms]", d)
		}
		sum0 += float64(d)
	}
	mean0 := sum0 / float64(samples)
	t.Logf("attempt=0 mean delay: %v", time.Duration(mean0))
	// mean should be roughly 250ms (half of 500ms for uniform full-jitter)
	if time.Duration(mean0) < 50*time.Millisecond || time.Duration(mean0) > 450*time.Millisecond {
		t.Errorf("attempt=0 mean delay %v out of sanity range [50ms, 450ms]", time.Duration(mean0))
	}

	// attempt=5: window = min(10s, 500ms * 2^5) = min(10s, 16s) = 10s; delay in [0, 10s]
	var sum5 float64
	for i := 0; i < samples; i++ {
		d := p.Delay(5, rng)
		if d < 0 || d > 10*time.Second {
			t.Errorf("attempt=5 delay %v out of [0, 10s]", d)
		}
		sum5 += float64(d)
	}
	mean5 := sum5 / float64(samples)
	t.Logf("attempt=5 mean delay: %v", time.Duration(mean5))
	// mean should be roughly 5s (half of 10s)
	if time.Duration(mean5) < 1*time.Second || time.Duration(mean5) > 9*time.Second {
		t.Errorf("attempt=5 mean delay %v out of sanity range [1s, 9s]", time.Duration(mean5))
	}
}

// TestBackoffParams_WithDefaults verifies that zero struct gets canonical defaults.
func TestBackoffParams_WithDefaults(t *testing.T) {
	p := httpfetch.BackoffParams{}.WithDefaults()
	if p.Base != 500*time.Millisecond {
		t.Errorf("Base = %v, want 500ms", p.Base)
	}
	if p.Factor != 2.0 {
		t.Errorf("Factor = %v, want 2.0", p.Factor)
	}
	if p.Ceil != 10*time.Second {
		t.Errorf("Ceil = %v, want 10s", p.Ceil)
	}
	if p.Attempts != 3 {
		t.Errorf("Attempts = %v, want 3", p.Attempts)
	}
}

// TestParseRetryAfter verifies parsing of integer-seconds and HTTP-date forms.
func TestParseRetryAfter(t *testing.T) {
	now := time.Now()

	futureDate := now.Add(2 * time.Second).UTC().Format(http.TimeFormat)
	pastDate := now.Add(-5 * time.Second).UTC().Format(http.TimeFormat)

	tests := []struct {
		value   string
		wantOK  bool
		wantMin time.Duration
		wantMax time.Duration
	}{
		{"0", true, 0, 0},
		{"30", true, 30 * time.Second, 30 * time.Second},
		{futureDate, true, 1 * time.Second, 3 * time.Second},
		{pastDate, true, 0, 0},
		{"not-a-date-or-int", false, 0, 0},
	}

	for _, tt := range tests {
		d, ok := httpfetch.ParseRetryAfter(tt.value, now)
		if ok != tt.wantOK {
			t.Errorf("ParseRetryAfter(%q) ok=%v, want %v", tt.value, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if d < tt.wantMin || d > tt.wantMax {
			t.Errorf("ParseRetryAfter(%q) = %v, want in [%v, %v]", tt.value, d, tt.wantMin, tt.wantMax)
		}
	}
}
