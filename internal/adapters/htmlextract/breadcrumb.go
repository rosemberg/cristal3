package htmlextract

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// rootLabels lists home/root labels to filter out of PathTitles.
var rootLabels = map[string]struct{}{
	"página inicial": {},
	"inicio":         {},
	"início":         {},
	"home":           {},
	"portal":         {},
}

// extractBreadcrumb extracts breadcrumb items, path titles, and section from the document.
// It tries several selectors in priority order and returns warnings if no breadcrumb is found.
func extractBreadcrumb(doc *goquery.Document, pageURL string) ([]domain.URLRef, []string, string, []string) {
	base, _ := url.Parse(pageURL)

	var items []domain.URLRef

	// Try selectors in order
	selectors := []string{
		"#portal-breadcrumbs ol.breadcrumb li a",
		"#breadcrumb ol.breadcrumb li a",
		"ol.breadcrumb li a",
		"nav.breadcrumb a",
		"[data-microdata-breadcrumb] a",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			href := s.AttrOr("href", "")

			// Resolve relative URLs
			resolved := resolveHref(base, href)

			items = append(items, domain.URLRef{
				Title: text,
				URL:   resolved,
			})
		})
		if len(items) > 0 {
			break
		}
	}

	// Also try active breadcrumb items (no <a>, just text in <li class="breadcrumb-item active">)
	// These are the current page, we still want them in PathTitles but not as URLRef
	var activeTitles []string
	doc.Find("ol.breadcrumb li.active, ol.breadcrumb li[aria-current='page']").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			activeTitles = append(activeTitles, t)
		}
	})

	var warnings []string
	if len(items) == 0 {
		warnings = append(warnings, "breadcrumb not found")
	}

	// Build PathTitles: titles from breadcrumb, skipping top-level root labels
	var pathTitles []string
	for _, item := range items {
		label := strings.ToLower(item.Title)
		if _, skip := rootLabels[label]; skip {
			continue
		}
		if item.Title != "" {
			pathTitles = append(pathTitles, item.Title)
		}
	}
	// Append active titles (current page not in <a>)
	for _, at := range activeTitles {
		// Avoid duplicates
		found := false
		for _, pt := range pathTitles {
			if pt == at {
				found = true
				break
			}
		}
		if !found && at != "" {
			pathTitles = append(pathTitles, at)
		}
	}

	// Section: PathTitles[1] if len >= 2, "" otherwise
	// (index 1 = second segment after home/root)
	// But since we've already filtered root labels, PathTitles[0] is the first real segment,
	// PathTitles[1] would be the second. We want the segment after the scope root.
	// Actually PathTitles[0] is the scope root section (like "Transparência e Prestação de Contas"),
	// and PathTitles[1] would be the first child section.
	section := ""
	if len(pathTitles) >= 2 {
		section = pathTitles[1]
	} else if len(pathTitles) == 1 {
		// Only one item — it IS the section itself
		section = ""
	}

	return items, pathTitles, section, warnings
}

// resolveHref resolves a possibly relative href against the base URL.
func resolveHref(base *url.URL, href string) string {
	if href == "" || strings.HasPrefix(href, "javascript:") {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	if base != nil {
		return base.ResolveReference(ref).String()
	}
	return ref.String()
}
