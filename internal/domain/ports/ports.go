// Package ports declares the interfaces implemented by adapters.
package ports

import (
	"context"
	"time"

	"github.com/bergmaia/site-research/internal/domain"
)

// URLCanonicalizer applies RF-03 canonicalization.
// excluded=true when the URL matches Plone exclusion patterns (@@ or ++theme++).
type URLCanonicalizer interface {
	Canonicalize(raw string) (canonical string, excluded bool, err error)
}

// SitemapSource fetches and parses the site's sitemap.
type SitemapSource interface {
	Fetch(ctx context.Context) ([]domain.SitemapEntry, error)
}

// FetchRequest is the input to Fetcher.Fetch.
type FetchRequest struct {
	URL             string
	IfNoneMatch     string    // ETag for If-None-Match
	IfModifiedSince time.Time // zero => no header
}

// FetchResult is the output of Fetcher.Fetch.
type FetchResult struct {
	URL          string // final URL after redirects
	OriginalURL  string // as requested, if different from URL
	StatusCode   int
	Headers      map[string]string
	Body         []byte
	ETag         string
	LastModified time.Time
	FetchedAt    time.Time
	DurationMs   int64
	NotModified  bool // true when server returned 304
}

// Fetcher performs rate-limited, retrying HTTP GETs, honouring robots.txt.
type Fetcher interface {
	Fetch(ctx context.Context, req FetchRequest) (*FetchResult, error)
}

// HTMLExtractor parses an HTML body into a partially-populated domain.Page
// (no page_type, no mini_summary, no derived hashes that require canonicalization).
type HTMLExtractor interface {
	Extract(ctx context.Context, url string, body []byte) (*domain.Page, error)
}

// PageClassifier assigns page_type to a Page based on its content and links.
type PageClassifier interface {
	Classify(p *domain.Page) domain.PageType
}

// PageStore persists and retrieves Page entities on the filesystem (_index.json tree).
type PageStore interface {
	Put(ctx context.Context, page *domain.Page) error
	Get(ctx context.Context, url string) (*domain.Page, error)
	Walk(ctx context.Context, fn func(p *domain.Page) error) error
	Delete(ctx context.Context, url string) error
}

// CatalogBuilder consolidates the page tree into a Catalog and writes catalog.json.
type CatalogBuilder interface {
	Build(ctx context.Context) (*domain.Catalog, error)
	WriteFile(ctx context.Context, catalog *domain.Catalog, path string) error
}

// SearchHit is a single result row returned by SearchIndex.Search.
type SearchHit struct {
	Path        string
	URL         string
	Title       string
	MiniSummary string
	Score       float64
	Section     string
}

// SearchIndex is the FTS index over the catalog.
type SearchIndex interface {
	Rebuild(ctx context.Context, catalog *domain.Catalog, pages []*domain.Page) error
	Search(ctx context.Context, query string, limit int) ([]SearchHit, error)
	Close() error
}

// GenerateRequest is the input to LLMProvider.Generate.
type GenerateRequest struct {
	System      string
	User        string
	MaxTokens   int
	Temperature float64
}

// GenerateResponse is the output of LLMProvider.Generate.
type GenerateResponse struct {
	Text         string
	TokensInput  int64
	TokensOutput int64
	Provider     string
	Model        string
}

// LLMProvider generates text from a prompt. Multiple providers are pluggable.
type LLMProvider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Name() string
	Model() string
}

// Clock is a mockable wall clock for tests.
type Clock interface {
	Now() time.Time
}
