package canonical

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ploneCopyRe matches path segments beginning with "copy_of" or "copy<N>_of"
// (where N is one or more digits). A segment start is the beginning of the
// string or a "/" character.
var ploneCopyRe = regexp.MustCompile(`(^|/)copy\d*_of`)

// trackingKeys lists exact-match query-parameter keys to strip (RF-03).
var trackingKeys = map[string]struct{}{
	"gclid":          {},
	"fbclid":         {},
	"_ga":            {},
	"portal_form_id": {},
}

// Canonicalizer applies deterministic canonicalization to URLs per BRIEF v2.1 RF-03.
// It is safe for concurrent use.
type Canonicalizer struct{}

// New returns a Canonicalizer with default behavior.
func New() *Canonicalizer { return &Canonicalizer{} }

// Canonicalize applies the rules in the order specified by RF-03 and returns:
//   - canonical: the canonical form of the URL
//   - excluded: true if the URL is in an excluded category (Plone views "@@" or theme assets "++theme++")
//   - err: non-nil only for malformed URLs (invalid scheme, unparseable)
//
// Rules applied in order:
//  1. Require absolute URL with scheme http or https; otherwise return error.
//  2. Lowercase scheme and host.
//  3. Remove default port (:80 for http, :443 for https).
//  4. Remove fragment always.
//  5. Remove tracking query params: keys matching "utm_*" (prefix), or exactly "gclid", "fbclid", "_ga", "portal_form_id".
//  6. Trailing slash: remove if path ends with "/" AND path is not "/".
//  7. If path contains "@@" or "++theme++", set excluded=true.
//  8. Path case is preserved (not lowercased).
//  9. Remaining query params are sorted alphabetically (url.Values.Encode behaviour).
func (c *Canonicalizer) Canonicalize(raw string) (canonical string, excluded bool, err error) {
	if raw == "" {
		return "", false, errors.New("canonical: empty URL")
	}

	u, parseErr := url.Parse(raw)
	if parseErr != nil {
		return "", false, fmt.Errorf("canonical: parse %q: %w", raw, parseErr)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false, fmt.Errorf("canonical: unsupported scheme %q", u.Scheme)
	}
	u.Scheme = scheme

	// Lowercase host, preserving port if present.
	host, port, _ := splitHostPort(u.Host)
	host = strings.ToLower(host)

	// Remove default ports.
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	if port != "" {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}

	// Strip fragment.
	u.Fragment = ""
	u.RawFragment = ""

	// Filter tracking query params; preserve remaining in alphabetical order.
	if u.RawQuery != "" {
		original := u.Query()
		filtered := url.Values{}
		for k, vs := range original {
			if strings.HasPrefix(k, "utm_") {
				continue
			}
			if _, bad := trackingKeys[k]; bad {
				continue
			}
			filtered[k] = vs
		}
		u.RawQuery = filtered.Encode() // Encode sorts keys alphabetically.
	}

	// Remove trailing slash unless path is exactly "/".
	if strings.HasSuffix(u.Path, "/") && u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	// Check Plone exclusion patterns.
	if strings.Contains(u.Path, "@@") || strings.Contains(u.Path, "++theme++") {
		excluded = true
	}

	return u.String(), excluded, nil
}

// IsPloneCopy returns true when the URL's path contains a segment that starts
// with "copy_of" or "copy<N>_of" (Plone duplicate marker). The match requires
// the pattern to appear at the beginning of a path segment (strict segment-start).
// Used by callers to set the is_plone_copy flag.
func IsPloneCopy(rawOrPath string) bool {
	// Extract just the path if a full URL is given.
	u, err := url.Parse(rawOrPath)
	if err == nil && u.Scheme != "" {
		return ploneCopyRe.MatchString(u.Path)
	}
	return ploneCopyRe.MatchString(rawOrPath)
}

// splitHostPort splits "host:port" into its components.
// It is tolerant of IPv6 addresses like "[::1]:8080".
// Returns ("host", "port", true) or ("host", "", false) when no port is present.
func splitHostPort(hostport string) (host, port string, hasPort bool) {
	if hostport == "" {
		return "", "", false
	}
	// IPv6 with port: "[::1]:8080"
	if hostport[0] == '[' {
		end := strings.LastIndex(hostport, "]")
		if end < 0 {
			return hostport, "", false
		}
		if end+1 < len(hostport) && hostport[end+1] == ':' {
			return hostport[:end+1], hostport[end+2:], true
		}
		return hostport, "", false
	}
	// Normal host or host:port.
	i := strings.LastIndex(hostport, ":")
	if i < 0 {
		return hostport, "", false
	}
	return hostport[:i], hostport[i+1:], true
}
