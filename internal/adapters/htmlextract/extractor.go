// Package htmlextract parses HTML bodies into partially-populated domain.Page values.
// It covers RF-04 (content extraction) and RF-06 (date extraction) fields. It does NOT
// classify page_type (classifier does that), set CanonicalURL (canonicalizer), or generate
// mini_summary (LLM). Metadata except CrawlerVersion is left to the HTTP caller.
package htmlextract

import (
	"bytes"
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// Options configures the extractor.
type Options struct {
	// CrawlerVersion set into Page.Metadata.CrawlerVersion.
	CrawlerVersion string
	// ScopePrefix is the URL prefix of the in-scope subsite, used to classify links
	// into children / internal / external.
	// E.g., "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas".
	ScopePrefix string
}

// Extractor parses HTML bodies.
type Extractor struct {
	opts Options
}

// New returns a new Extractor.
func New(opts Options) *Extractor {
	return &Extractor{opts: opts}
}

// Extract parses body as HTML and returns a *domain.Page populated with:
//
//	URL, Title, Description, Section, Breadcrumb, PathTitles, Lang,
//	Content (Summary, FullText, FullTextHash, ContentHash, ContentLength, KeywordsExtracted),
//	Links (Children, Internal, External),
//	Documents,
//	Dates (ContentDate, PageUpdatedAt),
//	Tags,
//	Metadata.CrawlerVersion.
//
// Left empty/zero (callers populate):
//
//	Schema, SchemaVersion, CanonicalURL, PageType, HasSubstantiveContent,
//	MiniSummary, Metadata.{Depth, ExtractedAt, LastModified, ETag, HTTPStatus,
//	ContentType, ParentURL, RedirectedFrom, CanonicalOf, FetchDurationMs,
//	DiscoveredVia, IsPloneCopy, StaleSince, ExtractionWarnings}.
//
// Never returns nil *Page when error is nil. Adds descriptive warnings into
// a local slice; the caller can attach them to Metadata.ExtractionWarnings.
// Convention: return warnings via page.Metadata.ExtractionWarnings — populate it
// with any non-fatal issues discovered during extraction (e.g., "breadcrumb not found").
func (e *Extractor) Extract(ctx context.Context, pageURL string, body []byte) (*domain.Page, error) {
	page := &domain.Page{
		URL: pageURL,
	}
	page.Metadata.CrawlerVersion = e.opts.CrawlerVersion

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		// goquery is tolerant; this is rare — still return a populated page
		return page, err
	}

	var warnings []string

	// Lang
	page.Lang = strings.TrimSpace(doc.Find("html").AttrOr("lang", ""))

	// Title
	rawTitle := strings.TrimSpace(doc.Find("title").Text())
	// Strip common suffix like " — Tribunal Regional Eleitoral do Piauí"
	if idx := strings.Index(rawTitle, " — "); idx > 0 {
		rawTitle = strings.TrimSpace(rawTitle[:idx])
	}
	page.Title = rawTitle

	// Description from meta
	doc.Find("meta[name='description']").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("content"); ok && page.Description == "" {
			page.Description = strings.TrimSpace(v)
		}
	})
	if page.Description == "" {
		doc.Find("meta[property='og:description']").Each(func(_ int, s *goquery.Selection) {
			if v, ok := s.Attr("content"); ok {
				page.Description = strings.TrimSpace(v)
			}
		})
	}

	// Tags from meta keywords or portal tags section
	doc.Find("meta[name='keywords']").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("content"); ok {
			for _, kw := range strings.Split(v, ",") {
				kw = strings.TrimSpace(kw)
				if kw != "" {
					page.Tags = append(page.Tags, kw)
				}
			}
		}
	})
	// Also extract tags from portal tags section
	doc.Find("#tags a, section#tags a").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		t = strings.TrimPrefix(t, "#")
		t = strings.TrimSpace(t)
		if t != "" {
			page.Tags = append(page.Tags, t)
		}
	})

	// Breadcrumb
	bc, pathTitles, section, bcWarnings := extractBreadcrumb(doc, pageURL)
	page.Breadcrumb = bc
	page.PathTitles = pathTitles
	page.Section = section
	warnings = append(warnings, bcWarnings...)

	// Content
	content := extractContent(doc, page.Title, page.Description)
	page.Content = content

	// Keywords
	page.Content.KeywordsExtracted = extractKeywords(content.FullText)

	// Links
	links := extractLinks(doc, pageURL, e.opts.ScopePrefix)
	page.Links = links

	// Documents
	page.Documents = extractDocuments(doc, pageURL)

	// Dates
	page.Dates = extractDates(doc)

	if len(warnings) > 0 {
		page.Metadata.ExtractionWarnings = warnings
	}

	return page, nil
}
