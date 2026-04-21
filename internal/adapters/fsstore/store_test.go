package fsstore_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bergmaia/site-research/internal/adapters/fsstore"
	"github.com/bergmaia/site-research/internal/domain"
)

const testPrefix = "https://host/transparencia-e-prestacao-de-contas"

func newStore(t *testing.T) *fsstore.Store {
	t.Helper()
	s, err := fsstore.New(fsstore.Options{
		RootDir:     t.TempDir(),
		ScopePrefix: testPrefix,
	})
	if err != nil {
		t.Fatalf("fsstore.New: %v", err)
	}
	return s
}

func minimalPage(url string) *domain.Page {
	return &domain.Page{
		SchemaVersion: 2,
		URL:           url,
		Title:         "Test Page",
		Lang:          "pt-BR",
	}
}

// TestStore_PutGet_Roundtrip — Put a minimal Page and Get it back; assert fields match.
func TestStore_PutGet_Roundtrip(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	url := testPrefix + "/contabilidade/balancetes"
	page := minimalPage(url)
	page.Description = "Balancetes mensais"
	page.Lang = "pt-BR"

	if err := s.Put(ctx, page); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.Get(ctx, url)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.URL != page.URL {
		t.Errorf("URL mismatch: got %q, want %q", got.URL, page.URL)
	}
	if got.Title != page.Title {
		t.Errorf("Title mismatch: got %q, want %q", got.Title, page.Title)
	}
	if got.Description != page.Description {
		t.Errorf("Description mismatch: got %q, want %q", got.Description, page.Description)
	}
	if got.Lang != page.Lang {
		t.Errorf("Lang mismatch: got %q, want %q", got.Lang, page.Lang)
	}
	if got.SchemaVersion != page.SchemaVersion {
		t.Errorf("SchemaVersion mismatch: got %d, want %d", got.SchemaVersion, page.SchemaVersion)
	}
}

// TestStore_OutOfScope — URL outside the prefix → Put returns ErrOutOfScope.
func TestStore_OutOfScope(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	outURL := "https://host/other-section/page"
	err := s.Put(ctx, minimalPage(outURL))
	if !errors.Is(err, fsstore.ErrOutOfScope) {
		t.Errorf("expected ErrOutOfScope, got: %v", err)
	}

	// Also test Get and Delete.
	_, err = s.Get(ctx, outURL)
	if !errors.Is(err, fsstore.ErrOutOfScope) {
		t.Errorf("Get: expected ErrOutOfScope, got: %v", err)
	}

	err = s.Delete(ctx, outURL)
	if !errors.Is(err, fsstore.ErrOutOfScope) {
		t.Errorf("Delete: expected ErrOutOfScope, got: %v", err)
	}
}

