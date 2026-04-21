package orchestrator

import (
	"fmt"
	"regexp"
	"strings"
)

// extractCitationsFrom parses markdown output from site-research MCP
// and extracts URL, title, and breadcrumb/section information
func extractCitationsFromMarkdown(markdown string, citationMap map[string]int, citations *[]Citation) {
	// Pattern for search results: ## N. Title\n...\n**Seção:** Section\n**URL:** url
	searchPattern := regexp.MustCompile(`(?m)^## \d+\.\s+(.+?)\n([\s\S]*?)\*\*URL:\*\*\s+(.+?)$`)
	matches := searchPattern.FindAllStringSubmatch(markdown, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		title := strings.TrimSpace(match[1])
		body := match[2]
		url := strings.TrimSpace(match[3])

		// Extract section/breadcrumb from body
		breadcrumb := extractBreadcrumb(body)

		addCitation(url, title, breadcrumb, citationMap, citations)
	}

	// Pattern for inspect_page results: # Title\n**URL:** url\n...\n## Breadcrumb\nbreadcrumb
	pagePattern := regexp.MustCompile(`(?m)^# (.+?)\n\*\*URL:\*\*\s+(.+?)$`)
	pageMatches := pagePattern.FindAllStringSubmatch(markdown, -1)

	for _, match := range pageMatches {
		if len(match) < 3 {
			continue
		}

		title := strings.TrimSpace(match[1])
		url := strings.TrimSpace(match[2])

		// Look for breadcrumb section
		breadcrumb := extractBreadcrumbSection(markdown)

		addCitation(url, title, breadcrumb, citationMap, citations)
	}
}

// extractBreadcrumb extracts breadcrumb from **Seção:** pattern
func extractBreadcrumb(text string) string {
	sectionPattern := regexp.MustCompile(`\*\*Seção:\*\*\s+(.+?)(?:\n|$)`)
	match := sectionPattern.FindStringSubmatch(text)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// extractBreadcrumbSection extracts breadcrumb from ## Breadcrumb section
func extractBreadcrumbSection(markdown string) string {
	breadcrumbPattern := regexp.MustCompile(`(?m)^## Breadcrumb\n(.+?)(?:\n\n|$)`)
	match := breadcrumbPattern.FindStringSubmatch(markdown)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// addCitation adds a citation to the map if not already present
// Returns the citation ID (1-indexed)
func addCitation(url, title, breadcrumb string, citationMap map[string]int, citations *[]Citation) int {
	// Check if already exists
	if id, exists := citationMap[url]; exists {
		return id
	}

	// Add new citation
	nextID := len(*citations) + 1
	citationMap[url] = nextID

	*citations = append(*citations, Citation{
		ID:         nextID,
		Title:      title,
		Breadcrumb: breadcrumb,
		URL:        url,
	})

	return nextID
}

// formatInlineCitations converts [text](url) to [text]^N format
// using the citationMap to find citation IDs
func formatInlineCitations(text string, citationMap map[string]int) string {
	// Pattern: [text](url)
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	result := linkPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := linkPattern.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		linkText := submatches[1]
		url := submatches[2]

		// Check if URL is in citationMap
		if id, exists := citationMap[url]; exists {
			// Convert to inline citation format: [text]^N
			return fmt.Sprintf("%s^%d", linkText, id)
		}

		// Not a citation, return as hyperlink (markdown)
		return match
	})

	return result
}
