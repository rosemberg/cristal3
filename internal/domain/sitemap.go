package domain

import "time"

// SitemapEntry is a single <url> entry from the sitemap (sitemaps.org protocol).
// LastMod is zero-valued when the sitemap did not provide <lastmod>.
type SitemapEntry struct {
	Loc     string
	LastMod time.Time
}