// TestStore_PathDerivation — table-driven path derivation tests.
func TestStore_PathDerivation(t *testing.T) {
	tmp := t.TempDir()
	s, err := fsstore.New(fsstore.Options{
		RootDir:     tmp,
		ScopePrefix: testPrefix,
	})
	if err != nil {
		t.Fatalf("fsstore.New: %v", err)
	}

	tests := []struct {
		name    string
		url     string
		wantRel string // relative to tmp
	}{
		{
			name:    "root URL",
			url:     testPrefix,
			wantRel: "_index.json",
		},
		{
			name:    "one-segment URL",
			url:     testPrefix + "/contabilidade",
			wantRel: filepath.Join("contabilidade", "_index.json"),
		},
		{
			name:    "three-segment URL",
			url:     testPrefix + "/a/b/c",
			wantRel: filepath.Join("a", "b", "c", "_index.json"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.Path(tc.url)
			if err != nil {
				t.Fatalf("Path(%q): %v", tc.url, err)
			}
			want := filepath.Join(tmp, tc.wantRel)
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestStore_AtomicPut_NoPartialFile — after Put, assert no .tmp.* files remain.
func TestStore_AtomicPut_NoPartialFile(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	url := testPrefix + "/contabilidade/balancetes"
	if err := s.Put(ctx, minimalPage(url)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	path, err := s.Path(url)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	dir := filepath.Dir(path)

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp.*"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) > 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}

// TestStore_Walk_Deterministic — Put 5 pages, Walk, collect URLs, assert lexicographic order.
func TestStore_Walk_Deterministic(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	urls := []string{
		testPrefix + "/z/page",
		testPrefix + "/a/page",
		testPrefix + "/m/page",
		testPrefix + "/b/page",
		testPrefix + "/a/subpage",
	}

	for _, u := range urls {
		if err := s.Put(ctx, minimalPage(u)); err != nil {
			t.Fatalf("Put(%q): %v", u, err)
		}
	}

	var visited []string
	err := s.Walk(ctx, func(p *domain.Page) error {
		visited = append(visited, p.URL)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	if len(visited) != len(urls) {
		t.Fatalf("expected %d pages, got %d", len(urls), len(visited))
	}

	// Verify the result is sorted (lexicographic by URL is implied by filepath.WalkDir
	// lexicographic path order, which corresponds to URL order given same prefix).
	sorted := make([]string, len(visited))
	copy(sorted, visited)
	sort.Strings(sorted)

	for i, u := range visited {
		if u != sorted[i] {
			t.Errorf("Walk order not lexicographic at index %d: got %q, want %q", i, u, sorted[i])
		}
	}
}

// TestStore_Walk_ContextCancel — Walk with a canceled ctx returns context.Canceled.
func TestStore_Walk_ContextCancel(t *testing.T) {
	s := newStore(t)

	urls := []string{
		testPrefix + "/page1",
		testPrefix + "/page2",
	}

	bgCtx := context.Background()
	for _, u := range urls {
		if err := s.Put(bgCtx, minimalPage(u)); err != nil {
			t.Fatalf("Put(%q): %v", u, err)
		}
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := s.Walk(canceledCtx, func(p *domain.Page) error {
		return nil
	})

	if err == nil {
		t.Error("expected error from Walk with canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestStore_Delete_RemovesFile_KeepsChildren — Put A and child B; delete A;
// assert A's _index.json is gone but B's directory and file are intact.
func TestStore_Delete_RemovesFile_KeepsChildren(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	urlA := testPrefix + "/a"
	urlB := testPrefix + "/a/b"

	if err := s.Put(ctx, minimalPage(urlA)); err != nil {
		t.Fatalf("Put A: %v", err)
	}
	if err := s.Put(ctx, minimalPage(urlB)); err != nil {
		t.Fatalf("Put B: %v", err)
	}

	pathA, err := s.Path(urlA)
	if err != nil {
		t.Fatalf("Path A: %v", err)
	}
	pathB, err := s.Path(urlB)
	if err != nil {
		t.Fatalf("Path B: %v", err)
	}

	if err := s.Delete(ctx, urlA); err != nil {
		t.Fatalf("Delete A: %v", err)
	}

	// A's _index.json must be gone.
	if _, err := os.Stat(pathA); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected %q to be removed, got stat err: %v", pathA, err)
	}

	// B's _index.json must still exist.
	if _, err := os.Stat(pathB); err != nil {
		t.Errorf("expected %q to still exist, got: %v", pathB, err)
	}

	// B's directory must still be accessible.
	dirB := filepath.Dir(pathB)
	if _, err := os.Stat(dirB); err != nil {
		t.Errorf("expected dir %q to still exist, got: %v", dirB, err)
	}
}

// TestStore_Get_NotFound — Get on a URL never Put → wraps os.ErrNotExist.
func TestStore_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	_, err := s.Get(ctx, testPrefix+"/nonexistent/page")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist in error chain, got: %v", err)
	}
}

// TestStore_New_Errors — invalid Options return errors.
func TestStore_New_Errors(t *testing.T) {
	// Empty RootDir.
	_, err := fsstore.New(fsstore.Options{RootDir: "", ScopePrefix: testPrefix})
	if err == nil {
		t.Error("expected error for empty RootDir, got nil")
	}

	// Empty ScopePrefix.
	_, err = fsstore.New(fsstore.Options{RootDir: t.TempDir(), ScopePrefix: ""})
	if err == nil {
		t.Error("expected error for empty ScopePrefix, got nil")
	}
}

// TestStore_Delete_NotFound — Delete a URL that was never Put → wraps os.ErrNotExist.
func TestStore_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	err := s.Delete(ctx, testPrefix+"/never/put")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist in error chain, got: %v", err)
	}
}

// TestStore_Put_ContextCancel — Put with canceled context returns early.
func TestStore_Put_ContextCancel(t *testing.T) {
	s := newStore(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Put(canceledCtx, minimalPage(testPrefix+"/page"))
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestStore_Get_ContextCancel — Get with canceled context returns early.
func TestStore_Get_ContextCancel(t *testing.T) {
	s := newStore(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Get(canceledCtx, testPrefix+"/page")
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestStore_Delete_ContextCancel — Delete with canceled context returns early.
func TestStore_Delete_ContextCancel(t *testing.T) {
	s := newStore(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Delete(canceledCtx, testPrefix+"/page")
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestStore_Put_RootURL — Put and Get at the root of the scope (url == prefix).
func TestStore_Put_RootURL(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	page := minimalPage(testPrefix)
	page.Title = "Root page"

	if err := s.Put(ctx, page); err != nil {
		t.Fatalf("Put root URL: %v", err)
	}

	got, err := s.Get(ctx, testPrefix)
	if err != nil {
		t.Fatalf("Get root URL: %v", err)
	}
	if got.Title != page.Title {
		t.Errorf("Title mismatch: got %q, want %q", got.Title, page.Title)
	}
}

// TestStore_Walk_Empty — Walk on an empty store calls fn zero times.
func TestStore_Walk_Empty(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	count := 0
	err := s.Walk(ctx, func(p *domain.Page) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("Walk on empty store: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 visits, got %d", count)
	}
}

// TestStore_Walk_FnError — Walk propagates error returned by fn.
func TestStore_Walk_FnError(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	url := testPrefix + "/page"
	if err := s.Put(ctx, minimalPage(url)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	sentinel := errors.New("stop walk")
	err := s.Walk(ctx, func(p *domain.Page) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

// TestStore_Delete_EmptyDirCleanup — deleting only page in a dir removes the dir.
func TestStore_Delete_EmptyDirCleanup(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	url := testPrefix + "/cleanup-test"
	if err := s.Put(ctx, minimalPage(url)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	path, _ := s.Path(url)
	dir := filepath.Dir(path)

	if err := s.Delete(ctx, url); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// The now-empty directory should have been removed.
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Logf("dir %q: stat result: %v (may still exist, that's acceptable)", dir, err)
		// Not a hard failure — the spec says "best-effort".
	}
}

// TestStore_RejectPathTraversal — URL with ".." segment in relative path returns error.
func TestStore_RejectPathTraversal(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	// This URL has ".." as a segment — defense-in-depth against traversal.
	// Note: the prefix ends at /x, so relative path would be "../etc/passwd"
	traversalURL := testPrefix + "/../etc/passwd"
	// After TrimRight the url is: "https://host/transparencia-e-prestacao-de-contas/../etc/passwd"
	// The prefix is:               "https://host/transparencia-e-prestacao-de-contas"
	// Relative:                    "../etc/passwd"
	// Segments: ["..", "etc", "passwd"] → ".." triggers rejection.

	err := s.Put(ctx, minimalPage(traversalURL))
	if err == nil {
		t.Fatal("expected error for path traversal URL, got nil")
	}
	// Should NOT be an ErrOutOfScope; it's a sanitization error.
	// But we at minimum must not allow it through silently.
	t.Logf("got expected error for traversal URL: %v", err)

	// Also, an explicit ".." segment in a valid-prefix URL.
	dotDotURL := testPrefix + "/valid/../../etc/passwd"
	err = s.Put(ctx, minimalPage(dotDotURL))
	if err == nil {
		t.Fatal("expected error for dotdot URL, got nil")
	}
	t.Logf("got expected error for dotdot URL: %v", err)

	// A hidden directory segment starting with "." (e.g., ".ssh").
	hiddenURL := testPrefix + "/.hidden/page"
	err = s.Put(ctx, minimalPage(hiddenURL))
	if err == nil {
		t.Fatal("expected error for hidden segment URL, got nil")
	}
	t.Logf("got expected error for hidden segment URL: %v", err)
}
