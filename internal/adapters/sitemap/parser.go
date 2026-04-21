package sitemap

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/bergmaia/site-research/internal/domain"
)

// xmlURLSet is the root element of a sitemaps.org 0.9 XML sitemap.
type xmlURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []xmlURL `xml:"url"`
}

// xmlURL is a single <url> entry in the sitemap.
type xmlURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// lastModFormats are the date/time formats tried in order when parsing <lastmod>.
var lastModFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05-07:00",
	"2006-01-02",
}

// parseSitemap detects gzip, decompresses if needed, and parses the XML urlset.
func parseSitemap(body []byte) ([]domain.SitemapEntry, error) {
	// Detect gzip by magic bytes 0x1f 0x8b.
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		gr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("sitemap: gzip open: %w", err)
		}
		decompressed, err := io.ReadAll(gr)
		if err != nil {
			return nil, fmt.Errorf("sitemap: gzip read: %w", err)
		}
		if err := gr.Close(); err != nil {
			return nil, fmt.Errorf("sitemap: gzip close: %w", err)
		}
		body = decompressed
	}

	var us xmlURLSet
	if err := xml.Unmarshal(body, &us); err != nil {
		return nil, fmt.Errorf("sitemap: xml parse: %w", err)
	}

	entries := make([]domain.SitemapEntry, 0, len(us.URLs))
	for _, u := range us.URLs {
		if u.Loc == "" {
			continue
		}

		entry := domain.SitemapEntry{Loc: u.Loc}

		if u.LastMod != "" {
			for _, format := range lastModFormats {
				t, err := time.Parse(format, u.LastMod)
				if err == nil {
					entry.LastMod = t
					break
				}
			}
			// If parse fails for all formats, LastMod remains zero — not an error.
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
