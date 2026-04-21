package sqlitefts

import (
	"context"
	"fmt"
	"strings"

	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

// derivePath returns the path component of pageURL relative to rootURL.
// It strips the rootURL prefix and any leading slash, falling back to pageURL
// when rootURL is not a prefix (defensive).
func derivePath(rootURL, pageURL string) string {
	if rootURL != "" && strings.HasPrefix(pageURL, rootURL) {
		rel := strings.TrimPrefix(pageURL, rootURL)
		return strings.TrimPrefix(rel, "/")
	}
	return pageURL
}

// Rebuild drops and re-creates the pages_fts FTS5 table, then bulk-inserts
// rows derived from pages. It is idempotent — safe to call repeatedly.
// A single transaction wraps the bulk insert.
func (s *Store) Rebuild(ctx context.Context, catalog *domain.Catalog, pages []*domain.Page) error {
	const dropSQL = `DROP TABLE IF EXISTS pages_fts;`
	const createSQL = `CREATE VIRTUAL TABLE pages_fts USING fts5(
  path UNINDEXED,
  url UNINDEXED,
  title,
  mini_summary,
  full_text,
  section UNINDEXED,
  page_type UNINDEXED,
  tokenize = "unicode61 remove_diacritics 2"
);`

	if _, err := s.db.ExecContext(ctx, dropSQL); err != nil {
		return fmt.Errorf("sqlitefts: drop pages_fts: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("sqlitefts: create pages_fts: %w", err)
	}

	if len(pages) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlitefts: begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insertSQL = `INSERT INTO pages_fts
	(path, url, title, mini_summary, full_text, section, page_type)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("sqlitefts: prepare insert: %w", err)
	}
	defer stmt.Close()

	rootURL := ""
	if catalog != nil {
		rootURL = catalog.RootURL
	}

	for _, p := range pages {
		path := derivePath(rootURL, p.URL)
		_, err = stmt.ExecContext(ctx,
			path,
			p.URL,
			p.Title,
			p.MiniSummary.Text,
			p.Content.FullText,
			p.Section,
			string(p.PageType),
		)
		if err != nil {
			return fmt.Errorf("sqlitefts: insert page %q: %w", p.URL, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("sqlitefts: commit: %w", err)
	}
	return nil
}

// sanitizeFTSQuery strips FTS5 syntax characters so natural-language input
// (from LLMs or humans) is treated as plain keywords joined by implicit AND.
// Without this, inputs like "TRE-PI" or "diarias-marco-2026" trigger FTS5's
// NOT/column-filter grammar and surface as SQL errors to the caller.
func sanitizeFTSQuery(q string) string {
	// Chars with special meaning in FTS5 query grammar:
	//   "  phrase delimiter
	//   -  NOT operator
	//   +  reserved
	//   *  prefix operator
	//   (  group begin
	//   )  group end
	//   :  column filter
	//   ^  initial-token anchor
	r := strings.NewReplacer(
		`"`, " ",
		`-`, " ",
		`+`, " ",
		`*`, " ",
		`(`, " ",
		`)`, " ",
		`:`, " ",
		`^`, " ",
	)
	return strings.Join(strings.Fields(r.Replace(q)), " ")
}

// Search executes an FTS5 MATCH query and returns the top-limit hits.
// Ranking uses the FTS5 built-in bm25 function. Lower bm25 values indicate
// better matches; the returned Score inverts that (Score = -bm25) so that
// higher Score means a better match. If query is the empty string (or
// becomes empty after FTS5 syntax sanitization), Search returns (nil, nil).
func (s *Store) Search(ctx context.Context, query string, limit int) ([]ports.SearchHit, error) {
	if query == "" {
		return nil, nil
	}
	cleaned := sanitizeFTSQuery(query)
	if cleaned == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	const searchSQL = `SELECT path, url, title, mini_summary, section, bm25(pages_fts) AS rank
FROM pages_fts
WHERE pages_fts MATCH ?
ORDER BY rank
LIMIT ?`

	rows, err := s.db.QueryContext(ctx, searchSQL, cleaned, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlitefts: search %q: %w", query, err)
	}
	defer rows.Close()

	var hits []ports.SearchHit
	for rows.Next() {
		// Honour context cancellation between rows so long scans are interruptible.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var h ports.SearchHit
		var rank float64
		if err := rows.Scan(&h.Path, &h.URL, &h.Title, &h.MiniSummary, &h.Section, &rank); err != nil {
			return nil, fmt.Errorf("sqlitefts: scan row: %w", err)
		}
		h.Score = -rank
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlitefts: rows error: %w", err)
	}
	return hits, nil
}
