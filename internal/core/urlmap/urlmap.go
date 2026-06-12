// Package urlmap maps URLs to on-disk markdown paths (R5 in
// docs/acceptance.md): <host>/<path>.md, root → index.md, trailing slash
// stripped, query strings disambiguated by a short hash suffix.
package urlmap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// Map maps a URL to its relative path beneath urls/.
func Map(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid URL %q: only http(s) URLs can be fetched", raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL %q: missing host", raw)
	}

	// Trailing slash is stripped before mapping: /blog ≡ /blog/ (AC-URL-07).
	trimmed := strings.Trim(u.EscapedPath(), "/")

	var rel string
	if trimmed == "" {
		// Root is the one path with no filename to derive (AC-URL-02).
		rel = "index"
	} else {
		segments := strings.Split(trimmed, "/")
		for i, seg := range segments {
			segments[i] = sanitizeSegment(seg)
		}
		rel = strings.Join(segments, "/")
	}

	if u.RawQuery != "" {
		rel += "__" + queryHash(u.RawQuery)
	}
	return u.Host + "/" + rel + ".md", nil
}

// queryHash is the first 8 hex chars of SHA-256(query string): stable,
// deterministic, distinct per query (AC-URL-06).
func queryHash(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])[:8]
}

// sanitizeSegment percent-decodes one path segment and replaces unsafe
// characters with `_`. Dot-only segments (".", "..") are neutralized so a
// decoded path can never escape the target tree.
func sanitizeSegment(seg string) string {
	decoded, err := url.PathUnescape(seg)
	if err != nil {
		decoded = seg
	}
	if strings.Trim(decoded, ".") == "" {
		return "_"
	}
	var sb strings.Builder
	for _, r := range decoded {
		if isSafe(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// isSafe reports whether a rune may appear literally in a filename segment.
func isSafe(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == '.':
		return true
	}
	return false
}
