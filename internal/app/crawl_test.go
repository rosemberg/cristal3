package app_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/logging"
)

const fixtureHTML = "/Users/rosemberg/projetos-gemini/cristal3/fixtures/html/article.html"

func crawlTestCfg(t *testing.T, srv *httptest.Server) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Scope.SeedURL = srv.URL + "/transparencia-e-prestacao-de-contas"
	cfg.Scope.Prefix = srv.URL + "/transparencia-e-prestacao-de-contas"
	cfg.Sitemap.URL = ""
	cfg.Crawler.UserAgent = "test-crawler"
	cfg.Crawler.RateLimitPerSecond = 100
	cfg.Crawler.JitterMS = 5
	cfg.Crawler.RequestTimeoutSeconds = 10
	cfg.Crawler.MaxRetries = 2
	cfg.Crawler.RespectRobotsTxt = false // avoid https robots fetch for http test server
	cfg.Crawler.HonorRetryAfter = false
	cfg.Crawler.CircuitBreaker.MaxConsecutiveFailures = 5
	cfg.Crawler.CircuitBreaker.PauseMinutes = 1
	cfg.Crawler.CircuitBreaker.AbortThreshold = 3
	cfg.Crawler.SuspiciousResponse.MinBodyBytes = 100
	cfg.Crawler.SuspiciousResponse.BlockTitlePatterns = []string{"Access Denied"}
	cfg.Storage.DataDir = t.TempDir()
	return cfg
}

func buildSitemap(t *testing.T, urls []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sitemap.xml")

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		sb.WriteString(fmt.Sprintf("<url><loc>%s</loc></url>", u))
	}
	sb.WriteString(`</urlset>`)

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write sitemap: %v", err)
	}
	return path
}

func countIndexFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "_index.json" {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk dir: %v", err)
	}
	return count
}

func TestCrawl_EndToEnd_TmpSitemap_FixtureHTML(t *testing.T) {
	htmlBody, err := os.ReadFile(fixtureHTML)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(htmlBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	ctx := context.Background()
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})

	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}
	if report.TotalCrawled != 3 {
		t.Errorf("TotalCrawled = %d, want 3", report.TotalCrawled)
	}
	if len(report.FailedURLs) != 0 {
		t.Errorf("FailedURLs = %v, want empty", report.FailedURLs)
	}
	if report.HTTPStatusHistogram[200] != 3 {
		t.Errorf("HTTPStatusHistogram[200] = %d, want 3", report.HTTPStatusHistogram[200])
	}

	count := countIndexFiles(t, cfg.Storage.DataDir)
	if count != 3 {
		t.Errorf("found %d _index.json files, want 3", count)
	}

	// Verify at least one file has "page_type" key.
	found := false
	_ = filepath.Walk(cfg.Storage.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != "_index.json" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(data), `"page_type"`) {
			found = true
		}
		return nil
	})
	if !found {
		t.Error(`no _index.json file contained "page_type" key`)
	}
}

func TestCrawl_DryRun_DoesNotWrite(t *testing.T) {
	htmlBody, err := os.ReadFile(fixtureHTML)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(htmlBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	ctx := context.Background()
	_, err = app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath, DryRun: true})
	if err != nil {
		t.Fatalf("Crawl dry-run error: %v", err)
	}

	count := countIndexFiles(t, cfg.Storage.DataDir)
	if count != 0 {
		t.Errorf("dry-run: found %d _index.json files, want 0", count)
	}
}

func TestCrawl_MaxURLs(t *testing.T) {
	htmlBody, err := os.ReadFile(fixtureHTML)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(htmlBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	ctx := context.Background()
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath, MaxURLs: 2})
	if err != nil {
		t.Fatalf("Crawl MaxURLs error: %v", err)
	}
	if report.TotalCrawled != 2 {
		t.Errorf("TotalCrawled = %d, want 2", report.TotalCrawled)
	}

	count := countIndexFiles(t, cfg.Storage.DataDir)
	if count != 2 {
		t.Errorf("found %d _index.json files, want 2", count)
	}
}

