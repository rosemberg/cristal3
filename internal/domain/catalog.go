package domain

import "time"

// Catalog is the consolidated catalog.json produced by build-catalog.
type Catalog struct {
	GeneratedAt   time.Time      `json:"generated_at"`
	RootURL       string         `json:"root_url"`
	SchemaVersion int            `json:"schema_version"`
	Stats         CatalogStats   `json:"stats"`
	Entries       []CatalogEntry `json:"entries"`
}

// CatalogStats is the aggregate statistics block inside catalog.json.
// JSON marshals int-keyed maps with string keys, matching the BRIEF example
// ("by_depth": {"2": 17, "3": 80, ...}).
type CatalogStats struct {
	TotalPages int              `json:"total_pages"`
	ByDepth    map[int]int      `json:"by_depth"`
	ByPageType map[PageType]int `json:"by_page_type"`
}

// CatalogEntry is the compact per-page row in catalog.json.
type CatalogEntry struct {
	Path                  string   `json:"path"`
	URL                   string   `json:"url"`
	Title                 string   `json:"title"`
	Depth                 int      `json:"depth"`
	Parent                string   `json:"parent"`
	Section               string   `json:"section"`
	PageType              PageType `json:"page_type"`
	HasSubstantiveContent bool     `json:"has_substantive_content"`
	MiniSummary           string   `json:"mini_summary"`
	ChildCount            int      `json:"child_count"`
	HasDocs               bool     `json:"has_docs"`
	ContentDate           *string  `json:"content_date"`
}
