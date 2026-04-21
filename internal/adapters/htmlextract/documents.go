package htmlextract

import (
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// docExtensions lists the file extensions that identify document attachments.
var docExtensions = map[string]struct{}{
	".pdf":  {},
	".csv":  {},
	".xlsx": {},
	".xls":  {},
	".docx": {},
	".doc":  {},
	".ods":  {},
	".odt":  {},
	".zip":  {},
	".rar":  {},
}

// extractDocuments finds all <a href> links pointing to document files in the document body.
func extractDocuments(doc *goquery.Document, pageURL string) []domain.Document {
	base, _ := url.Parse(pageURL)

	seen := make(map[string]struct{})
	var docs []domain.Document

	doc.Find("body a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}

		// Check if it's a document URL (check path only, before ? or #)
		ext := docExt(href)
		if ext == "" {
			return
		}

		// Resolve to absolute URL
		resolved := resolveHref(base, href)
		if resolved == "" {
			return
		}

		// Dedup by URL
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}

		// Title from anchor text or title attr
		title := strings.TrimSpace(s.Text())
		title = spaceRe.ReplaceAllString(title, " ")
		if title == "" {
			title = s.AttrOr("title", "")
		}
		if title == "" {
			title = href
		}

		// ContextText: parent paragraph/list-item text, up to 200 chars around link
		contextText := extractContext(s, title)

		docs = append(docs, domain.Document{
			Title:        title,
			URL:          resolved,
			Type:         strings.TrimPrefix(ext, "."),
			SizeBytes:    nil,
			DetectedFrom: "link_href",
			ContextText:  contextText,
		})
	})

	return docs
}

// docExt returns the lowercase file extension from the href path (before ? or #),
// or empty string if not a recognized document extension.
func docExt(href string) string {
	// Strip fragment and query
	clean := href
	if idx := strings.Index(clean, "#"); idx >= 0 {
		clean = clean[:idx]
	}
	if idx := strings.Index(clean, "?"); idx >= 0 {
		clean = clean[:idx]
	}
	ext := strings.ToLower(path.Ext(clean))
	if _, ok := docExtensions[ext]; ok {
		return ext
	}
	return ""
}

// extractContext returns up to 200 chars of context text around the link.
func extractContext(s *goquery.Selection, fallback string) string {
	// Try parent paragraph or list item
	parent := s.Closest("p, li, dd, dt, td, div")
	if parent.Length() > 0 {
		ctx := strings.TrimSpace(parent.Text())
		ctx = spaceRe.ReplaceAllString(ctx, " ")
		if len(ctx) > 200 {
			ctx = ctx[:200]
		}
		return ctx
	}
	return fallback
}