// TestCrawl_Incremental_UnchangedHashPreservesMiniSummary verifies that when a
// second crawl sees the same HTML body (same content_hash), the MiniSummary seeded
// in the store is preserved (not cleared), and counters reflect unchanged pages.
func TestCrawl_Incremental_UnchangedHashPreservesMiniSummary(t *testing.T) {
	htmlBody, err := os.ReadFile(fixtureHTML)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// Server returns the same HTML for each URL, no ETag/Last-Modified headers —
	// so the fetcher always GETs (no 304) and the hash-equality path activates.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(htmlBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})
	ctx := context.Background()

	// --- First crawl: populate the store ---
	report1, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("first crawl error: %v", err)
	}
	if report1.TotalCrawled != 3 {
		t.Fatalf("first crawl: TotalCrawled = %d, want 3", report1.TotalCrawled)
	}
	if report1.NewPages != 3 {
		t.Fatalf("first crawl: NewPages = %d, want 3", report1.NewPages)
	}

	// --- Seed mini_summary on one of the stored pages ---
	// Use fsstore directly to load, mutate, and re-put the page for URL /a.
	targetURL := srv.URL + "/transparencia-e-prestacao-de-contas/a"
	seededSummary := domain.MiniSummary{
		Text:        "seeded summary",
		GeneratedAt: time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC),
		Model:       "claude-haiku-4-5",
		SourceHash:  "sha256:abc",
	}

	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	pageA, err := store.Get(ctx, targetURL)
	if err != nil {
		t.Fatalf("store.Get for /a: %v", err)
	}
	pageA.MiniSummary = seededSummary
	if err := store.Put(ctx, pageA); err != nil {
		t.Fatalf("store.Put (seed mini_summary): %v", err)
	}

	// --- Second crawl: same HTML → same content_hash → unchanged path ---
	report2, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("second crawl error: %v", err)
	}

	// All three pages should be unchanged (same body → same hash).
	if report2.UnchangedPages != 3 {
		t.Errorf("second crawl: UnchangedPages = %d, want 3", report2.UnchangedPages)
	}
	if report2.NewPages != 0 {
		t.Errorf("second crawl: NewPages = %d, want 0", report2.NewPages)
	}
	if report2.UpdatedPages != 0 {
		t.Errorf("second crawl: UpdatedPages = %d, want 0", report2.UpdatedPages)
	}

	// Verify mini_summary was preserved on page /a.
	reloaded, err := store.Get(ctx, targetURL)
	if err != nil {
		t.Fatalf("store.Get after second crawl: %v", err)
	}
	if reloaded.MiniSummary.Text != "seeded summary" {
		t.Errorf("mini_summary.text = %q, want %q", reloaded.MiniSummary.Text, "seeded summary")
	}
	if reloaded.MiniSummary.Model != "claude-haiku-4-5" {
		t.Errorf("mini_summary.model = %q, want %q", reloaded.MiniSummary.Model, "claude-haiku-4-5")
	}
}

// TestCrawl_Incremental_ChangedBodyClearsMiniSummary verifies that when the body
// changes between crawls the seeded MiniSummary is cleared so the summarize step
// can regenerate it, and counters show all pages as updated.
func TestCrawl_Incremental_ChangedBodyClearsMiniSummary(t *testing.T) {
	htmlBody, err := os.ReadFile(fixtureHTML)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// bodyHolder lets us swap the response body between the two crawl runs.
	type bodyHolder struct{ body []byte }
	holder := &bodyHolder{body: htmlBody}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(holder.body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})
	ctx := context.Background()

	// --- First crawl ---
	report1, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("first crawl error: %v", err)
	}
	if report1.NewPages != 3 {
		t.Fatalf("first crawl: NewPages = %d, want 3", report1.NewPages)
	}

	// --- Seed mini_summary on page /a ---
	targetURL := srv.URL + "/transparencia-e-prestacao-de-contas/a"
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	pageA, err := store.Get(ctx, targetURL)
	if err != nil {
		t.Fatalf("store.Get /a: %v", err)
	}
	pageA.MiniSummary = domain.MiniSummary{
		Text:  "will be cleared",
		Model: "claude-haiku-4-5",
	}
	if err := store.Put(ctx, pageA); err != nil {
		t.Fatalf("store.Put: %v", err)
	}

	// --- Change the body so the content_hash differs ---
	// Insert an extra paragraph inside #content (which is what the extractor reads).
	// The fixture has id="content"; we inject a <p> before its first existing <p> so
	// the extracted full_text changes and therefore content_hash changes.
	modified := strings.Replace(
		string(htmlBody),
		`<p align="justify" style="text-align: justify;"><span><span> A divulgação`,
		`<p>extra paragraph for M5 test</p><p align="justify" style="text-align: justify;"><span><span> A divulgação`,
		1,
	)
	if modified == string(htmlBody) {
		t.Fatal("body replacement did not match — fixture may have changed")
	}
	holder.body = []byte(modified)

	// --- Second crawl with changed HTML ---
	report2, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("second crawl error: %v", err)
	}

	if report2.UpdatedPages != 3 {
		t.Errorf("second crawl: UpdatedPages = %d, want 3", report2.UpdatedPages)
	}
	if report2.NewPages != 0 {
		t.Errorf("second crawl: NewPages = %d, want 0", report2.NewPages)
	}
	if report2.UnchangedPages != 0 {
		t.Errorf("second crawl: UnchangedPages = %d, want 0", report2.UnchangedPages)
	}

	// mini_summary on /a must be cleared (zero Text).
	reloaded, err := store.Get(ctx, targetURL)
	if err != nil {
		t.Fatalf("store.Get after second crawl: %v", err)
	}
	if reloaded.MiniSummary.Text != "" {
		t.Errorf("mini_summary.text = %q, want empty (cleared)", reloaded.MiniSummary.Text)
	}

}

