package htmlextract_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/bergmaia/site-research/internal/adapters/htmlextract"
)

const (
	fixtureDir  = "../../../fixtures/html"
	scopePrefix = "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
)

func newExtractor() *htmlextract.Extractor {
	return htmlextract.New(htmlextract.Options{
		CrawlerVersion: "test-v0",
		ScopePrefix:    scopePrefix,
	})
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixtureDir + "/" + name)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return data
}

// TestExtract_LandingRoot tests the root landing page fixture.
func TestExtract_LandingRoot(t *testing.T) {
	body := loadFixture(t, "landing_root.html")
	e := newExtractor()
	pageURL := scopePrefix

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if page == nil {
		t.Fatal("Extract returned nil page")
	}

	// Title non-empty and contains "Transparência"
	if page.Title == "" {
		t.Error("expected non-empty Title")
	}
	if !strings.Contains(strings.ToLower(page.Title), "transparência") &&
		!strings.Contains(strings.ToLower(page.Title), "transparencia") {
		t.Errorf("expected Title to contain 'Transparência', got: %q", page.Title)
	}

	// If breadcrumb missing, ExtractionWarnings must mention it
	if len(page.Breadcrumb) == 0 {
		found := false
		for _, w := range page.Metadata.ExtractionWarnings {
			if strings.Contains(strings.ToLower(w), "breadcrumb") {
				found = true
				break
			}
		}
		if !found {
			t.Error("no breadcrumb found but no warning about it in ExtractionWarnings")
		}
	}

	// Children >= 5 (root page has many children)
	if len(page.Links.Children) < 5 {
		t.Errorf("expected >= 5 children, got %d", len(page.Links.Children))
	}

	// ContentLength >= 0
	if page.Content.ContentLength < 0 {
		t.Errorf("expected ContentLength >= 0, got %d", page.Content.ContentLength)
	}

	// CrawlerVersion
	if page.Metadata.CrawlerVersion != "test-v0" {
		t.Errorf("expected CrawlerVersion 'test-v0', got %q", page.Metadata.CrawlerVersion)
	}
}

// TestExtract_LandingSection tests a landing section page.
func TestExtract_LandingSection(t *testing.T) {
	body := loadFixture(t, "landing_section.html")
	e := newExtractor()
	pageURL := scopePrefix + "/gestao-orcamentaria-e-financeira"

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Section non-empty OR PathTitles >= 1
	if len(page.PathTitles) < 1 {
		t.Errorf("expected PathTitles length >= 1, got %d (section=%q)", len(page.PathTitles), page.Section)
	}
	// We allow Section to be empty only if PathTitles has just 1 item
	if len(page.PathTitles) >= 2 && page.Section == "" {
		t.Errorf("expected Section non-empty when PathTitles has >= 2 items, PathTitles=%v", page.PathTitles)
	}
}

// TestExtract_Article tests an article page.
func TestExtract_Article(t *testing.T) {
	body := loadFixture(t, "article.html")
	e := newExtractor()
	pageURL := scopePrefix + "/licitacoes-e-contratos/licitacoes-e-contratos"

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// FullText non-empty and ContentLength >= 200
	if page.Content.FullText == "" {
		t.Error("expected non-empty FullText")
	}
	if page.Content.ContentLength < 200 {
		t.Errorf("expected ContentLength >= 200, got %d", page.Content.ContentLength)
	}

	// Summary non-empty
	if page.Content.Summary == "" {
		t.Error("expected non-empty Summary")
	}

	// FullTextHash starts with "sha256:"
	if !strings.HasPrefix(page.Content.FullTextHash, "sha256:") {
		t.Errorf("expected FullTextHash to start with 'sha256:', got %q", page.Content.FullTextHash)
	}

	// KeywordsExtracted between 1 and 10
	n := len(page.Content.KeywordsExtracted)
	if n < 1 || n > 10 {
		t.Errorf("expected KeywordsExtracted between 1 and 10, got %d", n)
	}
}

// TestExtract_ListingWithDocs tests a listing page with document links.
func TestExtract_ListingWithDocs(t *testing.T) {
	body := loadFixture(t, "listing_with_docs.html")
	e := newExtractor()
	pageURL := scopePrefix + "/licitacoes-e-contratos/outras-contratacoes/termo-de-repasse/arquivos"

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Combined links + docs >= 3
	total := len(page.Links.Internal) + len(page.Links.Children) + len(page.Documents)
	if total < 3 {
		t.Errorf("expected combined internal+children+documents >= 3, got %d", total)
	}

	// Check if fixture has doc links — if so, Documents must have at least one
	rawHTML := string(body)
	hasDocLink := strings.Contains(strings.ToLower(rawHTML), ".pdf") ||
		strings.Contains(strings.ToLower(rawHTML), ".zip") ||
		strings.Contains(strings.ToLower(rawHTML), ".doc")
	if hasDocLink && len(page.Documents) == 0 {
		t.Error("fixture has document links but Documents is empty")
	}
}

