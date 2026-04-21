package catalog_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/adapters/catalog"
	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/domain"
)

const rootURL = "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"

func newStore(t *testing.T) *fsstore.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := fsstore.New(fsstore.Options{
		RootDir:     dir,
		ScopePrefix: rootURL,
	})
	if err != nil {
		t.Fatalf("fsstore.New: %v", err)
	}
	return st
}

func mkPage(url, title, section string, depth int, parentURL string, pt domain.PageType) *domain.Page {
	return &domain.Page{
		URL: url, CanonicalURL: url, Title: title, Section: section,
		PageType: pt, HasSubstantiveContent: pt == domain.PageTypeArticle,
		Content: domain.Content{ContentLength: 800, FullTextHash: "sha256:fake", ContentHash: "sha256:fake"},
		MiniSummary: domain.MiniSummary{Text: "mini " + title},
		Metadata:    domain.Metadata{Depth: depth, ParentURL: parentURL, ExtractedAt: time.Now()},
		Links:       domain.Links{},
	}
}

func newBuilder(t *testing.T, st *fsstore.Store) *catalog.Builder {
	t.Helper()
	b, err := catalog.New(catalog.Options{
		Store:   st,
		RootURL: rootURL,
		// Use real time.Now so that time.Since checks work correctly.
	})
	if err != nil {
		t.Fatalf("catalog.New: %v", err)
	}
	return b
}

func TestBuilder_BuildBasic(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	pages := []*domain.Page{
		mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding),
		mkPage(rootURL+"/licitacoes", "Licitações", "licitacoes", 1, rootURL, domain.PageTypeListing),
		mkPage(rootURL+"/contabilidade", "Contabilidade", "contabilidade", 1, rootURL, domain.PageTypeListing),
		mkPage(rootURL+"/contabilidade/balancetes", "Balancetes", "contabilidade", 2, rootURL+"/contabilidade", domain.PageTypeArticle),
		mkPage(rootURL+"/licitacoes/edital-001", "Edital 001", "licitacoes", 2, rootURL+"/licitacoes", domain.PageTypeArticle),
	}

	for _, p := range pages {
		if err := st.Put(ctx, p); err != nil {
			t.Fatalf("Put(%s): %v", p.URL, err)
		}
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := len(cat.Entries); got != 5 {
		t.Errorf("len(Entries) = %d; want 5", got)
	}

	// Entries must be sorted by Path.
	for i := 1; i < len(cat.Entries); i++ {
		if cat.Entries[i].Path < cat.Entries[i-1].Path {
			t.Errorf("entries not sorted: %q before %q", cat.Entries[i-1].Path, cat.Entries[i].Path)
		}
	}

	if cat.Stats.TotalPages != 5 {
		t.Errorf("TotalPages = %d; want 5", cat.Stats.TotalPages)
	}

	// ByDepth: depth 0→1, 1→2, 2→2
	wantByDepth := map[int]int{0: 1, 1: 2, 2: 2}
	for d, want := range wantByDepth {
		if got := cat.Stats.ByDepth[d]; got != want {
			t.Errorf("ByDepth[%d] = %d; want %d", d, got, want)
		}
	}

	// ByPageType: landing→1, listing→2, article→2
	wantByType := map[domain.PageType]int{
		domain.PageTypeLanding:  1,
		domain.PageTypeListing:  2,
		domain.PageTypeArticle:  2,
	}
	for pt, want := range wantByType {
		if got := cat.Stats.ByPageType[pt]; got != want {
			t.Errorf("ByPageType[%s] = %d; want %d", pt, got, want)
		}
	}

	if cat.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d; want 2", cat.SchemaVersion)
	}
	if cat.RootURL != rootURL {
		t.Errorf("RootURL = %q; want %q", cat.RootURL, rootURL)
	}
	if cat.GeneratedAt.IsZero() {
		t.Error("GeneratedAt is zero")
	}
	if time.Since(cat.GeneratedAt) > 5*time.Second {
		t.Errorf("GeneratedAt %v is too far in the past", cat.GeneratedAt)
	}
}

func TestBuilder_ChildCount(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	pages := []*domain.Page{
		mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding),
		mkPage(rootURL+"/a", "A", "a", 1, rootURL, domain.PageTypeListing),
		mkPage(rootURL+"/b", "B", "b", 1, rootURL, domain.PageTypeListing),
		mkPage(rootURL+"/c", "C", "c", 1, rootURL, domain.PageTypeListing),
		mkPage(rootURL+"/a/x", "AX", "a", 2, rootURL+"/a", domain.PageTypeArticle),
		mkPage(rootURL+"/a/y", "AY", "a", 2, rootURL+"/a", domain.PageTypeArticle),
	}

	for _, p := range pages {
		if err := st.Put(ctx, p); err != nil {
			t.Fatalf("Put(%s): %v", p.URL, err)
		}
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	byURL := make(map[string]domain.CatalogEntry, len(cat.Entries))
	for _, e := range cat.Entries {
		byURL[e.URL] = e
	}

	if got := byURL[rootURL].ChildCount; got != 3 {
		t.Errorf("root ChildCount = %d; want 3", got)
	}
	if got := byURL[rootURL+"/a"].ChildCount; got != 2 {
		t.Errorf("/a ChildCount = %d; want 2", got)
	}
	if got := byURL[rootURL+"/b"].ChildCount; got != 0 {
		t.Errorf("/b ChildCount = %d; want 0", got)
	}
	if got := byURL[rootURL+"/c"].ChildCount; got != 0 {
		t.Errorf("/c ChildCount = %d; want 0", got)
	}
}