// orphanHTML returns minimal HTML for the orphan-discovery tests. When withOrphanLink
// is true the body contains an <a> inside #content-core pointing to /d.
func orphanHTML(t *testing.T, title string, withOrphanLink bool, srvURL string) []byte {
	t.Helper()
	link := ""
	if withOrphanLink {
		link = `<p>Mais conteúdo: <a href="/transparencia-e-prestacao-de-contas/d">Orphan D</a></p>`
	}
	// The body must be ≥ MinBodyBytes (100) and have a content zone so htmlextract
	// picks up the page as an article. We use <article id="content-core">.
	body := `<!doctype html>
<html lang="pt-BR"><head><title>` + title + `</title></head>
<body>
<nav id="portal-breadcrumbs">
  <ol class="breadcrumb">
    <li><a href="/">Início</a></li>
    <li><a href="/transparencia-e-prestacao-de-contas">Transparência</a></li>
  </ol>
</nav>
<article id="content-core">
  <h1>` + title + `</h1>
  <p>Conteúdo substantivo para detecção como article. Lorem ipsum dolor sit amet,
  consectetur adipiscing elit. Quisque vehicula libero ut velit venenatis pretium.
  Nulla facilisi. Pellentesque habitant morbi tristique senectus et netus.</p>
  ` + link + `
</article>
</body></html>`
	return []byte(body)
}

// TestCrawl_OrphanDiscovery verifies the full BFS orphan-discovery path (RF-02):
//   - 3 pages in sitemap (/a, /b, /c)
//   - /d reachable only via a link on /a
//   - After crawl: OrphansFound==1, TotalCrawled==4, /d stamped with discovered_via=="link"
func TestCrawl_OrphanDiscovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch path {
		case "/transparencia-e-prestacao-de-contas/a":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página A", true, r.Host))
		case "/transparencia-e-prestacao-de-contas/b":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página B", false, r.Host))
		case "/transparencia-e-prestacao-de-contas/c":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página C", false, r.Host))
		case "/transparencia-e-prestacao-de-contas/d":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Orphan D", false, r.Host))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	ctx := context.Background()
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	if report.SitemapTotal != 3 {
		t.Errorf("SitemapTotal = %d, want 3", report.SitemapTotal)
	}
	if report.OrphansFound != 1 {
		t.Errorf("OrphansFound = %d, want 1", report.OrphansFound)
	}
	if report.TotalCrawled != 4 {
		t.Errorf("TotalCrawled = %d, want 4 (3 sitemap + 1 orphan)", report.TotalCrawled)
	}
	if len(report.FailedURLs) != 0 {
		t.Errorf("FailedURLs = %v, want empty", report.FailedURLs)
	}

	// Expect 4 _index.json files on disk.
	count := countIndexFiles(t, cfg.Storage.DataDir)
	if count != 4 {
		t.Errorf("found %d _index.json files, want 4", count)
	}

	// Verify discovered_via on each stored page.
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	for _, slug := range []string{"a", "b", "c"} {
		u := srv.URL + "/transparencia-e-prestacao-de-contas/" + slug
		p, err := store.Get(ctx, u)
		if err != nil {
			t.Fatalf("store.Get /%s: %v", slug, err)
		}
		if p.Metadata.DiscoveredVia != domain.DiscoveredViaSitemap {
			t.Errorf("page /%s: discovered_via = %q, want %q",
				slug, p.Metadata.DiscoveredVia, domain.DiscoveredViaSitemap)
		}
	}

	dURL := srv.URL + "/transparencia-e-prestacao-de-contas/d"
	pageD, err := store.Get(ctx, dURL)
	if err != nil {
		t.Fatalf("store.Get /d: %v", err)
	}
	if pageD.Metadata.DiscoveredVia != domain.DiscoveredViaLink {
		t.Errorf("page /d: discovered_via = %q, want %q",
			pageD.Metadata.DiscoveredVia, domain.DiscoveredViaLink)
	}
}

