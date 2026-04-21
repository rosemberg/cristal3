package canonical_test

import (
	"testing"

	"github.com/bergmaia/site-research/internal/canonical"
)

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		want     string
		wantExcl bool
		wantErr  bool
	}{
		// trailing slash (RF-03 decision 3.8)
		{"trailing slash removed", "https://example.com/a/b/", "https://example.com/a/b", false, false},
		{"no trailing slash preserved", "https://example.com/a/b", "https://example.com/a/b", false, false},
		{"root slash preserved", "https://example.com/", "https://example.com/", false, false},
		// fragment
		{"fragment stripped", "https://example.com/a/b#section", "https://example.com/a/b", false, false},
		{"fragment + trailing slash", "https://example.com/a/b/#section", "https://example.com/a/b", false, false},
		// tracking params
		{"utm_source stripped", "https://example.com/a?utm_source=x", "https://example.com/a", false, false},
		{"utm_* family stripped", "https://example.com/a?utm_source=x&utm_medium=y&utm_campaign=z", "https://example.com/a", false, false},
		{"gclid stripped", "https://example.com/a?gclid=abc", "https://example.com/a", false, false},
		{"fbclid stripped", "https://example.com/a?fbclid=abc", "https://example.com/a", false, false},
		{"_ga stripped", "https://example.com/a?_ga=1.2.3.4", "https://example.com/a", false, false},
		{"portal_form_id stripped", "https://example.com/a?portal_form_id=form1", "https://example.com/a", false, false},
		{"non-tracking param preserved", "https://example.com/a?id=42", "https://example.com/a?id=42", false, false},
		{"mix preserves only non-tracking", "https://example.com/a?id=42&utm_source=x&fbclid=y", "https://example.com/a?id=42", false, false},
		// combined (RF-03 example)
		{"combined utm+frag+trailing", "https://example.com/a/b/?utm_source=x#frag", "https://example.com/a/b", false, false},
		// scheme / host / port
		{"scheme lowercased", "HTTPS://WWW.EXAMPLE.COM/a", "https://www.example.com/a", false, false},
		{"host lowercased", "https://WWW.Example.COM/a", "https://www.example.com/a", false, false},
		{"http default port removed", "http://example.com:80/a", "http://example.com/a", false, false},
		{"https default port removed", "https://example.com:443/a", "https://example.com/a", false, false},
		{"non-default port preserved", "https://example.com:8443/a", "https://example.com:8443/a", false, false},
		{"path case preserved", "https://example.com/Foo/Bar", "https://example.com/Foo/Bar", false, false},
		// Plone exclusions
		{"@@ in path excluded", "https://example.com/@@advanced-search", "https://example.com/@@advanced-search", true, false},
		{"@@ mid-path excluded", "https://example.com/foo/@@view/bar", "https://example.com/foo/@@view/bar", true, false},
		{"++theme++ excluded", "https://example.com/++theme++default/logo.png", "https://example.com/++theme++default/logo.png", true, false},
		// errors
		{"empty string errors", "", "", false, true},
		{"non-http scheme errors", "ftp://example.com/a", "", false, true},
		{"malformed errors", "://::", "", false, true},
		// additional coverage cases
		{"multiple non-tracking params sorted", "https://example.com/a?z=1&a=2", "https://example.com/a?a=2&z=1", false, false},
		{"no path", "https://example.com", "https://example.com", false, false},
		{"root slash http", "http://example.com/", "http://example.com/", false, false},
		{"utm_medium stripped", "https://example.com/a?utm_medium=email", "https://example.com/a", false, false},
		{"utm_campaign stripped", "https://example.com/a?utm_campaign=launch", "https://example.com/a", false, false},
		{"https non-default port 8080", "https://example.com:8080/path", "https://example.com:8080/path", false, false},
		{"http non-default port 8080", "http://example.com:8080/path", "http://example.com:8080/path", false, false},
		{"all tracking stripped leaves no query", "https://example.com/a?utm_source=x&gclid=g&fbclid=f&_ga=1&portal_form_id=p", "https://example.com/a", false, false},
		{"excluded + tracking stripped canonical still set", "https://example.com/@@view?utm_source=x", "https://example.com/@@view", true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := canonical.New()
			got, excl, err := c.Canonicalize(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; canonical=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("canonical = %q, want %q", got, tc.want)
			}
			if excl != tc.wantExcl {
				t.Errorf("excluded = %v, want %v", excl, tc.wantExcl)
			}
		})
	}
}

func TestIsPloneCopy(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// full URLs with copy_of segments
		{"https://example.com/a/copy_of_something", true},
		{"https://example.com/a/copy2_of_something", true},
		{"https://example.com/a/copy10_of_something", true},
		// plain paths (no scheme)
		{"/a/copy_of_report", true},
		{"/copy_of_report", true},
		{"copy_of_report", true},
		// not a match
		{"https://example.com/a/b", false},
		{"https://example.com/a/copied-thing", false},
		// somecopy_of_x should NOT match because it does not start at a segment boundary
		{"https://example.com/a/somecopy_of_x", false},
		// no path at all
		{"https://example.com/a", false},
		// Plone real-world example from BRIEF
		{"https://example.com/copy5_of_relatorios-do-conselho-nacional-de-justica-cnj", true},
		// digits variant
		{"/a/b/copy99_of_foo", true},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := canonical.IsPloneCopy(tc.path)
			if got != tc.want {
				t.Errorf("IsPloneCopy(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
