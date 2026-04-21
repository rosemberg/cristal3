package httpfetch_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/bergmaia/site-research/internal/adapters/httpfetch"
)

// TestRobotsCache_Allowed_Default verifies that an empty robots.txt allows everything.
func TestRobotsCache_Allowed_Default(t *testing.T) {
	fetcher := func(_ context.Context, _ string) ([]byte, error) {
		return []byte(""), nil
	}
	cache := httpfetch.NewRobotsCache("TestBot", fetcher)

	tests := []string{
		"https://example.com/",
		"https://example.com/public/page",
		"https://example.com/private/secret",
	}
	for _, rawURL := range tests {
		allowed, err := cache.Allowed(context.Background(), rawURL)
		if err != nil {
			t.Errorf("Allowed(%q) unexpected error: %v", rawURL, err)
		}
		if !allowed {
			t.Errorf("Allowed(%q) = false with empty robots.txt, want true", rawURL)
		}
	}
}

// TestRobotsCache_Allowed_Disallow verifies that /private/ paths are blocked.
func TestRobotsCache_Allowed_Disallow(t *testing.T) {
	robotsTxt := `User-agent: *
Disallow: /private/
`
	fetcher := func(_ context.Context, _ string) ([]byte, error) {
		return []byte(robotsTxt), nil
	}
	cache := httpfetch.NewRobotsCache("TestBot", fetcher)

	tests := []struct {
		path    string
		want    bool
	}{
		{"/public/x", true},
		{"/private/x", false},
		{"/private/", false},
		{"/other", true},
	}
	for _, tt := range tests {
		rawURL := "https://example.com" + tt.path
		allowed, err := cache.Allowed(context.Background(), rawURL)
		if err != nil {
			t.Errorf("Allowed(%q) unexpected error: %v", rawURL, err)
			continue
		}
		if allowed != tt.want {
			t.Errorf("Allowed(%q) = %v, want %v", rawURL, allowed, tt.want)
		}
	}
}

// TestRobotsCache_PerHost_Cached verifies that the fetcher is called exactly once per host.
func TestRobotsCache_PerHost_Cached(t *testing.T) {
	var callCount atomic.Int32
	fetcher := func(_ context.Context, _ string) ([]byte, error) {
		callCount.Add(1)
		return []byte(""), nil
	}
	cache := httpfetch.NewRobotsCache("TestBot", fetcher)

	host := "https://example.com"
	for i := 0; i < 2; i++ {
		_, err := cache.Allowed(context.Background(), host+"/page")
		if err != nil {
			t.Fatalf("Allowed call %d error: %v", i, err)
		}
	}

	if got := callCount.Load(); got != 1 {
		t.Errorf("fetcher called %d times, want exactly 1", got)
	}
}

// TestRobotsCache_Different_Hosts verifies that each host gets its own robots.txt fetch.
func TestRobotsCache_Different_Hosts(t *testing.T) {
	var callCount atomic.Int32
	fetcher := func(_ context.Context, host string) ([]byte, error) {
		callCount.Add(1)
		return []byte(""), nil
	}
	cache := httpfetch.NewRobotsCache("TestBot", fetcher)

	hosts := []string{
		"https://host-a.example.com/page",
		"https://host-b.example.com/page",
	}
	for _, u := range hosts {
		_, err := cache.Allowed(context.Background(), u)
		if err != nil {
			t.Fatalf("Allowed(%q) error: %v", u, err)
		}
	}

	if got := callCount.Load(); got != 2 {
		t.Errorf("fetcher called %d times for 2 different hosts, want 2", got)
	}
}

// TestRobotsCache_FetcherError verifies that fetch errors allow access by default
// and propagate the error to the caller (allowing caller to log it).
func TestRobotsCache_FetcherError(t *testing.T) {
	fetchErr := errors.New("network timeout")
	fetcher := func(_ context.Context, _ string) ([]byte, error) {
		return nil, fetchErr
	}
	cache := httpfetch.NewRobotsCache("TestBot", fetcher)

	// On fetch error, Allowed returns (true, err): allows by default, exposes error.
	allowed, err := cache.Allowed(context.Background(), "https://example.com/some/path")
	if !allowed {
		t.Error("Allowed returned false on fetcher error, want true (allow by default)")
	}
	if err == nil {
		t.Error("Allowed returned nil error on fetcher error, want propagated error")
	}
	if !errors.Is(err, fetchErr) {
		t.Errorf("Allowed returned error %v, want to wrap %v", err, fetchErr)
	}
}
