package htmlextract

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bergmaia/site-research/internal/domain"
)

// dateFormats lists the formats to try when parsing date strings.
var dateFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// contentDateMetas lists meta tags to try for content date (in order).
var contentDateMetas = []struct{ attr, val string }{
	{"name", "DC.date"},
	{"name", "DC.date.created"},
	{"name", "DCTERMS.created"},
	{"property", "article:published_time"},
}

// updatedAtMetas lists meta tags to try for updated-at date (in order).
var updatedAtMetas = []struct{ attr, val string }{
	{"name", "DC.date.modified"},
	{"name", "DCTERMS.modified"},
	{"property", "article:modified_time"},
	{"property", "og:updated_time"},
}

// extractDates populates the Dates struct from meta tags.
func extractDates(doc *goquery.Document) domain.Dates {
	var dates domain.Dates

	// ContentDate
	for _, m := range contentDateMetas {
		val := metaContent(doc, m.attr, m.val)
		if val != "" {
			t, err := parseDate(val)
			if err == nil {
				s := t.Format("2006-01-02")
				dates.ContentDate = &s
				break
			}
		}
	}

	// PageUpdatedAt
	for _, m := range updatedAtMetas {
		val := metaContent(doc, m.attr, m.val)
		if val != "" {
			t, err := parseDate(val)
			if err == nil {
				dates.PageUpdatedAt = &t
				break
			}
		}
	}

	return dates
}

// metaContent finds a meta tag by attribute name/value and returns its content.
func metaContent(doc *goquery.Document, attr, val string) string {
	// Case-insensitive match
	selector := "meta[" + attr + "]"
	result := ""
	doc.Find(selector).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		attrVal := s.AttrOr(attr, "")
		if strings.EqualFold(attrVal, val) {
			content := s.AttrOr("content", "")
			if content != "" {
				result = content
				return false // break
			}
		}
		return true
	})
	return result
}

// parseDate attempts to parse a date string using multiple formats.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, f := range dateFormats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, &parseDateError{s}
}

type parseDateError struct{ s string }

func (e *parseDateError) Error() string { return "cannot parse date: " + e.s }
