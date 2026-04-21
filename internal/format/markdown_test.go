package format_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
	"github.com/bergmaia/site-research/internal/format"
)

var update = flag.Bool("update", false, "regenerate golden files")

// golden reads the golden file or, when -update is set, writes it and returns the content.
func golden(t *testing.T, name string, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", name, err)
		}
		t.Logf("updated golden %s", name)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", name, err)
	}
	if string(want) != got {
		t.Errorf("output mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}

// ---- fixtures ---------------------------------------------------------------

func makeHits() []ports.SearchHit {
	return []ports.SearchHit{
		{
			Path:        "contabilidade/balancetes",
			URL:         "https://www.example.com/transparencia/contabilidade/balancetes",
			Title:       "Balancetes",
			MiniSummary: "Balancetes contábeis mensais do TRE-PI.",
			Score:       3.5,
			Section:     "Contabilidade",
		},
		{
			Path:        "contabilidade/balancetes/2025",
			URL:         "https://www.example.com/transparencia/contabilidade/balancetes/2025",
			Title:       "Balancetes 2025",
			MiniSummary: "Demonstrativos contábeis do exercício de 2025.",
			Score:       2.8,
			Section:     "Contabilidade",
		},
		{
			Path:        "recursos-humanos/diarias",
			URL:         "https://www.example.com/transparencia/recursos-humanos/diarias",
			Title:       "Diárias de Servidores",
			MiniSummary: "Pagamentos de diárias a servidores e magistrados.",
			Score:       1.2,
			Section:     "Recursos Humanos",
		},
	}
}

func makePage() *domain.Page {
	extractedAt, _ := time.Parse("2006-01-02T15:04:05Z", "2026-04-20T18:00:00Z")
	return &domain.Page{
		URL:      "https://www.example.com/transparencia/contabilidade/balancetes",
		Title:    "Balancetes",
		Section:  "Contabilidade",
		PageType: domain.PageTypeLanding,
		PathTitles: []string{
			"Transparência e Prestação de Contas",
			"Contabilidade",
			"Balancetes",
		},
		MiniSummary: domain.MiniSummary{
			Text: "Índice dos balancetes contábeis do TRE-PI, organizados por exercício.",
		},
		Links: domain.Links{
			Children: []domain.ChildLink{
				{Title: "Balancetes 2025", URL: "https://www.example.com/transparencia/contabilidade/balancetes/2025"},
				{Title: "Balancetes 2024", URL: "https://www.example.com/transparencia/contabilidade/balancetes/2024"},
			},
		},
		Documents: []domain.Document{
			{Title: "Balancete Março 2026", Type: "pdf", URL: "https://www.example.com/transparencia/arquivo/balancete-marco-2026.pdf"},
		},
		Metadata: domain.Metadata{
			Depth:          2,
			ExtractedAt:    extractedAt,
			CrawlerVersion: "0.1.0",
			DiscoveredVia:  domain.DiscoveredViaSitemap,
		},
	}
}

func makeStats() app.StatsReport {
	return app.StatsReport{
		GeneratedAt:             "2026-04-20 18:00:00 UTC",
		RootURL:                 "https://www.example.com/transparencia",
		SchemaVersion:           2,
		TotalPages:              100,
		PagesWithoutMiniSummary: 5,
		PagesWithDocs:           30,
		TotalDocuments:          120,
		StalePages:              2,
		ByDepth: map[int]int{
			2: 10,
			3: 50,
			4: 40,
		},
		ByPageType: map[string]int{
			"landing": 10,
			"article": 70,
			"listing": 15,
			"empty":   5,
		},
		TopSections: []app.SectionCount{
			{Section: "Contabilidade", Count: 40},
			{Section: "Recursos Humanos", Count: 30},
			{Section: "Estratégia", Count: 30},
		},
	}
}

// ---- tests ------------------------------------------------------------------

func TestRenderSearchHits_Basic(t *testing.T) {
	hits := makeHits()
	got := format.RenderSearchHits("balancetes 2025", hits, len(hits), 10, "")
	golden(t, "search_basic.md", got)
}

func TestRenderSearchHits_Empty(t *testing.T) {
	got := format.RenderSearchHits("xpto irrelevante", nil, 0, 10, "")
	golden(t, "search_empty.md", got)
}

func TestRenderSearchHits_WithSectionFilter(t *testing.T) {
	// Only hits from Contabilidade section
	hits := makeHits()[:2]
	got := format.RenderSearchHits("balancetes", hits, 3, 10, "Contabilidade")
	golden(t, "search_with_section_filter.md", got)
}

func TestRenderPage(t *testing.T) {
	page := makePage()
	got := format.RenderPage(page)
	golden(t, "inspect_page.md", got)
}

func TestRenderStats(t *testing.T) {
	r := makeStats()
	got := format.RenderStats(r)
	golden(t, "stats.md", got)
}

// TestRenderSearchHits_Determinism verifies that the same input always produces
// the same output (no time.Now() calls).
func TestRenderSearchHits_Determinism(t *testing.T) {
	hits := makeHits()
	a := format.RenderSearchHits("balancetes", hits, len(hits), 10, "")
	b := format.RenderSearchHits("balancetes", hits, len(hits), 10, "")
	if a != b {
		t.Error("RenderSearchHits is not deterministic")
	}
}

// TestRenderStats_Determinism verifies RenderStats is deterministic.
func TestRenderStats_Determinism(t *testing.T) {
	r := makeStats()
	a := format.RenderStats(r)
	b := format.RenderStats(r)
	if a != b {
		t.Error("RenderStats is not deterministic")
	}
}

// TestTruncateLong verifies that a long mini-summary is truncated.
func TestTruncateLong(t *testing.T) {
	longText := ""
	for i := 0; i < 600; i++ {
		longText += "a"
	}
	hits := []ports.SearchHit{
		{
			URL:         "https://example.com/p",
			Title:       "Page",
			MiniSummary: longText,
			Section:     "S",
		},
	}
	got := format.RenderSearchHits("test", hits, 1, 10, "")
	// The rendered text should not contain 600 a's.
	if len(longText) > 500 && len(got) > 600 {
		// It was truncated if "mais)" appears in the output.
		if !containsStr(got, "mais)") {
			t.Error("expected truncation indicator in output")
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
