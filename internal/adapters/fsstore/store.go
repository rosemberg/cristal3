// Package fsstore implements ports.PageStore as a hierarchical filesystem tree
// mirroring the URL path structure under scope. Each page lives at
// <rootDir>/<segment1>/<segment2>/.../_index.json.
//
// The package guarantees:
//   - Writes are atomic via write-then-rename (RNF-05).
//   - JSON is pretty-printed with 2-space indentation for inspection.
//   - URLs outside the configured scope prefix return an error on Put/Get/Delete.
package fsstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bergmaia/site-research/internal/domain"
)

// ErrOutOfScope is returned when a URL falls outside the store's scope prefix.
var ErrOutOfScope = errors.New("fsstore: URL out of scope")

// Store implements ports.PageStore.
type Store struct {
	rootDir string
	prefix  string // canonical scope prefix (no trailing slash)
}

// Options configures a new Store.
type Options struct {
	// RootDir is the directory under which the tree is rooted (e.g., "./data").
	RootDir string
	// ScopePrefix is the URL prefix defining what's in scope (e.g.,
	// "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas").
	// Trailing slash, if present, is trimmed for normalization.
	ScopePrefix string
}

// New builds a Store. RootDir is created if missing.
func New(opts Options) (*Store, error) {
	if opts.RootDir == "" {
		return nil, fmt.Errorf("fsstore: RootDir must not be empty")
	}
	if opts.ScopePrefix == "" {
		return nil, fmt.Errorf("fsstore: ScopePrefix must not be empty")
	}

	prefix := strings.TrimRight(opts.ScopePrefix, "/")

	if err := os.MkdirAll(opts.RootDir, 0o755); err != nil {
		return nil, fmt.Errorf("fsstore: creating root dir: %w", err)
	}

	absRoot, err := filepath.Abs(opts.RootDir)
	if err != nil {
		return nil, fmt.Errorf("fsstore: resolving root dir: %w", err)
	}

	return &Store{
		rootDir: absRoot,
		prefix:  prefix,
	}, nil
}

// segments derives the path segments from the URL relative to the store prefix.
// Returns an empty slice for the root URL (url == prefix).
// Returns ErrOutOfScope if the URL is not under the prefix.
// Returns an error if any segment is unsafe (path traversal protection).
func (s *Store) segments(url string) ([]string, error) {
	u := strings.TrimRight(url, "/")

	if u == s.prefix {
		return []string{}, nil
	}

	if !strings.HasPrefix(u, s.prefix+"/") {
		return nil, ErrOutOfScope
	}

	relative := u[len(s.prefix)+1:]
	parts := strings.Split(relative, "/")

	for _, seg := range parts {
		if seg == "" || seg == ".." || strings.Contains(seg, "/") || strings.HasPrefix(seg, ".") {
			return nil, fmt.Errorf("fsstore: unsafe path segment %q", seg)
		}
	}

	return parts, nil
}

// Path returns the absolute filesystem path of the _index.json for the given URL.
// Returns ErrOutOfScope if the URL is not under the configured prefix.
// Exposed for debugging/inspection.
func (s *Store) Path(url string) (string, error) {
	segs, err := s.segments(url)
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, len(segs)+2)
	parts = append(parts, s.rootDir)
	parts = append(parts, segs...)
	parts = append(parts, "_index.json")

	return filepath.Join(parts...), nil
}

// Put writes the page to <path>/_index.json atomically (write-then-rename).
// Creates parent directories as needed.
func (s *Store) Put(ctx context.Context, page *domain.Page) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := s.Path(page.URL)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(page, "", "  ")
	if err != nil {
		return fmt.Errorf("fsstore: marshaling page: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("fsstore: creating directory: %w", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("fsstore: writing temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("fsstore: renaming temp file: %w", err)
	}

	return nil
}

// Get reads and unmarshals the _index.json for the URL.
// Returns os.ErrNotExist (wrapped) if the page is absent.
func (s *Store) Get(ctx context.Context, url string) (*domain.Page, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path, err := s.Path(url)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("fsstore: page not found for %q: %w", url, os.ErrNotExist)
		}
		return nil, fmt.Errorf("fsstore: reading file: %w", err)
	}

	var page domain.Page
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("fsstore: unmarshaling page: %w", err)
	}

	return &page, nil
}

// Walk visits every page in the tree. The walk is deterministic (lexicographic).
// Returning an error from fn stops the walk and is propagated.
// ctx cancellation stops the walk promptly between files.
func (s *Store) Walk(ctx context.Context, fn func(p *domain.Page) error) error {
	return filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation at the start of each iteration.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		if d.IsDir() || d.Name() != "_index.json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("fsstore: reading %s: %w", path, err)
		}

		var page domain.Page
		if err := json.Unmarshal(data, &page); err != nil {
			return fmt.Errorf("fsstore: unmarshaling %s: %w", path, err)
		}

		return fn(&page)
	})
}

// Delete removes the page and its parent directory IF the directory becomes empty
// after removing _index.json (don't recursively delete subdirectories — they may
// hold child pages).
func (s *Store) Delete(ctx context.Context, url string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := s.Path(url)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("fsstore: page not found for %q: %w", url, os.ErrNotExist)
		}
		return fmt.Errorf("fsstore: removing file: %w", err)
	}

	// Attempt to remove parent directory only if it becomes empty.
	dir := filepath.Dir(path)
	// Do not attempt to remove rootDir itself.
	if dir != s.rootDir {
		entries, err := os.ReadDir(dir)
		if err == nil && len(entries) == 0 {
			// Best-effort removal of empty directory; ignore error.
			_ = os.Remove(dir)
		}
	}

	return nil
}
