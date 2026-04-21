package sqlitefts_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bergmaia/site-research/internal/adapters/sqlitefts"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

// mkPage builds a minimal *domain.Page fixture for testing.
func mkPage(url, title, section, miniSummary, fullText string, pt domain.PageType) *domain.Page {
	return &domain.Page{
		URL:          url,
		CanonicalURL: url,
		Title:        title,
		Section:      section,
		PageType:     pt,
		Content:      domain.Content{FullText: fullText},
		MiniSummary:  domain.MiniSummary{Text: miniSummary},
	}
}

// scaffoldCatalog returns a minimal Catalog for tests.
func scaffoldCatalog() *domain.Catalog {
	return &domain.Catalog{
		RootURL:       "https://example.com/scope",
		SchemaVersion: 2,
	}
}

// openStore opens a Store in a temporary directory and returns it plus its path.
func openStore(t *testing.T) *sqlitefts.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sqlite")
	s, err := sqlitefts.Open(sqlitefts.Options{Path: path})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestOpenClose verifies that Open creates the file and Close succeeds.
func TestOpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.sqlite")

	s, err := sqlitefts.Open(sqlitefts.Options{Path: path})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected database file to exist after Open, but it does not")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestRebuild_Empty verifies that Rebuild with nil pages creates the table
// and that Search returns nil for any query.
func TestRebuild_Empty(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	if err := s.Rebuild(ctx, cat, nil); err != nil {
		t.Fatalf("Rebuild(nil pages): %v", err)
	}

	hits, err := s.Search(ctx, "anything", 10)
	if err != nil {
		t.Fatalf("Search after empty rebuild: %v", err)
	}
	if hits != nil {
		t.Errorf("expected nil hits, got %v", hits)
	}
}

// TestRebuild_ThreePages_SearchHits inserts three distinct pages and checks
// that a query matching only the first page returns it with the highest Score.
func TestRebuild_ThreePages_SearchHits(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	pages := []*domain.Page{
		mkPage("https://example.com/scope/alpha", "Alpha Page", "news", "summary alpha", "unique keyword xylophone here", domain.PageTypeArticle),
		mkPage("https://example.com/scope/beta", "Beta Page", "docs", "summary beta", "some other content about golang", domain.PageTypeArticle),
		mkPage("https://example.com/scope/gamma", "Gamma Page", "blog", "summary gamma", "completely different topic javascript", domain.PageTypeArticle),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "xylophone", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit, got none")
	}

	top := hits[0]
	if top.URL != "https://example.com/scope/alpha" {
		t.Errorf("top hit URL = %q; want %q", top.URL, "https://example.com/scope/alpha")
	}
	if top.Title != "Alpha Page" {
		t.Errorf("top hit Title = %q; want %q", top.Title, "Alpha Page")
	}
	if top.Score <= 0 {
		t.Errorf("expected Score > 0, got %f", top.Score)
	}
}

