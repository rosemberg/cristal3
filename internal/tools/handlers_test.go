package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/adapters/sqlitefts"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/logging"
	"github.com/bergmaia/site-research/internal/tools"
)

// buildTestFixture creates a temporary catalog for handler tests.
// Returns a *config.Config pointing to the fixture.
func buildTestFixture(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()

	scopePrefix := "https://www.example.com/transparencia"
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	// 1. Write _index.json pages via fsstore.
	store, err := fsstore.New(fsstore.Options{
		RootDir:     dataDir,
		ScopePrefix: scopePrefix,
	})
	if err != nil {
		t.Fatalf("fsstore.New: %v", err)
	}

	now := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	pages := []*domain.Page{
		{
			URL:      scopePrefix + "/contabilidade/balancetes",
			Title:    "Balancetes",
			Section:  "Contabilidade",
			PageType: domain.PageTypeLanding,
			PathTitles: []string{
				"Transparência e Prestação de Contas",
				"Contabilidade",
				"Balancetes",
			},
			MiniSummary: domain.MiniSummary{Text: "Balancetes contábeis mensais."},
			Metadata:    domain.Metadata{Depth: 2, ExtractedAt: now, CrawlerVersion: "0.1.0"},
			Links: domain.Links{
				Children: []domain.ChildLink{
					{Title: "Balancetes 2025", URL: scopePrefix + "/contabilidade/balancetes/2025"},
				},
			},
		},
		{
			URL:         scopePrefix + "/recursos-humanos/diarias",
			Title:       "Diárias",
			Section:     "Recursos Humanos",
			PageType:    domain.PageTypeArticle,
			MiniSummary: domain.MiniSummary{Text: "Pagamentos de diárias a servidores."},
			Metadata:    domain.Metadata{Depth: 2, ExtractedAt: now},
		},
	}

	ctx := context.Background()
	for _, p := range pages {
		if err := store.Put(ctx, p); err != nil {
			t.Fatalf("store.Put %s: %v", p.URL, err)
		}
	}

	// 2. Build catalog.json.
	cat := &domain.Catalog{
		GeneratedAt:   now,
		RootURL:       scopePrefix,
		SchemaVersion: 2,
		Stats: domain.CatalogStats{
			TotalPages: len(pages),
			ByDepth:    map[int]int{2: 2},
			ByPageType: map[domain.PageType]int{domain.PageTypeLanding: 1, domain.PageTypeArticle: 1},
		},
		Entries: []domain.CatalogEntry{
			{Path: "contabilidade/balancetes", URL: scopePrefix + "/contabilidade/balancetes", Title: "Balancetes", Section: "Contabilidade", PageType: domain.PageTypeLanding, MiniSummary: "Balancetes contábeis mensais.", Depth: 2},
			{Path: "recursos-humanos/diarias", URL: scopePrefix + "/recursos-humanos/diarias", Title: "Diárias", Section: "Recursos Humanos", PageType: domain.PageTypeArticle, MiniSummary: "Pagamentos de diárias a servidores.", Depth: 2},
		},
	}
	catalogData, _ := json.MarshalIndent(cat, "", "  ")
	catalogPath := filepath.Join(dir, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogData, 0o644); err != nil {
		t.Fatalf("write catalog.json: %v", err)
	}

	// 3. Build catalog.sqlite.
	dbPath := filepath.Join(dir, "catalog.sqlite")
	ftsStore, err := sqlitefts.Open(sqlitefts.Options{Path: dbPath})
	if err != nil {
		t.Fatalf("sqlitefts.Open: %v", err)
	}
	if err := ftsStore.Rebuild(ctx, cat, pages); err != nil {
		t.Fatalf("ftsStore.Rebuild: %v", err)
	}
	ftsStore.Close()

	return &config.Config{
		Scope: config.ScopeConfig{Prefix: scopePrefix},
		Storage: config.StorageConfig{
			CatalogPath: catalogPath,
			SQLitePath:  dbPath,
			DataDir:     dataDir,
		},
	}
}

func newRegistry(t *testing.T) (*tools.Registry, *config.Config) {
	t.Helper()
	cfg := buildTestFixture(t)
	logger := logging.New(logging.Config{Level: "error", Format: "json", Output: os.Stderr})
	return tools.DefaultRegistry(cfg, logger), cfg
}

