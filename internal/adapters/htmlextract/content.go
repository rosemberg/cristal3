package htmlextract

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// spaceRe collapses runs of whitespace to a single space.
var spaceRe = regexp.MustCompile(`\s+`)

// chromeSelectors are elements to remove before extracting text.
const chromeSelectors = "script, style, nav, header, footer, #portal-footer, #portal-header, .portlet, #portal-searchbox, aside, .nao-imprimir"

// extractContent extracts the main textual content from the document.
func extractContent(doc *goquery.Document, title, description string) domain.Content {
	container := findContentContainer(doc)

	// Remove chrome elements from container
	container.Find(chromeSelectors).Remove()

	// Extract text from content elements
	var sb strings.Builder
	container.Find("p, h1, h2, h3, h4, h5, h6, li, dt, dd").Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		t = spaceRe.ReplaceAllString(t, " ")
		if t != "" {
			sb.WriteString(t)
			sb.WriteString("\n\n")
		}
	})
	fullText := strings.TrimSpace(sb.String())

	// Summary: first 500 chars, clean cut at last sentence/paragraph within 450-500
	summary := makeSummary(fullText, 500, 450)

	// Hashes
	ftHash := hashText(fullText)
	contentInput := title + "\n" + description + "\n" + fullText
	contentHash := hashText(contentInput)

	return domain.Content{
		Summary:       summary,
		FullText:      fullText,
		FullTextHash:  ftHash,
		ContentHash:   contentHash,
		ContentLength: len([]rune(fullText)),
	}
}

// findContentContainer locates the main content container using a selector cascade.
func findContentContainer(doc *goquery.Document) *goquery.Selection {
	selectors := []string{
		"#content-core",
		"#content",
		"main",
		"article",
	}
	for _, sel := range selectors {
		c := doc.Find(sel)
		if c.Length() > 0 {
			return c.First()
		}
	}
	// Fallback: body minus known chrome
	body := doc.Find("body")
	return body
}

// makeSummary returns the first maxLen chars of text, cutting cleanly at
// a sentence/paragraph boundary within [minCut, maxLen].
func makeSummary(text string, maxLen, minCut int) string {
	if len(text) == 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	// Try to find a clean cut point between minCut and maxLen
	sub := string(runes[:maxLen])
	// Look for paragraph break first
	if idx := strings.LastIndex(sub[minCut:], "\n\n"); idx >= 0 {
		return strings.TrimSpace(sub[:minCut+idx])
	}
	// Look for sentence end (. ! ?)
	for i := maxLen - 1; i >= minCut; i-- {
		ch := runes[i]
		if ch == '.' || ch == '!' || ch == '?' {
			return strings.TrimSpace(string(runes[:i+1]))
		}
	}
	// Just cut at maxLen
	return strings.TrimSpace(sub)
}

// hashText computes sha256 of the text and returns "sha256:<hex>".
func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("sha256:%x", h)
}