func TestBuilder_PathRelativeToRoot(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	pages := []*domain.Page{
		mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding),
		mkPage(rootURL+"/contabilidade/balancetes", "Balancetes", "contabilidade", 2, rootURL+"/contabilidade", domain.PageTypeArticle),
	}

	for _, p := range pages {
		if err := st.Put(ctx, p); err != nil {
			t.Fatalf("Put(%s): %v", p.URL, err)
		}
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	byURL := make(map[string]domain.CatalogEntry, len(cat.Entries))
	for _, e := range cat.Entries {
		byURL[e.URL] = e
	}

	rootEntry, ok := byURL[rootURL]
	if !ok {
		t.Fatal("root entry not found")
	}
	if rootEntry.Path != "" {
		t.Errorf("root Path = %q; want empty string", rootEntry.Path)
	}

	subEntry, ok := byURL[rootURL+"/contabilidade/balancetes"]
	if !ok {
		t.Fatal("sub entry not found")
	}
	if subEntry.Path != "contabilidade/balancetes" {
		t.Errorf("sub Path = %q; want %q", subEntry.Path, "contabilidade/balancetes")
	}
	if strings.HasPrefix(subEntry.Path, "/") {
		t.Errorf("Path has leading slash: %q", subEntry.Path)
	}
}

func TestBuilder_HasDocs(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	withDocs := mkPage(rootURL+"/docs", "Docs", "docs", 1, rootURL, domain.PageTypeArticle)
	withDocs.Documents = []domain.Document{
		{Title: "Doc1", URL: rootURL + "/docs/file1.pdf", Type: "pdf"},
		{Title: "Doc2", URL: rootURL + "/docs/file2.pdf", Type: "pdf"},
		{Title: "Doc3", URL: rootURL + "/docs/file3.xlsx", Type: "xlsx"},
	}

	noDocs := mkPage(rootURL+"/nodocs", "No Docs", "nodocs", 1, rootURL, domain.PageTypeListing)

	for _, p := range []*domain.Page{withDocs, noDocs} {
		if err := st.Put(ctx, p); err != nil {
			t.Fatalf("Put(%s): %v", p.URL, err)
		}
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	byURL := make(map[string]domain.CatalogEntry, len(cat.Entries))
	for _, e := range cat.Entries {
		byURL[e.URL] = e
	}

	if !byURL[rootURL+"/docs"].HasDocs {
		t.Error("docs page: HasDocs = false; want true")
	}
	if byURL[rootURL+"/nodocs"].HasDocs {
		t.Error("nodocs page: HasDocs = true; want false")
	}
}

func TestBuilder_WriteFile(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	pages := []*domain.Page{
		mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding),
		mkPage(rootURL+"/licitacoes", "Licitações", "licitacoes", 1, rootURL, domain.PageTypeListing),
	}

	for _, p := range pages {
		if err := st.Put(ctx, p); err != nil {
			t.Fatalf("Put(%s): %v", p.URL, err)
		}
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "catalog.json")
	if err := b.WriteFile(ctx, cat, outPath); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var decoded domain.Catalog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SchemaVersion != 2 {
		t.Errorf("decoded SchemaVersion = %d; want 2", decoded.SchemaVersion)
	}
	if decoded.Stats.TotalPages != 2 {
		t.Errorf("decoded TotalPages = %d; want 2", decoded.Stats.TotalPages)
	}

	// Spot-check 2-space indent: at least one line must start with "  " (two spaces).
	lines := strings.Split(string(data), "\n")
	found2space := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			found2space = true
			break
		}
	}
	if !found2space {
		t.Error("expected 2-space indented JSON but found none")
	}
}

func TestBuilder_WriteFile_AtomicNoTmp(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	p := mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding)
	if err := st.Put(ctx, p); err != nil {
		t.Fatalf("Put: %v", err)
	}

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "catalog.json")
	if err := b.WriteFile(ctx, cat, outPath); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover .tmp file found: %s", e.Name())
		}
	}
}

func TestBuilder_EmptyStore(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	b := newBuilder(t, st)
	cat, err := b.Build(ctx)
	if err != nil {
		t.Fatalf("Build on empty store: %v", err)
	}
	if cat.Stats.TotalPages != 0 {
		t.Errorf("TotalPages = %d; want 0", cat.Stats.TotalPages)
	}
	if len(cat.Entries) != 0 {
		t.Errorf("len(Entries) = %d; want 0", len(cat.Entries))
	}
}

func TestBuilder_ContextCancel(t *testing.T) {
	st := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Put a page so Walk has something to iterate, then cancel before Build.
	p := mkPage(rootURL, "Root", "root", 0, "", domain.PageTypeLanding)
	if err := st.Put(context.Background(), p); err != nil {
		t.Fatalf("Put: %v", err)
	}

	cancel()

	b := newBuilder(t, st)
	_, err := b.Build(ctx)
	if err == nil {
		t.Error("Build with canceled ctx: expected error, got nil")
	}
}

func TestNew_MissingStore(t *testing.T) {
	_, err := catalog.New(catalog.Options{
		Store:   nil,
		RootURL: rootURL,
	})
	if err == nil {
		t.Error("expected error for nil Store")
	}
}

func TestNew_MissingRootURL(t *testing.T) {
	st := newStore(t)
	_, err := catalog.New(catalog.Options{
		Store:   st,
		RootURL: "",
	})
	if err == nil {
		t.Error("expected error for empty RootURL")
	}
}
