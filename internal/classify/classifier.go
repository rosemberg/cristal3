// Package classify implements the deterministic page-type classifier (RF-05).
// It inspects a (partially-populated) *domain.Page and returns the classified PageType.
//
// Precedence order:
//  1. redirect — HTTP status 3xx OR metadata.redirected_from set.
//  2. empty    — ContentLength < EmptyMaxContentLength AND no substantive children/docs.
//  3. listing  — ChildCount + DocumentCount exceed ListingMinItems AND few substantive paragraphs.
//  4. landing  — ContentLength < LandingMaxContentLength AND ratio links/text > LandingLinkTextRatio.
//  5. article  — default fallback (substantive content present).
package classify

import (
	"strings"

	"github.com/bergmaia/site-research/internal/domain"
)

// Config tunes the heuristic thresholds. Zero values trigger defaults via WithDefaults.
type Config struct {
	// EmptyMaxContentLength: if Content.ContentLength < this, and no children/docs/substantive links, classify as empty.
	// Default: 100.
	EmptyMaxContentLength int
	// LandingMaxContentLength: if content_length < this AND link/text ratio exceeds LandingLinkTextRatio, classify as landing.
	// Default: 500 (per BRIEF RF-05).
	LandingMaxContentLength int
	// LandingLinkTextRatio: chars-in-links / chars-in-text threshold above which a short page is a landing.
	// Default: 0.5 — i.e., anchors account for at least half the visible characters.
	LandingLinkTextRatio float64
	// ListingMinItems: minimum number of (Children + Documents) entries to qualify as listing.
	// Default: 8.
	ListingMinItems int
	// ListingMaxSubstantiveParagraphs: max number of paragraphs > 160 chars to still qualify as listing
	// (a listing may have a short intro; too many paragraphs means it's an article instead).
	// Default: 2.
	ListingMaxSubstantiveParagraphs int
}

// WithDefaults returns a copy with zero fields replaced by the default values above.
func (c Config) WithDefaults() Config {
	if c.EmptyMaxContentLength == 0 {
		c.EmptyMaxContentLength = 100
	}
	if c.LandingMaxContentLength == 0 {
		c.LandingMaxContentLength = 500
	}
	if c.LandingLinkTextRatio == 0 {
		c.LandingLinkTextRatio = 0.5
	}
	if c.ListingMinItems == 0 {
		c.ListingMinItems = 8
	}
	if c.ListingMaxSubstantiveParagraphs == 0 {
		c.ListingMaxSubstantiveParagraphs = 2
	}
	return c
}

// Classifier implements ports.PageClassifier.
type Classifier struct{ cfg Config }

// New returns a Classifier wired with the given config (zero-fields → defaults).
func New(cfg Config) *Classifier {
	return &Classifier{cfg: cfg.WithDefaults()}
}

// Classify returns the PageType for p, applying the precedence rules above.
// It does NOT mutate p. It does NOT populate HasSubstantiveContent (the caller does that).
func (c *Classifier) Classify(p *domain.Page) domain.PageType {
	// 1. redirect
	if p.Metadata.HTTPStatus >= 300 && p.Metadata.HTTPStatus < 400 {
		return domain.PageTypeRedirect
	}
	if p.Metadata.RedirectedFrom != nil && *p.Metadata.RedirectedFrom != "" {
		return domain.PageTypeRedirect
	}

	// 2. empty
	if p.Content.ContentLength < c.cfg.EmptyMaxContentLength &&
		len(p.Links.Children) == 0 &&
		len(p.Documents) == 0 &&
		len(p.Links.Internal) < 3 {
		return domain.PageTypeEmpty
	}

	// 3. listing
	items := len(p.Links.Children) + len(p.Documents)
	substParas := countSubstantiveParagraphs(p.Content.FullText)
	if items >= c.cfg.ListingMinItems && substParas <= c.cfg.ListingMaxSubstantiveParagraphs {
		return domain.PageTypeListing
	}

	// 4. landing
	charsInLinks := sumLinkTitleChars(p.Links.Children, p.Links.Internal)
	charsInText := p.Content.ContentLength
	ratio := float64(charsInLinks) / float64(max1(charsInText))
	if p.Content.ContentLength < c.cfg.LandingMaxContentLength && ratio > c.cfg.LandingLinkTextRatio {
		return domain.PageTypeLanding
	}

	// 5. article (fallback)
	return domain.PageTypeArticle
}

// countSubstantiveParagraphs splits text by "\n\n" (then "\n" for remaining blocks)
// and counts segments with >= 160 visible characters (trimmed).
func countSubstantiveParagraphs(text string) int {
	const minLen = 160
	count := 0
	// Split on double newlines first, then single newlines within each block.
	blocks := strings.Split(text, "\n\n")
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if len(strings.TrimSpace(line)) >= minLen {
				count++
			}
		}
	}
	return count
}

// sumLinkTitleChars returns the total number of trimmed title characters
// across Children and Internal links (External links excluded).
func sumLinkTitleChars(children []domain.ChildLink, internal []domain.URLRef) int {
	total := 0
	for _, cl := range children {
		total += len(strings.TrimSpace(cl.Title))
	}
	for _, r := range internal {
		total += len(strings.TrimSpace(r.Title))
	}
	return total
}

// max1 returns v if v >= 1, else 1 (guards division by zero).
func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}