// TestSearch_AccentInsensitive verifies that the FTS5 tokenizer
// (unicode61 remove_diacritics 2) allows searching without accents.
func TestSearch_AccentInsensitive(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	pages := []*domain.Page{
		mkPage(
			"https://example.com/scope/servidores",
			"Diárias dos Servidores",
			"rh",
			"Informações sobre diárias",
			"conteúdo sobre diárias dos servidores públicos",
			domain.PageTypeArticle,
		),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	// Search without accent — should still match "diárias" in full_text.
	hits, err := s.Search(ctx, "diarias", 10)
	if err != nil {
		t.Fatalf("Search(\"diarias\"): %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("TestSearch_AccentInsensitive: expected at least one hit for 'diarias' (no accent), got none — FTS5 tokenizer may not be stripping diacritics")
	}
	t.Logf("TestSearch_AccentInsensitive: found %d hit(s); top URL=%q Score=%f", len(hits), hits[0].URL, hits[0].Score)
}

// TestSearch_TitleAndMiniSummaryWeight verifies that both a term in the title
// and a term only in full_text produce hits.
func TestSearch_TitleAndMiniSummaryWeight(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	pages := []*domain.Page{
		mkPage(
			"https://example.com/scope/title-match",
			"Extraordinary Title Term",
			"a",
			"",
			"generic content",
			domain.PageTypeArticle,
		),
		mkPage(
			"https://example.com/scope/fulltext-match",
			"Normal Title",
			"b",
			"",
			"this body has the extraordinary term buried deep",
			domain.PageTypeArticle,
		),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "extraordinary", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) < 2 {
		t.Fatalf("expected 2 hits (title and fulltext), got %d", len(hits))
	}

	urls := make(map[string]bool)
	for _, h := range hits {
		urls[h.URL] = true
	}
	if !urls["https://example.com/scope/title-match"] {
		t.Error("expected hit for title-match page")
	}
	if !urls["https://example.com/scope/fulltext-match"] {
		t.Error("expected hit for fulltext-match page")
	}
}

// TestSearch_LimitRespected inserts 5 pages all matching the same query and
// verifies that Search with limit=3 returns exactly 3 hits.
func TestSearch_LimitRespected(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	var pages []*domain.Page
	for i := 1; i <= 5; i++ {
		pages = append(pages, mkPage(
			fmt.Sprintf("https://example.com/scope/page%d", i),
			fmt.Sprintf("Page %d about widget", i),
			"test",
			"widget summary",
			"widget content here",
			domain.PageTypeArticle,
		))
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "widget", 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 3 {
		t.Errorf("expected 3 hits with limit=3, got %d", len(hits))
	}
}

// TestSearch_EmptyQuery verifies that searching with "" returns (nil, nil).
func TestSearch_EmptyQuery(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	if err := s.Rebuild(ctx, cat, []*domain.Page{
		mkPage("https://example.com/scope/x", "X", "s", "m", "body", domain.PageTypeArticle),
	}); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf("Search(\"\") returned error: %v", err)
	}
	if hits != nil {
		t.Errorf("Search(\"\") should return nil, got %v", hits)
	}
}

// TestSearch_SanitizesFTSSyntax verifies that user input containing FTS5
// operator characters (-, :, ", (, ), *, +, ^) is treated as plain keywords
// rather than surfacing FTS5 grammar errors as SQL errors.
func TestSearch_SanitizesFTSSyntax(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	pages := []*domain.Page{
		mkPage("https://example.com/scope/a", "Diárias março 2026", "s", "m", "planilha de diárias de março", domain.PageTypeArticle),
	}
	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	cases := []string{
		"diarias-marco-2026", // hyphens must not trigger FTS5 NOT
		"TRE-PI",             // hyphen inside sigla
		"title:foo",          // colon must not trigger column filter
		"diárias (março)",    // parentheses
	}
	for _, q := range cases {
		if _, err := s.Search(ctx, q, 10); err != nil {
			t.Errorf("Search(%q) unexpected error: %v", q, err)
		}
	}

	// A query consisting only of FTS5 operators collapses to empty and
	// must return no results, no error.
	hits, err := s.Search(ctx, ")", 10)
	if err != nil {
		t.Fatalf("Search(\")\") returned error after sanitization: %v", err)
	}
	if hits != nil {
		t.Errorf("Search(\")\") should return nil hits, got %v", hits)
	}
}

// TestRebuild_Idempotent calls Rebuild twice and checks that results are consistent.
func TestRebuild_Idempotent(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	pages := []*domain.Page{
		mkPage("https://example.com/scope/one", "One", "s", "m", "idempotent content here", domain.PageTypeArticle),
	}

	for i := 0; i < 2; i++ {
		if err := s.Rebuild(ctx, cat, pages); err != nil {
			t.Fatalf("Rebuild iteration %d: %v", i, err)
		}
	}

	hits, err := s.Search(ctx, "idempotent", 10)
	if err != nil {
		t.Fatalf("Search after double rebuild: %v", err)
	}
	if len(hits) != 1 {
		t.Errorf("expected 1 hit after idempotent rebuild, got %d", len(hits))
	}
}

// TestSearch_Score_InvertedBM25 verifies that results are ordered by Score
// descending (i.e., best match first, Score = -bm25).
func TestSearch_Score_InvertedBM25(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	// Page A has the search term many times → should rank higher.
	// Page B has it once → lower rank.
	pages := []*domain.Page{
		mkPage(
			"https://example.com/scope/low",
			"Low relevance",
			"s",
			"",
			"nebula appears once",
			domain.PageTypeArticle,
		),
		mkPage(
			"https://example.com/scope/high",
			"Nebula study",          // title hit
			"s",
			"nebula overview",       // mini_summary hit
			"nebula nebula nebula",  // full_text hits
			domain.PageTypeArticle,
		),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "nebula", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) < 2 {
		t.Fatalf("expected at least 2 hits, got %d", len(hits))
	}

	// Verify scores are descending.
	for i := 1; i < len(hits); i++ {
		if hits[i].Score > hits[i-1].Score {
			t.Errorf("hits not ordered by Score desc: hits[%d].Score=%f > hits[%d].Score=%f",
				i, hits[i].Score, i-1, hits[i-1].Score)
		}
	}

	// The high-relevance page should be first.
	if hits[0].URL != "https://example.com/scope/high" {
		t.Errorf("expected high-relevance page first, got %q", hits[0].URL)
	}
}

// TestPath_Derivation checks that the path field strips the RootURL prefix correctly.
func TestPath_Derivation(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog() // RootURL = "https://example.com/scope"

	pages := []*domain.Page{
		mkPage("https://example.com/scope/news/article-1", "Article 1", "news", "m", "body content", domain.PageTypeArticle),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "body", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected a hit")
	}

	want := "news/article-1"
	if hits[0].Path != want {
		t.Errorf("Path = %q; want %q", hits[0].Path, want)
	}
}

// TestSearch_DefaultLimit verifies that limit <= 0 defaults to 10.
func TestSearch_DefaultLimit(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	var pages []*domain.Page
	for i := 0; i < 15; i++ {
		pages = append(pages, mkPage(
			fmt.Sprintf("https://example.com/scope/p%d", i),
			"Gadget page",
			"test",
			"gadget summary",
			"gadget content",
			domain.PageTypeArticle,
		))
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "gadget", 0) // limit=0 → default 10
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 10 {
		t.Errorf("expected 10 hits with limit=0 (default), got %d", len(hits))
	}
}

// TestOpen_EmptyPath verifies that Open returns an error for an empty path.
func TestOpen_EmptyPath(t *testing.T) {
	_, err := sqlitefts.Open(sqlitefts.Options{Path: ""})
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

// TestOpen_InvalidPath verifies that Open returns an error for an unusable path.
func TestOpen_InvalidPath(t *testing.T) {
	// A path whose parent directory does not exist should fail on Ping.
	_, err := sqlitefts.Open(sqlitefts.Options{Path: "/nonexistent/deeply/nested/path/db.sqlite"})
	if err == nil {
		t.Fatal("expected error for invalid/unreachable path, got nil")
	}
}

// TestDerivePath_NoPrefix verifies that derivePath falls back to the raw URL
// when the rootURL is not a prefix of pageURL.
func TestDerivePath_NoPrefix(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	// Use a catalog whose RootURL does NOT match the page URL.
	cat := &domain.Catalog{RootURL: "https://other.com", SchemaVersion: 2}

	pages := []*domain.Page{
		mkPage("https://example.com/different/path", "Different", "s", "", "fallback path content", domain.PageTypeArticle),
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "fallback", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected a hit")
	}
	// When rootURL is not a prefix, path falls back to the full URL.
	if hits[0].Path != "https://example.com/different/path" {
		t.Errorf("Path = %q; want full URL as fallback", hits[0].Path)
	}
}

// TestRebuild_NilCatalog verifies that Rebuild works when catalog is nil
// (rootURL defaults to empty string, path falls back to full URL).
func TestRebuild_NilCatalog(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	pages := []*domain.Page{
		mkPage("https://example.com/page", "Page", "s", "", "nilcatalog body", domain.PageTypeArticle),
	}

	if err := s.Rebuild(ctx, nil, pages); err != nil {
		t.Fatalf("Rebuild(nil catalog): %v", err)
	}

	hits, err := s.Search(ctx, "nilcatalog", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected a hit")
	}
	// With nil catalog / empty rootURL, path = full URL (fallback).
	if hits[0].Path != "https://example.com/page" {
		t.Errorf("Path = %q; want full URL", hits[0].Path)
	}
}

// TestSearch_NegativeLimitDefaultsTen verifies limit=-1 defaults to 10.
func TestSearch_NegativeLimitDefaultsTen(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	cat := scaffoldCatalog()

	var pages []*domain.Page
	for i := 0; i < 12; i++ {
		pages = append(pages, mkPage(
			fmt.Sprintf("https://example.com/scope/q%d", i),
			"Quartz page",
			"test",
			"quartz summary",
			"quartz content",
			domain.PageTypeArticle,
		))
	}

	if err := s.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	hits, err := s.Search(ctx, "quartz", -1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 10 {
		t.Errorf("expected 10 hits with limit=-1 (default), got %d", len(hits))
	}
}

// Compile-time interface check from the test package side.
var _ ports.SearchIndex = (*sqlitefts.Store)(nil)
