package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/llm"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

// testConfig builds a minimal config pointing to a temp data dir.
func testConfig(t *testing.T, dataDir string) *config.Config {
	t.Helper()
	return &config.Config{
		Scope: config.ScopeConfig{
			SeedURL: "https://example.com/site",
			Prefix:  "https://example.com/site",
		},
		Storage: config.StorageConfig{
			DataDir:     dataDir,
			CatalogPath: dataDir + "/catalog.json",
			SQLitePath:  dataDir + "/catalog.sqlite",
		},
		LLM: config.LLMConfig{
			Provider:              "anthropic",
			Model:                 "claude-haiku-4-5",
			Concurrency:           2,
			RequestTimeoutSeconds: 60,
		},
	}
}

// testPage builds a simple Page for testing.
func testPage(url, title string, pageType domain.PageType, fullText, fullTextHash string) *domain.Page {
	return &domain.Page{
		SchemaVersion: 2,
		URL:           url,
		CanonicalURL:  url,
		Title:         title,
		Section:       "test-section",
		PathTitles:    []string{"Home", title},
		PageType:      pageType,
		Content: domain.Content{
			FullText:     fullText,
			FullTextHash: fullTextHash,
		},
	}
}

// seedPage writes a single page to the temp store.
func seedPage(t *testing.T, dataDir, scopePrefix string, page *domain.Page) {
	t.Helper()

	// Compute relative path from scope prefix.
	u := strings.TrimRight(page.URL, "/")
	prefix := strings.TrimRight(scopePrefix, "/")

	var segments []string
	if u != prefix {
		if !strings.HasPrefix(u, prefix+"/") {
			t.Fatalf("URL %q is out of scope %q", page.URL, scopePrefix)
		}
		rel := u[len(prefix)+1:]
		segments = strings.Split(rel, "/")
	}

	parts := make([]string, 0, len(segments)+2)
	parts = append(parts, dataDir)
	parts = append(parts, segments...)
	parts = append(parts, "_index.json")

	path := filepath.Join(parts...)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	data, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		t.Fatalf("marshal page %s: %v", page.URL, err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write page %s: %v", page.URL, err)
	}
}

// seedStore writes pages into a temporary fsstore directory.
func seedStore(t *testing.T, dataDir string, pages []*domain.Page) {
	t.Helper()
	for _, p := range pages {
		seedPage(t, dataDir, "https://example.com/site", p)
	}
}

// readStoredPage reads and unmarshal a page from the store.
func readStoredPage(t *testing.T, dataDir string, url string) *domain.Page {
	t.Helper()

	u := strings.TrimRight(url, "/")
	prefix := strings.TrimRight("https://example.com/site", "/")

	var segments []string
	if u != prefix {
		rel := u[len(prefix)+1:]
		segments = strings.Split(rel, "/")
	}

	parts := make([]string, 0, len(segments)+2)
	parts = append(parts, dataDir)
	parts = append(parts, segments...)
	parts = append(parts, "_index.json")

	path := filepath.Join(parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read page %s: %v", url, err)
	}
	var p domain.Page
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal page %s: %v", url, err)
	}
	return &p
}

func TestSummarize_BasicFlow(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	pages := []*domain.Page{
		testPage("https://example.com/site/page1", "Article 1", domain.PageTypeArticle, "Full text for article 1", "hash1"),
		testPage("https://example.com/site/page2", "Article 2", domain.PageTypeArticle, "Full text for article 2", "hash2"),
		testPage("https://example.com/site/page3", "Article 3", domain.PageTypeArticle, "Full text for article 3", "hash3"),
	}
	seedStore(t, dataDir, pages)

	mockResp := func(text string) llm.MockResponse {
		return llm.MockResponse{Response: &ports.GenerateResponse{
			Text: text, TokensInput: 50, TokensOutput: 20, Provider: "anthropic", Model: "claude-haiku-4-5",
		}}
	}
	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			mockResp("Summary for article 1"),
			mockResp("Summary for article 2"),
			mockResp("Summary for article 3"),
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 3 {
		t.Errorf("Generated = %d, want 3", report.Generated)
	}
	if report.Failed != 0 {
		t.Errorf("Failed = %d, want 0", report.Failed)
	}
	if report.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", report.Skipped)
	}
	if report.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", report.TotalPages)
	}

	// Verify stored pages have summaries.
	for i, page := range pages {
		stored := readStoredPage(t, dataDir, page.URL)
		if stored.MiniSummary.Text == "" {
			t.Errorf("page %d (%s): MiniSummary.Text is empty", i, page.URL)
		}
		if stored.MiniSummary.SourceHash != page.Content.FullTextHash {
			t.Errorf("page %d (%s): SourceHash = %q, want %q", i, page.URL, stored.MiniSummary.SourceHash, page.Content.FullTextHash)
		}
		if stored.MiniSummary.Skipped != nil {
			t.Errorf("page %d (%s): Skipped should be nil, got %q", i, page.URL, *stored.MiniSummary.Skipped)
		}
	}
}

