package httpfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/temoto/robotstxt"
)

// RobotsFetcher fetches /robots.txt for a host. Injected for testability.
type RobotsFetcher func(ctx context.Context, host string) ([]byte, error)

// RobotsCache lazily fetches and caches /robots.txt per host and evaluates
// Allowed(url) against the configured User-Agent.
// Safe for concurrent use.
type RobotsCache struct {
	userAgent string
	fetcher   RobotsFetcher
	mu        sync.Mutex
	byHost    map[string]*robotstxt.RobotsData
}

// defaultRobotsFetcher constructs a RobotsFetcher that performs a real HTTP GET.
func defaultRobotsFetcher() RobotsFetcher {
	return func(ctx context.Context, host string) ([]byte, error) {
		robotsURL := fmt.Sprintf("https://%s/robots.txt", host)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return []byte{}, nil
		}
		return io.ReadAll(resp.Body)
	}
}

// NewRobotsCache returns a RobotsCache using the given user-agent for matching.
// If fetcher is nil, a default http.Get fetcher is used against scheme "https".
func NewRobotsCache(userAgent string, fetcher RobotsFetcher) *RobotsCache {
	if fetcher == nil {
		fetcher = defaultRobotsFetcher()
	}
	return &RobotsCache{
		userAgent: userAgent,
		fetcher:   fetcher,
		byHost:    make(map[string]*robotstxt.RobotsData),
	}
}

// Allowed returns true if the URL is permitted by the host's robots.txt for
// the configured User-Agent. Failure to fetch robots.txt (e.g., 404) is treated
// as "everything allowed" per sitemaps.org/RFC 9309 guidance.
//
// On transient fetch errors, this method returns (true, err): it allows the
// request by default but propagates the error so the caller can log it.
func (r *RobotsCache) Allowed(ctx context.Context, rawURL string) (bool, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("httpfetch: invalid URL %q: %w", rawURL, err)
	}
	host := u.Host

	r.mu.Lock()
	data, cached := r.byHost[host]
	r.mu.Unlock()

	if !cached {
		body, fetchErr := r.fetcher(ctx, host)
		if fetchErr != nil {
			// Transient or permanent fetch failure: allow by default, propagate error.
			return true, fetchErr
		}
		parsed, parseErr := robotstxt.FromBytes(body)
		if parseErr != nil {
			// Invalid robots.txt: treat as everything allowed.
			parsed, _ = robotstxt.FromBytes([]byte{})
		}
		r.mu.Lock()
		// Double-check in case another goroutine already stored it
		if _, exists := r.byHost[host]; !exists {
			r.byHost[host] = parsed
		}
		data = r.byHost[host]
		r.mu.Unlock()
	}

	group := data.FindGroup(r.userAgent)
	allowed := group.Test(u.Path)
	return allowed, nil
}