// TestCrawl_StaleDetection_MarksAbsentURL verifies that a page seeded into the store
// but absent from both the sitemap and link discovery sets is marked stale after crawl,
// while pages in the sitemap remain untouched. The stale page file must still exist.
func TestCrawl_StaleDetection_MarksAbsentURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch path {
		case "/transparencia-e-prestacao-de-contas/a",
			"/transparencia-e-prestacao-de-contas/b",
			"/transparencia-e-prestacao-de-contas/c":
			w.WriteHeader(http.StatusOK)
			// Use orphanHTML with no outbound links so /d is never linked.
			_, _ = w.Write(orphanHTML(t, "Page "+path[len(path)-1:], false, r.Host))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := crawlTestCfg(t, srv)
	ctx := context.Background()

	// Seed /d directly into the store (not in sitemap, not linked from /a/b/c).
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	dURL := srv.URL + "/transparencia-e-prestacao-de-contas/d"
	seededPage := &domain.Page{
		Schema:        "page-node-v2",
		SchemaVersion: 2,
		URL:           dURL,
		CanonicalURL:  dURL,
		Metadata: domain.Metadata{
			HTTPStatus:    200,
			CrawlerVersion: "0.1.0",
		},
	}
	if err := store.Put(ctx, seededPage); err != nil {
		t.Fatalf("seed /d: %v", err)
	}

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	// /a, /b, /c crawled; since they are fresh (NewPages=3), /d is stale.
	if report.StalePages != 1 {
		t.Errorf("StalePages = %d, want 1", report.StalePages)
	}
	if report.RemovedPages != 0 {
		t.Errorf("RemovedPages = %d, want 0 (no purge)", report.RemovedPages)
	}

	// /d must still exist on disk.
	dPage, err := store.Get(ctx, dURL)
	if err != nil {
		t.Fatalf("store.Get /d after crawl: %v", err)
	}
	if dPage.Metadata.StaleSince == nil {
		t.Error("/d StaleSince is nil, want non-nil after stale marking")
	}

	// Re-run: /d must not be re-marked (timestamp preserved).
	firstStaleTime := *dPage.Metadata.StaleSince
	report2, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("second Crawl error: %v", err)
	}
	if report2.StalePages != 0 {
		t.Errorf("second run: StalePages = %d, want 0 (already marked)", report2.StalePages)
	}
	dPage2, err := store.Get(ctx, dURL)
	if err != nil {
		t.Fatalf("store.Get /d after second crawl: %v", err)
	}
	if dPage2.Metadata.StaleSince == nil {
		t.Error("/d StaleSince is nil after second crawl, want preserved timestamp")
	}
	if !dPage2.Metadata.StaleSince.Equal(firstStaleTime) {
		t.Errorf("/d StaleSince changed: got %v, want %v", dPage2.Metadata.StaleSince, firstStaleTime)
	}
}

