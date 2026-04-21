package httpfetch

import (
	"net/http"
	"testing"
)

// newTestDetector creates a SuspiciousDetector with default configuration.
func newTestDetector() *SuspiciousDetector {
	return NewSuspiciousDetector(SuspiciousDetectorConfig{})
}

// TestSuspicious_TitleMatch verifies title pattern detection.
func TestSuspicious_TitleMatch(t *testing.T) {
	d := newTestDetector()

	// "Just a moment..." should match the "Just a moment" pattern.
	reason := d.Check("Just a moment...", 1000, http.Header{}, false)
	if reason != ReasonTitlePattern {
		t.Errorf("expected ReasonTitlePattern for 'Just a moment...', got %q", reason)
	}

	// Legitimate title should not match.
	reason = d.Check("Balancete Março 2026", 1000, http.Header{}, false)
	if reason != ReasonNone {
		t.Errorf("expected ReasonNone for 'Balancete Março 2026', got %q", reason)
	}
}

// TestSuspicious_BodyTooSmall_WithPrior verifies small body fires when hasPriorContent is true.
func TestSuspicious_BodyTooSmall_WithPrior(t *testing.T) {
	d := newTestDetector()
	reason := d.Check("", 100, http.Header{}, true)
	if reason != ReasonBodyTooSmall {
		t.Errorf("expected ReasonBodyTooSmall with prior content and small body, got %q", reason)
	}
}

// TestSuspicious_BodyTooSmall_WithoutPrior verifies small body is NOT suspicious without prior content.
func TestSuspicious_BodyTooSmall_WithoutPrior(t *testing.T) {
	d := newTestDetector()
	reason := d.Check("", 100, http.Header{}, false)
	if reason != ReasonNone {
		t.Errorf("expected ReasonNone without prior content, got %q", reason)
	}
}

// TestSuspicious_WAFHeader_Alone verifies Cf-Ray without Plone marker fires ReasonWAFHeaders.
func TestSuspicious_WAFHeader_Alone(t *testing.T) {
	d := newTestDetector()
	headers := http.Header{}
	headers.Set("Cf-Ray", "123abc")
	reason := d.Check("", 1000, headers, false)
	if reason != ReasonWAFHeaders {
		t.Errorf("expected ReasonWAFHeaders for Cf-Ray without Plone marker, got %q", reason)
	}
}

// TestSuspicious_WAFHeader_WithPloneMarker verifies Cf-Ray is canceled by X-Generator: Plone.
func TestSuspicious_WAFHeader_WithPloneMarker(t *testing.T) {
	d := newTestDetector()
	headers := http.Header{}
	headers.Set("Cf-Ray", "123abc")
	headers.Set("X-Generator", "Plone")
	reason := d.Check("", 1000, headers, false)
	if reason != ReasonNone {
		t.Errorf("expected ReasonNone when Plone marker is present, got %q", reason)
	}
}

// TestSuspicious_EvaluationOrder verifies title is checked first when all heuristics fire.
func TestSuspicious_EvaluationOrder(t *testing.T) {
	d := newTestDetector()
	// Title fires, body too small fires (hasPriorContent=true, bodyLen=10 < 500), WAF fires.
	headers := http.Header{}
	headers.Set("Cf-Ray", "abc123")
	reason := d.Check("Access Denied", 10, headers, true)
	if reason != ReasonTitlePattern {
		t.Errorf("expected ReasonTitlePattern (first in order), got %q", reason)
	}
}

// TestSuspicious_Defaults verifies SuspiciousDetectorConfig{}.WithDefaults() returns expected values.
func TestSuspicious_Defaults(t *testing.T) {
	cfg := SuspiciousDetectorConfig{}.WithDefaults()

	if cfg.MinBodyBytes != 500 {
		t.Errorf("MinBodyBytes: want 500, got %d", cfg.MinBodyBytes)
	}

	if len(cfg.BlockTitlePatterns) != 5 {
		t.Errorf("BlockTitlePatterns: want 5 entries, got %d", len(cfg.BlockTitlePatterns))
	}

	// Verify "Cloudflare" is one of the patterns.
	found := false
	for _, p := range cfg.BlockTitlePatterns {
		if p == "Cloudflare" {
			found = true
			break
		}
	}
	if !found {
		t.Error("BlockTitlePatterns: expected 'Cloudflare' to be included")
	}

	if len(cfg.WAFHeaders) != 2 {
		t.Errorf("WAFHeaders: want 2 entries, got %d", len(cfg.WAFHeaders))
	}

	if val, ok := cfg.PloneMarkerHeaders["X-Generator"]; !ok || val != "Plone" {
		t.Errorf("PloneMarkerHeaders: expected X-Generator=Plone, got %v", cfg.PloneMarkerHeaders)
	}
}
