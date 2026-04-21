package sitemap_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bergmaia/site-research/internal/adapters/sitemap"
)

const fixtureFile = "/Users/rosemberg/projetos-gemini/cristal3/fixtures/sitemap.xml.gz"

// TestFetch_FromFile_GzippedFixture tests loading and parsing the real gzipped fixture.
func TestFetch_FromFile_GzippedFixture(t *testing.T) {
	src := sitemap.New(sitemap.Options{FromFile: fixtureFile})
	entries, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const wantCount = 2367
	if len(entries) != wantCount {
		t.Errorf("got %d entries, want %d", len(entries), wantCount)
	}

	// At least one entry should have a non-zero LastMod.
	hasLastMod := false
	for _, e := range entries {
		if !e.LastMod.IsZero() {
			hasLastMod = true
			break
		}
	}
	if !hasLastMod {
		t.Error("expected at least one entry with non-zero LastMod")
	}

	// At least one entry should have Loc containing the expected path prefix.
	hasTransparencia := false
	for _, e := range entries {
		if strings.Contains(e.Loc, "transparencia-e-prestacao-de-contas") {
			hasTransparencia = true
			break
		}
	}
	if !hasTransparencia {
		t.Error("expected at least one entry with Loc containing 'transparencia-e-prestacao-de-contas'")
	}
}

// TestFetch_FromFile_MissingFile verifies error when file does not exist.
func TestFetch_FromFile_MissingFile(t *testing.T) {
	src := sitemap.New(sitemap.Options{FromFile: "/nonexistent/file.xml"})
	_, err := src.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "sitemap") && !strings.Contains(msg, "no such file") {
		t.Errorf("error %q does not contain 'sitemap' or 'no such file'", msg)
	}
}

// TestFetch_NoURLNoFile_Errors verifies error when neither URL nor FromFile is set.
func TestFetch_NoURLNoFile_Errors(t *testing.T) {
	src := sitemap.New(sitemap.Options{})
	_, err := src.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "URL or FromFile") {
		t.Errorf("error %q does not contain 'URL or FromFile'", err.Error())
	}
}

// TestFetch_FromHTTP_Synthetic tests fetching a synthetic plain-XML sitemap over HTTP.
func TestFetch_FromHTTP_Synthetic(t *testing.T) {
	const syntheticXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc><lastmod>2024-01-15</lastmod></url>
  <url><loc>https://example.com/page2</loc><lastmod>2024-02-20</lastmod></url>
  <url><loc>https://example.com/page3</loc></url>
</urlset>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(syntheticXML))
	}))
	defer srv.Close()

	src := sitemap.New(sitemap.Options{URL: srv.URL})
	entries, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3", len(entries))
	}
}

// TestFetch_FromHTTP_Non2xx_Errors verifies error on non-2xx HTTP response.
func TestFetch_FromHTTP_Non2xx_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src := sitemap.New(sitemap.Options{URL: srv.URL})
	_, err := src.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("error %q does not contain 'unexpected status'", err.Error())
	}
}

// TestFetch_FromHTTP_MalformedXML verifies error on malformed XML response.
func TestFetch_FromHTTP_MalformedXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<not valid xml"))
	}))
	defer srv.Close()

	src := sitemap.New(sitemap.Options{URL: srv.URL})
	_, err := src.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

// TestFetch_FromHTTP_GzippedSynthetic tests that gzip-compressed response bodies are handled.
func TestFetch_FromHTTP_GzippedSynthetic(t *testing.T) {
	const syntheticXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/gz1</loc><lastmod>2024-03-10T12:00:00Z</lastmod></url>
  <url><loc>https://example.com/gz2</loc></url>
</urlset>`

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte(syntheticXML))
	_ = gw.Close()
	gzBody := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gzBody)
	}))
	defer srv.Close()

	src := sitemap.New(sitemap.Options{URL: srv.URL})
	entries, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
	if entries[0].Loc != "https://example.com/gz1" {
		t.Errorf("unexpected first loc: %s", entries[0].Loc)
	}
	if entries[0].LastMod.IsZero() {
		t.Error("expected non-zero LastMod for first entry")
	}
}

// TestFetch_UserAgent verifies that the User-Agent header is sent correctly.
func TestFetch_UserAgent(t *testing.T) {
	const wantUA = "test-crawler/1.0"
	gotUA := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
	}))
	defer srv.Close()

	src := sitemap.New(sitemap.Options{URL: srv.URL, UserAgent: wantUA})
	_, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUA != wantUA {
		t.Errorf("got User-Agent %q, want %q", gotUA, wantUA)
	}
}
