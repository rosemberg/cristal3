package classify

import (
	"strings"
	"testing"

	"github.com/bergmaia/site-research/internal/domain"
)

// pageOpts holds the parameters used by mkPage to build synthetic domain.Page values.
type pageOpts struct {
	contentLen       int
	fullText         string
	children         int
	internalLinks    int
	internalTitleLen int // chars per link title (uniform across internal links)
	externalLinks    int
	documents        int
	httpStatus       int
	redirectedFrom   *string
}

// mkPage constructs a *domain.Page from the given opts.
// It creates uniform link titles of the requested length and fills in ContentLength.
func mkPage(_ *testing.T, opts pageOpts) *domain.Page {
	p := &domain.Page{}

	p.Content.ContentLength = opts.contentLen
	p.Content.FullText = opts.fullText

	// Children
	for i := 0; i < opts.children; i++ {
		p.Links.Children = append(p.Links.Children, domain.ChildLink{
			Title: strings.Repeat("a", 15), // 15 chars each
			URL:   "http://example.com/child",
		})
	}

	// Internal links
	titleLen := opts.internalTitleLen
	if titleLen == 0 {
		titleLen = 10
	}
	for i := 0; i < opts.internalLinks; i++ {
		p.Links.Internal = append(p.Links.Internal, domain.URLRef{
			Title: strings.Repeat("b", titleLen),
			URL:   "http://example.com/internal",
		})
	}

	// External links
	for i := 0; i < opts.externalLinks; i++ {
		p.Links.External = append(p.Links.External, domain.URLRef{
			Title: strings.Repeat("c", 10),
			URL:   "http://external.com/page",
		})
	}

	// Documents
	for i := 0; i < opts.documents; i++ {
		p.Documents = append(p.Documents, domain.Document{
			Title: "doc.pdf",
			URL:   "http://example.com/doc.pdf",
			Type:  "pdf",
		})
	}

	// Metadata
	p.Metadata.HTTPStatus = opts.httpStatus
	p.Metadata.RedirectedFrom = opts.redirectedFrom

	return p
}

// ptr returns a pointer to the given string (helper for redirectedFrom).
func ptr(s string) *string { return &s }

