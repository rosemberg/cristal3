package domain

import "time"

// CrawlReport summarizes a crawl run (initial or incremental).
type CrawlReport struct {
	StartedAt           time.Time
	CompletedAt         time.Time
	SitemapTotal        int
	OrphansFound        int
	TotalCrawled        int
	NewPages            int
	UpdatedPages        int
	UnchangedPages      int
	StalePages          int
	RemovedPages        int
	FailedURLs          []FailedURL
	HTTPStatusHistogram map[int]int
}

// SummarizeReport summarizes a summarize run: LLM calls, cost, failures.
type SummarizeReport struct {
	StartedAt     time.Time
	CompletedAt   time.Time
	TotalPages    int
	Generated     int
	Skipped       int
	Failed        int
	TokensInput   int64
	TokensOutput  int64
	EstimatedCost float64 // USD; heuristic per provider/model
	Provider      string
	Model         string
	FailedReasons map[string]int // error code → count
}

// FailedURL records a URL that failed during crawl or summarize.
type FailedURL struct {
	URL    string
	Error  string
	Status int // HTTP status if applicable; 0 otherwise
}