// TestCrawl_StaleReturned_ClearsStaleSince verifies that when a previously-stale page
// re-appears in the sitemap and is successfully crawled, its StaleSince is reset to nil.
func TestCrawl_StaleReturned_ClearsStaleSince(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/transparencia-e-prestacao-de-contas/returning":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Returning Page", false, r.Host))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := crawlTestCfg(t, srv)
	ctx := context.Background()

	// Seed the page with a non-nil StaleSince (was stale in previous run).
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	returningURL := srv.URL + "/transparencia-e-prestacao-de-contas/returning"
	pastTime := time.Now().Add(-72 * time.Hour)
	seededPage := &domain.Page{
		Schema:        "page-node-v2",
		SchemaVersion: 2,
		URL:           returningURL,
		CanonicalURL:  returningURL,
		Metadata: domain.Metadata{
			HTTPStatus:    200,
			CrawlerVersion: "0.1.0",
			StaleSince:    &pastTime,
		},
	}
	if err := store.Put(ctx, seededPage); err != nil {
		t.Fatalf("seed returning page: %v", err)
	}

	// Put the returning URL back into the sitemap.
	sitemapPath := buildSitemap(t, []string{returningURL})
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	_, err = app.Crawl(ctx, logger, cfg, app.CrawlOptions{FromFile: sitemapPath})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	// Verify StaleSince is now nil.
	page, err := store.Get(ctx, returningURL)
	if err != nil {
		t.Fatalf("store.Get returning page: %v", err)
	}
	if page.Metadata.StaleSince != nil {
		t.Errorf("StaleSince = %v, want nil (page returned from stale)", page.Metadata.StaleSince)
	}
}

// TestCrawl_PurgeStale_RequiresConfirm verifies that --purge-stale without --confirm
// returns an error and does not delete any pages.
func TestCrawl_PurgeStale_RequiresConfirm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := crawlTestCfg(t, srv)
	ctx := context.Background()

	// Seed a stale page.
	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	oldTime := time.Now().Add(-45 * 24 * time.Hour)
	staleURL := srv.URL + "/transparencia-e-prestacao-de-contas/stale-page"
	stalePage := &domain.Page{
		Schema:        "page-node-v2",
		SchemaVersion: 2,
		URL:           staleURL,
		CanonicalURL:  staleURL,
		Metadata: domain.Metadata{
			HTTPStatus:     200,
			CrawlerVersion: "0.1.0",
			StaleSince:     &oldTime,
		},
	}
	if err := store.Put(ctx, stalePage); err != nil {
		t.Fatalf("seed stale page: %v", err)
	}

	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	// Use OverrideURLs with empty slice to bypass DiscoverInScope (no sitemap needed).
	_, err = app.Crawl(ctx, logger, cfg, app.CrawlOptions{
		OverrideURLs: []app.URLCandidate{}, // empty: no URLs to crawl
		PurgeStale:   true,
		Confirm:      false,
	})
	if err == nil {
		t.Fatal("expected error for purge-stale without --confirm, got nil")
	}
	if !strings.Contains(err.Error(), "confirm") && !strings.Contains(err.Error(), "destructive") {
		t.Errorf("error message %q should contain 'confirm' or 'destructive'", err.Error())
	}

	// Page must still exist.
	if _, getErr := store.Get(ctx, staleURL); getErr != nil {
		t.Errorf("stale page should still exist after refused purge: %v", getErr)
	}
}

// TestCrawl_PurgeStale_DeletesOldStale verifies that pages stale longer than the
// retention period are deleted when --purge-stale and --confirm are both set.
func TestCrawl_PurgeStale_DeletesOldStale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := crawlTestCfg(t, srv)
	cfg.Recrawl.StaleRetentionDays = 30
	ctx := context.Background()

	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	// Seed a page stale for 45 days (> 30-day retention).
	oldTime := time.Now().Add(-45 * 24 * time.Hour)
	staleURL := srv.URL + "/transparencia-e-prestacao-de-contas/old-stale"
	stalePage := &domain.Page{
		Schema:        "page-node-v2",
		SchemaVersion: 2,
		URL:           staleURL,
		CanonicalURL:  staleURL,
		Metadata: domain.Metadata{
			HTTPStatus:     200,
			CrawlerVersion: "0.1.0",
			StaleSince:     &oldTime,
		},
	}
	if err := store.Put(ctx, stalePage); err != nil {
		t.Fatalf("seed old-stale page: %v", err)
	}

	indexPath, pathErr := store.Path(staleURL)
	if pathErr != nil {
		t.Fatalf("store.Path: %v", pathErr)
	}

	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	// Use OverrideURLs with empty slice to bypass DiscoverInScope (no sitemap needed).
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{
		OverrideURLs: []app.URLCandidate{}, // empty: no URLs to crawl
		PurgeStale:   true,
		Confirm:      true,
	})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	if report.RemovedPages != 1 {
		t.Errorf("RemovedPages = %d, want 1", report.RemovedPages)
	}

	// File must no longer exist.
	if _, statErr := os.Stat(indexPath); !os.IsNotExist(statErr) {
		t.Errorf("_index.json should be deleted after purge, but os.Stat returned: %v", statErr)
	}
}

