// Package sqlitefts is an adapter that persists a full-text searchable index
// of the page catalog into an embedded SQLite database using FTS5.
// It uses modernc.org/sqlite (pure Go) so the binary works with CGO_ENABLED=0.
package sqlitefts

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/bergmaia/site-research/internal/domain/ports"
)

// ensure Store implements ports.SearchIndex at compile time.
var _ ports.SearchIndex = (*Store)(nil)

// Options configures a Store.
type Options struct {
	// Path is the filesystem path of the SQLite file (e.g., "./data/catalog.sqlite").
	Path string
}

// Store implements ports.SearchIndex backed by an SQLite FTS5 table.
type Store struct {
	path string
	db   *sql.DB
}

// Open opens or creates the SQLite database at Options.Path.
// Callers MUST call Close when done.
func Open(opts Options) (*Store, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("sqlitefts: Options.Path must not be empty")
	}

	// Use the modernc pure-Go driver registered under the "sqlite" driver name.
	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("sqlitefts: open %q: %w", opts.Path, err)
	}

	// Verify the connection is live.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlitefts: ping %q: %w", opts.Path, err)
	}

	return &Store{path: opts.Path, db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("sqlitefts: close: %w", err)
	}
	return nil
}
