package httpfetch

import (
	"net/http"
	"strings"
)

// SuspiciousReason classifies why a response was flagged.
type SuspiciousReason string

const (
	// ReasonNone means no heuristic fired.
	ReasonNone SuspiciousReason = ""
	// ReasonTitlePattern fires when the page title matches a known bot-wall pattern.
	ReasonTitlePattern SuspiciousReason = "title_pattern"
	// ReasonBodyTooSmall fires when the body is shorter than MinBodyBytes and the URL has prior content.
	ReasonBodyTooSmall SuspiciousReason = "body_too_small"
	// ReasonWAFHeaders fires when WAF fingerprint headers are present without a Plone marker.
	ReasonWAFHeaders SuspiciousReason = "waf_headers"
)

// SuspiciousDetectorConfig mirrors crawler.suspicious_response in the YAML.
type SuspiciousDetectorConfig struct {
	// MinBodyBytes: bodies shorter than this are suspicious when hasPriorContent is true.
	MinBodyBytes int
	// BlockTitlePatterns: case-insensitive substring match against <title>.
	BlockTitlePatterns []string
	// WAFHeaders: list of header names whose presence triggers suspicion unless a Plone marker cancels it.
	// Defaults to ["Cf-Ray", "X-Sucuri-Id"].
	WAFHeaders []string
	// PloneMarkerHeaders: presence cancels the WAF-headers suspicion (e.g., X-Generator: Plone).
	PloneMarkerHeaders map[string]string
}

// WithDefaults returns a copy with zero fields replaced by defaults.
func (c SuspiciousDetectorConfig) WithDefaults() SuspiciousDetectorConfig {
	if c.MinBodyBytes == 0 {
		c.MinBodyBytes = 500
	}
	if len(c.BlockTitlePatterns) == 0 {
		c.BlockTitlePatterns = []string{
			"Access Denied",
			"Forbidden",
			"Captcha",
			"Cloudflare",
			"Just a moment",
		}
	}
	if len(c.WAFHeaders) == 0 {
		c.WAFHeaders = []string{"Cf-Ray", "X-Sucuri-Id"}
	}
	if c.PloneMarkerHeaders == nil {
		c.PloneMarkerHeaders = map[string]string{
			"X-Generator": "Plone",
		}
	}
	return c
}

// SuspiciousDetector evaluates a response for bot-wall / WAF fingerprints.
type SuspiciousDetector struct {
	cfg SuspiciousDetectorConfig
}

// NewSuspiciousDetector builds a detector with the given configuration.
func NewSuspiciousDetector(cfg SuspiciousDetectorConfig) *SuspiciousDetector {
	return &SuspiciousDetector{cfg: cfg.WithDefaults()}
}

// Check inspects the response. Returns a non-empty reason if any heuristic fires.
//
// title should be the raw <title> content (caller extracts it; detector does case-insensitive match).
// bodyLen is the total bytes of the response body.
// hasPriorContent is true when the URL has a local record with non-trivial content;
// without prior content, a short body alone is NOT suspicious (a short page may legitimately be small).
// headers is the response headers as returned by net/http.
//
// Evaluation order: title → body-too-small → WAF. Returns the first hit.
func (d *SuspiciousDetector) Check(title string, bodyLen int, headers http.Header, hasPriorContent bool) SuspiciousReason {
	// 1. Title pattern check.
	lowerTitle := strings.ToLower(title)
	for _, pattern := range d.cfg.BlockTitlePatterns {
		if strings.Contains(lowerTitle, strings.ToLower(pattern)) {
			return ReasonTitlePattern
		}
	}

	// 2. Body too small check.
	if hasPriorContent && bodyLen < d.cfg.MinBodyBytes {
		return ReasonBodyTooSmall
	}

	// 3. WAF header check.
	for _, wafHeader := range d.cfg.WAFHeaders {
		if headers.Get(wafHeader) == "" {
			continue
		}
		// WAF header present — check if any Plone marker cancels it.
		canceled := false
		for ph, expected := range d.cfg.PloneMarkerHeaders {
			if headers.Get(ph) == expected {
				canceled = true
				break
			}
		}
		if !canceled {
			return ReasonWAFHeaders
		}
	}

	return ReasonNone
}