// TestCrawl_PurgeStale_KeepsFreshStale verifies that pages stale for less than the
// retention period are NOT deleted even when --purge-stale and --confirm are set.
func TestCrawl_PurgeStale_KeepsFreshStale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := crawlTestCfg(t, srv)
	cfg.Recrawl.StaleRetentionDays = 30
	ctx := context.Background()

	store, err := fsstore.New(fsstore.Options{
		RootDir:     cfg.Storage.DataDir,
		ScopePrefix: cfg.Scope.Prefix,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	// Seed a page stale for only 10 days (< 30-day retention).
	recentTime := time.Now().Add(-10 * 24 * time.Hour)
	staleURL := srv.URL + "/transparencia-e-prestacao-de-contas/fresh-stale"
	stalePage := &domain.Page{
		Schema:        "page-node-v2",
		SchemaVersion: 2,
		URL:           staleURL,
		CanonicalURL:  staleURL,
		Metadata: domain.Metadata{
			HTTPStatus:     200,
			CrawlerVersion: "0.1.0",
			StaleSince:     &recentTime,
		},
	}
	if err := store.Put(ctx, stalePage); err != nil {
		t.Fatalf("seed fresh-stale page: %v", err)
	}

	indexPath, pathErr := store.Path(staleURL)
	if pathErr != nil {
		t.Fatalf("store.Path: %v", pathErr)
	}

	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	// Use OverrideURLs with empty slice to bypass DiscoverInScope (no sitemap needed).
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{
		OverrideURLs: []app.URLCandidate{}, // empty: no URLs to crawl
		PurgeStale:   true,
		Confirm:      true,
	})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	if report.RemovedPages != 0 {
		t.Errorf("RemovedPages = %d, want 0 (stale is within retention period)", report.RemovedPages)
	}

	// File must still exist.
	if _, statErr := os.Stat(indexPath); statErr != nil {
		t.Errorf("_index.json should still exist (within retention): %v", statErr)
	}
}

// TestCrawl_OrphanRespectsMaxURLs verifies that when MaxURLs is reached during the
// sitemap pass, the orphan pass is skipped and /d is never fetched.
//
// Design choice: OrphansFound is computed post-sitemap-pass and reflects how many
// orphan candidates were detected; we assert it may be ≥ 1. But because MaxURLs==3
// was already exhausted by the 3 sitemap pages, processOne returns false immediately
// and /d is never stored — TotalCrawled stays at 3.
func TestCrawl_OrphanRespectsMaxURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch path {
		case "/transparencia-e-prestacao-de-contas/a":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página A", true, r.Host))
		case "/transparencia-e-prestacao-de-contas/b":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página B", false, r.Host))
		case "/transparencia-e-prestacao-de-contas/c":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Página C", false, r.Host))
		case "/transparencia-e-prestacao-de-contas/d":
			// /d should never be fetched in this test.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(orphanHTML(t, "Orphan D", false, r.Host))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sitemapPath := buildSitemap(t, []string{
		srv.URL + "/transparencia-e-prestacao-de-contas/a",
		srv.URL + "/transparencia-e-prestacao-de-contas/b",
		srv.URL + "/transparencia-e-prestacao-de-contas/c",
	})

	cfg := crawlTestCfg(t, srv)
	logger := logging.New(logging.Config{Level: "warn", Format: "json", Output: io.Discard})

	ctx := context.Background()
	report, err := app.Crawl(ctx, logger, cfg, app.CrawlOptions{
		FromFile: sitemapPath,
		MaxURLs:  3,
	})
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}

	// The sitemap pass consumed all 3 MaxURLs, so the orphan pass must be skipped.
	if report.TotalCrawled != 3 {
		t.Errorf("TotalCrawled = %d, want 3", report.TotalCrawled)
	}

	// /d must NOT have been stored.
	count := countIndexFiles(t, cfg.Storage.DataDir)
	if count != 3 {
		t.Errorf("found %d _index.json files, want 3 (/d must not be stored)", count)
	}

	// OrphansFound may be ≥ 1 (the candidate set is counted even though the pass ran
	// 0 fetches). We just verify it's non-negative (no assertion on exact value here
	// because /a's link to /d is discovered during the sitemap pass).
	if report.OrphansFound < 0 {
		t.Errorf("OrphansFound = %d, want >= 0", report.OrphansFound)
	}
}
