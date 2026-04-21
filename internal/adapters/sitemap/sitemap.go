// Package sitemap is an adapter implementing the SitemapSource port.
// It can fetch a sitemap from HTTP or read it from a local file.
// Gzipped and plain XML bodies are both supported; the encoding is detected
// by magic bytes (0x1f 0x8b) at the start of the body.
package sitemap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bergmaia/site-research/internal/domain"
)

// Options configures a Source.
type Options struct {
	// URL is the sitemap URL to fetch via HTTP. Ignored if FromFile is non-empty.
	URL string
	// FromFile, when non-empty, is the path to a local sitemap file (.xml or .xml.gz).
	// Useful for offline tests and for the --from-file CLI flag.
	FromFile string
	// HTTPClient is the client used for HTTP fetches. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
	// UserAgent for the fetch request. Empty means Go default.
	UserAgent string
}

// Source implements domain.ports.SitemapSource.
type Source struct {
	opts Options
}

// New returns a Source wired with the given options.
func New(opts Options) *Source { return &Source{opts: opts} }

// Fetch retrieves the sitemap (from file if Options.FromFile is set, otherwise via HTTP),
// handles gzip transparently, parses the XML, and returns all <url> entries.
// No filtering is applied here — callers (e.g. the discover application service) filter
// by scope after canonicalization.
func (s *Source) Fetch(ctx context.Context) ([]domain.SitemapEntry, error) {
	var body []byte

	switch {
	case s.opts.FromFile != "":
		data, err := os.ReadFile(s.opts.FromFile)
		if err != nil {
			return nil, fmt.Errorf("sitemap: read file %s: %w", s.opts.FromFile, err)
		}
		body = data

	case s.opts.URL != "":
		client := s.opts.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.opts.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("sitemap: fetch %s: %w", s.opts.URL, err)
		}

		if s.opts.UserAgent != "" {
			req.Header.Set("User-Agent", s.opts.UserAgent)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("sitemap: fetch %s: %w", s.opts.URL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("sitemap: unexpected status %d from %s", resp.StatusCode, s.opts.URL)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("sitemap: fetch %s: %w", s.opts.URL, err)
		}
		body = data

	default:
		return nil, errors.New("sitemap: URL or FromFile must be set")
	}

	return parseSitemap(body)
}