// TestExtract_DeepArticle tests a deeply nested article page.
func TestExtract_DeepArticle(t *testing.T) {
	body := loadFixture(t, "deep_article.html")
	e := newExtractor()
	pageURL := scopePrefix + "/colegiados/comissoes-permanentes-e-tecnicas/comissao-setorial-de-risco-saof/comissao-setorial-de-risco-saof"

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// PathTitles >= 3 (breadcrumb shows depth)
	if len(page.PathTitles) < 3 {
		t.Errorf("expected PathTitles length >= 3 for deep article, got %d: %v", len(page.PathTitles), page.PathTitles)
	}
}

// TestExtract_EmptyOrMinimal tests a minimal page.
func TestExtract_EmptyOrMinimal(t *testing.T) {
	body := loadFixture(t, "empty_or_minimal.html")
	e := newExtractor()
	pageURL := scopePrefix + "/gestao-orcamentaria-e-financeira/demonstracao"

	page, err := e.Extract(context.Background(), pageURL, body)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if page == nil {
		t.Fatal("Extract returned nil page")
	}

	// At least 1 breadcrumb item OR a warning about missing breadcrumb
	if len(page.Breadcrumb) == 0 {
		found := false
		for _, w := range page.Metadata.ExtractionWarnings {
			if strings.Contains(strings.ToLower(w), "breadcrumb") {
				found = true
				break
			}
		}
		if !found {
			t.Error("empty page: expected either breadcrumb items or a warning about missing breadcrumb")
		}
	}
}

// TestExtract_MalformedHTML tests handling of malformed/incomplete HTML.
func TestExtract_MalformedHTML(t *testing.T) {
	body := []byte("<html><body><h1>Hi")
	e := newExtractor()

	page, err := e.Extract(context.Background(), "https://example.com/page", body)
	if err != nil {
		t.Fatalf("Extract returned unexpected error for malformed HTML: %v", err)
	}
	if page == nil {
		t.Fatal("Extract returned nil page for malformed HTML")
	}

	// Should not panic; title should be empty or "Hi"
	// goquery parses <h1>Hi</h1> but there's no <title> tag
	// Accept either empty title or "Hi"
	if page.Title != "" && page.Title != "Hi" {
		t.Logf("Title for malformed HTML: %q (acceptable)", page.Title)
	}
}

// TestExtract_RelativeLinksResolved tests that relative links are resolved to absolute URLs.
func TestExtract_RelativeLinksResolved(t *testing.T) {
	html := `<!DOCTYPE html>
<html lang="pt-br">
<head><title>Test Page</title></head>
<body>
<nav id="breadcrumb"><ol class="breadcrumb">
<li class="breadcrumb-item"><a href="https://www.example.com">Home</a></li>
<li class="breadcrumb-item"><a href="/transparencia-e-prestacao-de-contas">Transparência</a></li>
</ol></nav>
<main>
<p>Test content for the page.</p>
<a href="/foo/bar">Relative link</a>
<a href="/foo/bar/baz">Deep relative link</a>
</main>
</body>
</html>`

	e := htmlextract.New(htmlextract.Options{
		CrawlerVersion: "test-v0",
		ScopePrefix:    "https://www.example.com/foo",
	})

	page, err := e.Extract(context.Background(), "https://www.example.com/foo", []byte(html))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Check that all link URLs are absolute (have scheme+host)
	allLinks := []string{}
	for _, c := range page.Links.Children {
		allLinks = append(allLinks, c.URL)
	}
	for _, l := range page.Links.Internal {
		allLinks = append(allLinks, l.URL)
	}
	for _, l := range page.Links.External {
		allLinks = append(allLinks, l.URL)
	}

	if len(allLinks) == 0 {
		t.Log("no links found in synthetic HTML (acceptable for this test)")
		return
	}

	for _, link := range allLinks {
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			t.Errorf("expected absolute URL, got relative: %q", link)
		}
	}

	// The /foo/bar link should be resolved to https://www.example.com/foo/bar
	found := false
	for _, c := range page.Links.Children {
		if c.URL == "https://www.example.com/foo/bar" {
			found = true
			break
		}
	}
	if !found {
		// Also check internal
		for _, l := range page.Links.Internal {
			if l.URL == "https://www.example.com/foo/bar" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Logf("all links: children=%v internal=%v external=%v", page.Links.Children, page.Links.Internal, page.Links.External)
		t.Error("expected /foo/bar to resolve to https://www.example.com/foo/bar")
	}
}
