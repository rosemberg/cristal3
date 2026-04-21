package htmlextract

import (
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// extractLinks processes all <a href> tags in the document and classifies them
// into Children, Internal, and External buckets.
func extractLinks(doc *goquery.Document, pageURL, scopePrefix string) domain.Links {
	base, _ := url.Parse(pageURL)
	pageURLClean := strings.TrimRight(stripFragment(pageURL), "/")

	// childrenSeen keyed by first-segment URL (pageURL/segment) to dedup children
	childrenSeen := make(map[string]struct{})
	internalSeen := make(map[string]struct{})
	externalSeen := make(map[string]struct{})

	var children []domain.ChildLink
	var internal []domain.URLRef
	var external []domain.URLRef

	doc.Find("body a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}

		// Skip non-http schemes and fragments
		lhref := strings.ToLower(href)
		if strings.HasPrefix(lhref, "mailto:") ||
			strings.HasPrefix(lhref, "tel:") ||
			strings.HasPrefix(lhref, "javascript:") ||
			strings.HasPrefix(lhref, "#") {
			return
		}

		// Resolve to absolute URL
		resolved := resolveHref(base, href)
		if resolved == "" {
			return
		}

		// Skip pure fragment anchors on the same page
		if strings.HasPrefix(resolved, "#") {
			return
		}

		// Parse resolved URL
		parsedResolved, err := url.Parse(resolved)
		if err != nil {
			return
		}

		// Clear fragment for dedup purposes
		parsedResolved.Fragment = ""
		cleanURL := parsedResolved.String()

		// Skip document file URLs (handled by documents extractor)
		if isDocumentURL(cleanURL) {
			return
		}

		// Title: first non-empty of text, title attr, or href
		title := strings.TrimSpace(s.Text())
		title = spaceRe.ReplaceAllString(title, " ")
		if title == "" {
			title = s.AttrOr("title", "")
		}
		if title == "" {
			title = href
		}

		// Classify the link
		childKey, isChild := childLinkKey(cleanURL, pageURLClean)
		if isChild {
			if _, seen := childrenSeen[childKey]; !seen {
				childrenSeen[childKey] = struct{}{}
				children = append(children, domain.ChildLink{
					Title:     title,
					URL:       cleanURL,
					LocalPath: "",
				})
			}
		} else if isInScope(cleanURL, scopePrefix) {
			if _, seen := internalSeen[cleanURL]; !seen {
				internalSeen[cleanURL] = struct{}{}
				internal = append(internal, domain.URLRef{
					Title: title,
					URL:   cleanURL,
				})
			}
		} else {
			// External: only include if it's an http(s) URL
			if strings.HasPrefix(strings.ToLower(cleanURL), "http") {
				if _, seen := externalSeen[cleanURL]; !seen {
					externalSeen[cleanURL] = struct{}{}
					external = append(external, domain.URLRef{
						Title: title,
						URL:   cleanURL,
					})
				}
			}
		}
	})

	return domain.Links{
		Children: children,
		Internal: internal,
		External: external,
	}
}

// childLinkKey returns the dedup key (pageURL/firstSegment) and true if cleanURL
// is a direct child of pageURL. A "child" is any URL that starts with pageURL/
// and whose first path segment after pageURL is non-empty. We dedup by that first
// segment, so /parent/child/anything deduplicates to /parent/child.
func childLinkKey(resolvedURL, pageURL string) (string, bool) {
	base := strings.TrimRight(pageURL, "/")
	if !strings.HasPrefix(resolvedURL, base+"/") {
		return "", false
	}
	// Get the suffix after base/
	suffix := resolvedURL[len(base)+1:]
	suffix = strings.TrimRight(suffix, "/")
	if suffix == "" {
		return "", false
	}
	// Extract the first path segment
	firstSlash := strings.Index(suffix, "/")
	firstSegment := suffix
	if firstSlash >= 0 {
		firstSegment = suffix[:firstSlash]
	}
	if firstSegment == "" {
		return "", false
	}
	return base + "/" + firstSegment, true
}

// isInScope returns true if the URL is within the scope prefix.
func isInScope(resolvedURL, scopePrefix string) bool {
	if scopePrefix == "" {
		return false
	}
	sp := strings.TrimRight(scopePrefix, "/")
	return strings.HasPrefix(resolvedURL, sp+"/") || resolvedURL == sp
}

// stripFragment removes the fragment portion of a URL string.
func stripFragment(rawURL string) string {
	if idx := strings.Index(rawURL, "#"); idx >= 0 {
		return rawURL[:idx]
	}
	return rawURL
}

// isDocumentURL returns true if the URL path has a document file extension.
func isDocumentURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	p := strings.ToLower(path.Ext(u.Path))
	switch p {
	case ".pdf", ".csv", ".xlsx", ".xls", ".docx", ".doc", ".ods", ".odt", ".zip", ".rar":
		return true
	}
	return false
}
