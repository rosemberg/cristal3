// Package domain — continued from doc.go. This file declares the Page entity
// and its subtypes matching schema_version 2 of _index.json.
package domain

import "time"

// Page is the schema v2 representation of a crawled page.
type Page struct {
	Schema                string           `json:"$schema"`
	SchemaVersion         int              `json:"schema_version"`
	URL                   string           `json:"url"`
	CanonicalURL          string           `json:"canonical_url"`
	Title                 string           `json:"title"`
	Description           string           `json:"description"`
	Section               string           `json:"section"`
	Breadcrumb            []URLRef         `json:"breadcrumb"`
	PathTitles            []string         `json:"path_titles"`
	Lang                  string           `json:"lang"`
	PageType              PageType         `json:"page_type"`
	HasSubstantiveContent bool             `json:"has_substantive_content"`
	Content               Content          `json:"content"`
	MiniSummary           MiniSummary      `json:"mini_summary"`
	Dates                 Dates            `json:"dates"`
	Metadata              Metadata         `json:"metadata"`
	Links                 Links            `json:"links"`
	Documents             []Document       `json:"documents"`
	Tags                  []string         `json:"tags"`
}

// PageType is the classification produced by RF-05.
type PageType string

const (
	PageTypeLanding  PageType = "landing"
	PageTypeArticle  PageType = "article"
	PageTypeListing  PageType = "listing"
	PageTypeRedirect PageType = "redirect"
	PageTypeEmpty    PageType = "empty"
)

// DiscoverySource indicates how a page entered the catalog.
type DiscoverySource string

const (
	DiscoveredViaSitemap DiscoverySource = "sitemap"
	DiscoveredViaLink    DiscoverySource = "link"
)

// URLRef is a (title, url) tuple used in breadcrumbs and link lists.
type URLRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ChildLink extends URLRef with the local path of the child page's _index.json.
type ChildLink struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	LocalPath string `json:"local_path"`
}

// Links groups hierarchical children, other scope-internal links, and external links.
type Links struct {
	Children []ChildLink `json:"children"`
	Internal []URLRef    `json:"internal"`
	External []URLRef    `json:"external"`
}

// Content holds extracted textual content and derived hashes / signals.
type Content struct {
	Summary           string   `json:"summary"`
	FullText          string   `json:"full_text"`
	FullTextHash      string   `json:"full_text_hash"`
	ContentHash       string   `json:"content_hash"`
	ContentLength     int      `json:"content_length"`
	KeywordsExtracted []string `json:"keywords_extracted"`
}

// Metadata captures crawl-time and schema-level metadata for a page.
type Metadata struct {
	Depth              int             `json:"depth"`
	ExtractedAt        time.Time       `json:"extracted_at"`
	LastModified       string          `json:"last_modified"`
	ETag               string          `json:"etag"`
	HTTPStatus         int             `json:"http_status"`
	ContentType        string          `json:"content_type"`
	ParentURL          string          `json:"parent_url"`
	RedirectedFrom     *string         `json:"redirected_from"`
	CanonicalOf        *string         `json:"canonical_of"`
	FetchDurationMs    int64           `json:"fetch_duration_ms"`
	CrawlerVersion     string          `json:"crawler_version"`
	DiscoveredVia      DiscoverySource `json:"discovered_via"`
	IsPloneCopy        bool            `json:"is_plone_copy"`
	StaleSince         *time.Time      `json:"stale_since"`
	ExtractionWarnings []string        `json:"extraction_warnings"`
}

// MiniSummary is the LLM-generated 1-2 line description for routing.
type MiniSummary struct {
	Text        string    `json:"text"`
	GeneratedAt time.Time `json:"generated_at"`
	Model       string    `json:"model"`
	SourceHash  string    `json:"source_hash"`
	Skipped     *string   `json:"skipped"` // null when generated successfully; string reason when skipped
}

// Dates holds content-level dates extracted from the page.
type Dates struct {
	ContentDate   *string    `json:"content_date"`    // ISO-8601 date or null
	PageUpdatedAt *time.Time `json:"page_updated_at"` // null when not available
}

// Document is a detected attachment (PDF, CSV, XLSX, DOCX, ODS). No download in Phase 1.
type Document struct {
	Title        string `json:"title"`
	URL          string `json:"url"`
	Type         string `json:"type"`
	SizeBytes    *int64 `json:"size_bytes"`
	DetectedFrom string `json:"detected_from"`
	ContextText  string `json:"context_text"`
}