func TestSummarize_SkipsEmptyPages(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	pages := []*domain.Page{
		testPage("https://example.com/site/empty", "Empty Page", domain.PageTypeEmpty, "", ""),
		testPage("https://example.com/site/art1", "Article 1", domain.PageTypeArticle, "Article text", "hash-art1"),
		testPage("https://example.com/site/art2", "Article 2", domain.PageTypeArticle, "Article text 2", "hash-art2"),
	}
	seedStore(t, dataDir, pages)

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "Summary 1", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
			{Response: &ports.GenerateResponse{Text: "Summary 2", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 2 {
		t.Errorf("Generated = %d, want 2", report.Generated)
	}
	if report.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", report.Skipped)
	}

	// Empty page should have Skipped set.
	storedEmpty := readStoredPage(t, dataDir, "https://example.com/site/empty")
	if storedEmpty.MiniSummary.Skipped == nil {
		t.Error("empty page: Skipped should not be nil")
	} else if *storedEmpty.MiniSummary.Skipped != "empty_content" {
		t.Errorf("empty page: Skipped = %q, want %q", *storedEmpty.MiniSummary.Skipped, "empty_content")
	}
}

func TestSummarize_SkipsAlreadySummarized(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	page := testPage("https://example.com/site/page1", "Article 1", domain.PageTypeArticle, "Article content", "known-hash")
	page.MiniSummary = domain.MiniSummary{
		Text:        "Existing summary",
		GeneratedAt: time.Now(),
		Model:       "claude-haiku-4-5",
		SourceHash:  "known-hash", // matches FullTextHash
	}
	seedStore(t, dataDir, []*domain.Page{page})

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "New summary", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
		Loop: true,
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{Force: false}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 0 {
		t.Errorf("Generated = %d, want 0 (page was already summarized)", report.Generated)
	}
	if report.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", report.Skipped)
	}

	// Provider should not have been called.
	if calls := provider.Calls(); len(calls) != 0 {
		t.Errorf("provider Calls() = %d, want 0", len(calls))
	}
}

func TestSummarize_ForceRegenerates(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	page := testPage("https://example.com/site/page1", "Article 1", domain.PageTypeArticle, "Article content", "known-hash")
	page.MiniSummary = domain.MiniSummary{
		Text:       "Existing summary",
		SourceHash: "known-hash",
	}
	seedStore(t, dataDir, []*domain.Page{page})

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "Regenerated summary", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{Force: true}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 1 {
		t.Errorf("Generated = %d, want 1 (force should regenerate)", report.Generated)
	}
	if report.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", report.Skipped)
	}

	if calls := provider.Calls(); len(calls) != 1 {
		t.Errorf("provider Calls() = %d, want 1", len(calls))
	}
}

func TestSummarize_FailuresIsolated(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)
	cfg.LLM.Concurrency = 1 // sequential for determinism

	pages := []*domain.Page{
		testPage("https://example.com/site/p1", "Page 1", domain.PageTypeArticle, "Content 1", "h1"),
		testPage("https://example.com/site/p2", "Page 2", domain.PageTypeArticle, "Content 2", "h2"),
		testPage("https://example.com/site/p3", "Page 3", domain.PageTypeArticle, "Content 3", "h3"),
	}
	seedStore(t, dataDir, pages)

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "Summary 1", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
			{Err: errors.New("provider error for page 2")},
			{Response: &ports.GenerateResponse{Text: "Summary 3", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 2 {
		t.Errorf("Generated = %d, want 2", report.Generated)
	}
	if report.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Failed)
	}
	if len(report.FailedReasons) == 0 {
		t.Error("FailedReasons should not be empty")
	}
}