// TestSearchHandler_Basic verifies a valid search returns markdown hits.
func TestSearchHandler_Basic(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("search")
	if !ok {
		t.Fatal("search handler not found")
	}

	args, _ := json.Marshal(map[string]any{"query": "balancetes"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "# Resultados para:") {
		t.Errorf("expected markdown header, got:\n%s", text)
	}
	if !strings.Contains(text, "Balancetes") {
		t.Errorf("expected Balancetes in results, got:\n%s", text)
	}
}

// TestSearchHandler_Empty verifies that zero hits returns a "Nenhum resultado" response (not isError).
func TestSearchHandler_Empty(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("search")
	if !ok {
		t.Fatal("search handler not found")
	}

	args, _ := json.Marshal(map[string]any{"query": "xpto_irrelevante_inexistente_12345"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("zero hits should not be isError, got: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "Nenhum resultado") {
		t.Errorf("expected 'Nenhum resultado' section, got:\n%s", text)
	}
}

// TestSearchHandler_SectionFilter verifies section filter works.
func TestSearchHandler_SectionFilter(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("search")
	if !ok {
		t.Fatal("search handler not found")
	}

	args, _ := json.Marshal(map[string]any{"query": "diarias", "section": "Recursos Humanos"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "Diárias") {
		t.Errorf("expected Diárias in results, got:\n%s", text)
	}
}

// TestSearchHandler_MissingQuery verifies that missing query returns isError.
func TestSearchHandler_MissingQuery(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("search")
	if !ok {
		t.Fatal("search handler not found")
	}

	args, _ := json.Marshal(map[string]any{})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for missing query")
	}
	if !strings.Contains(result.Content[0].Text, "**Erro:**") {
		t.Errorf("error message should start with **Erro:**, got: %s", result.Content[0].Text)
	}
}

// TestSearchHandler_InvalidArgs verifies malformed JSON args returns isError.
func TestSearchHandler_InvalidArgs(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("search")
	if !ok {
		t.Fatal("search handler not found")
	}

	result, err := h(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for invalid args")
	}
}

// TestInspectHandler_Valid verifies a valid target returns markdown.
func TestInspectHandler_Valid(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("inspect_page")
	if !ok {
		t.Fatal("inspect_page handler not found")
	}

	args, _ := json.Marshal(map[string]any{"target": "contabilidade/balancetes"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "# Balancetes") {
		t.Errorf("expected page title in markdown, got:\n%s", text)
	}
	if !strings.Contains(text, "## Metadados") {
		t.Errorf("expected Metadados section, got:\n%s", text)
	}
}

// TestInspectHandler_NotFound verifies that a missing page returns isError.
func TestInspectHandler_NotFound(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("inspect_page")
	if !ok {
		t.Fatal("inspect_page handler not found")
	}

	args, _ := json.Marshal(map[string]any{"target": "pagina/que/nao/existe"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for missing page")
	}
	if !strings.Contains(result.Content[0].Text, "não encontrada") {
		t.Errorf("expected 'não encontrada' message, got: %s", result.Content[0].Text)
	}
}

// TestInspectHandler_TraversalRejected verifies path traversal is rejected.
func TestInspectHandler_TraversalRejected(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("inspect_page")
	if !ok {
		t.Fatal("inspect_page handler not found")
	}

	args, _ := json.Marshal(map[string]any{"target": "../etc/passwd"})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for traversal target")
	}
}

// TestInspectHandler_MissingTarget verifies that missing target returns isError.
func TestInspectHandler_MissingTarget(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("inspect_page")
	if !ok {
		t.Fatal("inspect_page handler not found")
	}

	args, _ := json.Marshal(map[string]any{})
	result, err := h(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for missing target")
	}
}

// TestStatsHandler verifies catalog_stats returns markdown with required sections.
func TestStatsHandler(t *testing.T) {
	reg, _ := newRegistry(t)
	h, ok := reg.Handler("catalog_stats")
	if !ok {
		t.Fatal("catalog_stats handler not found")
	}

	result, err := h(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content[0].Text)
	}
	text := result.Content[0].Text
	for _, section := range []string{
		"# Catálogo site-research — estatísticas",
		"## Totais",
		"## Por profundidade",
		"## Por tipo de página",
	} {
		if !strings.Contains(text, section) {
			t.Errorf("expected section %q in output, got:\n%s", section, text)
		}
	}
}

// TestRegistryHasThreeTools verifies the registry exposes exactly 3 tools.
func TestRegistryHasThreeTools(t *testing.T) {
	reg, _ := newRegistry(t)
	if len(reg.Tools()) != 3 {
		t.Errorf("expected 3 tools, got %d", len(reg.Tools()))
	}
	names := map[string]bool{}
	for _, tool := range reg.Tools() {
		names[tool.Name] = true
	}
	for _, want := range []string{"search", "inspect_page", "catalog_stats"} {
		if !names[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}
