// Package linkheader parses RFC-5988 Link headers as used by the GitHub and
// GitLab REST APIs for pagination.
package linkheader

import "strings"

// Next returns the rel="next" URL from a Link header, or "" when there is
// no next page.
func Next(header string) string {
	for _, part := range strings.Split(header, ",") {
		urlPart, relPart, ok := strings.Cut(part, ";")
		if !ok {
			continue
		}
		url := strings.Trim(strings.TrimSpace(urlPart), "<>")
		if url == "" {
			continue
		}
		for _, attr := range strings.Split(relPart, ";") {
			k, v, ok := strings.Cut(strings.TrimSpace(attr), "=")
			if !ok {
				continue
			}
			if strings.TrimSpace(k) == "rel" && strings.Trim(strings.TrimSpace(v), `"`) == "next" {
				return url
			}
		}
	}
	return ""
}