func TestSummarize_ConcurrencyRespected(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)
	cfg.LLM.Concurrency = 2

	pages := make([]*domain.Page, 5)
	for i := range pages {
		pages[i] = testPage(
			fmt.Sprintf("https://example.com/site/page%d", i+1),
			fmt.Sprintf("Page %d", i+1),
			domain.PageTypeArticle,
			fmt.Sprintf("Content %d", i+1),
			fmt.Sprintf("hash%d", i+1),
		)
	}
	seedStore(t, dataDir, pages)

	// Build a mock that sleeps 20ms per response.
	sleepDuration := 20 * time.Millisecond
	responses := make([]llm.MockResponse, 5)
	for i := range responses {
		resp := &ports.GenerateResponse{
			Text:         fmt.Sprintf("Summary %d", i+1),
			TokensInput:  10,
			TokensOutput: 5,
			Provider:     "anthropic",
			Model:        "claude-haiku-4-5",
		}
		responses[i] = llm.MockResponse{Response: resp}
	}

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:      "anthropic",
		Model:     "claude-haiku-4-5",
		Responses: responses,
	})

	// Wrap provider to add sleep.
	sleepProvider := &sleepyProvider{inner: provider, delay: sleepDuration}

	start := time.Now()
	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, sleepProvider)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if report.Generated != 5 {
		t.Errorf("Generated = %d, want 5", report.Generated)
	}

	// Soft timing check: should complete well within 500ms.
	if elapsed > 500*time.Millisecond {
		t.Errorf("Summarize took %v, expected under 500ms", elapsed)
	}
}

// sleepyProvider wraps any LLMProvider and adds a delay per call.
type sleepyProvider struct {
	inner ports.LLMProvider
	delay time.Duration
}

func (s *sleepyProvider) Generate(ctx context.Context, req ports.GenerateRequest) (*ports.GenerateResponse, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return s.inner.Generate(ctx, req)
}
func (s *sleepyProvider) Name() string  { return s.inner.Name() }
func (s *sleepyProvider) Model() string { return s.inner.Model() }

func TestSummarize_CostEstimated(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)
	cfg.LLM.Provider = "anthropic"
	cfg.LLM.Model = "claude-haiku-4-5"

	page := testPage("https://example.com/site/p1", "Page 1", domain.PageTypeArticle, "Content 1", "h1")
	seedStore(t, dataDir, []*domain.Page{page})

	// Known tokens: 1M input = $1, 1M output = $5.
	// Use 100 tokens in, 50 tokens out → cost = 0.0001 + 0.00025 = 0.00035
	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{
				Text:         "Summary",
				TokensInput:  100,
				TokensOutput: 50,
				Provider:     "anthropic",
				Model:        "claude-haiku-4-5",
			}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.EstimatedCost <= 0 {
		t.Errorf("EstimatedCost = %f, want > 0", report.EstimatedCost)
	}

	// Exact calculation: 100/1M * $1 + 50/1M * $5 = 0.0001 + 0.00025 = 0.00035
	expected := 0.00035
	diff := report.EstimatedCost - expected
	if diff < -1e-9 || diff > 1e-9 {
		t.Errorf("EstimatedCost = %.8f, want %.8f", report.EstimatedCost, expected)
	}
}

func TestSummarize_MaxPages(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	pages := make([]*domain.Page, 5)
	for i := range pages {
		pages[i] = testPage(
			fmt.Sprintf("https://example.com/site/p%d", i+1),
			fmt.Sprintf("Page %d", i+1),
			domain.PageTypeArticle,
			fmt.Sprintf("Content %d", i+1),
			fmt.Sprintf("hash%d", i+1),
		)
	}
	seedStore(t, dataDir, pages)

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "s1", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
			{Response: &ports.GenerateResponse{Text: "s2", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
			{Response: &ports.GenerateResponse{Text: "s3", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{MaxPages: 3}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 3 {
		t.Errorf("Generated = %d, want 3 (MaxPages=3)", report.Generated)
	}
}

func TestSummarize_SkipsRedirectPages(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)

	redirect := testPage("https://example.com/site/redir", "Redirect Page", domain.PageTypeRedirect, "", "")
	article := testPage("https://example.com/site/art", "Article", domain.PageTypeArticle, "Content", "hash-art")
	seedStore(t, dataDir, []*domain.Page{redirect, article})

	provider := llm.NewMockProvider(llm.MockOptions{
		Name:  "anthropic",
		Model: "claude-haiku-4-5",
		Responses: []llm.MockResponse{
			{Response: &ports.GenerateResponse{Text: "Summary", TokensInput: 10, TokensOutput: 5, Provider: "anthropic", Model: "claude-haiku-4-5"}},
		},
	})

	report, err := Summarize(context.Background(), slog.Default(), cfg, SummarizeOptions{}, provider)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	if report.Generated != 1 {
		t.Errorf("Generated = %d, want 1", report.Generated)
	}
	if report.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (redirect page)", report.Skipped)
	}

	storedRedir := readStoredPage(t, dataDir, "https://example.com/site/redir")
	if storedRedir.MiniSummary.Skipped == nil {
		t.Error("redirect page: Skipped should not be nil")
	} else if *storedRedir.MiniSummary.Skipped != "redirect" {
		t.Errorf("redirect page: Skipped = %q, want %q", *storedRedir.MiniSummary.Skipped, "redirect")
	}
}
