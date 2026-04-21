package domain

import "errors"

// Sentinel errors raised by domain-level logic. Adapters may wrap these.
var (
	ErrPageNotFound       = errors.New("domain: page not found")
	ErrInvalidURL         = errors.New("domain: invalid URL")
	ErrOutOfScope         = errors.New("domain: URL out of scope")
	ErrBlockedByRobotsTxt = errors.New("domain: blocked by robots.txt")
	ErrNotFoundInSitemap  = errors.New("domain: URL not found in sitemap")
)