func TestClassify(t *testing.T) {
	defaultClassifier := New(Config{})

	tests := []struct {
		name string
		page *domain.Page
		want domain.PageType
	}{
		// 1. Redirect via HTTP 301
		{
			name: "redirect_via_http_301",
			page: mkPage(t, pageOpts{
				contentLen: 50,
				httpStatus: 301,
			}),
			want: domain.PageTypeRedirect,
		},
		// 2. Redirect via RedirectedFrom
		{
			name: "redirect_via_redirected_from",
			page: mkPage(t, pageOpts{
				contentLen:     50,
				httpStatus:     200,
				redirectedFrom: ptr("http://example.com/old-path"),
			}),
			want: domain.PageTypeRedirect,
		},
		// 3. Empty page (contentLength=10, no children/docs)
		{
			name: "empty_no_content_no_links",
			page: mkPage(t, pageOpts{
				contentLen:    10,
				httpStatus:    200,
				internalLinks: 0,
			}),
			want: domain.PageTypeEmpty,
		},
		// 4. ContentLength=50 but 5 internal links → NOT empty (≥3 internal).
		// With contentLen=50 < 500 and internal links contributing titles (10 chars each, 5 links = 50 chars),
		// ratio = 50/50 = 1.0 > 0.5 → classified as landing (not article, not empty).
		{
			name: "not_empty_due_to_internal_links",
			page: mkPage(t, pageOpts{
				contentLen:    50,
				httpStatus:    200,
				internalLinks: 5,
			}),
			want: domain.PageTypeLanding,
		},
		// 5. Listing via many documents (15 docs, short content)
		{
			name: "listing_via_documents",
			page: mkPage(t, pageOpts{
				contentLen: 200,
				fullText:   "Short intro text.",
				httpStatus: 200,
				documents:  15,
			}),
			want: domain.PageTypeListing,
		},
		// 6. Listing via many children (10 children, short content, 1 intro paragraph)
		{
			name: "listing_via_children_short_intro",
			page: mkPage(t, pageOpts{
				contentLen: 200,
				fullText:   "A short intro paragraph.\n\nAnother short line.",
				httpStatus: 200,
				children:   10,
			}),
			want: domain.PageTypeListing,
		},
		// 7. Not a listing — 10 children but 5 substantive paragraphs → falls to article
		{
			name: "not_listing_too_many_paragraphs",
			page: mkPage(t, pageOpts{
				contentLen: 3000,
				fullText: strings.Join([]string{
					strings.Repeat("x", 200), // para 1 >= 160 chars
					strings.Repeat("x", 200), // para 2
					strings.Repeat("x", 200), // para 3
					strings.Repeat("x", 200), // para 4
					strings.Repeat("x", 200), // para 5
				}, "\n"),
				httpStatus: 200,
				children:   10,
			}),
			want: domain.PageTypeArticle,
		},
		// 8. Landing (content_length=300, many menu links with avg title 20 chars → ratio high).
		// Only 3 children (below ListingMinItems=8) to avoid listing classification.
		// 3 * 15 = 45 chars from children, 10 * 20 = 200 chars from internal links.
		// total link chars = 245, text = 300 → ratio ≈ 0.82 > 0.5, contentLen=300 < 500 → landing.
		{
			name: "landing_high_link_ratio",
			page: mkPage(t, pageOpts{
				contentLen:       300,
				fullText:         "Welcome to our site.",
				httpStatus:       200,
				children:         3,  // below ListingMinItems to avoid listing
				internalLinks:    10, // 10 * 20 = 200 chars from internal
				internalTitleLen: 20,
			}),
			want: domain.PageTypeLanding,
		},
		// 9. Article (content_length=2000, 2 children, moderate links)
		{
			name: "article_fallback",
			page: mkPage(t, pageOpts{
				contentLen:    2000,
				fullText:      strings.Repeat("word ", 400), // plenty of text
				httpStatus:    200,
				children:      2,
				internalLinks: 3,
			}),
			want: domain.PageTypeArticle,
		},
		// Edge: redirect with status 302 (other 3xx)
		{
			name: "redirect_302",
			page: mkPage(t, pageOpts{
				contentLen: 0,
				httpStatus: 302,
			}),
			want: domain.PageTypeRedirect,
		},
		// Edge: redirect with status 399 (upper boundary of 3xx)
		{
			name: "redirect_399",
			page: mkPage(t, pageOpts{
				contentLen: 0,
				httpStatus: 399,
			}),
			want: domain.PageTypeRedirect,
		},
		// Edge: status 400 is NOT redirect
		{
			name: "status_400_not_redirect",
			page: mkPage(t, pageOpts{
				contentLen:    0,
				httpStatus:    400,
				internalLinks: 0,
			}),
			want: domain.PageTypeEmpty,
		},
		// Edge: redirectedFrom set but empty string → NOT redirect
		{
			name: "redirected_from_empty_string_not_redirect",
			page: mkPage(t, pageOpts{
				contentLen:     50,
				httpStatus:     200,
				redirectedFrom: ptr(""),
				internalLinks:  0,
			}),
			want: domain.PageTypeEmpty,
		},
		// Edge: exactly at listing threshold (8 items)
		{
			name: "listing_exact_threshold",
			page: mkPage(t, pageOpts{
				contentLen: 100,
				fullText:   "Short intro.",
				httpStatus: 200,
				children:   4,
				documents:  4,
			}),
			want: domain.PageTypeListing,
		},
		// Edge: one below listing threshold (7 items) with enough content for article
		{
			name: "below_listing_threshold_article",
			page: mkPage(t, pageOpts{
				contentLen:    2000,
				fullText:      strings.Repeat("word ", 400),
				httpStatus:    200,
				children:      3,
				documents:     4,
				internalLinks: 5,
			}),
			want: domain.PageTypeArticle,
		},
		// Edge: empty page with exactly 2 internal links (< 3) → still empty
		{
			name: "empty_with_two_internal_links",
			page: mkPage(t, pageOpts{
				contentLen:    50,
				httpStatus:    200,
				internalLinks: 2,
			}),
			want: domain.PageTypeEmpty,
		},
		// Edge: listing with exactly ListingMaxSubstantiveParagraphs (2 subst. paras) → listing
		{
			name: "listing_at_max_subst_paras",
			page: mkPage(t, pageOpts{
				contentLen: 500,
				fullText: strings.Join([]string{
					strings.Repeat("x", 200), // para 1
					strings.Repeat("x", 200), // para 2
				}, "\n\n"),
				httpStatus: 200,
				children:   10,
			}),
			want: domain.PageTypeListing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultClassifier.Classify(tc.page)
			if got != tc.want {
				t.Errorf("Classify() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestConfigWithDefaults verifies that Config{}.WithDefaults() produces the documented values.
func TestConfigWithDefaults(t *testing.T) {
	cfg := Config{}.WithDefaults()

	if cfg.EmptyMaxContentLength != 100 {
		t.Errorf("EmptyMaxContentLength = %d, want 100", cfg.EmptyMaxContentLength)
	}
	if cfg.LandingMaxContentLength != 500 {
		t.Errorf("LandingMaxContentLength = %d, want 500", cfg.LandingMaxContentLength)
	}
	if cfg.LandingLinkTextRatio != 0.5 {
		t.Errorf("LandingLinkTextRatio = %f, want 0.5", cfg.LandingLinkTextRatio)
	}
	if cfg.ListingMinItems != 8 {
		t.Errorf("ListingMinItems = %d, want 8", cfg.ListingMinItems)
	}
	if cfg.ListingMaxSubstantiveParagraphs != 2 {
		t.Errorf("ListingMaxSubstantiveParagraphs = %d, want 2", cfg.ListingMaxSubstantiveParagraphs)
	}
}

// TestConfigWithDefaultsPreservesNonZero verifies that non-zero values are preserved.
func TestConfigWithDefaultsPreservesNonZero(t *testing.T) {
	cfg := Config{
		EmptyMaxContentLength:           200,
		LandingMaxContentLength:         1000,
		LandingLinkTextRatio:            0.8,
		ListingMinItems:                 15,
		ListingMaxSubstantiveParagraphs: 5,
	}.WithDefaults()

	if cfg.EmptyMaxContentLength != 200 {
		t.Errorf("EmptyMaxContentLength = %d, want 200 (should not override non-zero)", cfg.EmptyMaxContentLength)
	}
	if cfg.LandingMaxContentLength != 1000 {
		t.Errorf("LandingMaxContentLength = %d, want 1000", cfg.LandingMaxContentLength)
	}
	if cfg.LandingLinkTextRatio != 0.8 {
		t.Errorf("LandingLinkTextRatio = %f, want 0.8", cfg.LandingLinkTextRatio)
	}
	if cfg.ListingMinItems != 15 {
		t.Errorf("ListingMinItems = %d, want 15", cfg.ListingMinItems)
	}
	if cfg.ListingMaxSubstantiveParagraphs != 5 {
		t.Errorf("ListingMaxSubstantiveParagraphs = %d, want 5", cfg.ListingMaxSubstantiveParagraphs)
	}
}

// TestCustomConfig verifies that a custom Config is respected.
func TestCustomConfig(t *testing.T) {
	// Low EmptyMaxContentLength so a page with contentLen=200 doesn't become empty.
	cfg := Config{
		EmptyMaxContentLength:           50,
		LandingMaxContentLength:         300,
		LandingLinkTextRatio:            0.5,
		ListingMinItems:                 5,
		ListingMaxSubstantiveParagraphs: 2,
	}
	cl := New(cfg)

	// With ListingMinItems=5, a page with 5 children should become listing.
	p := mkPage(t, pageOpts{
		contentLen: 200,
		fullText:   "Short intro.",
		httpStatus: 200,
		children:   5,
	})
	got := cl.Classify(p)
	if got != domain.PageTypeListing {
		t.Errorf("custom config: Classify() = %q, want %q", got, domain.PageTypeListing)
	}
}

// TestNewReturnsNonNil ensures the constructor returns a usable *Classifier.
func TestNewReturnsNonNil(t *testing.T) {
	cl := New(Config{})
	if cl == nil {
		t.Fatal("New() returned nil")
	}
}

// TestExternalLinksNotCountedInLandingRatio verifies that external link titles
// do NOT contribute to the landing ratio calculation.
func TestExternalLinksNotCountedInLandingRatio(t *testing.T) {
	// Page has lots of external links with long titles but low children/internal.
	// ratio should stay low and NOT classify as landing.
	p := mkPage(t, pageOpts{
		contentLen:    400,
		fullText:      "Some text.",
		httpStatus:    200,
		externalLinks: 50, // many external, but these don't count
		internalLinks: 0,
		children:      0,
	})
	cl := New(Config{})
	got := cl.Classify(p)
	// Should fall through to article (no listing, no landing signal from internal/children).
	if got != domain.PageTypeArticle {
		t.Errorf("external links should not affect landing ratio: got %q, want %q", got, domain.PageTypeArticle)
	}
}
